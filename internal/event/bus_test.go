package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestPublishSubscribe(t *testing.T) {
	bus := NewBus(testLogger())
	var received plugin.Event

	bus.Subscribe("test.topic", func(ctx context.Context, e plugin.Event) {
		received = e
	})

	event := plugin.Event{
		Topic:     "test.topic",
		Source:    "test",
		Timestamp: time.Now(),
		Payload:   "hello",
	}

	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if received.Topic != "test.topic" {
		t.Errorf("received.Topic = %q, want %q", received.Topic, "test.topic")
	}
	if received.Payload != "hello" {
		t.Errorf("received.Payload = %v, want %q", received.Payload, "hello")
	}
}

func TestSubscribeAll(t *testing.T) {
	bus := NewBus(testLogger())
	var count int32

	bus.SubscribeAll(func(ctx context.Context, e plugin.Event) {
		atomic.AddInt32(&count, 1)
	})

	bus.Publish(context.Background(), plugin.Event{Topic: "a"})
	bus.Publish(context.Background(), plugin.Event{Topic: "b"})

	if got := atomic.LoadInt32(&count); got != 2 {
		t.Errorf("SubscribeAll handler called %d times, want 2", got)
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus(testLogger())
	var count int32

	unsub := bus.Subscribe("test", func(ctx context.Context, e plugin.Event) {
		atomic.AddInt32(&count, 1)
	})

	bus.Publish(context.Background(), plugin.Event{Topic: "test"})
	unsub()
	bus.Publish(context.Background(), plugin.Event{Topic: "test"})

	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("handler called %d times after unsubscribe, want 1", got)
	}
}

func TestUnsubscribeAll(t *testing.T) {
	bus := NewBus(testLogger())
	var count int32

	unsub := bus.SubscribeAll(func(ctx context.Context, e plugin.Event) {
		atomic.AddInt32(&count, 1)
	})

	bus.Publish(context.Background(), plugin.Event{Topic: "test"})
	unsub()
	bus.Publish(context.Background(), plugin.Event{Topic: "test"})

	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("handler called %d times after unsubscribe, want 1", got)
	}
}

func TestPublishAsync(t *testing.T) {
	bus := NewBus(testLogger())
	var wg sync.WaitGroup
	var count int32

	wg.Add(2)
	bus.Subscribe("async.test", func(ctx context.Context, e plugin.Event) {
		atomic.AddInt32(&count, 1)
		wg.Done()
	})
	bus.SubscribeAll(func(ctx context.Context, e plugin.Event) {
		atomic.AddInt32(&count, 1)
		wg.Done()
	})

	bus.PublishAsync(context.Background(), plugin.Event{Topic: "async.test"})

	wg.Wait()
	if got := atomic.LoadInt32(&count); got != 2 {
		t.Errorf("async handlers called %d times, want 2", got)
	}
}

func TestHandlerPanicRecovery(t *testing.T) {
	bus := NewBus(testLogger())
	var count int32

	bus.Subscribe("panic.test", func(ctx context.Context, e plugin.Event) {
		panic("test panic")
	})
	bus.Subscribe("panic.test", func(ctx context.Context, e plugin.Event) {
		atomic.AddInt32(&count, 1)
	})

	// Should not panic, and second handler should still run.
	bus.Publish(context.Background(), plugin.Event{Topic: "panic.test"})

	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("second handler called %d times, want 1", got)
	}
}

func TestNoSubscribersOK(t *testing.T) {
	bus := NewBus(testLogger())

	// Publishing with no subscribers should not error.
	if err := bus.Publish(context.Background(), plugin.Event{Topic: "empty"}); err != nil {
		t.Fatalf("Publish() with no subscribers error = %v", err)
	}
}
