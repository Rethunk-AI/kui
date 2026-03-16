//go:build libvirt
// +build libvirt

package libvirtconn

import (
	"context"
	"testing"
	"time"
)

func TestConnectAndListOperations(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	conn, err := Connect(ctx, "test:///default", "")
	if err != nil {
		t.Skipf("libvirt not available for integration test: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Fatalf("close libvirt connection: %v", err)
		}
	})

	domains, err := conn.ListDomains(ctx)
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}
	if domains == nil {
		t.Fatalf("list domains should return a slice")
	}

	networks, err := conn.ListNetworks(ctx)
	if err != nil {
		t.Fatalf("list networks: %v", err)
	}
	if networks == nil {
		t.Fatalf("list networks should return a slice")
	}

	pools, err := conn.ListPools(ctx)
	if err != nil {
		t.Fatalf("list pools: %v", err)
	}
	if pools == nil {
		t.Fatalf("list pools should return a slice")
	}
}
