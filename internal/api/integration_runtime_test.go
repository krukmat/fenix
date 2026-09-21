package api

import (
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/flowinterop"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

func TestLoadCrossPlatformRuntimeSettingsDefaultsDisabled(t *testing.T) {
	clearIntegrationEnv(t)

	settings, err := loadCrossPlatformRuntimeSettings()
	if err != nil {
		t.Fatalf("loadCrossPlatformRuntimeSettings: %v", err)
	}
	if settings.M2SF.Enabled || settings.VEL.Enabled {
		t.Fatalf("providers unexpectedly enabled: %#v", settings)
	}
	if settings.Timeout != 15*time.Second {
		t.Fatalf("timeout = %v", settings.Timeout)
	}
}

func TestLoadCrossPlatformRuntimeSettingsFailsClosedOnPartialProviderConfig(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv(envM2SFURL, "http://127.0.0.1:4000")

	_, err := loadCrossPlatformRuntimeSettings()
	if !errors.Is(err, errIntegrationConfig) {
		t.Fatalf("expected integration config error, got %v", err)
	}
}

func TestConfigureCrossPlatformRuntimeRegistersM2SFCapabilities(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv(envM2SFURL, "http://127.0.0.1:4000")
	t.Setenv(envM2SFToken, "test-token")
	t.Setenv(envIntegrationMS, "2500")

	registry := tool.NewToolRegistry(nil)
	if err := configureCrossPlatformRuntime(registry); err != nil {
		t.Fatalf("configureCrossPlatformRuntime: %v", err)
	}

	for _, operation := range []flowinterop.Operation{
		flowinterop.OperationImport,
		flowinterop.OperationExport,
		flowinterop.OperationValidate,
		flowinterop.OperationCompare,
	} {
		if _, err := registry.Get(string(operation)); err != nil {
			t.Fatalf("capability %s not registered: %v", operation, err)
		}
	}
}

func clearIntegrationEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		envM2SFURL,
		envM2SFToken,
		envVELURL,
		envVELToken,
		envIntegrationMS,
	} {
		t.Setenv(key, "")
	}
}
