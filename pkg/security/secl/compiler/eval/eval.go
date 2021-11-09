// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:generate go run github.com/DataDog/datadog-agent/pkg/security/secl/compiler/generators/operators -output eval_operators.go

package eval

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/alecthomas/participle/lexer"
	"github.com/pkg/errors"

	"github.com/DataDog/datadog-agent/pkg/security/secl/compiler/ast"
)

// defines factor applied by specific operator
const (
	FunctionWeight       = 5
	InArrayWeight        = 10
	HandlerWeight        = 50
	RegexpWeight         = 100
	InPatternArrayWeight = 1000
	IteratorWeight       = 2000
)

// Opts are the options to be passed to the evaluator
type Opts struct {
	LegacyAttributes map[Field]Field
	Constants        map[string]interface{}
	Macros           map[MacroID]*Macro
}

// OpOverride defines a operator override function suite
type OpOverrides struct {
	StringEquals func(a *StringEvaluator, b *StringEvaluator, opts *Opts, state *State) (*BoolEvaluator, error)
}

// BoolEvalFnc describe a eval function return a boolean
type BoolEvalFnc = func(ctx *Context) bool

func extractField(field string) (Field, Field, RegisterID, error) {
	var regID RegisterID

	re := regexp.MustCompile(`\[([^\]]*)\]`)
	ids := re.FindStringSubmatch(field)

	switch len(ids) {
	case 0:
		return field, "", "", nil
	case 2:
		regID = ids[1]
	default:
		return "", "", "", fmt.Errorf("wrong register format for fields: %s", field)
	}

	re = regexp.MustCompile(`(.+)\[[^\]]+\](.*)`)
	field, itField := re.ReplaceAllString(field, `$1$2`), re.ReplaceAllString(field, `$1`)

	return field, itField, regID, nil
}

type ident struct {
	Pos   lexer.Position
	Ident *string
}

func identToEvaluator(obj *ident, opts *Opts, state *State) (interface{}, lexer.Position, error) {
	if accessor, ok := opts.Constants[*obj.Ident]; ok {
		return accessor, obj.Pos, nil
	}

	if state.macros != nil {
		if macro, ok := state.macros[*obj.Ident]; ok {
			return macro.Value, obj.Pos, nil
		}
	}

	field, itField, regID, err := extractField(*obj.Ident)
	if err != nil {
		return nil, obj.Pos, err
	}

	// transform extracted field to support legacy SECL attributes
	if opts.LegacyAttributes != nil {
		if newField, ok := opts.LegacyAttributes[field]; ok {
			field = newField
		}
		if newField, ok := opts.LegacyAttributes[field]; ok {
			itField = newField
		}
	}

	// extract iterator
	var iterator Iterator
	if itField != "" {
		if iterator, err = state.model.GetIterator(itField); err != nil {
			return nil, obj.Pos, err
		}
	} else {
		// detect whether a iterator is along the path
		var candidate string
		for _, node := range strings.Split(field, ".") {
			if candidate == "" {
				candidate = node
			} else {
				candidate = candidate + "." + node
			}

			iterator, err = state.model.GetIterator(candidate)
			if err == nil {
				break
			}
		}
	}

	if iterator != nil {
		// Force "_" register for now.
		if regID != "" && regID != "_" {
			return nil, obj.Pos, NewRegisterNameNotAllowed(obj.Pos, regID, errors.New("only `_` is supported"))
		}

		// regID not specified or `_` generate one
		if regID == "" || regID == "_" {
			regID = RandString(8)
		}

		if info, exists := state.registersInfo[regID]; exists {
			if info.field != itField {
				return nil, obj.Pos, NewRegisterMultipleFields(obj.Pos, regID, errors.New("used by multiple fields"))
			}

			info.subFields[field] = true
		} else {
			info = &registerInfo{
				field:    itField,
				iterator: iterator,
				subFields: map[Field]bool{
					field: true,
				},
			}
			state.registersInfo[regID] = info
		}
	}

	accessor, err := state.model.GetEvaluator(field, regID)
	if err != nil {
		return nil, obj.Pos, err
	}

	state.UpdateFields(field)

	return accessor, obj.Pos, nil
}

func arrayToEvaluator(array *ast.Array, opts *Opts, state *State) (interface{}, lexer.Position, error) {
	if len(array.Numbers) != 0 {
		var evaluator IntArrayEvaluator
		evaluator.AppendValues(array.Numbers...)
		return &evaluator, array.Pos, nil
	} else if len(array.StringMembers) != 0 {
		var evaluator StringValuesEvaluator
		if err := evaluator.AppendMembers(array.StringMembers...); err != nil {
			return nil, array.Pos, NewError(array.Pos, err.Error())
		}
		return &evaluator, array.Pos, nil
	} else if array.Ident != nil {
		if state.macros != nil {
			if macro, ok := state.macros[*array.Ident]; ok {
				return macro.Value, array.Pos, nil
			}
		}

		// could be an iterator
		return identToEvaluator(&ident{Pos: array.Pos, Ident: array.Ident}, opts, state)
	}

	return nil, array.Pos, NewError(array.Pos, "unknow array element type")
}

func nodeToEvaluator(obj interface{}, opts *Opts, state *State) (interface{}, lexer.Position, error) {
	var err error
	var boolEvaluator *BoolEvaluator
	var pos lexer.Position
	var cmp, unary, next interface{}

	switch obj := obj.(type) {
	case *ast.BooleanExpression:
		return nodeToEvaluator(obj.Expression, opts, state)
	case *ast.Expression:
		cmp, pos, err = nodeToEvaluator(obj.Comparison, opts, state)
		if err != nil {
			return nil, pos, err
		}

		if obj.Op != nil {
			cmpBool, ok := cmp.(*BoolEvaluator)
			if !ok {
				return nil, obj.Pos, NewTypeError(obj.Pos, reflect.Bool)
			}

			next, pos, err = nodeToEvaluator(obj.Next, opts, state)
			if err != nil {
				return nil, pos, err
			}

			nextBool, ok := next.(*BoolEvaluator)
			if !ok {
				return nil, pos, NewTypeError(pos, reflect.Bool)
			}

			switch *obj.Op {
			case "||", "or":
				boolEvaluator, err = Or(cmpBool, nextBool, opts, state)
				if err != nil {
					return nil, obj.Pos, err
				}
				return boolEvaluator, obj.Pos, nil
			case "&&", "and":
				boolEvaluator, err = And(cmpBool, nextBool, opts, state)
				if err != nil {
					return nil, obj.Pos, err
				}
				return boolEvaluator, obj.Pos, nil
			}
			return nil, pos, NewOpUnknownError(obj.Pos, *obj.Op)
		}
		return cmp, obj.Pos, nil
	case *ast.BitOperation:
		unary, pos, err = nodeToEvaluator(obj.Unary, opts, state)
		if err != nil {
			return nil, pos, err
		}

		if obj.Op != nil {
			bitInt, ok := unary.(*IntEvaluator)
			if !ok {
				return nil, obj.Pos, NewTypeError(obj.Pos, reflect.Int)
			}

			next, pos, err = nodeToEvaluator(obj.Next, opts, state)
			if err != nil {
				return nil, pos, err
			}

			nextInt, ok := next.(*IntEvaluator)
			if !ok {
				return nil, pos, NewTypeError(pos, reflect.Int)
			}

			switch *obj.Op {
			case "&":
				intEvaluator, err := IntAnd(bitInt, nextInt, opts, state)
				if err != nil {
					return nil, pos, err
				}
				return intEvaluator, obj.Pos, nil
			case "|":
				IntEvaluator, err := IntOr(bitInt, nextInt, opts, state)
				if err != nil {
					return nil, pos, err
				}
				return IntEvaluator, obj.Pos, nil
			case "^":
				IntEvaluator, err := IntXor(bitInt, nextInt, opts, state)
				if err != nil {
					return nil, pos, err
				}
				return IntEvaluator, obj.Pos, nil
			}
			return nil, pos, NewOpUnknownError(obj.Pos, *obj.Op)
		}
		return unary, obj.Pos, nil

	case *ast.Comparison:
		unary, pos, err = nodeToEvaluator(obj.BitOperation, opts, state)
		if err != nil {
			return nil, pos, err
		}

		if obj.ArrayComparison != nil {
			next, pos, err = nodeToEvaluator(obj.ArrayComparison, opts, state)
			if err != nil {
				return nil, pos, err
			}

			switch unary := unary.(type) {
			case *BoolEvaluator:
				switch nextBool := next.(type) {
				case *BoolArrayEvaluator:
					boolEvaluator, err = ArrayBoolContains(unary, nextBool, opts, state)
					if err != nil {
						return nil, pos, err
					}
					if *obj.ArrayComparison.Op == "notin" {
						return Not(boolEvaluator, opts, state), obj.Pos, nil
					}
					return boolEvaluator, obj.Pos, nil
				default:
					return nil, pos, NewArrayTypeError(pos, reflect.Array, reflect.Bool)
				}
			case *StringEvaluator:
				switch nextString := next.(type) {
				case *StringArrayEvaluator:
					boolEvaluator, err = StringArrayContains(unary, nextString, opts, state)
					if err != nil {
						return nil, pos, err
					}
				case *StringValuesEvaluator:
					boolEvaluator, err = StringValuesContains(unary, nextString, opts, state)
					if err != nil {
						return nil, pos, err
					}
				default:
					fmt.Printf("AAAAA\n")
					return nil, pos, NewArrayTypeError(pos, reflect.Array, reflect.String)
				}
				if *obj.ArrayComparison.Op == "notin" {
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				}
				return boolEvaluator, obj.Pos, nil
			case *StringValuesEvaluator:
				switch nextStringArray := next.(type) {
				case *StringArrayEvaluator:
					boolEvaluator, err = StringArrayMatches(nextStringArray, unary, opts, state)
					if err != nil {
						return nil, pos, err
					}
					if *obj.ArrayComparison.Op == "notin" {
						return Not(boolEvaluator, opts, state), obj.Pos, nil
					}
					return boolEvaluator, obj.Pos, nil
				default:
					return nil, pos, NewArrayTypeError(pos, reflect.Array, reflect.String)
				}
			case *IntEvaluator:
				switch nextInt := next.(type) {
				case *IntArrayEvaluator:
					boolEvaluator, err = IntArrayEquals(unary, nextInt, opts, state)
					if err != nil {
						return nil, pos, err
					}
					if *obj.ArrayComparison.Op == "notin" {
						return Not(boolEvaluator, opts, state), obj.Pos, nil
					}
					return boolEvaluator, obj.Pos, nil
				default:
					return nil, pos, NewArrayTypeError(pos, reflect.Array, reflect.Int)
				}
			case *IntArrayEvaluator:
				switch nextIntArray := next.(type) {
				case *IntArrayEvaluator:
					boolEvaluator, err = IntArrayMatches(unary, nextIntArray, opts, state)
					if err != nil {
						return nil, pos, err
					}
					if *obj.ArrayComparison.Op == "notin" {
						return Not(boolEvaluator, opts, state), obj.Pos, nil
					}
					return boolEvaluator, obj.Pos, nil
				default:
					return nil, pos, NewArrayTypeError(pos, reflect.Array, reflect.Int)
				}
			default:
				return nil, pos, NewTypeError(pos, reflect.Array)
			}
		} else if obj.ScalarComparison != nil {
			next, pos, err = nodeToEvaluator(obj.ScalarComparison, opts, state)
			if err != nil {
				return nil, pos, err
			}

			switch unary := unary.(type) {
			case *BoolEvaluator:
				nextBool, ok := next.(*BoolEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.Bool)
				}

				switch *obj.ScalarComparison.Op {
				case "!=":
					boolEvaluator, err = BoolEquals(unary, nextBool, opts, state)
					if err != nil {
						return nil, pos, err
					}
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				case "==":
					boolEvaluator, err = BoolEquals(unary, nextBool, opts, state)
					if err != nil {
						return nil, pos, err
					}
					return boolEvaluator, obj.Pos, nil
				}
				return nil, pos, NewOpUnknownError(obj.Pos, *obj.ScalarComparison.Op)
			case *BoolArrayEvaluator:
				nextBool, ok := next.(*BoolEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.Bool)
				}

				switch *obj.ScalarComparison.Op {
				case "!=":
					boolEvaluator, err = ArrayBoolEquals(nextBool, unary, opts, state)
					if err != nil {
						return nil, pos, err
					}
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				case "==":
					boolEvaluator, err = ArrayBoolEquals(nextBool, unary, opts, state)
					if err != nil {
						return nil, pos, err
					}
					return boolEvaluator, obj.Pos, nil
				}
				return nil, pos, NewOpUnknownError(obj.Pos, *obj.ScalarComparison.Op)
			case *StringEvaluator:
				nextString, ok := next.(*StringEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.String)
				}

				switch *obj.ScalarComparison.Op {
				case "!=":
					boolEvaluator, err = StringEquals(unary, nextString, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				case "!~":
					if nextString.EvalFnc != nil {
						return nil, obj.Pos, &ErrNonStaticPattern{Field: nextString.Field}
					}

					if err := compilePattern(nextString); err != nil {
						return nil, obj.Pos, NewError(obj.Pos, err.Error())
					}

					boolEvaluator, err = StringEquals(unary, nextString, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				case "==":
					if unary.OpOverrides != nil && unary.OpOverrides.StringEquals != nil {
						boolEvaluator, err = unary.OpOverrides.StringEquals(unary, nextString, opts, state)
					} else if nextString.OpOverrides != nil && nextString.OpOverrides.StringEquals != nil {
						boolEvaluator, err = nextString.OpOverrides.StringEquals(unary, nextString, opts, state)
					} else {
						boolEvaluator, err = StringEquals(unary, nextString, opts, state)
					}
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				case "=~":
					if nextString.EvalFnc != nil {
						return nil, obj.Pos, &ErrNonStaticPattern{Field: nextString.Field}
					}

					if err := compilePattern(nextString); err != nil {
						return nil, obj.Pos, NewError(obj.Pos, err.Error())
					}

					boolEvaluator, err = StringEquals(unary, nextString, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				}
				return nil, pos, NewOpUnknownError(obj.Pos, *obj.ScalarComparison.Op)
			case *StringArrayEvaluator:
				nextString, ok := next.(*StringEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.String)
				}

				switch *obj.ScalarComparison.Op {
				case "!=":
					boolEvaluator, err = StringArrayContains(nextString, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				case "==":
					boolEvaluator, err = StringArrayContains(nextString, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				case "!~":
					if nextString.EvalFnc != nil {
						return nil, obj.Pos, &ErrNonStaticPattern{Field: nextString.Field}
					}

					if err := compilePattern(nextString); err != nil {
						return nil, obj.Pos, NewError(obj.Pos, err.Error())
					}

					boolEvaluator, err = StringArrayContains(nextString, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				case "=~":
					if nextString.EvalFnc != nil {
						return nil, obj.Pos, &ErrNonStaticPattern{Field: nextString.Field}
					}

					if err := compilePattern(nextString); err != nil {
						return nil, obj.Pos, NewError(obj.Pos, err.Error())
					}

					boolEvaluator, err = StringArrayContains(nextString, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				}
			case *IntEvaluator:
				switch nextInt := next.(type) {
				case *IntEvaluator:
					if nextInt.isDuration {
						switch *obj.ScalarComparison.Op {
						case "<":
							boolEvaluator, err = DurationLesserThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						case "<=":
							boolEvaluator, err = DurationLesserOrEqualThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						case ">":
							boolEvaluator, err = DurationGreaterThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						case ">=":
							boolEvaluator, err = DurationGreaterOrEqualThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						}
					} else {
						switch *obj.ScalarComparison.Op {
						case "<":
							boolEvaluator, err = LesserThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						case "<=":
							boolEvaluator, err = LesserOrEqualThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						case ">":
							boolEvaluator, err = GreaterThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						case ">=":
							boolEvaluator, err = GreaterOrEqualThan(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						case "!=":
							boolEvaluator, err = IntEquals(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}

							return Not(boolEvaluator, opts, state), obj.Pos, nil
						case "==":
							boolEvaluator, err = IntEquals(unary, nextInt, opts, state)
							if err != nil {
								return nil, obj.Pos, err
							}
							return boolEvaluator, obj.Pos, nil
						default:
							return nil, pos, NewOpUnknownError(obj.Pos, *obj.ScalarComparison.Op)
						}
					}
				case *IntArrayEvaluator:
					nextIntArray := next.(*IntArrayEvaluator)

					switch *obj.ScalarComparison.Op {
					case "<":
						boolEvaluator, err = IntArrayLesserThan(unary, nextIntArray, opts, state)
						if err != nil {
							return nil, obj.Pos, err
						}
						return boolEvaluator, obj.Pos, nil
					case "<=":
						boolEvaluator, err = IntArrayLesserOrEqualThan(unary, nextIntArray, opts, state)
						if err != nil {
							return nil, obj.Pos, err
						}
						return boolEvaluator, obj.Pos, nil
					case ">":
						boolEvaluator, err = IntArrayGreaterThan(unary, nextIntArray, opts, state)
						if err != nil {
							return nil, obj.Pos, err
						}
						return boolEvaluator, obj.Pos, nil
					case ">=":
						boolEvaluator, err = IntArrayGreaterOrEqualThan(unary, nextIntArray, opts, state)
						if err != nil {
							return nil, obj.Pos, err
						}
						return boolEvaluator, obj.Pos, nil
					case "!=":
						boolEvaluator, err = IntArrayEquals(unary, nextIntArray, opts, state)
						if err != nil {
							return nil, obj.Pos, err
						}
						return Not(boolEvaluator, opts, state), obj.Pos, nil
					case "==":
						boolEvaluator, err = IntArrayEquals(unary, nextIntArray, opts, state)
						if err != nil {
							return nil, obj.Pos, err
						}
						return boolEvaluator, obj.Pos, nil
					default:
						return nil, pos, NewOpUnknownError(obj.Pos, *obj.ScalarComparison.Op)
					}
				}
				return nil, pos, NewTypeError(pos, reflect.Int)
			case *IntArrayEvaluator:
				nextInt, ok := next.(*IntEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.Int)
				}

				switch *obj.ScalarComparison.Op {
				case "<":
					boolEvaluator, err = IntArrayGreaterThan(nextInt, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				case "<=":
					boolEvaluator, err = IntArrayGreaterOrEqualThan(nextInt, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				case ">":
					boolEvaluator, err = IntArrayLesserThan(nextInt, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				case ">=":
					boolEvaluator, err = IntArrayLesserOrEqualThan(nextInt, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				case "!=":
					boolEvaluator, err = IntArrayEquals(nextInt, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return Not(boolEvaluator, opts, state), obj.Pos, nil
				case "==":
					boolEvaluator, err = IntArrayEquals(nextInt, unary, opts, state)
					if err != nil {
						return nil, obj.Pos, err
					}
					return boolEvaluator, obj.Pos, nil
				}
				return nil, pos, NewOpUnknownError(obj.Pos, *obj.ScalarComparison.Op)
			}
		} else {
			return unary, pos, nil
		}

	case *ast.ArrayComparison:
		return nodeToEvaluator(obj.Array, opts, state)

	case *ast.ScalarComparison:
		return nodeToEvaluator(obj.Next, opts, state)

	case *ast.Unary:
		if obj.Op != nil {
			unary, pos, err = nodeToEvaluator(obj.Unary, opts, state)
			if err != nil {
				return nil, pos, err
			}

			switch *obj.Op {
			case "!", "not":
				unaryBool, ok := unary.(*BoolEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.Bool)
				}

				return Not(unaryBool, opts, state), obj.Pos, nil
			case "-":
				unaryInt, ok := unary.(*IntEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.Int)
				}

				return Minus(unaryInt, opts, state), pos, nil
			case "^":
				unaryInt, ok := unary.(*IntEvaluator)
				if !ok {
					return nil, pos, NewTypeError(pos, reflect.Int)
				}

				return IntNot(unaryInt, opts, state), pos, nil
			}
			return nil, pos, NewOpUnknownError(obj.Pos, *obj.Op)
		}

		return nodeToEvaluator(obj.Primary, opts, state)
	case *ast.Primary:
		switch {
		case obj.Ident != nil:
			return identToEvaluator(&ident{Pos: obj.Pos, Ident: obj.Ident}, opts, state)
		case obj.Number != nil:
			return &IntEvaluator{
				Value: *obj.Number,
			}, obj.Pos, nil
		case obj.Duration != nil:
			return &IntEvaluator{
				Value:      *obj.Duration,
				isDuration: true,
			}, obj.Pos, nil
		case obj.String != nil:
			return &StringEvaluator{
				Value:     *obj.String,
				ValueType: ScalarValueType,
			}, obj.Pos, nil
		case obj.Pattern != nil:
			evaluator := &StringEvaluator{
				Value:     *obj.Pattern,
				ValueType: PatternValueType,
			}
			if err := evaluator.Compile(); err != nil {
				return nil, obj.Pos, NewError(obj.Pos, err.Error())
			}
			return evaluator, obj.Pos, nil
		case obj.Regexp != nil:
			evaluator := &StringEvaluator{
				Value:     *obj.Regexp,
				ValueType: RegexpValueType,
			}
			if err := evaluator.Compile(); err != nil {
				return nil, obj.Pos, NewError(obj.Pos, err.Error())
			}
			return evaluator, obj.Pos, nil
		case obj.SubExpression != nil:
			return nodeToEvaluator(obj.SubExpression, opts, state)
		default:
			return nil, obj.Pos, NewError(obj.Pos, fmt.Sprintf("unknown primary '%s'", reflect.TypeOf(obj)))
		}
	case *ast.Array:
		return arrayToEvaluator(obj, opts, state)
	}

	return nil, lexer.Position{}, NewError(lexer.Position{}, fmt.Sprintf("unknown entity '%s'", reflect.TypeOf(obj)))
}
