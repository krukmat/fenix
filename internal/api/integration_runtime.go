package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/flowinterop"
	"github.com/matiasleandrokruk/fenix/internal/domain/governance"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	evidenceoutbox "github.com/matiasleandrokruk/fenix/internal/infra/integration/evidenceoutbox"
	m2sfintegration "github.com/matiasleandrokruk/fenix/internal/infra/integration/m2sf"
	velintegration "github.com/matiasleandrokruk/fenix/internal/infra/integration/vel"
)

const (
	envM2SFURL             = "FENIX_M2SF_URL"
	envM2SFToken           = "FENIX_M2SF_TOKEN" // #nosec G101 -- environment variable name, not a credential
	envVELURL              = "FENIX_VEL_URL"
	envVELToken            = "FENIX_VEL_TOKEN" // #nosec G101 -- environment variable name, not a credential
	envIntegrationMS       = "FENIX_INTEGRATION_TIMEOUT_MS"
	envM2SFEvidenceEnabled = "FENIX_M2SF_EVIDENCE_ENABLED"
	defaultIntegrationMS   = 15000
)

var errIntegrationConfig = errors.New("invalid cross-platform integration configuration")

type providerRuntimeSettings struct {
	BaseURL string
	Token   string
	Enabled bool
}

type crossPlatformRuntimeSettings struct {
	M2SF                 providerRuntimeSettings
	VEL                  providerRuntimeSettings
	Timeout              time.Duration
	M2SFOptionalEvidence bool
}

var flowCapabilityInputSchema = json.RawMessage("{\"type\":\"object\",\"required\":[\"contract_version\",\"operation\",\"input\"],\"properties\":{\"contract_version\":{\"type\":\"string\"},\"operation\":{\"type\":\"string\"},\"input\":{\"type\":\"object\"},\"compare_to\":{\"type\":\"object\"}},\"additionalProperties\":false}")

func configureCrossPlatformRuntime(
	registry *tool.ToolRegistry,
	settings crossPlatformRuntimeSettings,
) error {
	return configureCrossPlatformRuntimeWithServices(registry, settings, nil, RouterRuntime{})
}

func configureCrossPlatformRuntimeWithServices(
	registry *tool.ToolRegistry,
	settings crossPlatformRuntimeSettings,
	db *sql.DB,
	runtime RouterRuntime,
) error {
	if settings.M2SF.Enabled {
		if err := registerM2SFCapabilities(registry, settings); err != nil {
			return err
		}
	}
	if settings.VEL.Enabled {
		if err := configureVELEvidence(registry, settings, db, runtime); err != nil {
			return err
		}
	}
	return nil
}

func registerM2SFCapabilities(
	registry *tool.ToolRegistry,
	settings crossPlatformRuntimeSettings,
) error {
	client := &http.Client{Timeout: settings.Timeout}
	for _, spec := range flowinterop.Catalog() {
		executor, err := m2sfintegration.NewExecutor(
			settings.M2SF.BaseURL,
			settings.M2SF.Token,
			flowinterop.Operation(spec.Descriptor.Name),
			client,
		)
		if err != nil {
			return fmt.Errorf("api: create M2SF executor %s: %w", spec.Descriptor.Name, err)
		}
		descriptor := spec.Descriptor
		descriptor.RetryPolicy = tool.CapabilityRetryPolicy{MaxAttempts: 2}
		definition := tool.CapabilityToolDefinition{
			Description:         "Governed Mermaid2SF capability " + descriptor.Name,
			InputSchema:         flowCapabilityInputSchema,
			RequiredPermissions: []string{"tools:" + descriptor.Name},
		}
		if registerErr := registry.RegisterCapabilityWithDefinition(
			descriptor,
			executor,
			definition,
		); registerErr != nil {
			return fmt.Errorf("api: register M2SF capability %s: %w", descriptor.Name, registerErr)
		}
	}
	return nil
}

func configureVELEvidence(
	registry *tool.ToolRegistry,
	settings crossPlatformRuntimeSettings,
	db *sql.DB,
	runtime RouterRuntime,
) error {
	if db == nil || runtime.BackgroundContext == nil || runtime.StartBackground == nil {
		return fmt.Errorf("%w: VEL durable evidence runtime is unavailable", errIntegrationConfig)
	}
	sink, err := velintegration.NewSink(
		settings.VEL.BaseURL,
		settings.VEL.Token,
		&http.Client{Timeout: settings.Timeout},
	)
	if err != nil {
		return fmt.Errorf("api: create VEL sink: %w", err)
	}
	recorder, err := evidenceoutbox.NewRecorder(db, sink)
	if err != nil {
		return fmt.Errorf("api: create VEL durable recorder: %w", err)
	}
	registry.SetCapabilityEvidenceRecorder(recorder)
	runtime.StartBackground(func() {
		recorder.Start(runtime.BackgroundContext)
	})
	return nil
}

func loadCrossPlatformRuntimeSettings() (crossPlatformRuntimeSettings, error) {
	m2sf, err := providerSettingsFromEnv(envM2SFURL, envM2SFToken)
	if err != nil {
		return crossPlatformRuntimeSettings{}, fmt.Errorf("%w: M2SF: %w", errIntegrationConfig, err)
	}
	vel, err := providerSettingsFromEnv(envVELURL, envVELToken)
	if err != nil {
		return crossPlatformRuntimeSettings{}, fmt.Errorf("%w: VEL: %w", errIntegrationConfig, err)
	}
	timeout, err := integrationTimeout()
	if err != nil {
		return crossPlatformRuntimeSettings{}, err
	}
	optionalEvidence, err := optionalM2SFEvidence()
	if err != nil {
		return crossPlatformRuntimeSettings{}, err
	}
	if optionalEvidence && !vel.Enabled {
		return crossPlatformRuntimeSettings{}, fmt.Errorf(
			"%w: M2SF optional evidence requires VEL configuration",
			errIntegrationConfig,
		)
	}
	return crossPlatformRuntimeSettings{
		M2SF:                 m2sf,
		VEL:                  vel,
		Timeout:              timeout,
		M2SFOptionalEvidence: optionalEvidence,
	}, nil
}

func providerSettingsFromEnv(urlKey, tokenKey string) (providerRuntimeSettings, error) {
	baseURL := strings.TrimSpace(os.Getenv(urlKey))
	token := strings.TrimSpace(os.Getenv(tokenKey))
	if providerConfigEmpty(baseURL, token) {
		return providerRuntimeSettings{}, nil
	}
	if validationErr := validateProviderConfig(baseURL, token); validationErr != nil {
		return providerRuntimeSettings{}, validationErr
	}
	return providerRuntimeSettings{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		Enabled: true,
	}, nil
}

func providerConfigEmpty(baseURL, token string) bool {
	return baseURL == "" && token == ""
}

func validateProviderConfig(baseURL, token string) error {
	if baseURL == "" || token == "" {
		return errors.New("URL and token must be configured together")
	}
	return validateProviderURL(baseURL)
}

func validateProviderURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || !validProviderScheme(parsed.Scheme) {
		return errors.New("URL must use http or https")
	}
	return nil
}

func validProviderScheme(scheme string) bool {
	return scheme == "http" || scheme == "https"
}

func integrationTimeout() (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(envIntegrationMS))
	if raw == "" {
		return defaultIntegrationMS * time.Millisecond, nil
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 100 || ms > 120000 {
		return 0, fmt.Errorf("%w: integration timeout must be 100..120000 ms", errIntegrationConfig)
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func optionalM2SFEvidence() (bool, error) {
	raw := strings.TrimSpace(os.Getenv(envM2SFEvidenceEnabled))
	if raw == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf(
			"%w: %s must be a boolean",
			errIntegrationConfig,
			envM2SFEvidenceEnabled,
		)
	}
	return enabled, nil
}

type crossPlatformPolicySelector struct {
	optionalM2SFEvidence bool
}

func (s crossPlatformPolicySelector) SelectRuntimePolicy(
	_ context.Context,
	profile governance.Profile,
) (governance.RuntimePolicySelection, error) {
	return governance.RuntimePolicySelection{
		Allowed:          true,
		PolicyReference:  "fenix:w1-policy-gate",
		OptionalEvidence: s.optionalM2SFEvidence &&
			profile.EvidenceRequirement == governance.EvidenceOptional,
	}, nil
}
