package proxmox

import (
	"reflect"
	"strings"
	"testing"
)

func TestProxmoxClientExposesNoMutations(t *testing.T) {
	// Verify the ProxmoxClient interface has only read methods
	iface := reflect.TypeOf((*ProxmoxClient)(nil)).Elem()

	// Collect all method names
	methods := make(map[string]bool)
	for i := 0; i < iface.NumMethod(); i++ {
		methods[iface.Method(i).Name] = true
	}

	// Verify expected read methods exist
	for _, expected := range []string{"ListNodes", "ListVMs"} {
		if !methods[expected] {
			t.Errorf("expected read method %s not found on ProxmoxClient", expected)
		}
	}

	// Verify only read methods exist (List* prefix only)
	for name := range methods {
		if !strings.HasPrefix(name, "List") && !strings.HasPrefix(name, "Get") {
			t.Errorf("non-read method %s found on ProxmoxClient — interface must be read-only", name)
		}
	}
}

func TestFakeProxmoxClientImplementsReadonly(t *testing.T) {
	// Compile-time check
	var _ ProxmoxClient = &FakeProxmoxClient{}
}

func TestHandlerExposesNoMutationMethods(t *testing.T) {
	handlerType := reflect.TypeOf(&Handler{}).Elem()

	mutationNames := map[string]bool{
		"Start": true, "Stop": true, "Restart": true, "Migrate": true,
		"DeleteVM": true, "Reboot": true, "Suspend": true, "Resize": true,
		"Snapshot": true, "Clone": true, "Firewall": true, "Network": true,
		"SetConfig": true, "Update": true,
	}
	for i := 0; i < handlerType.NumMethod(); i++ {
		name := handlerType.Method(i).Name
		if mutationNames[name] {
			t.Errorf("Handler has mutation method %s — must be read-only", name)
		}
	}
}

func TestProxmoxTokenNeverInAudit(t *testing.T) {
	// This is a design assertion — the Proxmox handler never receives tokens directly,
	// only a ProxmoxClient interface. Verify the Handler struct has no token fields.
	handlerType := reflect.TypeOf(&Handler{}).Elem()
	for i := 0; i < handlerType.NumField(); i++ {
		field := handlerType.Field(i)
		name := strings.ToLower(field.Name)
		if strings.Contains(name, "token") || strings.Contains(name, "secret") || strings.Contains(name, "password") || strings.Contains(name, "apikey") {
			t.Errorf("Handler has sensitive field %s (%s) — token/secret must not be stored", field.Name, field.Type)
		}
	}
}
