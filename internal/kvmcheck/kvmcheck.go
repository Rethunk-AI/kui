package kvmcheck

import (
	"bufio"
	"os"
	"strings"
)

const (
	defaultDevKVMPath   = "/dev/kvm"
	defaultOSReleasePath = "/etc/os-release"
)

// CheckKVM verifies that KVM is available on the local host. It checks that
// /dev/kvm exists and is accessible. If not, it returns a distro-specific
// package installation suggestion.
//
// This is intended for local hosts only (qemu:///system, qemu+unix:). For
// remote hosts (qemu+ssh), KVM availability cannot be checked from the KUI
// server.
func CheckKVM() (ok bool, suggestion string, err error) {
	return checkKVMWithPaths(defaultDevKVMPath, defaultOSReleasePath)
}

// checkKVMWithPaths is the internal implementation that accepts paths for testing.
func checkKVMWithPaths(devKVMPath, osReleasePath string) (ok bool, suggestion string, err error) {
	info, err := os.Stat(devKVMPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, buildSuggestion(osReleasePath), nil
		}
		return false, "", err
	}
	if info.Mode()&os.ModeDevice == 0 {
		return false, buildSuggestion(osReleasePath), nil
	}
	return true, "", nil
}

func buildSuggestion(osReleasePath string) string {
	id := detectDistroID(osReleasePath)
	switch id {
	case "ubuntu", "debian":
		return "KVM not available. Install: sudo apt install qemu-kvm libvirt-daemon-system (Ubuntu/Debian)"
	case "fedora", "rhel", "centos", "rocky", "alma":
		return "KVM not available. Install: sudo dnf install qemu-kvm libvirt (Fedora/RHEL)"
	case "arch":
		return "KVM not available. Install: sudo pacman -S qemu libvirt (Arch)"
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles":
		return "KVM not available. Install: sudo zypper install qemu-kvm libvirt (openSUSE/SLES)"
	default:
		return "KVM not available. Install qemu-kvm and libvirt for your distribution."
	}
}

func detectDistroID(osReleasePath string) string {
	f, err := os.Open(osReleasePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, `"`)
			id = strings.ToLower(id)
			return id
		}
	}
	return ""
}
