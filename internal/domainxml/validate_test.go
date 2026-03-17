package domainxml

import (
	"strings"
	"testing"
)

func TestValidateSafe_ValidXML(t *testing.T) {
	xml := `<?xml version="1.0"?>
<domain type="kvm">
  <name>test-vm</name>
  <uuid>uuid-vm</uuid>
  <memory unit="KiB">1048576</memory>
  <vcpu>1</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type></os>
  <devices><disk type="file"><source file="/var/lib/libvirt/images/test.qcow2"/></disk></devices>
</domain>`
	if err := ValidateSafe(xml, "uuid-vm"); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidateSafe_InvalidXML(t *testing.T) {
	xml := `<domain><name>x</name></domain`
	if err := ValidateSafe(xml, "uuid-vm"); err == nil {
		t.Error("expected error for invalid XML")
	} else if !strings.Contains(err.Error(), "invalid domain XML") {
		t.Errorf("expected 'invalid domain XML', got %q", err.Error())
	}
}

func TestValidateSafe_UUIDMismatch(t *testing.T) {
	xml := `<?xml version="1.0"?>
<domain type="kvm">
  <name>test-vm</name>
  <uuid>other-uuid</uuid>
  <memory unit="KiB">1048576</memory>
  <vcpu>1</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type></os>
  <devices><disk type="file"><source file="/var/lib/libvirt/images/test.qcow2"/></disk></devices>
</domain>`
	if err := ValidateSafe(xml, "uuid-vm"); err == nil {
		t.Error("expected error for UUID mismatch")
	} else if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("expected UUID mismatch message, got %q", err.Error())
	}
}

func TestValidateSafe_ForbiddenQemuCommandline(t *testing.T) {
	xml := `<?xml version="1.0"?>
<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>test-vm</name>
  <uuid>uuid-vm</uuid>
  <memory unit="KiB">1048576</memory>
  <vcpu>1</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type></os>
  <devices><disk type="file"><source file="/var/lib/libvirt/images/test.qcow2"/></disk></devices>
  <qemu:commandline>
    <qemu:arg value="-some-arg"/>
  </qemu:commandline>
</domain>`
	if err := ValidateSafe(xml, "uuid-vm"); err == nil {
		t.Error("expected error for qemu:commandline")
	} else if !strings.Contains(err.Error(), "qemu:commandline") {
		t.Errorf("expected forbidden elements message, got %q", err.Error())
	}
}

func TestValidateSafe_ForbiddenQemuArg(t *testing.T) {
	xml := `<?xml version="1.0"?>
<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>test-vm</name>
  <uuid>uuid-vm</uuid>
  <memory unit="KiB">1048576</memory>
  <vcpu>1</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type></os>
  <devices><disk type="file"><source file="/var/lib/libvirt/images/test.qcow2"/></disk></devices>
  <qemu:commandline>
    <qemu:arg value="-init"/>
    <qemu:arg value="/bin/sh"/>
  </qemu:commandline>
</domain>`
	if err := ValidateSafe(xml, "uuid-vm"); err == nil {
		t.Error("expected error for qemu:arg")
	} else {
		if !strings.Contains(err.Error(), "qemu:arg") || !strings.Contains(err.Error(), "qemu:commandline") {
			t.Errorf("expected forbidden elements (qemu:arg, qemu:commandline), got %q", err.Error())
		}
	}
}

func TestValidateSafe_ForbiddenQemuEnv(t *testing.T) {
	xml := `<?xml version="1.0"?>
<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>test-vm</name>
  <uuid>uuid-vm</uuid>
  <memory unit="KiB">1048576</memory>
  <vcpu>1</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type></os>
  <devices><disk type="file"><source file="/var/lib/libvirt/images/test.qcow2"/></disk></devices>
  <qemu:commandline>
    <qemu:env name="LD_PRELOAD" value="/evil.so"/>
  </qemu:commandline>
</domain>`
	if err := ValidateSafe(xml, "uuid-vm"); err == nil {
		t.Error("expected error for qemu:env")
	} else if !strings.Contains(err.Error(), "qemu:env") {
		t.Errorf("expected forbidden qemu:env, got %q", err.Error())
	}
}

func TestValidateSafe_ForbiddenQemuInit(t *testing.T) {
	xml := `<?xml version="1.0"?>
<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>test-vm</name>
  <uuid>uuid-vm</uuid>
  <memory unit="KiB">1048576</memory>
  <vcpu>1</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type></os>
  <devices><disk type="file"><source file="/var/lib/libvirt/images/test.qcow2"/></disk></devices>
  <qemu:init>/bin/evil-init</qemu:init>
</domain>`
	if err := ValidateSafe(xml, "uuid-vm"); err == nil {
		t.Error("expected error for qemu:init")
	} else if !strings.Contains(err.Error(), "qemu:init") {
		t.Errorf("expected forbidden qemu:init, got %q", err.Error())
	}
}
