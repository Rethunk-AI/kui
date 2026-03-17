package libvirtconn

import (
	"context"
	"errors"
	"io"
	"os"
)

// SetupTestConnector returns a Connector with one pool and one network.
// Used when KUI_TEST_SETUP_MOCK=1 to allow integration tests to pass
// without a real libvirt with pools/networks.
func SetupTestConnector() Connector {
	return &setupTestConnector{}
}

// SetupTestConnectorEnabled returns true when KUI_TEST_SETUP_MOCK=1.
func SetupTestConnectorEnabled() bool {
	return os.Getenv("KUI_TEST_SETUP_MOCK") == "1"
}

type setupTestConnector struct{}

func (c *setupTestConnector) Close() error { return nil }

func (c *setupTestConnector) ListPools(ctx context.Context) ([]StoragePoolInfo, error) {
	return []StoragePoolInfo{{Name: "default", UUID: "p1", Active: true}}, nil
}

func (c *setupTestConnector) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	return []NetworkInfo{{Name: "default", UUID: "n1", Active: true}}, nil
}

func (c *setupTestConnector) ListDomains(ctx context.Context) ([]DomainInfo, error) {
	return nil, errors.New("setup test connector: ListDomains not implemented")
}
func (c *setupTestConnector) LookupByUUID(ctx context.Context, uuid string) (DomainInfo, error) {
	return DomainInfo{}, errors.New("setup test connector: LookupByUUID not implemented")
}
func (c *setupTestConnector) GetDomainXML(ctx context.Context, uuid string) (string, error) {
	return "", errors.New("setup test connector: GetDomainXML not implemented")
}
func (c *setupTestConnector) DefineXML(ctx context.Context, xmlConfig string) (DomainInfo, error) {
	return DomainInfo{}, errors.New("setup test connector: DefineXML not implemented")
}
func (c *setupTestConnector) Create(ctx context.Context, uuid string) error {
	return errors.New("setup test connector: Create not implemented")
}
func (c *setupTestConnector) Shutdown(ctx context.Context, uuid string) error {
	return errors.New("setup test connector: Shutdown not implemented")
}
func (c *setupTestConnector) Destroy(ctx context.Context, uuid string) error {
	return errors.New("setup test connector: Destroy not implemented")
}
func (c *setupTestConnector) Undefine(ctx context.Context, uuid string) error {
	return errors.New("setup test connector: Undefine not implemented")
}
func (c *setupTestConnector) Suspend(ctx context.Context, uuid string) error {
	return errors.New("setup test connector: Suspend not implemented")
}
func (c *setupTestConnector) Resume(ctx context.Context, uuid string) error {
	return errors.New("setup test connector: Resume not implemented")
}
func (c *setupTestConnector) GetState(ctx context.Context, uuid string) (DomainLifecycleState, error) {
	return "", errors.New("setup test connector: GetState not implemented")
}
func (c *setupTestConnector) ListVolumes(ctx context.Context, pool string) ([]StorageVolumeInfo, error) {
	return nil, errors.New("setup test connector: ListVolumes not implemented")
}
func (c *setupTestConnector) ValidatePool(ctx context.Context, pool string) error {
	return errors.New("setup test connector: ValidatePool not implemented")
}
func (c *setupTestConnector) ValidatePath(ctx context.Context, pool, path string) error {
	return errors.New("setup test connector: ValidatePath not implemented")
}
func (c *setupTestConnector) ValidateVolume(ctx context.Context, pool, name string) error {
	return errors.New("setup test connector: ValidateVolume not implemented")
}
func (c *setupTestConnector) CreateVolumeFromXML(ctx context.Context, pool, xml string) (StorageVolumeInfo, error) {
	return StorageVolumeInfo{}, errors.New("setup test connector: CreateVolumeFromXML not implemented")
}
func (c *setupTestConnector) CloneVolume(ctx context.Context, pool, sourceName, targetName string) error {
	return errors.New("setup test connector: CloneVolume not implemented")
}
func (c *setupTestConnector) CreateStoragePoolFromXML(ctx context.Context, xml string) (StoragePoolInfo, error) {
	return StoragePoolInfo{}, errors.New("setup test connector: CreateStoragePoolFromXML not implemented")
}
func (c *setupTestConnector) CreateNetworkFromXML(ctx context.Context, xml string) (NetworkInfo, error) {
	return NetworkInfo{}, errors.New("setup test connector: CreateNetworkFromXML not implemented")
}
func (c *setupTestConnector) CopyVolume(ctx context.Context, pool, volumeName string) ([]byte, error) {
	return nil, errors.New("setup test connector: CopyVolume not implemented")
}
func (c *setupTestConnector) CreateVolumeFromBytes(ctx context.Context, pool, name string, data []byte, format string) (StorageVolumeInfo, error) {
	return StorageVolumeInfo{}, errors.New("setup test connector: CreateVolumeFromBytes not implemented")
}
func (c *setupTestConnector) OpenSerialConsole(ctx context.Context, uuid string) (io.ReadWriteCloser, error) {
	return nil, errors.New("setup test connector: OpenSerialConsole not implemented")
}
