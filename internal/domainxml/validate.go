package domainxml

import (
	"encoding/xml"
	"fmt"
	"strings"

	"libvirt.org/go/libvirtxml"
)

// QEMU namespace for forbidden elements
const qemuNamespace = "http://libvirt.org/schemas/domain/qemu/1.0"

// Forbidden elements (local name, namespace). Reject if present.
var forbiddenElements = []struct {
	Local  string
	NS     string
	Reason string
}{
	{"commandline", qemuNamespace, "qemu:commandline"},
	{"arg", qemuNamespace, "qemu:arg"},
	{"env", qemuNamespace, "qemu:env"},
	{"init", qemuNamespace, "qemu:init"},
}

// ValidateResult holds validation result.
type ValidateResult struct {
	Valid   bool
	Message string
}

// ValidateSafe parses XML, checks UUID match, and rejects forbidden elements.
// Returns error message if invalid.
func ValidateSafe(xmlStr string, expectedUUID string) error {
	dom := &libvirtxml.Domain{}
	if err := dom.Unmarshal(xmlStr); err != nil {
		return fmt.Errorf("invalid domain XML")
	}
	uuid := strings.TrimSpace(dom.UUID)
	expected := strings.TrimSpace(expectedUUID)
	if uuid != expected {
		return fmt.Errorf("domain UUID %q does not match path param %q", uuid, expected)
	}
	forbidden := findForbiddenElements(xmlStr)
	if len(forbidden) > 0 {
		return fmt.Errorf("domain XML contains forbidden elements: %s", strings.Join(forbidden, ", "))
	}
	return nil
}

// findForbiddenElements walks the XML and returns a list of forbidden element names found.
// Go's xml.Decoder resolves namespaces; StartElement.Name.Space is the namespace URL.
func findForbiddenElements(xmlStr string) []string {
	dec := xml.NewDecoder(strings.NewReader(xmlStr))
	var found []string
	seen := make(map[string]bool)
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		ns := start.Name.Space
		local := start.Name.Local
		for _, f := range forbiddenElements {
			if local == f.Local && ns == f.NS {
				key := f.Reason
				if !seen[key] {
					seen[key] = true
					found = append(found, key)
				}
				break
			}
		}
	}
	return found
}
