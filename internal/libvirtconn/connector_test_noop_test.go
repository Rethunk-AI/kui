//go:build !libvirt
// +build !libvirt

package libvirtconn

import "testing"

func TestConnectAndListOperations(t *testing.T) {
	t.Skip("libvirt connector tests are disabled without the libvirt build tag")
}
