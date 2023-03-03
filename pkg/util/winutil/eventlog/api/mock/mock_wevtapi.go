// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.
//go:build windows
// +build windows

package mockevtapi

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/DataDog/datadog-agent/pkg/util/winutil/eventlog/api"

	"go.uber.org/atomic"
	"golang.org/x/sys/windows"
)

type API struct {
	eventLogs map[string]*mockEventLog

	nextHandle *atomic.Uint64
	subscriptions map[evtapi.EventResultSetHandle]*mockSubscription
	eventHandles map[evtapi.EventRecordHandle]*mockEventRecord
	sourceHandles map[evtapi.EventSourceHandle]string
}

type mockEventLog struct {
	name string
	events []*mockEventRecord

	// For notifying of new events
	subscriptions map[evtapi.EventResultSetHandle]*mockSubscription
}

type mockSubscription struct {
	channel string
	query string
	handle evtapi.EventResultSetHandle
	// owned by caller, not closed by this lib
	signalEventHandle evtapi.WaitEventHandle

	nextEvent uint
}

type mockEventRecord struct {
	handle evtapi.EventRecordHandle

	// Must be exported so template can render them
	Type uint
	Category uint
	EventID uint
	Strings []string
	RawData []uint8
	EventLog string
}

func New() *API {
	var api API

	api.nextHandle = atomic.NewUint64(0)

	api.subscriptions = make(map[evtapi.EventResultSetHandle]*mockSubscription)
	api.eventHandles = make(map[evtapi.EventRecordHandle]*mockEventRecord)
	api.sourceHandles = make(map[evtapi.EventSourceHandle]string)

	api.eventLogs = make(map[string]*mockEventLog)

	return &api
}

func newMockEventLog(name string) *mockEventLog {
	var e mockEventLog
	e.name = name
	e.subscriptions = make(map[evtapi.EventResultSetHandle]*mockSubscription)
	return &e
}

func newMockSubscription(channel string, query string) *mockSubscription {
	var s mockSubscription
	s.channel = channel
	s.query = query
	return &s
}

func newMockEventRecord(Type uint, category uint, eventID uint, eventLog string, strings []string, data []uint8) *mockEventRecord {
	var e mockEventRecord
	e.Type = Type
	e.Category = category
	e.EventID = eventID
	e.Strings = strings
	e.RawData = data
	e.EventLog = eventLog
	return &e
}

//
// Mock helpers
//
func (api *API) AddEventLog(name string) error {
	// does it exist
	_, err := api.getMockEventLog(name)
	if err == nil {
		return fmt.Errorf("Event log %v already exists", name)
	}

	api.addMockEventLog(newMockEventLog(name))
	return nil
}

func (api *API) RemoveEventLog(name string) error {
	// Get event log
	_, err := api.getMockEventLog(name)
	if err != nil {
		return err
	}
	delete(api.eventLogs, name)
	return nil
}

func (api *API) GenerateEvents(eventLogName string, numEvents uint) error {
	// Get event log
	eventLog, err := api.getMockEventLog(eventLogName)
	if err != nil {
		return err
	}

	// Add junk events
	for i := uint(0); i < numEvents; i+=1 {
		event := eventLog.reportEvent(api, windows.EVENTLOG_INFORMATION_TYPE,
			0, 1000, []string{"teststring"}, nil)
		// TODO: Should only create a handle in the API when EvtNext is called
		api.addEventRecord(event)
	}

	return nil
}

//
// internal mock functions
//
func (api *API) addSubscription(sub *mockSubscription) {
	h := api.nextHandle.Inc()
	sub.handle = evtapi.EventResultSetHandle(h)
	api.subscriptions[sub.handle] = sub
}

func (api *API) addEventRecord(event *mockEventRecord) {
	h := api.nextHandle.Inc()
	event.handle = evtapi.EventRecordHandle(h)
	api.eventHandles[event.handle] = event
}

func (api *API) getMockSubscriptionByHandle(subHandle evtapi.EventResultSetHandle) (*mockSubscription, error) {
	v, ok := api.subscriptions[subHandle]
	if !ok {
		return nil, fmt.Errorf("Subscription not found: %#x", subHandle)
	}
	return v, nil
}

func (api *API) getMockEventRecordByHandle(eventHandle evtapi.EventRecordHandle) (*mockEventRecord, error) {
	v, ok := api.eventHandles[eventHandle]
	if !ok {
		return nil, fmt.Errorf("Event not found: %#x", eventHandle)
	}
	return v, nil
}

func (api *API) getMockEventLog(name string) (*mockEventLog, error) {
	v, ok := api.eventLogs[name]
	if !ok {
		return nil, fmt.Errorf("The Log name \"%v\" does not exist", name)
	}
	return v, nil
}

func (api *API) getMockEventLogByHandle(sourceHandle evtapi.EventSourceHandle) (*mockEventLog, error) {
	// lookup name using handle
	v, ok := api.sourceHandles[sourceHandle]
	if !ok {
		return nil, fmt.Errorf("Invalid source handle: %#x", sourceHandle)
	}

	return api.getMockEventLog(v)
}

func (api *API) addMockEventLog(eventLog *mockEventLog) {
	api.eventLogs[eventLog.name] = eventLog
}

func (e *mockEventLog) addEventRecord(event *mockEventRecord) {
	e.events = append(e.events, event)
}

func (e *mockEventLog) reportEvent(
	api *API,
	Type uint,
	Category uint,
	EventID uint,
	Strings []string,
	RawData []uint8) *mockEventRecord {

	event := newMockEventRecord(
		Type,
		Category,
		EventID,
		e.name,
		Strings,
		RawData)
	e.addEventRecord(event)

	// notify subscriptions
	for _, sub := range e.subscriptions {
		windows.SetEvent(windows.Handle(sub.signalEventHandle))
	}
	return event
}

//
// Mock Windows APIs
//
func (api *API) EvtSubscribe(
	SignalEvent evtapi.WaitEventHandle,
	ChannelPath string,
	Query string,
	Bookmark evtapi.EventBookmarkHandle,
	Flags uint) (evtapi.EventResultSetHandle, error) {

	if Query != "" && Query != "*" {
		return evtapi.EventResultSetHandle(0), fmt.Errorf("Mock does not support query syntax")
	}

	// ensure channel exists
	evtlog, err := api.getMockEventLog(ChannelPath)
	if err != nil {
		return evtapi.EventResultSetHandle(0), err
	}

	// create sub
	sub := newMockSubscription(ChannelPath, Query)
	sub.signalEventHandle = SignalEvent
	api.addSubscription(sub)
	evtlog.subscriptions[sub.handle] = sub
	return sub.handle, nil
}

func (api *API) EvtNext(
	Session evtapi.EventResultSetHandle,
	EventsArray []evtapi.EventRecordHandle,
	EventsSize uint,
	Timeout uint) ([]evtapi.EventRecordHandle, error) {

	// get subscription
	sub, err := api.getMockSubscriptionByHandle(Session)
	if err != nil {
		return nil, err
	}

	// get event log
	eventLog, err := api.getMockEventLog(sub.channel)
	if err != nil {
		return nil, err
	}

	// is event log empty
	if len(eventLog.events) == 0 {
		return nil, windows.ERROR_NO_MORE_ITEMS
	}
	// if we are at end
	if sub.nextEvent >= uint(len(eventLog.events)) {
		return nil, windows.ERROR_NO_MORE_ITEMS
	}

	// get next events from log
	end := sub.nextEvent+EventsSize
	if end > uint(len(eventLog.events)) {
		end = uint(len(eventLog.events))
	}
	events := eventLog.events[sub.nextEvent:end]
	eventHandles := make([]evtapi.EventRecordHandle, len(events))
	for i, e := range events {
		eventHandles[i] = e.handle
	}
	sub.nextEvent = end

	return eventHandles, nil
}

func (api *API) EvtClose(h windows.Handle) {
	// is handle an event?
	// TODO
	// Is handle a subscription?
	sub, err := api.getMockSubscriptionByHandle(evtapi.EventResultSetHandle(h))
	if err == nil {
		eventLog, err := api.getMockEventLog(sub.channel)
		if err != nil {
			return
		}
		delete(eventLog.subscriptions, sub.handle)
		return
	}
}

// EvtRenderEventXmlText renders EvtRenderEventXml
// https://learn.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtrender
func (api *API) EvtRenderEventXml(Fragment evtapi.EventRecordHandle) ([]uint16, error) {
	// get event object
	event, err := api.getMockEventRecordByHandle(Fragment)
	if err != nil {
		return nil, err
	}

	// Format event
	tstr := `<Event xmlns="http://scemas.microsoft.com/win/2004/08/events/event">
  <System>
	<EventID>{{ .EventID }}</EventID>
	<Channel>{{ .EventLog }}</Channel>
  </System>
  <EventData>
    <Data>{{ .Data }}</Data>
  </EventData>
</Event>
`
	t, err := template.New("eventRenderXML").Parse(tstr)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)

	err = t.Execute(buf, event)
	if err != nil {
		return nil, err
	}

	// convert from utf-8 to utf-16
	res, err := windows.UTF16FromString(buf.String())
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EvtRenderEventXmlText renders EvtRenderEventBookmark
// https://learn.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtrender
func (api *API) EvtRenderBookmark(Fragment evtapi.EventBookmarkHandle) ([]uint16, error) {
	return nil, fmt.Errorf("not implemented")
}

func (api *API) RegisterEventSource(SourceName string) (evtapi.EventSourceHandle, error) {
	// Ensure source/eventLog exists
	eventLog, err := api.getMockEventLog(SourceName)
	if err != nil {
		return evtapi.EventSourceHandle(0), err
	}

	// Create a handle
	h := evtapi.EventSourceHandle(api.nextHandle.Inc())
	api.sourceHandles[h] = eventLog.name
	return h, nil
}

func (api *API) DeregisterEventSource(sourceHandle evtapi.EventSourceHandle) error {
	_, err := api.getMockEventLogByHandle(sourceHandle)
	if err != nil {
		return err
	}
	delete(api.sourceHandles, sourceHandle)
	return nil
}

func (api *API) ReportEvent(
	EventLog evtapi.EventSourceHandle,
	Type uint,
	Category uint,
	EventID uint,
	Strings []string,
	RawData []uint8) error {

	// get event log
	eventLog, err := api.getMockEventLogByHandle(EventLog)
	if err != nil {
		return err
	}

	event := eventLog.reportEvent(
		api,
		Type,
		Category,
		EventID,
		Strings,
		RawData)
	// TODO: Should only create a handle in the API when EvtNext is called
	api.addEventRecord(event)

	return nil
}

// https://learn.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtclearlog
func (api *API) EvtClearLog(ChannelPath string) error {
	// Ensure eventlog exists
	eventLog, err := api.getMockEventLog(ChannelPath)
	if err != nil {
		return err
	}

	// clear the log
	eventLog.events = nil
	return nil
}
