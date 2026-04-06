package gobdd

import (
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

func registerSalesCopilotScenarios(ctx *godog.ScenarioContext, state *scenarioState) {
	ctx.Step(`^an account record has grounded CRM timeline evidence$`, func() error {
		return setupSalesBriefScenario(state, "account", knowledge.ConfidenceHigh, []float64{0.92, 0.88}, []string{
			`{"summary":"Account renewal is blocked by procurement.","risks":["Procurement delay","Missing executive sponsor"]}`,
			`{"actions":[
				{"title":"Schedule sponsor review","description":"Align next step","tool":"create_task","params":{"entity_type":"account","entity_id":"placeholder"}},
				{"title":"Confirm renewal timeline","description":"Validate blockers","tool":"create_task","params":{"entity_type":"account","entity_id":"placeholder"}},
				{"title":"Prepare commercial follow-up","description":"Coordinate owner response","tool":"create_task","params":{"entity_type":"account","entity_id":"placeholder"}}
			]}`,
		})
	})
	ctx.Step(`^the workspace user requests a sales brief for the account$`, func() error {
		return requestSalesBrief(state, "account")
	})
	ctx.Step(`^the sales brief outcome is completed$`, func() error {
		return expectSalesBriefOutcome(state, "completed")
	})
	ctx.Step(`^the sales brief summary reflects the account context$`, func() error {
		return expectSalesBriefSummaryContains(state, "Account renewal")
	})
	ctx.Step(`^the sales brief includes evidence-backed next best actions$`, func() error {
		return expectSalesBriefActions(state, 1)
	})

	ctx.Step(`^a deal record has grounded stage, owner, and activity evidence$`, func() error {
		return setupSalesBriefScenario(state, "deal", knowledge.ConfidenceHigh, []float64{0.94, 0.86}, []string{
			`{"summary":"Deal stalled after a pricing objection.","risks":["Pricing pressure","Procurement review"]}`,
			fmt.Sprintf(`{"actions":[
				{"title":"Advance deal stage","description":"Move to proposal review","tool":"update_deal","params":{"deal_id":"%s","stage_id":"stage_follow_up"}},
				{"title":"Coordinate buyer call","description":"Create owner follow-up","tool":"create_task","params":{"entity_type":"deal","entity_id":"%s"}},
				{"title":"Document pricing objection","description":"Create mitigation follow-up","tool":"create_task","params":{"entity_type":"deal","entity_id":"%s"}}
			]}`, placeholderEntityID, placeholderEntityID, placeholderEntityID),
		})
	})
	ctx.Step(`^the workspace user requests a sales brief for the deal$`, func() error {
		return requestSalesBrief(state, "deal")
	})
	ctx.Step(`^the sales brief summary reflects the deal context$`, func() error {
		return expectSalesBriefSummaryContains(state, "Deal stalled")
	})

	ctx.Step(`^a CRM record lacks enough grounded evidence for a sales brief$`, func() error {
		return setupSalesBriefScenario(state, "account", knowledge.ConfidenceLow, []float64{}, nil)
	})
	ctx.Step(`^the workspace user requests a sales brief for the record$`, func() error {
		return requestSalesBrief(state, "account")
	})
	ctx.Step(`^the sales brief outcome is abstained$`, func() error {
		return expectSalesBriefOutcome(state, "abstained")
	})
	ctx.Step(`^the sales brief explains that more evidence is required$`, func() error {
		data, err := salesBriefData(state)
		if err != nil {
			return err
		}
		reason, _ := data["abstentionReason"].(string)
		if reason == "" {
			return fmt.Errorf("expected abstentionReason in sales brief")
		}
		return nil
	})

	ctx.Step(`^a completed sales brief is requested for a CRM record$`, func() error {
		if err := setupSalesBriefScenario(state, "deal", knowledge.ConfidenceHigh, []float64{0.91, 0.84}, []string{
			`{"summary":"Deal still has budget risk.","risks":["Budget freeze"]}`,
			fmt.Sprintf(`{"actions":[
				{"title":"Open budget thread","description":"Align finance stakeholders","tool":"create_task","params":{"entity_type":"deal","entity_id":"%s"}},
				{"title":"Refresh proposal","description":"Update commercial package","tool":"create_task","params":{"entity_type":"deal","entity_id":"%s"}},
				{"title":"Move follow-up stage","description":"Advance to next stage","tool":"update_deal","params":{"deal_id":"%s","stage_id":"stage_budget_review"}}
			]}`, placeholderEntityID, placeholderEntityID, placeholderEntityID),
		}); err != nil {
			return err
		}
		return requestSalesBrief(state, "deal")
	})
	ctx.Step(`^the response exposes the evidence pack contract$`, func() error {
		data, err := salesBriefData(state)
		if err != nil {
			return err
		}
		evidencePack, ok := data["evidencePack"].(map[string]any)
		if !ok {
			return fmt.Errorf("expected evidencePack payload")
		}
		if evidencePack["schema_version"] != knowledge.EvidencePackSchemaVersion {
			return fmt.Errorf("schema_version = %v, want %s", evidencePack["schema_version"], knowledge.EvidencePackSchemaVersion)
		}
		if _, ok := evidencePack["built_at"].(string); !ok {
			return fmt.Errorf("expected built_at in evidence pack")
		}
		return nil
	})
	ctx.Step(`^the response exposes the proposed next best actions$`, func() error {
		return expectSalesBriefActions(state, 1)
	})
}

const placeholderEntityID = "placeholder"

func setupSalesBriefScenario(state *scenarioState, entityType string, confidence knowledge.ConfidenceLevel, scores []float64, llmResponses []string) error {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return err
	}
	runtime.llm.responses = nil
	runtime.evidence.packs = map[string]*knowledge.EvidencePack{}

	var entityID string
	switch entityType {
	case "account":
		entityID, err = runtime.createSalesAccount()
	case "deal":
		entityID, err = runtime.createSalesDeal()
	default:
		err = fmt.Errorf("unsupported entity type %s", entityType)
	}
	if err != nil {
		return err
	}

	query := salesEvidenceQuery(entityType, entityID)
	pack := newBDDEvidencePack(query, confidence, scores...)
	runtime.evidence.set(query, pack)

	for _, response := range llmResponses {
		runtime.llm.queue(stringsReplaceEntityPlaceholder(response, entityID))
	}

	state.lastEntityID = entityID
	return nil
}

func requestSalesBrief(state *scenarioState, entityType string) error {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return err
	}
	status, body, err := runtime.request("POST", "/api/v1/copilot/sales-brief", runtime.userID, map[string]any{
		"entityType": entityType,
		"entityId":   state.lastEntityID,
	})
	if err != nil {
		return err
	}
	state.lastStatusCode = status
	state.lastResponseBody = body
	return nil
}

func expectSalesBriefOutcome(state *scenarioState, want string) error {
	data, err := salesBriefData(state)
	if err != nil {
		return err
	}
	if got, _ := data["outcome"].(string); got != want {
		return fmt.Errorf("sales brief outcome = %q, want %q", got, want)
	}
	return nil
}

func expectSalesBriefSummaryContains(state *scenarioState, needle string) error {
	data, err := salesBriefData(state)
	if err != nil {
		return err
	}
	summary, _ := data["summary"].(string)
	if summary == "" || !contains(summary, needle) {
		return fmt.Errorf("sales brief summary = %q, want substring %q", summary, needle)
	}
	return nil
}

func expectSalesBriefActions(state *scenarioState, min int) error {
	data, err := salesBriefData(state)
	if err != nil {
		return err
	}
	actions, ok := data["nextBestActions"].([]any)
	if !ok || len(actions) < min {
		return fmt.Errorf("nextBestActions = %#v, want at least %d actions", data["nextBestActions"], min)
	}
	return nil
}

func salesBriefData(state *scenarioState) (map[string]any, error) {
	if state.lastStatusCode != 200 {
		return nil, fmt.Errorf("last sales brief status = %d, want 200", state.lastStatusCode)
	}
	decoded, err := decodeBDDEnvelope(state.lastResponseBody)
	if err != nil {
		return nil, err
	}
	data, ok := decoded["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing data envelope")
	}
	return data, nil
}

func stringsReplaceEntityPlaceholder(raw, entityID string) string {
	return strings.ReplaceAll(raw, placeholderEntityID, entityID)
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
