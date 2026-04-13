package eventsource

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/kui/kui/internal/broadcaster"
	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/libvirtconn"
)

// mockConnector implements libvirtconn.Connector for monitor tests.
type mockConnector struct {
	listDomainsErr error
	domains        []libvirtconn.DomainInfo
}

func (m *mockConnector) Close() error { return nil }

func (m *mockConnector) ListDomains(ctx context.Context) ([]libvirtconn.DomainInfo, error) {
	if m.listDomainsErr != nil {
		return nil, m.listDomainsErr
	}
	return m.domains, nil
}

func (m *mockConnector) LookupByUUID(ctx context.Context, uuid string) (libvirtconn.DomainInfo, error) {
	return libvirtconn.DomainInfo{}, errors.New("not implemented")
}
func (m *mockConnector) GetDomainXML(ctx context.Context, uuid string) (string, error) {
	return "", errors.New("not implemented")
}
func (m *mockConnector) DefineXML(ctx context.Context, xmlConfig string) (libvirtconn.DomainInfo, error) {
	return libvirtconn.DomainInfo{}, errors.New("not implemented")
}
func (m *mockConnector) Create(ctx context.Context, uuid string) error   { return nil }
func (m *mockConnector) Shutdown(ctx context.Context, uuid string) error { return nil }
func (m *mockConnector) Destroy(ctx context.Context, uuid string) error  { return nil }
func (m *mockConnector) Undefine(ctx context.Context, uuid string) error { return nil }
func (m *mockConnector) Suspend(ctx context.Context, uuid string) error  { return nil }
func (m *mockConnector) Resume(ctx context.Context, uuid string) error   { return nil }
func (m *mockConnector) GetState(ctx context.Context, uuid string) (libvirtconn.DomainLifecycleState, error) {
	return "", errors.New("not implemented")
}
func (m *mockConnector) ListNetworks(ctx context.Context) ([]libvirtconn.NetworkInfo, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConnector) ListPools(ctx context.Context) ([]libvirtconn.StoragePoolInfo, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConnector) ListVolumes(ctx context.Context, pool string) ([]libvirtconn.StorageVolumeInfo, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConnector) ValidatePool(ctx context.Context, pool string) error         { return nil }
func (m *mockConnector) ValidatePath(ctx context.Context, pool, path string) error   { return nil }
func (m *mockConnector) ValidateVolume(ctx context.Context, pool, name string) error { return nil }
func (m *mockConnector) CreateVolumeFromXML(ctx context.Context, pool, xml string) (libvirtconn.StorageVolumeInfo, error) {
	return libvirtconn.StorageVolumeInfo{}, errors.New("not implemented")
}
func (m *mockConnector) CloneVolume(ctx context.Context, pool, sourceName, targetName string) error {
	return errors.New("not implemented")
}
func (m *mockConnector) CreateStoragePoolFromXML(ctx context.Context, xml string) (libvirtconn.StoragePoolInfo, error) {
	return libvirtconn.StoragePoolInfo{}, errors.New("not implemented")
}
func (m *mockConnector) CreateNetworkFromXML(ctx context.Context, xml string) (libvirtconn.NetworkInfo, error) {
	return libvirtconn.NetworkInfo{}, errors.New("not implemented")
}
func (m *mockConnector) CopyVolume(ctx context.Context, pool, volumeName string) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConnector) CreateVolumeFromBytes(ctx context.Context, pool, name string, data []byte, format string) (libvirtconn.StorageVolumeInfo, error) {
	return libvirtconn.StorageVolumeInfo{}, errors.New("not implemented")
}
func (m *mockConnector) OpenSerialConsole(ctx context.Context, uuid string) (io.ReadWriteCloser, error) {
	return nil, errors.New("not implemented")
}

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
		{libvirtconn.DomainStateBlocked, "blocked"},
		{libvirtconn.DomainStateSuspended, "suspended"},
		{libvirtconn.DomainStateNoState, "nostate"},
		{libvirtconn.DomainStatePMSuspend, "pmsuspended"},
		{libvirtconn.DomainStateUnknown, "unknown"},
		{libvirtconn.DomainLifecycleState("custom"), "unknown"},
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

// TestPollHost_ErrLibvirtDisabled exercises pollHost when libvirt returns ErrLibvirtDisabled.
// When built without libvirt tag, libvirtconn.Connect returns ErrLibvirtDisabled; pollHost
// returns silently without broadcasting.
func TestPollHost_ErrLibvirtDisabled(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.Host{{ID: "testhost", URI: "qemu:///system"}},
	}
	bc := broadcaster.NewBroadcaster()
	mon := NewMonitor(cfg, bc, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Run one poll cycle; with libvirt disabled, pollHost returns on ErrLibvirtDisabled
	// without panicking or blocking.
	done := make(chan struct{})
	go func() {
		mon.Run(ctx)
		close(done)
	}()
	select {
	case <-done:
		// Run completed (ctx cancelled)
	case <-time.After(500 * time.Millisecond):
		cancel()
		<-done
	}
}

func TestPollHost_ConnectErrorWhenWasOnline_BroadcastsOffline(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.Host{{ID: "h1", URI: "qemu:///system"}},
	}
	bc := broadcaster.NewBroadcaster()
	firstCall := true
	mon := NewMonitorWithOptions(MonitorOptions{
		Config:       cfg,
		Broadcaster:  bc,
		Logger:       nil,
		PollInterval: 50 * time.Millisecond,
		ConnectorProvider: func(ctx context.Context, h config.Host) (libvirtconn.Connector, error) {
			if firstCall {
				firstCall = false
				return &mockConnector{domains: []libvirtconn.DomainInfo{{Name: "v1", UUID: "u1", State: libvirtconn.DomainStateRunning}}}, nil
			}
			return nil, errors.New("connection refused")
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	sub := bc.Subscribe(ctx)
	defer sub.Done()

	go mon.Run(ctx)

	// Drain: Subscribe placeholder, first poll host.online, first poll vm.state_changed (domain Running)
	_, _, _ = <-sub.C, <-sub.C, <-sub.C
	// Second poll: connect fails, host goes offline
	select {
	case ev := <-sub.C:
		if ev.Type != "host.offline" {
			t.Fatalf("expected host.offline, got %s", ev.Type)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for host.offline")
	}
}

func TestPollHost_ConnectSuccessWhenNotOnline_BroadcastsOnline(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.Host{{ID: "h1", URI: "qemu:///system"}},
	}
	bc := broadcaster.NewBroadcaster()
	mon := NewMonitorWithOptions(MonitorOptions{
		Config:      cfg,
		Broadcaster: bc,
		Logger:      nil,
		ConnectorProvider: func(ctx context.Context, h config.Host) (libvirtconn.Connector, error) {
			return &mockConnector{domains: []libvirtconn.DomainInfo{}}, nil
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	sub := bc.Subscribe(ctx)
	defer sub.Done()

	go mon.Run(ctx)

	// Subscribe sends placeholder host.online; first poll sends host.online for h1
	ev := <-sub.C
	if ev.Type != "host.online" {
		t.Fatalf("expected host.online, got %s", ev.Type)
	}
}

func TestPollHost_ListDomainsError_BroadcastsOffline(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.Host{{ID: "h1", URI: "qemu:///system"}},
	}
	bc := broadcaster.NewBroadcaster()
	mon := NewMonitorWithOptions(MonitorOptions{
		Config:       cfg,
		Broadcaster:  bc,
		Logger:       nil,
		PollInterval: 50 * time.Millisecond,
		ConnectorProvider: func(ctx context.Context, h config.Host) (libvirtconn.Connector, error) {
			return &mockConnector{listDomainsErr: errors.New("list failed")}, nil
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	sub := bc.Subscribe(ctx)
	defer sub.Done()

	go mon.Run(ctx)

	// Subscribe placeholder host.online, then connect succeeds -> host.online, ListDomains fails -> host.offline
	_, _ = <-sub.C, <-sub.C
	select {
	case ev := <-sub.C:
		if ev.Type != "host.offline" {
			t.Fatalf("expected host.offline, got %s", ev.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for host.offline")
	}
}

func TestPollHost_DomainStateChange_BroadcastsStateChanged(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.Host{{ID: "h1", URI: "qemu:///system"}},
	}
	bc := broadcaster.NewBroadcaster()
	pollCount := 0
	mon := NewMonitorWithOptions(MonitorOptions{
		Config:       cfg,
		Broadcaster:  bc,
		Logger:       nil,
		PollInterval: 50 * time.Millisecond,
		ConnectorProvider: func(ctx context.Context, h config.Host) (libvirtconn.Connector, error) {
			pollCount++
			if pollCount == 1 {
				return &mockConnector{domains: []libvirtconn.DomainInfo{
					{Name: "v1", UUID: "u1", State: libvirtconn.DomainStateShutoff},
				}}, nil
			}
			return &mockConnector{domains: []libvirtconn.DomainInfo{
				{Name: "v1", UUID: "u1", State: libvirtconn.DomainStateRunning},
			}}, nil
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	sub := bc.Subscribe(ctx)
	defer sub.Done()

	go mon.Run(ctx)

	// Drain: Subscribe placeholder, first poll host.online, first poll vm.state_changed (shutoff)
	_, _, _ = <-sub.C, <-sub.C, <-sub.C
	// Second poll: vm.state_changed (shutoff -> running)
	select {
	case ev := <-sub.C:
		if ev.Type != "vm.state_changed" {
			t.Fatalf("expected vm.state_changed, got %s", ev.Type)
		}
		if data, ok := ev.Data.(map[string]string); !ok || data["state"] != "running" {
			t.Errorf("expected state=running, got %v", ev.Data)
		}
	case <-time.After(350 * time.Millisecond):
		t.Fatal("timeout waiting for vm.state_changed")
	}
}
