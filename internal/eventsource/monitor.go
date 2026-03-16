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

// Monitor polls hosts for domain state changes and connection status, emitting
// vm.state_changed, host.online, and host.offline events via the Broadcaster.
type Monitor struct {
	config     *config.Config
	broadcaster *broadcaster.Broadcaster
	logger     *slog.Logger
	mu         sync.Mutex
	hostState  map[string]bool // hostID -> was online last poll
	domainState map[string]libvirtconn.DomainLifecycleState // "hostID:uuid" -> state
}

// NewMonitor creates a monitor. Pass nil config to disable (no hosts to poll).
func NewMonitor(cfg *config.Config, bc *broadcaster.Broadcaster, logger *slog.Logger) *Monitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Monitor{
		config:      cfg,
		broadcaster: bc,
		logger:      logger,
		hostState:   make(map[string]bool),
		domainState: make(map[string]libvirtconn.DomainLifecycleState),
	}
}

// Run starts the poll loop. Blocks until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) {
	if m.config == nil || len(m.config.Hosts) == 0 || m.broadcaster == nil {
		return
	}
	ticker := time.NewTicker(pollInterval)
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
	keyfile := ""
	if h.Keyfile != nil {
		keyfile = *h.Keyfile
	}
	conn, err := libvirtconn.Connect(ctx, h.URI, keyfile)
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
	defer conn.Close()

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
					"vm_id":  d.UUID,
					"state":  stateStr,
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
