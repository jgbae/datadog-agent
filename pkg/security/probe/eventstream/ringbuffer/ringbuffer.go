// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package ringbuffer holds ringbuffer related files
package ringbuffer

import (
	"errors"
	"sync"

	manager "github.com/DataDog/ebpf-manager"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"

	ddebpf "github.com/DataDog/datadog-agent/pkg/ebpf"
	"github.com/DataDog/datadog-agent/pkg/security/probe/config"
	"github.com/DataDog/datadog-agent/pkg/security/probe/eventstream"
)

// RingBuffer implements the EventStream interface
// using an eBPF map of type BPF_MAP_TYPE_RINGBUF
type RingBuffer struct {
	ringBuffer *manager.RingBuffer
	handler    func(int, []byte)
	record     ringbuf.Record
}

// Init the ring buffer
func (rb *RingBuffer) Init(mgr *manager.Manager, config *config.Config) error {
	var ok bool
	if rb.ringBuffer, ok = mgr.GetRingBuffer("events"); !ok {
		return errors.New("couldn't find events ring buffer")
	}

	rb.ringBuffer.RingBufferOptions = manager.RingBufferOptions{
		RecordHandler: rb.handleEvent,
		RecordGetter: func() *ringbuf.Record {
			return &rb.record
		},
	}

	if config.EventStreamBufferSize != 0 {
		rb.ringBuffer.RingBufferOptions.RingBufferSize = config.EventStreamBufferSize
	}

	return nil
}

// Start the event stream.
func (rb *RingBuffer) Start(wg *sync.WaitGroup) error {
	return rb.ringBuffer.Start()
}

// SetMonitor set the monitor
func (rb *RingBuffer) SetMonitor(counter eventstream.LostEventCounter) {}

func (rb *RingBuffer) handleEvent(rec *ringbuf.Record, ringBuffer *manager.RingBuffer, _ *manager.Manager) {
	ddebpf.ReportPerfMetrics(ringBuffer.Name, ebpf.RingBuf.String(), 0, rec.Remaining, ringBuffer.BufferSize())
	rb.handler(0, rec.RawSample)
}

// Pause the event stream. Do nothing when using ring buffer
func (rb *RingBuffer) Pause() error {
	return nil
}

// Resume the event stream. Do nothing when using ring buffer
func (rb *RingBuffer) Resume() error {
	return nil
}

// New returns a new ring buffer based event stream.
func New(handler func(int, []byte)) *RingBuffer {
	return &RingBuffer{
		handler: handler,
	}
}
