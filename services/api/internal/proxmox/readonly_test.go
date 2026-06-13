package proxmox

import (
	"reflect"
	"strings"
	"testing"
)

func TestProxmoxClientExposesOnlyAllowedMethods(t *testing.T) {
	// Verify the ProxmoxClient interface has only allowed methods
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

	// Verify allowed mutation methods exist (v1.0)
	for _, expected := range []string{"StartVM", "ShutdownVM", "StopVM", "SnapshotVM", "GetTaskStatus"} {
		if !methods[expected] {
			t.Errorf("expected mutation method %s not found on ProxmoxClient", expected)
		}
	}

	// Verify FORBIDDEN mutation methods do NOT exist
	forbidden := []string{
		"DeleteVM", "MigrateVM", "CloneVM", "ResetVM", "RebootVM",
		"ResizeVM", "SetConfig", "UpdateConfig",
		"ModifyFirewall", "UpdateNetwork", "ModifyStorage",
		"CreateCertificate", "BulkStart", "BulkStop", "BulkShutdown",
	}
	for _, name := range forbidden {
		if methods[name] {
			t.Errorf("forbidden method %s found on ProxmoxClient — must not exist", name)
		}
	}
}

func TestFakeProxmoxClientImplementsInterface(t *testing.T) {
	// Compile-time check
	var _ ProxmoxClient = &FakeProxmoxClient{}
}

func TestHandlerExposesNoForbiddenMethods(t *testing.T) {
	handlerType := reflect.TypeOf(&Handler{}).Elem()

	forbiddenNames := map[string]bool{
		"Delete": true, "Migrate": true, "Clone": true, "Reset": true,
		"Reboot": true, "Suspend": true, "Resize": true,
		"Firewall": true, "Network": true, "SetConfig": true,
		"BulkAction": true, "ModifyCertificate": true,
	}
	for i := 0; i < handlerType.NumMethod(); i++ {
		name := handlerType.Method(i).Name
		if forbiddenNames[name] {
			t.Errorf("Handler has forbidden method %s", name)
		}
	}
}

func TestProxmoxTokenNeverInHandlerFields(t *testing.T) {
	// Verify the Handler struct has no token/secret fields
	handlerType := reflect.TypeOf(&Handler{}).Elem()
	for i := 0; i < handlerType.NumField(); i++ {
		field := handlerType.Field(i)
		name := strings.ToLower(field.Name)
		if strings.Contains(name, "token") || strings.Contains(name, "secret") || strings.Contains(name, "password") || strings.Contains(name, "apikey") {
			t.Errorf("Handler has sensitive field %s (%s) — token/secret must not be stored", field.Name, field.Type)
		}
	}
}
