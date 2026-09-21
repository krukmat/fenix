package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
	"github.com/matiasleandrokruk/fenix/internal/domain/flowinterop"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	m2sfintegration "github.com/matiasleandrokruk/fenix/internal/infra/integration/m2sf"
	velintegration "github.com/matiasleandrokruk/fenix/internal/infra/integration/vel"
)

const (
	envM2SFURL           = "FENIX_M2SF_URL"
	envM2SFToken         = "FENIX_M2SF_TOKEN"
	envVELURL            = "FENIX_VEL_URL"
	envVELToken          = "FENIX_VEL_TOKEN"
	envIntegrationMS     = "FENIX_INTEGRATION_TIMEOUT_MS"
	defaultIntegrationMS = 15000
)

var errIntegrationConfig = errors.New("invalid cross-platform integration configuration")

type providerRuntimeSettings struct {
	BaseURL string
	Token   string
	Enabled bool
}

type crossPlatformRuntimeSettings struct {
	M2SF    providerRuntimeSettings
	VEL     providerRuntimeSettings
	Timeout time.Duration
}

var flowCapabilityInputSchema = json.RawMessage("{\"type\":\"object\",\"required\":[\"contract_version\",\"operation\",\"input\"],\"properties\":{\"contract_version\":{\"type\":\"string\"},\"operation\":{\"type\":\"string\"},\"input\":{\"type\":\"object\"},\"compare_to\":{\"type\":\"object\"}},\"additionalProperties\":false}")

func configureCrossPlatformRuntime(registry *tool.ToolRegistry) error {
	settings, err := loadCrossPlatformRuntimeSettings()
	if err != nil {
		return err
	}
	if settings.M2SF.Enabled {
		if err := registerM2SFCapabilities(registry, settings); err != nil {
			return err
		}
	}
	if settings.VEL.Enabled {
		if err := configureVELEvidence(registry, settings); err != nil {
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
		if err := registry.RegisterCapabilityWithDefinition(
			descriptor,
			executor,
			definition,
		); err != nil {
			return fmt.Errorf("api: register M2SF capability %s: %w", descriptor.Name, err)
		}
	}
	return nil
}

func configureVELEvidence(
	registry *tool.ToolRegistry,
	settings crossPlatformRuntimeSettings,
) error {
	sink, err := velintegration.NewSink(
		settings.VEL.BaseURL,
		settings.VEL.Token,
		&http.Client{Timeout: settings.Timeout},
	)
	if err != nil {
		return fmt.Errorf("api: create VEL sink: %w", err)
	}
	registry.SetCapabilityEvidenceRecorder(evidence.NewRuntimeRecorder(sink))
	return nil
}

func loadCrossPlatformRuntimeSettings() (crossPlatformRuntimeSettings, error) {
	m2sf, err := providerSettingsFromEnv(envM2SFURL, envM2SFToken)
	if err != nil {
		return crossPlatformRuntimeSettings{}, fmt.Errorf("%w: M2SF: %v", errIntegrationConfig, err)
	}
	vel, err := providerSettingsFromEnv(envVELURL, envVELToken)
	if err != nil {
		return crossPlatformRuntimeSettings{}, fmt.Errorf("%w: VEL: %v", errIntegrationConfig, err)
	}
	timeout, err := integrationTimeout()
	if err != nil {
		return crossPlatformRuntimeSettings{}, err
	}
	return crossPlatformRuntimeSettings{M2SF: m2sf, VEL: vel, Timeout: timeout}, nil
}

func providerSettingsFromEnv(urlKey, tokenKey string) (providerRuntimeSettings, error) {
	baseURL := strings.TrimSpace(os.Getenv(urlKey))
	token := strings.TrimSpace(os.Getenv(tokenKey))
	if baseURL == "" && token == "" {
		return providerRuntimeSettings{}, nil
	}
	if baseURL == "" || token == "" {
		return providerRuntimeSettings{}, errors.New("URL and token must be configured together")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return providerRuntimeSettings{}, errors.New("URL must use http or https")
	}
	return providerRuntimeSettings{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		Enabled: true,
	}, nil
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
