package kvmcheck

import (
	"bufio"
	"os"
	"strings"
)

const (
	defaultDevKVMPath    = "/dev/kvm"
	defaultOSReleasePath = "/etc/os-release"
	defaultModulesPath   = "/proc/modules"
	defaultCPUInfoPath   = "/proc/cpuinfo"
)

// CheckKVM verifies that KVM is available on the local host. It checks that
// /dev/kvm exists and is accessible. If not, it returns a distro-specific
// package installation suggestion and, when applicable, a modprobe hint for
// loading the processor-specific KVM module (kvm_intel or kvm_amd).
//
// This is intended for local hosts only (qemu:///system, qemu+unix:). For
// remote hosts (qemu+ssh), KVM availability cannot be checked from the KUI
// server.
func CheckKVM() (ok bool, suggestion string, err error) {
	return checkKVMWithPaths(defaultDevKVMPath, defaultOSReleasePath, defaultModulesPath, defaultCPUInfoPath)
}

// checkKVMWithPaths is the internal implementation that accepts paths for testing.
func checkKVMWithPaths(devKVMPath, osReleasePath, modulesPath, cpuInfoPath string) (ok bool, suggestion string, err error) {
	info, err := os.Stat(devKVMPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, buildSuggestion(osReleasePath, modulesPath, cpuInfoPath), nil
		}
		return false, "", err
	}
	if info.Mode()&os.ModeDevice == 0 {
		return false, buildSuggestion(osReleasePath, modulesPath, cpuInfoPath), nil
	}
	return true, "", nil
}

func buildSuggestion(osReleasePath, modulesPath, cpuInfoPath string) string {
	parts := []string{"KVM not available."}

	// Check if kvm_intel/kvm_amd is loaded; if not, suggest modprobe first
	kvmIntelLoaded := isModuleLoaded(modulesPath, "kvm_intel")
	kvmAmdLoaded := isModuleLoaded(modulesPath, "kvm_amd")
	if !kvmIntelLoaded && !kvmAmdLoaded {
		if isLikelyNestedVM(cpuInfoPath) {
			parts = append(parts, "This host appears to be a VM; enable nested virtualization on the physical host.")
		}
		parts = append(parts, "Load KVM module: sudo modprobe kvm_intel (Intel) or sudo modprobe kvm_amd (AMD)")
	}

	// Add package suggestion if packages might be missing
	id := detectDistroID(osReleasePath)
	switch id {
	case "ubuntu", "debian":
		parts = append(parts, "If needed, install: sudo apt install qemu-kvm libvirt-daemon-system")
	case "fedora", "rhel", "centos", "rocky", "alma":
		parts = append(parts, "If needed, install: sudo dnf install qemu-kvm libvirt")
	case "arch":
		parts = append(parts, "If needed, install: sudo pacman -S qemu libvirt")
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles":
		parts = append(parts, "If needed, install: sudo zypper install qemu-kvm libvirt")
	default:
		parts = append(parts, "If needed, install qemu-kvm and libvirt for your distribution.")
	}

	return strings.Join(parts, " ")
}

func isModuleLoaded(modulesPath, name string) bool {
	f, err := os.Open(modulesPath)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, name+" ") {
			return true
		}
	}
	return false
}

func isLikelyNestedVM(cpuInfoPath string) bool {
	f, err := os.Open(cpuInfoPath)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "flags") && strings.Contains(line, " hypervisor ") {
			return true
		}
	}
	return false
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
