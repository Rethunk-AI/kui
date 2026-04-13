package eventsource

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kui/kui/internal/broadcaster"
	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/libvirtconn"
)

const pollInterval = 5 * time.Second

// ConnectorProvider returns a libvirt connector for the given host. When nil, the monitor
// uses libvirtconn.Connect. Inject a mock in tests.
type ConnectorProvider func(ctx context.Context, host config.Host) (libvirtconn.Connector, error)

// Monitor polls hosts for domain state changes and connection status, emitting
// vm.state_changed, host.online, and host.offline events via the Broadcaster.
type Monitor struct {
	config            *config.Config
	broadcaster       *broadcaster.Broadcaster
	logger            *slog.Logger
	connectorProvider ConnectorProvider
	pollInterval      time.Duration
	mu                sync.Mutex
	hostState         map[string]bool                             // hostID -> was online last poll
	domainState       map[string]libvirtconn.DomainLifecycleState // "hostID:uuid" -> state
}

// MonitorOptions configures a Monitor.
type MonitorOptions struct {
	Config            *config.Config
	Broadcaster       *broadcaster.Broadcaster
	Logger            *slog.Logger
	ConnectorProvider ConnectorProvider
	PollInterval      time.Duration // 0 = default pollInterval
}

// NewMonitor creates a monitor. Pass nil config to disable (no hosts to poll).
func NewMonitor(cfg *config.Config, bc *broadcaster.Broadcaster, logger *slog.Logger) *Monitor {
	return NewMonitorWithOptions(MonitorOptions{Config: cfg, Broadcaster: bc, Logger: logger})
}

// NewMonitorWithOptions creates a monitor with optional ConnectorProvider for testing.
func NewMonitorWithOptions(opts MonitorOptions) *Monitor {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	m := &Monitor{
		config:            opts.Config,
		broadcaster:       opts.Broadcaster,
		logger:            opts.Logger,
		connectorProvider: opts.ConnectorProvider,
		hostState:         make(map[string]bool),
		domainState:       make(map[string]libvirtconn.DomainLifecycleState),
	}
	if opts.PollInterval > 0 {
		m.pollInterval = opts.PollInterval
	} else {
		m.pollInterval = pollInterval
	}
	return m
}

// Run starts the poll loop. Blocks until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) {
	interval := m.pollInterval
	if interval == 0 {
		interval = pollInterval
	}
	m.runWithInterval(ctx, interval)
}

func (m *Monitor) runWithInterval(ctx context.Context, interval time.Duration) {
	if m.config == nil || len(m.config.Hosts) == 0 || m.broadcaster == nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	m.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.poll(ctx)
		}
	}
}

func (m *Monitor) poll(ctx context.Context) {
	for _, h := range m.config.Hosts {
		m.pollHost(ctx, h)
	}
}

func (m *Monitor) pollHost(ctx context.Context, h config.Host) {
	var conn libvirtconn.Connector
	var err error
	if m.connectorProvider != nil {
		conn, err = m.connectorProvider(ctx, h)
	} else {
		keyfile := ""
		if h.Keyfile != nil {
			keyfile = *h.Keyfile
		}
		conn, err = libvirtconn.Connect(ctx, h.URI, keyfile)
	}
	if err != nil {
		if errors.Is(err, libvirtconn.ErrLibvirtDisabled) {
			return
		}
		m.mu.Lock()
		wasOnline := m.hostState[h.ID]
		m.hostState[h.ID] = false
		m.mu.Unlock()
		if wasOnline {
			m.broadcaster.Broadcast(broadcaster.Event{
				Type: "host.offline",
				Data: map[string]string{"host_id": h.ID, "reason": err.Error()},
			})
			m.logger.Debug("host offline", "host_id", h.ID, "error", err)
		}
		return
	}
	defer func() { _ = conn.Close() }()

	m.mu.Lock()
	wasOnline := m.hostState[h.ID]
	m.hostState[h.ID] = true
	m.mu.Unlock()
	if !wasOnline {
		m.broadcaster.Broadcast(broadcaster.Event{
			Type: "host.online",
			Data: map[string]string{"host_id": h.ID},
		})
		m.logger.Debug("host online", "host_id", h.ID)
	}

	domains, err := conn.ListDomains(ctx)
	if err != nil {
		m.mu.Lock()
		m.hostState[h.ID] = false
		m.mu.Unlock()
		m.broadcaster.Broadcast(broadcaster.Event{
			Type: "host.offline",
			Data: map[string]string{"host_id": h.ID, "reason": err.Error()},
		})
		m.logger.Debug("host offline after list domains", "host_id", h.ID, "error", err)
		return
	}

	for _, d := range domains {
		m.mu.Lock()
		key := h.ID + ":" + d.UUID
		prev := m.domainState[key]
		m.domainState[key] = d.State
		m.mu.Unlock()
		if prev != d.State {
			stateStr := domainStateToSpec(d.State)
			m.broadcaster.Broadcast(broadcaster.Event{
				Type: "vm.state_changed",
				Data: map[string]string{
					"host_id": h.ID,
					"vm_id":   d.UUID,
					"state":   stateStr,
				},
			})
			m.logger.Debug("vm state changed", "host_id", h.ID, "vm_id", d.UUID, "state", stateStr)
		}
	}

	// Remove domains that no longer exist from our state map
	m.mu.Lock()
	seen := make(map[string]struct{})
	for _, d := range domains {
		seen[h.ID+":"+d.UUID] = struct{}{}
	}
	for k := range m.domainState {
		if strings.HasPrefix(k, h.ID+":") {
			if _, ok := seen[k]; !ok {
				delete(m.domainState, k)
			}
		}
	}
	m.mu.Unlock()
}

func domainStateToSpec(s libvirtconn.DomainLifecycleState) string {
	switch s {
	case libvirtconn.DomainStateRunning:
		return "running"
	case libvirtconn.DomainStatePaused:
		return "paused"
	case libvirtconn.DomainStateShutoff:
		return "shut off"
	case libvirtconn.DomainStateShutting:
		return "shutting down"
	case libvirtconn.DomainStateCrashed:
		return "crashed"
	case libvirtconn.DomainStateBlocked:
		return "blocked"
	case libvirtconn.DomainStateSuspended:
		return "suspended"
	case libvirtconn.DomainStateNoState:
		return "nostate"
	case libvirtconn.DomainStatePMSuspend:
		return "pmsuspended"
	default:
		return "unknown"
	}
}
