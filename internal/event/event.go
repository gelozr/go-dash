package event

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type Publisher interface {
	Publish(ctx context.Context, event any) error
}

type KeyedPublisher interface {
	Publisher
	Key() string
}

type Broker struct {
	mu    sync.RWMutex
	buses map[string]Publisher
}

func NewBroker() *Broker {
	return &Broker{
		buses: make(map[string]Publisher),
	}
}

func (r *Broker) RegisterBus(bus KeyedPublisher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buses[bus.Key()] = bus
}

func (r *Broker) Publish(ctx context.Context, event any) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	t := reflect.TypeOf(event).String()

	r.mu.RLock()
	bus, ok := r.buses[t]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("event bus [%s] not found", t)
	}

	return bus.Publish(ctx, event)
}
