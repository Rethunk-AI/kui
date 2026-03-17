//go:build !libvirt
// +build !libvirt

package libvirtconn

import (
	"context"
	"errors"
	"testing"
)

func TestConnect_ReturnsErrLibvirtDisabled(t *testing.T) {
	conn, err := Connect(context.Background(), "qemu:///system", "")
	if conn != nil {
		t.Error("expected nil connector")
	}
	if !errors.Is(err, ErrLibvirtDisabled) {
		t.Errorf("expected ErrLibvirtDisabled, got %v", err)
	}
}

func TestConnectWithHostConfig_ReturnsErrLibvirtDisabled(t *testing.T) {
	conn, err := ConnectWithHostConfig(context.Background(), HostConfig{URI: "qemu+ssh://host/system"})
	if conn != nil {
		t.Error("expected nil connector")
	}
	if !errors.Is(err, ErrLibvirtDisabled) {
		t.Errorf("expected ErrLibvirtDisabled, got %v", err)
	}
}
