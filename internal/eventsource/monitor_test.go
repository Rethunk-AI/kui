package eventsource

import (
	"context"
	"testing"
	"time"

	"github.com/kui/kui/internal/broadcaster"
	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/libvirtconn"
)

func TestMonitor_NilConfigDoesNotRun(t *testing.T) {
	bc := broadcaster.NewBroadcaster()
	mon := NewMonitor(nil, bc, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	mon.Run(ctx)
}

func TestMonitor_EmptyHostsDoesNotRun(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{}}
	bc := broadcaster.NewBroadcaster()
	mon := NewMonitor(cfg, bc, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	mon.Run(ctx)
}

func TestMonitor_NilBroadcasterDoesNotRun(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{{ID: "h1", URI: "qemu:///system"}}}
	mon := NewMonitor(cfg, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	mon.Run(ctx)
}

func TestDomainStateToSpec(t *testing.T) {
	tests := []struct {
		state libvirtconn.DomainLifecycleState
		want  string
	}{
		{libvirtconn.DomainStateRunning, "running"},
		{libvirtconn.DomainStatePaused, "paused"},
		{libvirtconn.DomainStateShutoff, "shut off"},
		{libvirtconn.DomainStateShutting, "shutting down"},
		{libvirtconn.DomainStateCrashed, "crashed"},
		{libvirtconn.DomainStateUnknown, "unknown"},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := domainStateToSpec(tt.state)
			if got != tt.want {
				t.Errorf("domainStateToSpec(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}
