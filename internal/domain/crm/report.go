package crm

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
)

// Task 4.5e - Reporting base service.
type ReportService struct {
	querier sqlcgen.Querier
}

func NewReportService(db *sql.DB) *ReportService {
	return &ReportService{querier: sqlcgen.New(db)}
}

type SalesFunnelReport struct {
	GeneratedAt time.Time     `json:"generatedAt"`
	WorkspaceID string        `json:"workspaceId"`
	Stages      []FunnelStage `json:"stages"`
}

type FunnelStage struct {
	Name        string  `json:"name"`
	Order       int     `json:"order"`
	DealCount   int     `json:"dealCount"`
	TotalValue  float64 `json:"totalValue"`
	Probability float64 `json:"probability"`
}

type DealAgingRow struct {
	Name    string  `json:"name"`
	AvgDays float64 `json:"avgDays"`
}

type CaseVolumeRow struct {
	Priority string `json:"priority"`
	Status   string `json:"status"`
	Count    int    `json:"count"`
}

type SupportBacklogReport struct {
	GeneratedAt  time.Time            `json:"generatedAt"`
	OpenTotal    int                  `json:"openTotal"`
	AgingBuckets []AgingBucket        `json:"agingBuckets"`
	MTTR         map[string]float64   `json:"mttr"`
	Items        []SupportBacklogItem `json:"items"`
}

type SupportBacklogItem struct {
	ID        string `json:"id"`
	Priority  string `json:"priority"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	AgingDays int    `json:"agingDays"`
}

type AgingBucket struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Count int    `json:"count"`
}

func (s *ReportService) GetSalesFunnel(ctx context.Context, workspaceID string, _, _ *time.Time) (*SalesFunnelReport, error) {
	rows, err := s.querier.SalesFunnelByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("sales funnel query: %w", err)
	}

	stages := make([]FunnelStage, 0, len(rows))
	for _, r := range rows {
		stages = append(stages, FunnelStage{
			Name:        r.Name,
			Order:       int(r.StageOrder),
			DealCount:   int(r.DealCount),
			TotalValue:  numberToFloat64(r.TotalValue),
			Probability: r.Probability,
		})
	}

	return &SalesFunnelReport{
		GeneratedAt: time.Now().UTC(),
		WorkspaceID: workspaceID,
		Stages:      stages,
	}, nil
}

func (s *ReportService) GetDealAging(ctx context.Context, workspaceID string) ([]DealAgingRow, error) {
	rows, err := s.querier.DealAgingByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("deal aging query: %w", err)
	}
	out := make([]DealAgingRow, 0, len(rows))
	for _, r := range rows {
		avg := 0.0
		if r.AvgDays != nil {
			avg = *r.AvgDays
		}
		out = append(out, DealAgingRow{Name: r.Name, AvgDays: avg})
	}
	return out, nil
}

func (s *ReportService) GetCaseVolume(ctx context.Context, workspaceID string, _, _ *time.Time) ([]CaseVolumeRow, error) {
	rows, err := s.querier.CaseVolumeByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("case volume query: %w", err)
	}
	out := make([]CaseVolumeRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, CaseVolumeRow{Priority: r.Priority, Status: r.Status, Count: int(r.Count)})
	}
	return out, nil
}

func (s *ReportService) GetSupportBacklog(ctx context.Context, workspaceID string, agingDays int) (*SupportBacklogReport, error) {
	rows, err := s.querier.CaseBacklogByWorkspace(ctx, sqlcgen.CaseBacklogByWorkspaceParams{
		WorkspaceID: workspaceID,
		AgingDays:   strconv.Itoa(agingDays),
	})
	if err != nil {
		return nil, fmt.Errorf("case backlog query: %w", err)
	}
	mttrRows, err := s.querier.CaseMTTRByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("case mttr query: %w", err)
	}

	items := make([]SupportBacklogItem, 0, len(rows))
	buckets := []AgingBucket{
		{Label: "0-7d", Min: 0, Max: 7},
		{Label: "8-30d", Min: 8, Max: 30},
		{Label: "31d+", Min: 31, Max: -1},
	}
	for _, r := range rows {
		age := int(r.AgingDays)
		items = append(items, SupportBacklogItem{ID: r.ID, Priority: r.Priority, Status: r.Status, CreatedAt: r.CreatedAt, AgingDays: age})
		switch {
		case age <= 7:
			buckets[0].Count++
		case age <= 30:
			buckets[1].Count++
		default:
			buckets[2].Count++
		}
	}

	mttr := make(map[string]float64, len(mttrRows))
	for _, row := range mttrRows {
		if row.AvgResolutionDays == nil {
			mttr[row.Priority] = 0
			continue
		}
		mttr[row.Priority] = *row.AvgResolutionDays
	}

	return &SupportBacklogReport{
		GeneratedAt:  time.Now().UTC(),
		OpenTotal:    len(items),
		AgingBuckets: buckets,
		MTTR:         mttr,
		Items:        items,
	}, nil
}

func (s *ReportService) ExportSalesFunnelCSV(ctx context.Context, workspaceID string, from, to *time.Time) (io.Reader, error) {
	report, err := s.GetSalesFunnel(ctx, workspaceID, from, to)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	go func() {
		w := csv.NewWriter(pw)
		_ = w.Write([]string{"stage", "order", "deal_count", "total_value", "probability"})
		for _, row := range report.Stages {
			_ = w.Write([]string{
				row.Name,
				strconv.Itoa(row.Order),
				strconv.Itoa(row.DealCount),
				fmt.Sprintf("%.2f", row.TotalValue),
				fmt.Sprintf("%.2f", row.Probability),
			})
		}
		w.Flush()
		_ = pw.CloseWithError(w.Error())
	}()
	return pr, nil
}

func (s *ReportService) ExportSupportBacklogCSV(ctx context.Context, workspaceID string) (io.Reader, error) {
	report, err := s.GetSupportBacklog(ctx, workspaceID, -1)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	go func() {
		w := csv.NewWriter(pw)
		_ = w.Write([]string{"id", "priority", "status", "created_at", "aging_days"})
		for _, row := range report.Items {
			_ = w.Write([]string{row.ID, row.Priority, row.Status, row.CreatedAt, strconv.Itoa(row.AgingDays)})
		}
		w.Flush()
		_ = pw.CloseWithError(w.Error())
	}()
	return pr, nil
}

func numberToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case []byte:
		f, _ := strconv.ParseFloat(string(val), 64)
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}