package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/flowinterop"
	"github.com/matiasleandrokruk/fenix/internal/domain/governance"
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
	settings, err := loadCrossPlatformRuntimeSettings()
	if err != nil {
		t.Fatalf("loadCrossPlatformRuntimeSettings: %v", err)
	}
	if err := configureCrossPlatformRuntime(registry, settings); err != nil {
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
		envM2SFEvidenceEnabled,
	} {
		t.Setenv(key, "")
	}
}
func TestLoadCrossPlatformRuntimeSettingsOptionalEvidenceRequiresVEL(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv(envM2SFURL, "http://127.0.0.1:4000")
	t.Setenv(envM2SFToken, "test-token")
	t.Setenv(envM2SFEvidenceEnabled, "true")

	_, err := loadCrossPlatformRuntimeSettings()
	if !errors.Is(err, errIntegrationConfig) {
		t.Fatalf("expected integration config error, got %v", err)
	}
}

func TestCrossPlatformPolicySelectorPlansOnlyOptionalEvidence(t *testing.T) {
	selector := crossPlatformPolicySelector{optionalM2SFEvidence: true}

	optional, err := selector.SelectRuntimePolicy(
		context.Background(),
		governance.Profile{EvidenceRequirement: governance.EvidenceOptional},
	)
	if err != nil {
		t.Fatalf("SelectRuntimePolicy optional: %v", err)
	}
	if !optional.OptionalEvidence {
		t.Fatal("expected optional evidence selection")
	}

	none, err := selector.SelectRuntimePolicy(
		context.Background(),
		governance.Profile{EvidenceRequirement: governance.EvidenceNone},
	)
	if err != nil {
		t.Fatalf("SelectRuntimePolicy none: %v", err)
	}
	if none.OptionalEvidence {
		t.Fatal("evidence NONE must not be selected")
	}
}

