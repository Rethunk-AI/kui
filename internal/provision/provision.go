package provision

import (
	"fmt"
	"os"
	"strings"

	"github.com/kui/kui/internal/prefix"
	"libvirt.org/go/libvirtxml"
)

const (
	DefaultPoolPath      = "/var/lib/libvirt/images"
	DefaultKuiPoolPath   = "/var/lib/kui/images"
	DefaultPoolName      = "default"
	DefaultNetworkName   = "default"
	DefaultNetworkSubnet = "192.168.122.0/24"
)

// PoolPathResult holds the chosen pool path and whether the directory was created.
type PoolPathResult struct {
	Path    string
	Created bool
}

// SelectPoolPath chooses the pool path per plan: use /var/lib/libvirt/images if it
// exists and is non-empty; otherwise use /var/lib/kui/images.
// When KUI_TEST_PROVISION_POOL_PATH is set (e.g. in tests), returns that path.
// When pathPrefix is non-empty (after TrimSpace), paths are resolved under the
// runtime prefix; when empty, behavior matches the legacy unprefixed resolution.
func SelectPoolPath(pathPrefix string) (PoolPathResult, error) {
	pref := strings.TrimSpace(pathPrefix)
	if override := os.Getenv("KUI_TEST_PROVISION_POOL_PATH"); override != "" {
		if pref == "" {
			return PoolPathResult{Path: override, Created: false}, nil
		}
		trimmed := strings.TrimSpace(override)
		return PoolPathResult{Path: prefix.Resolve(pref, trimmed), Created: false}, nil
	}
	libvirtPath := DefaultPoolPath
	kuiPath := DefaultKuiPoolPath
	if pref != "" {
		libvirtPath = prefix.Resolve(pref, DefaultPoolPath)
		kuiPath = prefix.Resolve(pref, DefaultKuiPoolPath)
	}
	entries, err := os.ReadDir(libvirtPath)
	if err == nil && len(entries) > 0 {
		return PoolPathResult{Path: libvirtPath, Created: false}, nil
	}
	return PoolPathResult{Path: kuiPath, Created: false}, nil
}

// EnsurePoolDir creates the directory for the pool.
// Caller should call this before CreateStoragePoolFromXML for dir-type pools.
func EnsurePoolDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create pool directory %q: %w", path, err)
	}
	return nil
}

// BuildDirPoolXML returns libvirt XML for a dir-type storage pool.
func BuildDirPoolXML(name, path string) (string, error) {
	pool := &libvirtxml.StoragePool{
		Type: "dir",
		Name: name,
		Target: &libvirtxml.StoragePoolTarget{
			Path: path,
		},
	}
	return pool.Marshal()
}

// ParseSubnetToGateway parses "a.b.c.d/n" and returns gateway (a.b.c.1) and prefix.
func ParseSubnetToGateway(subnet string) (gateway string, prefix uint) {
	gateway = "192.168.122.1"
	prefix = 24
	if subnet == "" {
		return gateway, prefix
	}
	parts := strings.SplitN(subnet, "/", 2)
	if len(parts) == 2 {
		_, _ = fmt.Sscanf(parts[1], "%d", &prefix)
	}
	var a, b, c int
	if n, _ := fmt.Sscanf(parts[0], "%d.%d.%d.%d", &a, &b, &c, new(int)); n >= 4 {
		gateway = fmt.Sprintf("%d.%d.%d.1", a, b, c)
	}
	return gateway, prefix
}

// BuildNATNetworkXML returns libvirt XML for a NAT network.
func BuildNATNetworkXML(name, subnet string) (string, error) {
	gateway, prefix := ParseSubnetToGateway(subnet)
	net := &libvirtxml.Network{
		Name: name,
		Forward: &libvirtxml.NetworkForward{
			Mode: "nat",
		},
		IPs: []libvirtxml.NetworkIP{
			{
				Address: gateway,
				Prefix:  prefix,
			},
		},
	}
	return net.Marshal()
}
