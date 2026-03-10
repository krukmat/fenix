package workflow

import (
	"database/sql"
	"time"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func parseRFC3339Time(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}

func parseOptionalRFC3339(value *string) *time.Time {
	if value == nil {
		return nil
	}
	t := parseRFC3339Time(*value)
	return &t
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func scanWorkflow(scanner interface {
	Scan(dest ...any) error
}) (*Workflow, error) {
	var row workflowRow
	if err := scanner.Scan(
		&row.ID,
		&row.WorkspaceID,
		&row.AgentDefinitionID,
		&row.ParentVersionID,
		&row.Name,
		&row.Description,
		&row.DSLSource,
		&row.SpecSource,
		&row.Version,
		&row.Status,
		&row.CreatedByUserID,
		&row.ArchivedAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return rowToWorkflow(row), nil
}

func scanWorkflowRows(rows *sql.Rows) ([]*Workflow, error) {
	var out []*Workflow
	for rows.Next() {
		wf, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wf)
	}
	return out, rows.Err()
}

type workflowRow struct {
	ID                string
	WorkspaceID       string
	AgentDefinitionID *string
	ParentVersionID   *string
	Name              string
	Description       *string
	DSLSource         string
	SpecSource        *string
	Version           int64
	Status            string
	CreatedByUserID   *string
	ArchivedAt        *string
	CreatedAt         string
	UpdatedAt         string
}

func rowToWorkflow(row workflowRow) *Workflow {
	return &Workflow{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		AgentDefinitionID: row.AgentDefinitionID,
		ParentVersionID:   row.ParentVersionID,
		Name:              row.Name,
		Description:       row.Description,
		DSLSource:         row.DSLSource,
		SpecSource:        row.SpecSource,
		Version:           int(row.Version),
		Status:            Status(row.Status),
		CreatedByUserID:   row.CreatedByUserID,
		ArchivedAt:        parseOptionalRFC3339(row.ArchivedAt),
		CreatedAt:         parseRFC3339Time(row.CreatedAt),
		UpdatedAt:         parseRFC3339Time(row.UpdatedAt),
	}
}
