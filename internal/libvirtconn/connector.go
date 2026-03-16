//go:build libvirt
// +build libvirt

package libvirtconn

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"libvirt.org/go/libvirt"
)

type HostConfig struct {
	URI     string
	Keyfile *string
}

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

type DomainInfo struct {
	Name  string
	UUID  string
	State DomainLifecycleState
}

type NetworkInfo struct {
	Name   string
	UUID   string
	Active bool
}

type StoragePoolInfo struct {
	Name   string
	UUID   string
	Active bool
}

type StorageVolumeInfo struct {
	Name     string
	Path     string
	Capacity uint64
}

type Connector interface {
	Close() error

	ListDomains(ctx context.Context) ([]DomainInfo, error)
	LookupByUUID(ctx context.Context, uuid string) (DomainInfo, error)
	DefineXML(ctx context.Context, xmlConfig string) (DomainInfo, error)
	Create(ctx context.Context, uuid string) error
	Shutdown(ctx context.Context, uuid string) error
	Destroy(ctx context.Context, uuid string) error
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
}

// Connect opens a libvirt connection and returns a connection-scoped connector.
func Connect(ctx context.Context, uri string, keyfile string) (Connector, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	resolvedURI, err := resolveURI(uri, keyfile)
	if err != nil {
		return nil, fmt.Errorf("prepare connection URI: %w", err)
	}

	conn, err := libvirt.NewConnect(resolvedURI)
	if err != nil {
		return nil, fmt.Errorf("connect to %q: %w", resolvedURI, err)
	}

	return &connector{
		conn: conn,
		uri:  resolvedURI,
	}, nil
}

func ConnectWithHostConfig(ctx context.Context, cfg HostConfig) (Connector, error) {
	keyfile := ""
	if cfg.Keyfile != nil {
		keyfile = *cfg.Keyfile
	}
	return Connect(ctx, cfg.URI, keyfile)
}

type connector struct {
	conn *libvirt.Connect
	uri  string
}

func (c *connector) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	if _, err := c.conn.Close(); err != nil {
		return c.wrapErr("close", err)
	}

	return nil
}

func (c *connector) ListDomains(ctx context.Context) ([]DomainInfo, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	libvirtDomains, err := c.conn.ListAllDomains(0)
	if err != nil {
		return nil, c.wrapErr("list domains", err)
	}

	domains := make([]DomainInfo, 0, len(libvirtDomains))
	for _, libvirtDomain := range libvirtDomains {
		domain := libvirtDomain
		defer func() {
			_ = domain.Free()
		}()

		info, err := c.readDomainInfo(domain)
		if err != nil {
			return nil, err
		}
		domains = append(domains, info)
	}

	return domains, nil
}

func (c *connector) LookupByUUID(ctx context.Context, uuid string) (DomainInfo, error) {
	if err := checkContext(ctx); err != nil {
		return DomainInfo{}, err
	}

	domain, err := c.lookupDomain(ctx, "lookup domain by uuid", uuid)
	if err != nil {
		return DomainInfo{}, err
	}
	defer func() {
		_ = domain.Free()
	}()

	return c.readDomainInfo(domain)
}

func (c *connector) DefineXML(ctx context.Context, xmlConfig string) (DomainInfo, error) {
	if err := checkContext(ctx); err != nil {
		return DomainInfo{}, err
	}

	domain, err := c.conn.DomainDefineXML(xmlConfig)
	if err != nil {
		return DomainInfo{}, c.wrapErr("define domain", err)
	}
	defer func() {
		_ = domain.Free()
	}()

	return c.readDomainInfo(domain)
}

func (c *connector) Create(ctx context.Context, uuid string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	domain, err := c.lookupDomain(ctx, "create domain", uuid)
	if err != nil {
		return err
	}
	defer func() {
		_ = domain.Free()
	}()

	if err = domain.Create(); err != nil {
		return c.wrapErr(fmt.Sprintf("create domain uuid=%q", uuid), err)
	}

	return nil
}

func (c *connector) Shutdown(ctx context.Context, uuid string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	domain, err := c.lookupDomain(ctx, "shutdown domain", uuid)
	if err != nil {
		return err
	}
	defer func() {
		_ = domain.Free()
	}()

	if err = domain.Shutdown(); err != nil {
		return c.wrapErr(fmt.Sprintf("shutdown domain uuid=%q", uuid), err)
	}

	return nil
}

func (c *connector) Destroy(ctx context.Context, uuid string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	domain, err := c.lookupDomain(ctx, "destroy domain", uuid)
	if err != nil {
		return err
	}
	defer func() {
		_ = domain.Free()
	}()

	if err = domain.Destroy(); err != nil {
		return c.wrapErr(fmt.Sprintf("destroy domain uuid=%q", uuid), err)
	}

	return nil
}

func (c *connector) Suspend(ctx context.Context, uuid string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	domain, err := c.lookupDomain(ctx, "suspend domain", uuid)
	if err != nil {
		return err
	}
	defer func() {
		_ = domain.Free()
	}()

	if err = domain.Suspend(); err != nil {
		return c.wrapErr(fmt.Sprintf("suspend domain uuid=%q", uuid), err)
	}

	return nil
}

func (c *connector) Resume(ctx context.Context, uuid string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	domain, err := c.lookupDomain(ctx, "resume domain", uuid)
	if err != nil {
		return err
	}
	defer func() {
		_ = domain.Free()
	}()

	if err = domain.Resume(); err != nil {
		return c.wrapErr(fmt.Sprintf("resume domain uuid=%q", uuid), err)
	}

	return nil
}

func (c *connector) GetState(ctx context.Context, uuid string) (DomainLifecycleState, error) {
	if err := checkContext(ctx); err != nil {
		return DomainStateUnknown, err
	}

	domain, err := c.lookupDomain(ctx, "read domain state", uuid)
	if err != nil {
		return DomainStateUnknown, err
	}
	defer func() {
		_ = domain.Free()
	}()

	state, _, err := domain.GetState()
	if err != nil {
		return DomainStateUnknown, c.wrapErr(fmt.Sprintf("get state uuid=%q", uuid), err)
	}

	return normalizeDomainState(state), nil
}

func (c *connector) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	networks, err := c.conn.ListAllNetworks(0)
	if err != nil {
		return nil, c.wrapErr("list networks", err)
	}

	list := make([]NetworkInfo, 0, len(networks))
	for _, libvirtNetwork := range networks {
		network := libvirtNetwork
		defer func() {
			_ = network.Free()
		}()

		name, err := network.GetName()
		if err != nil {
			return nil, c.wrapErr("read network name", err)
		}
		uuid, err := network.GetUUIDString()
		if err != nil {
			return nil, c.wrapErr("read network uuid", err)
		}
		active, err := network.IsActive()
		if err != nil {
			return nil, c.wrapErr("read network state", err)
		}

		list = append(list, NetworkInfo{
			Name:   name,
			UUID:   uuid,
			Active: active,
		})
	}

	return list, nil
}

func (c *connector) ListPools(ctx context.Context) ([]StoragePoolInfo, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	pools, err := c.conn.ListAllStoragePools(0)
	if err != nil {
		return nil, c.wrapErr("list storage pools", err)
	}

	list := make([]StoragePoolInfo, 0, len(pools))
	for _, libvirtPool := range pools {
		pool := libvirtPool
		defer func() {
			_ = pool.Free()
		}()

		active, err := pool.IsActive()
		if err != nil {
			return nil, c.wrapErr("read pool state", err)
		}
		name, err := pool.GetName()
		if err != nil {
			return nil, c.wrapErr("read pool name", err)
		}
		uuid, err := pool.GetUUIDString()
		if err != nil {
			return nil, c.wrapErr("read pool uuid", err)
		}

		list = append(list, StoragePoolInfo{
			Name:   name,
			UUID:   uuid,
			Active: active,
		})
	}

	return list, nil
}

func (c *connector) ListVolumes(ctx context.Context, poolName string) ([]StorageVolumeInfo, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	pool, err := c.lookupPool(ctx, "lookup storage pool", poolName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = pool.Free()
	}()

	volumes, err := pool.ListAllStorageVolumes(0)
	if err != nil {
		return nil, c.wrapErr("list storage volumes", err)
	}

	list := make([]StorageVolumeInfo, 0, len(volumes))
	for _, libvirtVolume := range volumes {
		volume := libvirtVolume
		defer func() {
			_ = volume.Free()
		}()

		name, err := volume.GetName()
		if err != nil {
			return nil, c.wrapErr("read volume name", err)
		}
		path, err := volume.GetPath()
		if err != nil {
			return nil, c.wrapErr("read volume path", err)
		}
		info, err := volume.GetInfo()
		if err != nil {
			return nil, c.wrapErr("read volume info", err)
		}

		list = append(list, StorageVolumeInfo{
			Name:     name,
			Path:     path,
			Capacity: info.Capacity,
		})
	}

	return list, nil
}

func (c *connector) ValidatePool(ctx context.Context, name string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	pool, err := c.lookupPool(ctx, "lookup storage pool", name)
	if err != nil {
		return err
	}
	defer func() {
		_ = pool.Free()
	}()

	active, err := pool.IsActive()
	if err != nil {
		return c.wrapErr("read pool activity", err)
	}
	if !active {
		return c.wrapErr(fmt.Sprintf("validate pool %q", name), fmt.Errorf("pool is inactive"))
	}

	return nil
}

func (c *connector) ValidatePath(ctx context.Context, pool string, path string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		return c.wrapErr("validate volume path", fmt.Errorf("path is required"))
	}

	volumes, err := c.ListVolumes(ctx, pool)
	if err != nil {
		return c.wrapErr("validate volume path", err)
	}

	for _, volume := range volumes {
		if volume.Path == path {
			return nil
		}
	}

	return c.wrapErr("validate volume path", fmt.Errorf("volume path %q not found in pool %q", path, pool))
}

func (c *connector) ValidateVolume(ctx context.Context, pool string, name string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return c.wrapErr("validate volume", fmt.Errorf("volume name is required"))
	}

	poolRef, err := c.lookupPool(ctx, "lookup storage pool", pool)
	if err != nil {
		return err
	}
	defer func() {
		_ = poolRef.Free()
	}()

	volume, err := poolRef.LookupStorageVolByName(name)
	if err != nil {
		return c.wrapErr(fmt.Sprintf("validate volume %q in pool %q", name, pool), err)
	}
	defer func() {
		_ = volume.Free()
	}()

	return nil
}

func (c *connector) CreateVolumeFromXML(ctx context.Context, poolName string, xml string) (StorageVolumeInfo, error) {
	if err := checkContext(ctx); err != nil {
		return StorageVolumeInfo{}, err
	}
	if strings.TrimSpace(xml) == "" {
		return StorageVolumeInfo{}, c.wrapErr("create volume", fmt.Errorf("volume XML is required"))
	}

	pool, err := c.lookupPool(ctx, "create volume", poolName)
	if err != nil {
		return StorageVolumeInfo{}, err
	}
	defer func() {
		_ = pool.Free()
	}()

	vol, err := pool.CreateStorageVolFromXML(xml, 0)
	if err != nil {
		return StorageVolumeInfo{}, c.wrapErr("create volume from XML", err)
	}
	defer func() {
		_ = vol.Free()
	}()

	name, err := vol.GetName()
	if err != nil {
		return StorageVolumeInfo{}, c.wrapErr("read created volume name", err)
	}
	path, err := vol.GetPath()
	if err != nil {
		return StorageVolumeInfo{}, c.wrapErr("read created volume path", err)
	}
	info, err := vol.GetInfo()
	if err != nil {
		return StorageVolumeInfo{}, c.wrapErr("read created volume info", err)
	}

	return StorageVolumeInfo{
		Name:     name,
		Path:     path,
		Capacity: info.Capacity,
	}, nil
}

func (c *connector) CloneVolume(ctx context.Context, poolName string, sourceName string, targetName string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(sourceName) == "" || strings.TrimSpace(targetName) == "" {
		return c.wrapErr("clone volume", fmt.Errorf("source and target volume names are required"))
	}

	pool, err := c.lookupPool(ctx, "clone volume", poolName)
	if err != nil {
		return err
	}
	defer func() {
		_ = pool.Free()
	}()

	sourceVol, err := pool.LookupStorageVolByName(sourceName)
	if err != nil {
		return c.wrapErr("clone volume: lookup source", err)
	}
	defer func() {
		_ = sourceVol.Free()
	}()

	cloneXML := fmt.Sprintf(`<volume><name>%s</name></volume>`, targetName)
	_, err = pool.CreateStorageVolFromXMLFrom(cloneXML, sourceVol, 0)
	if err != nil {
		return c.wrapErr("clone volume", err)
	}

	return nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (c *connector) wrapErr(op string, err error) error {
	return fmt.Errorf("%s on %q: %w", op, c.uri, err)
}

func (c *connector) lookupDomain(ctx context.Context, operation string, uuid string) (libvirt.Domain, error) {
	if err := checkContext(ctx); err != nil {
		return libvirt.Domain{}, err
	}
	if strings.TrimSpace(uuid) == "" {
		return libvirt.Domain{}, c.wrapErr(operation, fmt.Errorf("uuid is required"))
	}

	domain, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return libvirt.Domain{}, c.wrapErr(operation, err)
	}

	return *domain, nil
}

func (c *connector) lookupPool(ctx context.Context, operation string, name string) (libvirt.StoragePool, error) {
	if err := checkContext(ctx); err != nil {
		return libvirt.StoragePool{}, err
	}
	if strings.TrimSpace(name) == "" {
		return libvirt.StoragePool{}, c.wrapErr(operation, fmt.Errorf("pool is required"))
	}

	pool, err := c.conn.LookupStoragePoolByName(name)
	if err != nil {
		return libvirt.StoragePool{}, c.wrapErr(operation, err)
	}

	return *pool, nil
}

func (c *connector) readDomainInfo(domain libvirt.Domain) (DomainInfo, error) {
	name, err := domain.GetName()
	if err != nil {
		return DomainInfo{}, c.wrapErr("read domain name", err)
	}
	uuid, err := domain.GetUUIDString()
	if err != nil {
		return DomainInfo{}, c.wrapErr("read domain uuid", err)
	}
	state, _, err := domain.GetState()
	if err != nil {
		return DomainInfo{}, c.wrapErr(fmt.Sprintf("read domain state for uuid %q", uuid), err)
	}

	return DomainInfo{
		Name:  name,
		UUID:  uuid,
		State: normalizeDomainState(state),
	}, nil
}

func resolveURI(uri string, keyfile string) (string, error) {
	rawURI := strings.TrimSpace(uri)
	if rawURI == "" {
		return "", fmt.Errorf("uri is required")
	}

	if !strings.HasPrefix(rawURI, "qemu+") {
		return rawURI, nil
	}

	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", fmt.Errorf("parse uri %q: %w", rawURI, err)
	}
	if parsed.Scheme != "qemu+ssh" {
		return rawURI, nil
	}

	query := parsed.Query()
	if strings.TrimSpace(keyfile) != "" {
		query.Set("keyfile", strings.TrimSpace(keyfile))
		parsed.RawQuery = query.Encode()
	}

	if strings.TrimSpace(query.Get("keyfile")) == "" {
		return "", fmt.Errorf("keyfile is required for qemu+ssh uri %q", rawURI)
	}

	return parsed.String(), nil
}

func normalizeDomainState(state libvirt.DomainState) DomainLifecycleState {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return DomainStateRunning
	case libvirt.DOMAIN_PAUSED:
		return DomainStatePaused
	case libvirt.DOMAIN_SHUTDOWN:
		return DomainStateShutting
	case libvirt.DOMAIN_SHUTOFF:
		return DomainStateShutoff
	case libvirt.DOMAIN_CRASHED:
		return DomainStateCrashed
	case libvirt.DOMAIN_BLOCKED:
		return DomainStateBlocked
	case libvirt.DOMAIN_PMSUSPENDED:
		return DomainStatePMSuspend
	case libvirt.DOMAIN_NOSTATE:
		return DomainStateNoState
	default:
		return DomainStateUnknown
	}
}
