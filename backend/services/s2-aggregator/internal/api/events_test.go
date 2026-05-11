package api

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryEventSubscriber_DispatchesToHandler(t *testing.T) {
	s := NewInMemoryEventSubscriber()
	var got Event
	err := s.Subscribe(context.Background(), TopicRecommendationDetected, func(_ context.Context, e Event) error {
		got = e
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	err = s.Publish(context.Background(), Event{Topic: TopicRecommendationDetected, Payload: map[string]any{"id": "x"}})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if got.Topic != TopicRecommendationDetected {
		t.Fatalf("handler did not receive event; got=%q", got.Topic)
	}
}

func TestInMemoryEventSubscriber_NilHandlerRejected(t *testing.T) {
	s := NewInMemoryEventSubscriber()
	err := s.Subscribe(context.Background(), TopicRecommendationDrafted, nil)
	if err == nil {
		t.Fatal("nil handler should be rejected")
	}
}

func TestInMemoryEventSubscriber_StopBlocksSubscribe(t *testing.T) {
	s := NewInMemoryEventSubscriber()
	_ = s.Stop()
	err := s.Subscribe(context.Background(), TopicRecommendationAccepted, func(context.Context, Event) error { return nil })
	if err == nil {
		t.Fatal("Subscribe after Stop should fail")
	}
}

func TestInMemoryEventSubscriber_HandlerErrorReturned(t *testing.T) {
	s := NewInMemoryEventSubscriber()
	want := errors.New("boom")
	_ = s.Subscribe(context.Background(), TopicRecommendationDeclined, func(context.Context, Event) error { return want })
	err := s.Publish(context.Background(), Event{Topic: TopicRecommendationDeclined})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestNoopEventSubscriber(t *testing.T) {
	s := NewNoopEventSubscriber()
	if err := s.Subscribe(context.Background(), "anything", func(context.Context, Event) error { return nil }); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
}
