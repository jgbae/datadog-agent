// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

package activity_tree

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"golang.org/x/sys/unix"

	"github.com/DataDog/datadog-agent/pkg/security/resolvers/process"
	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/security/utils"
)

// NodeDroppedReason is used to list the reasons to drop a node
type NodeDroppedReason string

var (
	eventTypeReason       NodeDroppedReason = "event_type"
	invalidRootNodeReason NodeDroppedReason = "invalid_root_node"
	bindFamilyReason      NodeDroppedReason = "bind_family"
	brokenEventReason     NodeDroppedReason = "broken_event"
	allDropReasons                          = []NodeDroppedReason{
		eventTypeReason,
		invalidRootNodeReason,
		bindFamilyReason,
		brokenEventReason,
	}
)

// NodeGenerationType is used to indicate if a node was generated by a runtime or snapshot event
// IMPORTANT: IT MUST STAY IN SYNC WITH `adproto.GenerationType`
type NodeGenerationType byte

const (
	// Unknown is a node that was added at an unknown time
	Unknown NodeGenerationType = 0
	// Runtime is a node that was added at runtime
	Runtime NodeGenerationType = 1
	// Snapshot is a node that was added during the snapshot
	Snapshot NodeGenerationType = 2
	// ProfileDrift is a node that was added because of a drift from a security profile
	ProfileDrift NodeGenerationType = 3
	// WorkloadWarmup is a node that was added of a drift in a warming up profile
	WorkloadWarmup NodeGenerationType = 4
	// MaxNodeGenerationType is the maximum node type
	MaxNodeGenerationType NodeGenerationType = 4
)

func (genType NodeGenerationType) String() string {
	switch genType {
	case Runtime:
		return "runtime"
	case Snapshot:
		return "snapshot"
	case ProfileDrift:
		return "profile_drift"
	case WorkloadWarmup:
		return "workload_warmup"
	default:
		return "unknown"
	}
}

// ActivityTreeOwner is used to communicate with the owner of the activity tree
type ActivityTreeOwner interface {
	MatchesSelector(entry *model.ProcessCacheEntry) bool
	IsEventTypeValid(evtType model.EventType) bool
	NewProcessNodeCallback(p *ProcessNode)
}

// ActivityTree contains a process tree and its activities. This structure has no locks.
type ActivityTree struct {
	Stats *ActivityTreeStats

	treeType          string
	differentiateArgs bool

	validator ActivityTreeOwner

	CookieToProcessNode map[uint32]*ProcessNode `json:"-"`
	ProcessNodes        []*ProcessNode          `json:"-"`

	// top level lists used to summarize the content of the tree
	DNSNames     *utils.StringKeys
	SyscallsMask map[int]int
}

// NewActivityTree returns a new ActivityTree instance
func NewActivityTree(validator ActivityTreeOwner, treeType string) *ActivityTree {
	at := &ActivityTree{
		treeType:            treeType,
		validator:           validator,
		Stats:               NewActivityTreeNodeStats(),
		CookieToProcessNode: make(map[uint32]*ProcessNode),
		SyscallsMask:        make(map[int]int),
		DNSNames:            utils.NewStringKeys(nil),
	}
	return at
}

// ComputeSyscallsList computes the top level list of syscalls
func (at *ActivityTree) ComputeSyscallsList() []uint32 {
	output := make([]uint32, 0, len(at.SyscallsMask))
	for key := range at.SyscallsMask {
		output = append(output, uint32(key))
	}
	sort.Slice(output, func(i, j int) bool {
		return output[i] < output[j]
	})
	return output
}

// ComputeActivityTreeStats computes the initial counts of the activity tree stats
func (at *ActivityTree) ComputeActivityTreeStats() {
	pnodes := at.ProcessNodes
	var fnodes []*FileNode

	for len(pnodes) > 0 {
		node := pnodes[0]

		at.Stats.ProcessNodes += 1
		pnodes = append(pnodes, node.Children...)

		at.Stats.DNSNodes += int64(len(node.DNSNames))
		at.Stats.SocketNodes += int64(len(node.Sockets))

		for _, f := range node.Files {
			fnodes = append(fnodes, f)
		}

		pnodes = pnodes[1:]
	}

	for len(fnodes) > 0 {
		node := fnodes[0]

		if node.File != nil {
			at.Stats.FileNodes += 1
		}

		for _, f := range node.Children {
			fnodes = append(fnodes, f)
		}

		fnodes = fnodes[1:]
	}
}

// IsEmpty returns true if the tree is empty
func (at *ActivityTree) IsEmpty() bool {
	return len(at.ProcessNodes) == 0
}

// nolint: unused
func (at *ActivityTree) debug(w io.Writer) {
	for _, root := range at.ProcessNodes {
		root.debug(w, "")
	}
}

// ScrubProcessArgsEnvs scrubs and retains process args and envs
func (at *ActivityTree) ScrubProcessArgsEnvs(resolver *process.Resolver) {
	// iterate through all the process nodes
	openList := make([]*ProcessNode, len(at.ProcessNodes))
	copy(openList, at.ProcessNodes)

	for len(openList) != 0 {
		current := openList[len(openList)-1]
		current.scrubAndReleaseArgsEnvs(resolver)
		openList = append(openList[:len(openList)-1], current.Children...)
	}
}

// DifferentiateArgs enables the args differentiation feature
func (at *ActivityTree) DifferentiateArgs() {
	at.differentiateArgs = true
}

// isValidRootNode evaluates if the provided process entry is allowed to become a root node of an Activity Dump
func (at *ActivityTree) isValidRootNode(entry *model.ProcessContext) bool {
	// TODO: evaluate if the same issue affects other container runtimes
	return !(strings.HasPrefix(entry.FileEvent.BasenameStr, "runc") || strings.HasPrefix(entry.FileEvent.BasenameStr, "containerd-shim"))
}

// isEventValid evaluates if the provided event is valid
func (at *ActivityTree) isEventValid(event *model.Event, dryRun bool) (bool, error) {
	// check event type
	if !at.validator.IsEventTypeValid(event.GetEventType()) {
		if !dryRun {
			at.Stats.droppedCount[event.GetEventType()][eventTypeReason].Inc()
		}
		return false, fmt.Errorf("event type not valid: %s", event.GetEventType())
	}

	// event specific filtering
	switch event.GetEventType() {
	case model.BindEventType:
		// ignore non IPv4 / IPv6 bind events for now
		if event.Bind.AddrFamily != unix.AF_INET && event.Bind.AddrFamily != unix.AF_INET6 {
			if !dryRun {
				at.Stats.droppedCount[model.BindEventType][bindFamilyReason].Inc()
			}
			return false, fmt.Errorf("invalid bind family")
		}
	}
	return true, nil
}

// Insert inserts the event in the activity tree
func (at *ActivityTree) Insert(event *model.Event, generationType NodeGenerationType) (bool, error) {
	newEntry, err := at.insert(event, false, generationType)
	if newEntry {
		// this doesn't count the exec events which are counted separately
		at.Stats.addedCount[event.GetEventType()][generationType].Inc()
	}
	return newEntry, err
}

// Contains looks up the event in the activity tree
func (at *ActivityTree) Contains(event *model.Event, generationType NodeGenerationType) (bool, error) {
	newEntry, err := at.insert(event, true, generationType)
	return !newEntry, err
}

// insert inserts the event in the activity tree, returns true if the event generated a new entry in the tree
func (at *ActivityTree) insert(event *model.Event, dryRun bool, generationType NodeGenerationType) (bool, error) {
	// sanity check
	if generationType == Unknown || generationType > MaxNodeGenerationType {
		return false, fmt.Errorf("invalid generation type: %v", generationType)
	}

	// check if this event type is traced
	if valid, err := at.isEventValid(event, dryRun); !valid || err != nil {
		return false, fmt.Errorf("invalid event: %s", err)
	}

	node, newProcessNode, err := at.CreateProcessNode(event.ProcessCacheEntry, generationType, dryRun)
	if err != nil {
		return false, err
	}
	if newProcessNode && dryRun {
		return true, nil
	}
	if node == nil {
		// a process node couldn't be found or created for this event, ignore it
		return false, err
	}

	// resolve fields
	event.ResolveFieldsForAD()

	// ignore events with an error
	if event.Error != nil {
		at.Stats.droppedCount[event.GetEventType()][brokenEventReason].Inc()
		return false, event.Error
	}

	// the count of processed events is the count of events that matched the activity dump selector = the events for
	// which we successfully found a process activity node
	at.Stats.processedCount[event.GetEventType()].Inc()

	// insert the event based on its type
	switch event.GetEventType() {
	case model.ExecEventType:
		// tag the matched rules if any
		node.MatchedRules = model.AppendMatchedRule(node.MatchedRules, event.Rules)
		return newProcessNode, nil
	case model.FileOpenEventType:
		return node.InsertFileEvent(&event.Open.File, event, generationType, at.Stats, dryRun), nil
	case model.DNSEventType:
		return node.InsertDNSEvent(event, generationType, at.Stats, at.DNSNames, dryRun), nil
	case model.BindEventType:
		return node.InsertBindEvent(event, generationType, at.Stats, dryRun), nil
	case model.SyscallsEventType:
		return node.InsertSyscalls(event, at.SyscallsMask), nil
	}

	return false, nil
}

// CreateProcessNode finds or a create a new process activity node in the activity dump if the entry
// matches the activity dump selector.
func (at *ActivityTree) CreateProcessNode(entry *model.ProcessCacheEntry, generationType NodeGenerationType, dryRun bool) (node *ProcessNode, newProcessNode bool, err error) {
	if entry == nil {
		return nil, false, nil
	}

	// look for a ProcessActivityNode by process cookie
	if entry.Cookie > 0 {
		var found bool
		node, found = at.CookieToProcessNode[entry.Cookie]
		if found {
			return node, false, nil
		}
	}

	defer func() {
		// if a node was found, and if the entry has a valid cookie, insert a cookie shortcut
		if entry.Cookie > 0 && node != nil {
			at.CookieToProcessNode[entry.Cookie] = node
		}
	}()

	// find or create a ProcessActivityNode for the parent of the input ProcessCacheEntry. If the parent is a fork entry,
	// jump immediately to the next ancestor.
	parentNode, newProcessNode, err := at.CreateProcessNode(entry.GetNextAncestorBinary(), Snapshot, dryRun)
	if err != nil || (newProcessNode && dryRun) {
		// Explanation of (newProcessNode && dryRun): when dryRun is on, we can return as soon as we
		// see something new in the tree. Although `newProcessNode` and `err` seem to be tied (i.e. newProcessNode is
		// always false when err != nil), the important case is when err == nil, where newProcessNode can be either
		// true or false.
		return parentNode, newProcessNode, err
	}

	// if parentNode is nil, the parent of the current node is out of tree (either because the parent is null, or it
	// doesn't match the dump tags).
	if parentNode == nil {

		// since the parent of the current entry wasn't inserted, we need to know if the current entry needs to be inserted.
		if !at.validator.MatchesSelector(entry) {
			return nil, false, nil
		}

		// go through the root nodes and check if one of them matches the input ProcessCacheEntry:
		for _, root := range at.ProcessNodes {
			if root.Matches(&entry.Process, at.differentiateArgs) {
				return root, false, nil
			}
		}

		// we're about to add a root process node, make sure this root node passes the root node sanitizer
		if !at.isValidRootNode(&entry.ProcessContext) {
			if !dryRun {
				at.Stats.droppedCount[model.ExecEventType][invalidRootNodeReason].Inc()
			}
			return nil, false, fmt.Errorf("invalid root node")
		}

		// if it doesn't, create a new ProcessActivityNode for the input ProcessCacheEntry
		if !dryRun {
			node = NewProcessNode(entry, generationType)
			// insert in the list of root entries
			at.ProcessNodes = append(at.ProcessNodes, node)
			at.Stats.ProcessNodes++
		}

	} else {

		// if parentNode wasn't nil, then (at least) the parent is part of the activity dump. This means that we need
		// to add the current entry no matter if it matches the selector or not. Go through the root children of the
		// parent node and check if one of them matches the input ProcessCacheEntry.
		for _, child := range parentNode.Children {
			if child.Matches(&entry.Process, at.differentiateArgs) {
				return child, false, nil
			}
		}

		// if none of them matched, create a new ProcessActivityNode for the input processCacheEntry
		if !dryRun {
			node = NewProcessNode(entry, generationType)
			// insert in the list of children
			parentNode.Children = append(parentNode.Children, node)
			at.Stats.ProcessNodes++
		}
	}

	// count new entry
	if !dryRun {
		at.Stats.addedCount[model.ExecEventType][generationType].Inc()
		// propagate the entry matching process cache entry
		at.validator.NewProcessNodeCallback(node)
	}

	return node, true, nil
}

func (at *ActivityTree) FindMatchingRootNodes(basename string) []*ProcessNode {
	var res []*ProcessNode
	for _, node := range at.ProcessNodes {
		if node.Process.FileEvent.BasenameStr == basename {
			res = append(res, node)
		}
	}
	return res
}

// Snapshot uses procfs to snapshot the nodes of the tree
func (at *ActivityTree) Snapshot(newEvent func() *model.Event) error {
	for _, pn := range at.ProcessNodes {
		if err := pn.snapshot(at.validator, at.Stats, newEvent); err != nil {
			return err
		}
		// iterate slowly
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

// SendStats sends the tree statistics
func (at *ActivityTree) SendStats(client statsd.ClientInterface) error {
	return at.Stats.SendStats(client, at.treeType)
}
