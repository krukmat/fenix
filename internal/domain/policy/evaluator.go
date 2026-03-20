package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

// Cache is the minimal cache contract used by PolicyEngine.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration)
}

// Filter represents a SQL WHERE fragment + args for permission filtering.
type Filter struct {
	Where string
	Args  []any
}

// PolicyDecisionTrace captures deterministic rule resolution context.
//nolint:revive // API name kept for backward compatibility in policy package.
type PolicyDecisionTrace struct {
	PolicySetID      string
	PolicyVersionID  string
	PolicyVersionNum int64
	Resource         string
	Action           string
	MatchedRuleID    string
	MatchedEffect    string
	RuleTrace        []string
}

// PolicyDecision contains final allow/deny and optional trace metadata.
//nolint:revive // API name kept for backward compatibility in policy package.
type PolicyDecision struct {
	Allow bool
	Trace *PolicyDecisionTrace
}

// AuditLogEvent captures the minimum event data required for EP4 logging.
type AuditLogEvent struct {
	WorkspaceID string
	ActorID     string
	ActorType   audit.ActorType
	Action      string
	EntityType  *string
	EntityID    *string
	Outcome     audit.Outcome
	Details     map[string]any
}

// PolicyEngine implements the Week 7, Task 3.1 enforcement points.
//
//nolint:revive // nombre mantenido por compatibilidad interna del módulo
type PolicyEngine struct {
	db    *sql.DB
	cache Cache
	audit *audit.AuditService
}

type policyRule struct {
	ID         string         `json:"id"`
	Resource   string         `json:"resource"`
	Action     string         `json:"action"`
	Effect     string         `json:"effect"`
	Priority   int            `json:"priority"`
	Conditions map[string]any `json:"conditions"`
}

type policyDoc struct {
	Rules []policyRule `json:"rules"`
}

type activePolicyVersion struct {
	PolicySetID    string
	PolicyVersion  string
	VersionNumber  int64
	RawPolicyRules string
}

func NewPolicyEngine(db *sql.DB, cache Cache, auditService *audit.AuditService) *PolicyEngine {
	if auditService == nil {
		auditService = audit.NewAuditService(db)
	}
	return &PolicyEngine{db: db, cache: cache, audit: auditService}
}

// BuildPermissionFilter (EP1): builds a SQL filter for retrieval paths.
// Baseline: always isolate by workspace. If user has no broad-read grant,
// restricts further to owner_id = userID.
func (p *PolicyEngine) BuildPermissionFilter(ctx context.Context, userID string) (Filter, error) {
	workspaceID, rolePerms, err := p.loadUserWorkspaceAndRolePermissions(ctx, userID)
	if err != nil {
		return Filter{}, err
	}

	if hasPermission(rolePerms, permGlobal, "read_all") || hasPermission(rolePerms, "records", "read_all") {
		return Filter{Where: "workspace_id = ?", Args: []any{workspaceID}}, nil
	}

	return Filter{Where: "workspace_id = ? AND owner_id = ?", Args: []any{workspaceID, userID}}, nil
}

var (
	emailRe = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	phoneRe = regexp.MustCompile(`\b(?:\+?\d{1,3}[\s.-]?)?(?:\(?\d{2,4}\)?[\s.-]?)?\d{3}[\s.-]?\d{3,4}\b`)
	ssnRe   = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
)

const (
	permGlobal = "global"
	permAPI    = "api"
	permTools  = "tools"

	auditActionPolicyEvaluated = "policy.evaluated"

	toolsPrefix = "tools:"
)

// RedactPII (EP2): redacts PII in evidence snippets.
// Reverse mapping is cached (if cache is configured) for controlled post-processing.
func (p *PolicyEngine) RedactPII(ctx context.Context, evidence []knowledge.Evidence) ([]knowledge.Evidence, error) {
	if len(evidence) == 0 {
		return evidence, nil
	}

	redacted := make([]knowledge.Evidence, len(evidence))
	copy(redacted, evidence)

	reverseMap := map[string]string{}
	countByType := map[string]int{"EMAIL": 0, "PHONE": 0, "SSN": 0}

	for i := range redacted {
		redactEvidenceSnippet(&redacted[i], reverseMap, countByType)
	}

	p.cacheReverseMap(ctx, reverseMap)

	return redacted, nil
}

// CheckToolPermission (EP3): verifies user permissions before tool execution.
func (p *PolicyEngine) CheckToolPermission(ctx context.Context, userID, toolID string) (bool, error) {
	requiredPerms, err := p.loadToolRequiredPermissions(ctx, toolID)
	if err != nil {
		return false, err
	}

	for _, req := range requiredPerms {
		allowed, checkErr := p.anyActionAllowed(ctx, userID, candidateToolActions(req))
		if checkErr != nil {
			return false, checkErr
		}
		if allowed {
			return true, nil
		}
	}

	return false, nil
}

// anyActionAllowed returns true if the user is allowed any of the given actions on the tools resource.
func (p *PolicyEngine) anyActionAllowed(ctx context.Context, userID string, actions []string) (bool, error) {
	for _, action := range actions {
		ok, err := p.CheckActionPermission(ctx, userID, permTools, action, nil)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// CheckAgentPermission verifies whether a user may dispatch a sub-agent.
func (p *PolicyEngine) CheckAgentPermission(ctx context.Context, userID, agentID, agentName string) (bool, error) {
	attrs := map[string]string{}
	if strings.TrimSpace(agentID) != "" {
		attrs["agent_id"] = strings.TrimSpace(agentID)
	}
	if strings.TrimSpace(agentName) != "" {
		attrs["agent_name"] = strings.TrimSpace(agentName)
	}
	return p.CheckActionPermission(ctx, userID, "agents", "execute", attrs)
}

// CheckActionPermission performs unified RBAC/ABAC evaluation for API/tool actions.
// 1) If an active policy exists, its decision is authoritative.
// 2) If no active policy is available, fallback to role permissions.
func (p *PolicyEngine) CheckActionPermission(
	ctx context.Context,
	userID, resource, action string,
	attrs map[string]string,
) (bool, error) {
	workspaceID, rolePerms, err := p.loadUserWorkspaceAndRolePermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	decision, decisionErr := p.EvaluatePolicyDecision(ctx, workspaceID, userID, resource, action, attrs)
	if decisionErr != nil {
		return false, decisionErr
	}
	if decision.Trace != nil {
		return decision.Allow, nil
	}

	return roleAllowsAction(rolePerms, resource, action), nil
}

// EvaluatePolicyDecision resolves allow/deny for resource+action against active policy set/version.
// If no active policy is found or no rule matches, Trace is nil and callers may fallback.
func (p *PolicyEngine) EvaluatePolicyDecision(
	ctx context.Context,
	workspaceID, actorID, resource, action string,
	attrs map[string]string,
) (PolicyDecision, error) {
	active, ok, err := p.loadActivePolicyVersion(ctx, workspaceID)
	if err != nil {
		return PolicyDecision{}, err
	}
	if !ok {
		return PolicyDecision{Allow: false, Trace: nil}, nil
	}

	rules, err := parsePolicyRules(active.RawPolicyRules)
	if err != nil {
		return PolicyDecision{}, err
	}

	matched := filterMatchingRules(rules, resource, action, attrs)
	if len(matched) == 0 {
		decision := PolicyDecision{
			Allow: false,
			Trace: &PolicyDecisionTrace{
				PolicySetID:      active.PolicySetID,
				PolicyVersionID:  active.PolicyVersion,
				PolicyVersionNum: active.VersionNumber,
				Resource:         resource,
				Action:           action,
				MatchedEffect:    decisionDeny,
				RuleTrace:        []string{},
			},
		}
		p.logPolicyDecision(ctx, workspaceID, actorID, decision)
		return decision, nil
	}

	resolved := resolveDeterministicRule(matched)

	trace := &PolicyDecisionTrace{
		PolicySetID:      active.PolicySetID,
		PolicyVersionID:  active.PolicyVersion,
		PolicyVersionNum: active.VersionNumber,
		Resource:         resource,
		Action:           action,
		MatchedRuleID:    resolved.ID,
		MatchedEffect:    normalizeEffect(resolved.Effect),
		RuleTrace:        makeRuleTrace(matched),
	}

	decision := PolicyDecision{Allow: normalizeEffect(resolved.Effect) == "allow", Trace: trace}
	p.logPolicyDecision(ctx, workspaceID, actorID, decision)

	return decision, nil
}

// LogAuditEvent (EP4): appends an immutable audit event.
func (p *PolicyEngine) LogAuditEvent(ctx context.Context, event AuditLogEvent) error {
	var details *audit.EventDetails
	if len(event.Details) > 0 {
		details = &audit.EventDetails{Metadata: event.Details}
	}

	return p.audit.LogWithDetails(
		ctx,
		event.WorkspaceID,
		event.ActorID,
		event.ActorType,
		event.Action,
		event.EntityType,
		event.EntityID,
		details,
		event.Outcome,
	)
}

func redactWithToken(
	input, tokenType string,
	re *regexp.Regexp,
	reverseMap map[string]string,
	countByType map[string]int,
) (string, bool) {
	changed := false
	output := re.ReplaceAllStringFunc(input, func(match string) string {
		countByType[tokenType]++
		token := fmt.Sprintf("[%s_%d]", tokenType, countByType[tokenType])
		reverseMap[token] = match
		changed = true
		return token
	})
	return output, changed
}

func (p *PolicyEngine) loadUserWorkspaceAndRolePermissions(
	ctx context.Context,
	userID string,
) (string, map[string][]string, error) {
	workspaceID, err := p.loadWorkspaceID(ctx, userID)
	if err != nil {
		return "", nil, err
	}

	rawPerms, err := p.loadRolePermissionRows(ctx, userID, workspaceID)
	if err != nil {
		return "", nil, err
	}

	return workspaceID, mergePermissions(rawPerms), nil
}

func (p *PolicyEngine) loadToolRequiredPermissions(ctx context.Context, toolID string) ([]string, error) {
	exists, err := p.tableExists(ctx, "tool_definition")
	if err != nil {
		return nil, err
	}
	if !exists {
		return []string{toolID}, nil
	}

	raw, ok, err := p.fetchToolRequiredPermissions(ctx, toolID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []string{toolID}, nil
	}

	return parseRequiredPermissions(raw, toolID), nil
}

func redactEvidenceSnippet(item *knowledge.Evidence, reverseMap map[string]string, countByType map[string]int) {
	if item.Snippet == nil || *item.Snippet == "" {
		return
	}

	newText, changed := redactSnippetText(*item.Snippet, reverseMap, countByType)
	if changed {
		item.Snippet = &newText
		item.PiiRedacted = true
	}
}

func redactSnippetText(text string, reverseMap map[string]string, countByType map[string]int) (string, bool) {
	newText, changed := redactWithToken(text, "EMAIL", emailRe, reverseMap, countByType)
	newText, changedPhone := redactWithToken(newText, "PHONE", phoneRe, reverseMap, countByType)
	newText, changedSSN := redactWithToken(newText, "SSN", ssnRe, reverseMap, countByType)
	return newText, changed || changedPhone || changedSSN
}

func (p *PolicyEngine) cacheReverseMap(ctx context.Context, reverseMap map[string]string) {
	if p.cache == nil || len(reverseMap) == 0 {
		return
	}

	if raw, err := json.Marshal(reverseMap); err == nil {
		p.cache.Set(ctx, fmt.Sprintf("pii-map:%d", time.Now().UnixNano()), raw, 5*time.Minute)
	}
}

func (p *PolicyEngine) loadWorkspaceID(ctx context.Context, userID string) (string, error) {
	var workspaceID string
	if err := p.db.QueryRowContext(ctx, `SELECT workspace_id FROM user_account WHERE id = ? LIMIT 1`, userID).Scan(&workspaceID); err != nil {
		return "", fmt.Errorf("policy: load user workspace: %w", err)
	}
	return workspaceID, nil
}

func (p *PolicyEngine) loadRolePermissionRows(ctx context.Context, userID, workspaceID string) ([]string, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT r.permissions
		FROM role r
		JOIN user_role ur ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.workspace_id = ?
	`, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("policy: load role permissions: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var raw string
		if scanErr := rows.Scan(&raw); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, raw)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}

	return out, nil
}

func mergePermissions(rawPerms []string) map[string][]string {
	acc := map[string][]string{}
	for _, raw := range rawPerms {
		var m map[string][]string
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			continue
		}
		for k, vals := range m {
			acc[k] = append(acc[k], vals...)
		}
	}
	return acc
}

func (p *PolicyEngine) tableExists(ctx context.Context, name string) (bool, error) {
	var exists int
	if err := p.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type='table' AND name=?
	`, name).Scan(&exists); err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (p *PolicyEngine) fetchToolRequiredPermissions(ctx context.Context, toolID string) (string, bool, error) {
	var raw sql.NullString
	dbErr := p.db.QueryRowContext(ctx, `
		SELECT required_permissions
		FROM tool_definition
		WHERE id = ?
		LIMIT 1
	`, toolID).Scan(&raw)
	if dbErr == sql.ErrNoRows {
		return "", false, nil
	}
	if dbErr != nil {
		return "", false, dbErr
	}
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return "", false, nil
	}
	return raw.String, true, nil
}

func parseRequiredPermissions(raw, fallback string) []string {
	var perms []string
	if err := json.Unmarshal([]byte(raw), &perms); err != nil || len(perms) == 0 {
		return []string{fallback}
	}
	return perms
}

func candidateToolActions(required string) []string {
	required = strings.TrimSpace(required)
	if required == "" {
		return nil
	}
	out := []string{required}
	if strings.HasPrefix(required, toolsPrefix) {
		trimmed := strings.TrimSpace(strings.TrimPrefix(required, toolsPrefix))
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func roleAllowsAction(perms map[string][]string, resource, action string) bool {
	return hasGlobalAdminPermission(perms) ||
		hasDirectResourcePermission(perms, resource, action) ||
		hasWildcardPermission(perms, action) ||
		hasAPIAdminPermission(perms, resource, action)
}

func hasGlobalAdminPermission(perms map[string][]string) bool {
	return hasPermission(perms, permGlobal, "*") || hasPermission(perms, permGlobal, "admin")
}

func hasDirectResourcePermission(perms map[string][]string, resource, action string) bool {
	return hasPermission(perms, resource, action) || hasPermission(perms, resource, "*")
}

func hasWildcardPermission(perms map[string][]string, action string) bool {
	return hasPermission(perms, "*", action) || hasPermission(perms, "*", "*")
}

func hasAPIAdminPermission(perms map[string][]string, resource, action string) bool {
	if resource != permAPI || !strings.HasPrefix(action, "admin.") {
		return false
	}
	return hasPermission(perms, permAPI, "admin")
}

func hasPermission(perms map[string][]string, key, value string) bool {
	vals := perms[key]
	for _, v := range vals {
		if v == value {
			return true
		}
	}
	return false
}

func (p *PolicyEngine) loadActivePolicyVersion(ctx context.Context, workspaceID string) (activePolicyVersion, bool, error) {
	var out activePolicyVersion
	dbErr := p.db.QueryRowContext(ctx, `
		SELECT pv.policy_set_id, pv.id, pv.version_number, pv.policy_json
		FROM policy_set ps
		JOIN policy_version pv ON pv.policy_set_id = ps.id
		WHERE ps.workspace_id = ?
		  AND ps.is_active = 1
		  AND pv.workspace_id = ?
		  AND pv.status = 'active'
		ORDER BY pv.version_number DESC, pv.created_at DESC
		LIMIT 1
	`, workspaceID, workspaceID).Scan(&out.PolicySetID, &out.PolicyVersion, &out.VersionNumber, &out.RawPolicyRules)
	if dbErr == sql.ErrNoRows {
		return activePolicyVersion{}, false, nil
	}
	if dbErr != nil {
		return activePolicyVersion{}, false, dbErr
	}
	return out, true, nil
}

func parsePolicyRules(raw string) ([]policyRule, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var doc policyDoc
	if err := json.Unmarshal([]byte(raw), &doc); err == nil && len(doc.Rules) > 0 {
		return doc.Rules, nil
	}

	var arr []policyRule
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil, fmt.Errorf("policy: parse policy rules: %w", err)
	}
	return arr, nil
}

func filterMatchingRules(rules []policyRule, resource, action string, attrs map[string]string) []policyRule {
	matched := make([]policyRule, 0, len(rules))
	for _, rule := range rules {
		if !matchesResourceAction(rule, resource, action) || !matchesConditions(rule.Conditions, attrs) {
			continue
		}
		matched = append(matched, rule)
	}
	return matched
}

func matchesResourceAction(rule policyRule, resource, action string) bool {
	resourceRule := strings.TrimSpace(rule.Resource)
	actionRule := strings.TrimSpace(rule.Action)
	if resourceRule == "" {
		resourceRule = "*"
	}
	if actionRule == "" {
		actionRule = "*"
	}
	return matchToken(resourceRule, resource) && matchToken(actionRule, action)
}

func matchToken(pattern, value string) bool {
	return pattern == "*" || pattern == value
}

func matchesConditions(conditions map[string]any, attrs map[string]string) bool {
	if len(conditions) == 0 {
		return true
	}
	for key, expected := range conditions {
		expectedStr, ok := expected.(string)
		if !ok || attrs == nil || attrs[key] != expectedStr {
			return false
		}
	}
	return true
}

func resolveDeterministicRule(rules []policyRule) policyRule {
	sorted := append([]policyRule(nil), rules...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		eI := normalizeEffect(sorted[i].Effect)
		eJ := normalizeEffect(sorted[j].Effect)
		if eI != eJ {
			return eI == decisionDeny
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted[0]
}

func makeRuleTrace(rules []policyRule) []string {
	out := make([]string, len(rules))
	for i, rule := range rules {
		id := rule.ID
		if strings.TrimSpace(id) == "" {
			id = fmt.Sprintf("rule_%d", i+1)
		}
		out[i] = fmt.Sprintf("%s:%s:p=%d", id, normalizeEffect(rule.Effect), rule.Priority)
	}
	return out
}

func normalizeEffect(effect string) string {
	e := strings.ToLower(strings.TrimSpace(effect))
	if e == decisionDeny {
		return decisionDeny
	}
	return "allow"
}

func (p *PolicyEngine) logPolicyDecision(ctx context.Context, workspaceID, actorID string, decision PolicyDecision) {
	if p.audit == nil || decision.Trace == nil {
		return
	}
	entityType := "policy_set"
	entityID := decision.Trace.PolicySetID
	_ = p.audit.LogWithDetails(
		ctx,
		workspaceID,
		actorID,
		audit.ActorTypeUser,
		auditActionPolicyEvaluated,
		&entityType,
		&entityID,
		&audit.EventDetails{Metadata: map[string]any{
			"policy_set_id":      decision.Trace.PolicySetID,
			"policy_version_id":  decision.Trace.PolicyVersionID,
			"policy_version_num": decision.Trace.PolicyVersionNum,
			"resource":           decision.Trace.Resource,
			"action":             decision.Trace.Action,
			"matched_rule_id":    decision.Trace.MatchedRuleID,
			"matched_effect":     decision.Trace.MatchedEffect,
			"rule_trace":         decision.Trace.RuleTrace,
			"allow":              decision.Allow,
		}},
		resolvePolicyOutcome(decision.Allow),
	)
}

func resolvePolicyOutcome(allow bool) audit.Outcome {
	if allow {
		return audit.OutcomeSuccess
	}
	return audit.OutcomeDenied
}
