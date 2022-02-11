// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package workloadmeta

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/errors"
	"github.com/DataDog/datadog-agent/pkg/status/health"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/util/retry"
	"github.com/DataDog/datadog-agent/pkg/workloadmeta/telemetry"
)

var (
	globalStore Store
	initOnce    sync.Once
)

const (
	retryCollectorInterval = 30 * time.Second
	pullCollectorInterval  = 5 * time.Second
	eventBundleChTimeout   = 1 * time.Second
	eventChBufferSize      = 50
)

type subscriber struct {
	name     string
	priority SubscriberPriority
	ch       chan EventBundle
	filter   *Filter
}

// store is a central storage of metadata about workloads. A workload is any
// unit of work being done by a piece of software, like a process, a container,
// a kubernetes pod, or a task in any cloud provider.
type store struct {
	storeMut sync.RWMutex
	store    map[Kind]map[string]*cachedEntity // store[entity.Kind][entity.ID] = &cachedEntity{}

	subscribersMut sync.RWMutex
	subscribers    []subscriber

	collectorMut sync.RWMutex
	candidates   map[string]Collector
	collectors   map[string]Collector

	eventCh chan []CollectorEvent
}

var _ Store = &store{}

// NewStore creates a new workload metadata store, building a new instance of
// each collector in the catalog. Call Start to start the store and its
// collectors.
func NewStore(catalog map[string]collectorFactory) Store {
	return newStore(catalog)
}

func newStore(catalog map[string]collectorFactory) *store {
	candidates := make(map[string]Collector)
	for id, c := range catalog {
		candidates[id] = c()
	}

	return &store{
		store:      make(map[Kind]map[string]*cachedEntity),
		candidates: candidates,
		collectors: make(map[string]Collector),
		eventCh:    make(chan []CollectorEvent, eventChBufferSize),
	}
}

// Start starts the workload metadata store.
func (s *store) Start(ctx context.Context) {
	go func() {
		health := health.RegisterLiveness("workloadmeta-store")
		for {
			select {
			case <-health.C:

			case evs := <-s.eventCh:
				s.handleEvents(evs)

			case <-ctx.Done():
				err := health.Deregister()
				if err != nil {
					log.Warnf("error de-registering health check: %s", err)
				}

				return
			}
		}
	}()

	go func() {
		retryTicker := time.NewTicker(retryCollectorInterval)
		pullTicker := time.NewTicker(pullCollectorInterval)
		health := health.RegisterLiveness("workloadmeta-puller")
		pullCtx, pullCancel := context.WithTimeout(ctx, pullCollectorInterval)

		// Start a pull immediately to fill the store without waiting for the
		// next tick.
		s.pull(pullCtx)

		for {
			select {
			case <-health.C:

			case <-pullTicker.C:
				// pullCtx will always be expired at this point
				// if pullTicker has the same duration as
				// pullCtx, so we cancel just as good practice
				pullCancel()

				pullCtx, pullCancel = context.WithTimeout(ctx, pullCollectorInterval)
				s.pull(pullCtx)

			case <-retryTicker.C:
				stop := s.startCandidates(ctx)

				if stop {
					retryTicker.Stop()
				}

			case <-ctx.Done():
				retryTicker.Stop()
				pullTicker.Stop()

				pullCancel()

				err := health.Deregister()
				if err != nil {
					log.Warnf("error de-registering health check: %s", err)
				}

				s.unsubscribeAll()

				log.Infof("stopped workloadmeta store")

				return
			}
		}
	}()

	s.startCandidates(ctx)

	log.Info("workloadmeta store initialized successfully")
}

// Subscribe returns a channel where workload metadata events will be streamed
// as they happen. On first subscription, it will also generate an EventTypeSet
// event for each entity present in the store that matches filter.
func (s *store) Subscribe(name string, priority SubscriberPriority, filter *Filter) chan EventBundle {
	// ch needs to be buffered since we'll send it events before the
	// subscriber has the chance to start receiving from it. if it's
	// unbuffered, it'll deadlock.
	sub := subscriber{
		name:     name,
		priority: priority,
		ch:       make(chan EventBundle, 1),
		filter:   filter,
	}

	var events []Event

	// lock the store and only unlock once the subscriber has been added,
	// otherwise the subscriber can lose events. adding the subscriber
	// before locking the store can cause deadlocks if the store sends
	// events before this function returns.
	s.storeMut.RLock()
	defer s.storeMut.RUnlock()

	for kind, entitiesOfKind := range s.store {
		if !sub.filter.MatchKind(kind) {
			continue
		}

		for _, cachedEntity := range entitiesOfKind {
			entity := cachedEntity.get(sub.filter.Source())
			if entity != nil {
				events = append(events, Event{
					Type:   EventTypeSet,
					Entity: entity,
				})
			}
		}
	}

	// sort events by kind and ID for deterministic ordering
	sort.Slice(events, func(i, j int) bool {
		a := events[i].Entity.GetID()
		b := events[j].Entity.GetID()

		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}

		return a.ID < b.ID
	})

	// notifyChannel should not wait when doing the first subscription, as
	// the subscriber is not ready to receive events yet
	notifyChannel(sub.name, sub.ch, events, false)

	s.subscribersMut.Lock()
	defer s.subscribersMut.Unlock()

	s.subscribers = append(s.subscribers, sub)
	sort.SliceStable(s.subscribers, func(i, j int) bool {
		return s.subscribers[i].priority < s.subscribers[j].priority
	})

	telemetry.Subscribers.Inc()

	return sub.ch
}

// Unsubscribe ends a subscription to entity events and closes its channel.
func (s *store) Unsubscribe(ch chan EventBundle) {
	s.subscribersMut.Lock()
	defer s.subscribersMut.Unlock()

	for i, sub := range s.subscribers {
		if sub.ch == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			telemetry.Subscribers.Dec()
			close(ch)
			return
		}
	}
}

// GetContainer implements Store#GetContainer.
func (s *store) GetContainer(id string) (*Container, error) {
	entity, err := s.getEntityByKind(KindContainer, id)
	if err != nil {
		return nil, err
	}

	return entity.(*Container), nil
}

// ListContainers implements Store#ListContainers.
func (s *store) ListContainers() ([]*Container, error) {
	entities, err := s.listEntitiesByKind(KindContainer)
	if err != nil {
		return nil, err
	}

	// Not very efficient
	containers := make([]*Container, 0, len(entities))
	for _, entity := range entities {
		containers = append(containers, entity.(*Container))
	}

	return containers, nil
}

// GetKubernetesPod implements Store#GetKubernetesPod
func (s *store) GetKubernetesPod(id string) (*KubernetesPod, error) {
	entity, err := s.getEntityByKind(KindKubernetesPod, id)
	if err != nil {
		return nil, err
	}

	return entity.(*KubernetesPod), nil
}

// GetKubernetesPodForContainer implements Store#GetKubernetesPodForContainer
func (s *store) GetKubernetesPodForContainer(containerID string) (*KubernetesPod, error) {
	entities, ok := s.store[KindKubernetesPod]
	if !ok {
		return nil, errors.NewNotFound(containerID)
	}

	for _, e := range entities {
		pod := e.cached.(*KubernetesPod)
		for _, podContainer := range pod.Containers {
			if podContainer.ID == containerID {
				return pod, nil
			}
		}
	}

	return nil, errors.NewNotFound(containerID)
}

// GetECSTask implements Store#GetECSTask
func (s *store) GetECSTask(id string) (*ECSTask, error) {
	entity, err := s.getEntityByKind(KindECSTask, id)
	if err != nil {
		return nil, err
	}

	return entity.(*ECSTask), nil
}

// Notify implements Store#Notify
func (s *store) Notify(events []CollectorEvent) {
	if len(events) > 0 {
		s.eventCh <- events
	}
}

func (s *store) startCandidates(ctx context.Context) bool {
	s.collectorMut.Lock()
	defer s.collectorMut.Unlock()

	for id, c := range s.candidates {
		err := c.Start(ctx, s)

		// Leave candidates that returned a retriable error to be
		// re-started in the next tick
		if err != nil && retry.IsErrWillRetry(err) {
			log.Debugf("workloadmeta collector %q could not start, but will retry. error: %s", id, err)
			continue
		}

		// Store successfully started collectors for future reference
		if err == nil {
			log.Infof("workloadmeta collector %q started successfully", id)
			s.collectors[id] = c
		} else {
			log.Infof("workloadmeta collector %q could not start. error: %s", id, err)
		}

		// Remove non-retriable and successfully started collectors
		// from the list of candidates so they're not retried in the
		// next tick
		delete(s.candidates, id)
	}

	return len(s.candidates) == 0
}

func (s *store) pull(ctx context.Context) {
	s.collectorMut.RLock()
	defer s.collectorMut.RUnlock()

	for id, c := range s.collectors {
		// Run each pull in its own separate goroutine to reduce
		// latency and unlock the main goroutine to do other work.
		go func(id string, c Collector) {
			err := c.Pull(ctx)
			if err != nil {
				log.Warnf("error pulling from collector %q: %s", id, err.Error())
				telemetry.PullErrors.Inc(id)
			}
		}(id, c)
	}
}

func (s *store) handleEvents(evs []CollectorEvent) {
	s.storeMut.Lock()

	for _, ev := range evs {
		meta := ev.Entity.GetID()

		telemetry.EventsReceived.Inc(string(meta.Kind), string(ev.Source))

		entitiesOfKind, ok := s.store[meta.Kind]
		if !ok {
			s.store[meta.Kind] = make(map[string]*cachedEntity)
			entitiesOfKind = s.store[meta.Kind]
		}

		cachedEntity, ok := entitiesOfKind[meta.ID]

		switch ev.Type {
		case EventTypeSet:
			if !ok {
				entitiesOfKind[meta.ID] = newCachedEntity()
				cachedEntity = entitiesOfKind[meta.ID]
			}

			if found := cachedEntity.set(ev.Source, ev.Entity); !found {
				telemetry.StoredEntities.Inc(
					string(meta.Kind),
					string(ev.Source),
				)
			}
		case EventTypeUnset:
			if ok && cachedEntity.unset(ev.Source) {
				telemetry.StoredEntities.Dec(
					string(meta.Kind),
					string(ev.Source),
				)

				if len(cachedEntity.sources) == 0 {
					delete(entitiesOfKind, meta.ID)
				}
			}
		default:
			log.Errorf("cannot handle event of type %d. event dump: %+v", ev)
		}
	}

	// unlock the store before notifying subscribers, as they might need to
	// read it for related entities (such as a pod's containers) while they
	// process an event.
	s.storeMut.Unlock()

	// copy the list of subscribers to hold locks for as little as
	// possible, since notifyChannel is a blocking operation.
	s.subscribersMut.RLock()
	subscribers := append([]subscriber{}, s.subscribers...)
	s.subscribersMut.RUnlock()

	for _, sub := range subscribers {
		filter := sub.filter
		filteredEvents := make([]Event, 0, len(evs))

		for _, ev := range evs {
			entityID := ev.Entity.GetID()
			if !filter.MatchKind(entityID.Kind) || !filter.MatchSource(ev.Source) {
				// event should be filtered out because it
				// doesn't match the filter
				continue
			}

			cachedEntity, ok := s.store[entityID.Kind][entityID.ID]

			var entity Entity
			if ok {
				entity = cachedEntity.get(filter.Source())
			}

			if entity != nil {
				// setting an entity (EventTypeSet) or entity
				// had one source removed, but others remain
				filteredEvents = append(filteredEvents, Event{
					Type:   EventTypeSet,
					Entity: entity,
				})
			} else {
				// entity has been removed entirely, unsetting
				// is straight forward too
				filteredEvents = append(filteredEvents, Event{
					Type:   EventTypeUnset,
					Entity: ev.Entity,
				})
			}
		}

		notifyChannel(sub.name, sub.ch, filteredEvents, true)
	}
}

func (s *store) getEntityByKind(kind Kind, id string) (Entity, error) {
	entitiesOfKind, ok := s.store[kind]
	if !ok {
		return nil, errors.NewNotFound(string(kind))
	}

	s.storeMut.RLock()
	defer s.storeMut.RUnlock()

	entity, ok := entitiesOfKind[id]
	if !ok {
		return nil, errors.NewNotFound(id)
	}

	return entity.cached, nil
}

func (s *store) listEntitiesByKind(kind Kind) ([]Entity, error) {
	s.storeMut.RLock()
	defer s.storeMut.RUnlock()

	entitiesOfKind, ok := s.store[kind]
	if !ok {
		if _, found := allKinds[kind]; !found {
			return nil, errors.NewNotFound(string(kind))
		}

		return nil, nil
	}

	entities := make([]Entity, 0, len(entitiesOfKind))
	for _, entity := range entitiesOfKind {
		entities = append(entities, entity.cached)
	}

	return entities, nil
}

func (s *store) unsubscribeAll() {
	s.subscribersMut.Lock()
	defer s.subscribersMut.Unlock()

	for _, sub := range s.subscribers {
		close(sub.ch)
	}

	telemetry.Subscribers.Set(0)
}

func notifyChannel(name string, ch chan EventBundle, events []Event, wait bool) {
	if len(events) == 0 {
		return
	}

	bundle := EventBundle{
		Ch:     make(chan struct{}),
		Events: events,
	}

	ch <- bundle

	if wait {
		timer := time.NewTimer(eventBundleChTimeout)

		select {
		case <-bundle.Ch:
			timer.Stop()
			telemetry.NotificationsSent.Inc(name, telemetry.StatusSuccess)
		case <-timer.C:
			log.Warnf("collector %q did not close the event bundle channel in time, continuing with downstream collectors. bundle dump: %+v", name, bundle)
			telemetry.NotificationsSent.Inc(name, telemetry.StatusError)
		}
	}
}

// GetGlobalStore returns a global instance of the workloadmeta store,
// creating one if it doesn't exist. Start() needs to be called before any data
// collection happens.
func GetGlobalStore() Store {
	initOnce.Do(func() {
		if globalStore == nil {
			globalStore = NewStore(collectorCatalog)
		}
	})

	return globalStore
}
