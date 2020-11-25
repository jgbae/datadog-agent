package snmp

import (
	"github.com/soniah/gosnmp"
	"strings"
)

func getValueFromPDU(pduVariable gosnmp.SnmpPDU) (string, snmpValue) {
	var value interface{}
	name := strings.TrimLeft(pduVariable.Name, ".") // remove leading dot
	switch pduVariable.Type {
	case gosnmp.OctetString:
		// TODO: Return custom val struct
		value = string(pduVariable.Value.([]byte))
	default:
		value = float64(gosnmp.ToBigInt(pduVariable.Value).Int64())
	}
	valueType := gosnmpTypeToSimpleType(pduVariable.Type)
	return name, snmpValue{valType: valueType, val: value}
}

func resultToScalarValues(result *gosnmp.SnmpPacket) (values map[string]snmpValue) {
	// TODO: test me
	returnValues := make(map[string]snmpValue)
	for _, pduVariable := range result.Variables {
		name, value := getValueFromPDU(pduVariable)
		returnValues[name] = value
	}
	return returnValues
}

func resultToColumnValues(oids []string, result *gosnmp.SnmpPacket) (map[string]map[string]snmpValue, map[string]string) {
	// TODO: test me
	returnValues := make(map[string]map[string]snmpValue)
	nextOidsMap := make(map[string]string)
	for i, pduVariable := range result.Variables {
		name, value := getValueFromPDU(pduVariable)
		oid := oids[i%len(oids)]
		if _, ok := returnValues[oid]; !ok {
			returnValues[oids[i]] = make(map[string]snmpValue)
		}
		prefix := oid + "."
		if strings.HasPrefix(name, prefix) {
			returnValues[oid][name[len(prefix):]] = value
			nextOidsMap[oid] = name
		} else {
			delete(nextOidsMap, oid)
		}
	}
	return returnValues, nextOidsMap
}

func gosnmpTypeToSimpleType(gosnmpType gosnmp.Asn1BER) valueType {
	switch gosnmpType {
	case gosnmp.Counter32, gosnmp.Counter64:
		// TODO: ZeroBasedCounter64
		//   https://github.com/DataDog/integrations-core/blob/d6add1dfcd99c3610f45390b8d4cd97390af1f69/snmp/datadog_checks/snmp/pysnmp_inspect.py#L37-L38
		return Counter
	}
	return Other
}
