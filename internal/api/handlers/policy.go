// GO-POLICY-READ-01: Read-only HTTP handlers for policy_set and policy_version.
package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// PolicyHandler serves read-only endpoints for policy_set and policy_version.
type PolicyHandler struct {
	db *sql.DB
}

// NewPolicyHandler constructs a PolicyHandler backed by the given DB.
func NewPolicyHandler(db *sql.DB) *PolicyHandler {
	return &PolicyHandler{db: db}
}

// policySetRow is the shape returned by ListPolicySets.
type policySetRow struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// policyVersionRow is the shape returned by GetPolicyVersions.
type policyVersionRow struct {
	ID            string    `json:"id"`
	PolicySetID   string    `json:"policy_set_id"`
	WorkspaceID   string    `json:"workspace_id"`
	VersionNumber int64     `json:"version_number"`
	PolicyJSON    string    `json:"policy_json"`
	Status        string    `json:"status"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

const (
	errPolicySetIDRequired     = "policy set id is required"
	errFailedToQueryPolicySets = "failed to query policy sets: %v"
	errFailedToQueryVersions   = "failed to query policy versions: %v"
	queryParamIsActive         = "is_active"
)

// ListPolicySets handles GET /api/v1/policy/sets.
// Supports optional ?is_active=true|false filter and standard pagination.
// GO-POLICY-READ-01
func (h *PolicyHandler) ListPolicySets(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	page := parsePaginationParams(r)

	isActiveFilter := r.URL.Query().Get(queryParamIsActive)
	sets, err := h.queryPolicySets(r, wsID, isActiveFilter, page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFailedToQueryPolicySets, err))
		return
	}
	_ = writePaginatedOr500(w, sets, len(sets), page)
}

func (h *PolicyHandler) queryPolicySets(r *http.Request, wsID, isActiveFilter string, page paginationParams) ([]policySetRow, error) {
	query, args := buildPolicySetsQuery(wsID, isActiveFilter, page)
	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPolicySets(rows)
}

func buildPolicySetsQuery(wsID, isActiveFilter string, page paginationParams) (string, []any) {
	base := `SELECT id, workspace_id, name, description, is_active, created_by, created_at, updated_at
	         FROM policy_set WHERE workspace_id = ?`
	args := []any{wsID}

	if isActiveFilter == "true" {
		base += ` AND is_active = 1`
	} else if isActiveFilter == "false" {
		base += ` AND is_active = 0`
	}
	base += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, page.Limit, page.Offset)
	return base, args
}

func scanPolicySets(rows *sql.Rows) ([]policySetRow, error) {
	sets := make([]policySetRow, 0)
	for rows.Next() {
		var s policySetRow
		var isActiveInt int
		scanErr := rows.Scan(
			&s.ID, &s.WorkspaceID, &s.Name, &s.Description,
			&isActiveInt, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		)
		if scanErr != nil {
			return nil, scanErr
		}
		s.IsActive = isActiveInt == 1
		sets = append(sets, s)
	}
	return sets, rows.Err()
}

// GetPolicyVersions handles GET /api/v1/policy/sets/{id}/versions.
// Returns all versions for the given policy_set_id scoped to the workspace.
// GO-POLICY-READ-01
func (h *PolicyHandler) GetPolicyVersions(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	setID := chi.URLParam(r, paramID)
	if setID == "" {
		writeError(w, http.StatusBadRequest, errPolicySetIDRequired)
		return
	}
	page := parsePaginationParams(r)

	versions, err := h.queryPolicyVersions(r, wsID, setID, page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFailedToQueryVersions, err))
		return
	}
	_ = writePaginatedOr500(w, versions, len(versions), page)
}

func (h *PolicyHandler) queryPolicyVersions(r *http.Request, wsID, setID string, page paginationParams) ([]policyVersionRow, error) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, policy_set_id, workspace_id, version_number, policy_json, status, created_by, created_at
		FROM policy_version
		WHERE workspace_id = ? AND policy_set_id = ?
		ORDER BY version_number DESC
		LIMIT ? OFFSET ?`,
		wsID, setID, page.Limit, page.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPolicyVersions(rows)
}

func scanPolicyVersions(rows *sql.Rows) ([]policyVersionRow, error) {
	versions := make([]policyVersionRow, 0)
	for rows.Next() {
		var v policyVersionRow
		scanErr := rows.Scan(
			&v.ID, &v.PolicySetID, &v.WorkspaceID, &v.VersionNumber,
			&v.PolicyJSON, &v.Status, &v.CreatedBy, &v.CreatedAt,
		)
		if scanErr != nil {
			return nil, scanErr
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}
