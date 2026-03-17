package broadcaster

import (
	"context"
	"testing"
)

func TestNewBroadcaster(t *testing.T) {
	b := NewBroadcaster()
	if b == nil {
		t.Fatal("NewBroadcaster returned nil")
	}
}

func TestSubscribe_ReceivesPlaceholderEvent(t *testing.T) {
	b := NewBroadcaster()
	ctx := context.Background()

	sub := b.Subscribe(ctx)
	defer sub.Done()

	ev := <-sub.C
	if ev.Type != "host.online" {
		t.Errorf("expected host.online, got %q", ev.Type)
	}
	if ev.Data == nil {
		t.Error("expected non-nil Data")
	}
}

func TestSubscribe_ContextDoneBeforeSend(t *testing.T) {
	b := NewBroadcaster()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sub := b.Subscribe(ctx)
	// done is nil when ctx was already done; no need to call Done
	_, ok := <-sub.C
	if ok {
		t.Error("expected channel to be closed")
	}
}

func TestSubscribe_Done_ClosesChannel(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe(context.Background())
	<-sub.C // consume placeholder so channel is empty

	sub.Done()
	_, ok := <-sub.C
	if ok {
		t.Error("expected channel to be closed after Done()")
	}
}

func TestSubscribe_Done_Idempotent(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe(context.Background())
	<-sub.C // consume placeholder

	sub.Done()
	sub.Done()
	sub.Done()
	_, ok := <-sub.C
	if ok {
		t.Error("expected channel to be closed")
	}
}

func TestBroadcast_DeliversToSubscribers(t *testing.T) {
	b := NewBroadcaster()
	ctx := context.Background()

	sub1 := b.Subscribe(ctx)
	defer sub1.Done()
	<-sub1.C // consume placeholder

	sub2 := b.Subscribe(ctx)
	defer sub2.Done()
	<-sub2.C // consume placeholder

	ev := Event{Type: "vm.state_changed", Data: map[string]string{"id": "vm-1"}}
	b.Broadcast(ev)

	got1 := <-sub1.C
	got2 := <-sub2.C

	if got1.Type != ev.Type || got2.Type != ev.Type {
		t.Errorf("expected %q, got %q and %q", ev.Type, got1.Type, got2.Type)
	}
}

func TestBroadcast_NonBlockingWhenChannelFull(t *testing.T) {
	b := NewBroadcaster()
	ctx := context.Background()

	sub := b.Subscribe(ctx)
	defer sub.Done()
	<-sub.C // consume placeholder

	// Fill channel (buffer size 8)
	for i := 0; i < 8; i++ {
		b.Broadcast(Event{Type: "test", Data: i})
	}

	// This should not block; extra events are dropped
	b.Broadcast(Event{Type: "dropped", Data: nil})
	b.Broadcast(Event{Type: "dropped", Data: nil})

	// Consume what we can
	for i := 0; i < 8; i++ {
		<-sub.C
	}
}
