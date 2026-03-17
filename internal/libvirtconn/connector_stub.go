//go:build !libvirt
// +build !libvirt

package libvirtconn

import (
	"context"
	"errors"
	"io"
)

// HostConfig holds libvirt connection parameters.
type HostConfig struct {
	URI     string
	Keyfile *string
}

// DomainLifecycleState represents domain state.
type DomainLifecycleState string

const (
	DomainStateUnknown   DomainLifecycleState = "unknown"
	DomainStateRunning   DomainLifecycleState = "running"
	DomainStatePaused    DomainLifecycleState = "paused"
	DomainStateShutoff   DomainLifecycleState = "shutoff"
	DomainStateShutting  DomainLifecycleState = "shutdown"
	DomainStateCrashed   DomainLifecycleState = "crashed"
	DomainStateBlocked   DomainLifecycleState = "blocked"
	DomainStateSuspended DomainLifecycleState = "suspended"
	DomainStateNoState   DomainLifecycleState = "nostate"
	DomainStatePMSuspend DomainLifecycleState = "pmsuspended"
)

// DomainInfo holds domain metadata.
type DomainInfo struct {
	Name  string
	UUID  string
	State DomainLifecycleState
}

// NetworkInfo holds network metadata.
type NetworkInfo struct {
	Name   string
	UUID   string
	Active bool
}

// StoragePoolInfo holds storage pool metadata.
type StoragePoolInfo struct {
	Name   string
	UUID   string
	Active bool
}

// StorageVolumeInfo holds storage volume metadata.
type StorageVolumeInfo struct {
	Name     string
	Path     string
	Capacity uint64
}

// Connector is the libvirt connector interface.
type Connector interface {
	Close() error

	ListDomains(ctx context.Context) ([]DomainInfo, error)
	LookupByUUID(ctx context.Context, uuid string) (DomainInfo, error)
	GetDomainXML(ctx context.Context, uuid string) (string, error)
	DefineXML(ctx context.Context, xmlConfig string) (DomainInfo, error)
	Create(ctx context.Context, uuid string) error
	Shutdown(ctx context.Context, uuid string) error
	Destroy(ctx context.Context, uuid string) error
	Undefine(ctx context.Context, uuid string) error
	Suspend(ctx context.Context, uuid string) error
	Resume(ctx context.Context, uuid string) error
	GetState(ctx context.Context, uuid string) (DomainLifecycleState, error)

	ListNetworks(ctx context.Context) ([]NetworkInfo, error)

	ListPools(ctx context.Context) ([]StoragePoolInfo, error)
	ListVolumes(ctx context.Context, pool string) ([]StorageVolumeInfo, error)
	ValidatePool(ctx context.Context, pool string) error
	ValidatePath(ctx context.Context, pool string, path string) error
	ValidateVolume(ctx context.Context, pool string, name string) error

	CreateVolumeFromXML(ctx context.Context, pool string, xml string) (StorageVolumeInfo, error)
	CloneVolume(ctx context.Context, pool string, sourceName string, targetName string) error

	CopyVolume(ctx context.Context, pool string, volumeName string) ([]byte, error)
	CreateVolumeFromBytes(ctx context.Context, pool string, name string, data []byte, format string) (StorageVolumeInfo, error)

	OpenSerialConsole(ctx context.Context, uuid string) (io.ReadWriteCloser, error)
}

// ErrLibvirtDisabled is returned when the libvirt build tag is not set.
var ErrLibvirtDisabled = errors.New("libvirt connector disabled: build with -tags libvirt and install libvirt-dev")

// Connect opens a libvirt connection. Returns ErrLibvirtDisabled without -tags libvirt.
func Connect(ctx context.Context, uri string, keyfile string) (Connector, error) {
	return nil, ErrLibvirtDisabled
}

// ConnectWithHostConfig opens a libvirt connection from HostConfig. Returns ErrLibvirtDisabled without -tags libvirt.
func ConnectWithHostConfig(ctx context.Context, cfg HostConfig) (Connector, error) {
	return nil, ErrLibvirtDisabled
}
