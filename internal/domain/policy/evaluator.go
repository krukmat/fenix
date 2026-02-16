package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
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
//nolint:revive // nombre mantenido por compatibilidad interna del módulo
type PolicyEngine struct {
	db    *sql.DB
	cache Cache
	audit *audit.AuditService
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

	if hasPermission(rolePerms, "global", "read_all") || hasPermission(rolePerms, "records", "read_all") {
		return Filter{Where: "workspace_id = ?", Args: []any{workspaceID}}, nil
	}

	return Filter{Where: "workspace_id = ? AND owner_id = ?", Args: []any{workspaceID, userID}}, nil
}

var (
	emailRe = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	phoneRe = regexp.MustCompile(`\b(?:\+?\d{1,3}[\s.-]?)?(?:\(?\d{2,4}\)?[\s.-]?)?\d{3}[\s.-]?\d{3,4}\b`)
	ssnRe   = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
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
	_, rolePerms, err := p.loadUserWorkspaceAndRolePermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	requiredPerms, err := p.loadToolRequiredPermissions(ctx, toolID)
	if err != nil {
		return false, err
	}

	for _, req := range requiredPerms {
		if hasPermission(rolePerms, "tools", req) || hasPermission(rolePerms, "tools", "*") {
			return true, nil
		}
	}

	return false, nil
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
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		out = append(out, raw)
	}

	if err := rows.Err(); err != nil {
		return nil, err
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

func hasPermission(perms map[string][]string, key, value string) bool {
	vals := perms[key]
	for _, v := range vals {
		if v == value {
			return true
		}
	}
	return false
}
