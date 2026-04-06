package gobdd

import (
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format: "pretty",
			Paths:  []string{filepath.Join("..", "..", "..", "features")},
			Tags:   "@stack-go && not @deferred",
		},
	}

	if suite.Run() != 0 {
		t.Fatal("go BDD suite failed")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &scenarioState{}

	ctx.BeforeScenario(func(*godog.Scenario) {
		state.reset()
	})

	registerBaselineScenarioSteps(ctx, state)
	registerSalesCopilotScenarios(ctx, state)
	registerSupportAgentScenarios(ctx, state)
	registerGovernanceScenarios(ctx, state)
	initWorkflowRuntimeScenarios(ctx, state)
}
