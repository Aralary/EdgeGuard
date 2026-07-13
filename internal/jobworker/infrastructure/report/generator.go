package report

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/jobworker/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProjectNotFound    = errors.New("report project not found")
	ErrInvalidProjectID   = errors.New("invalid report project id")
	ErrInvalidReportRange = errors.New("invalid report range")
	ErrInvalidFormat      = errors.New("invalid report format")
)

type Generator struct {
	pool      *pgxpool.Pool
	directory string
}

type row struct {
	BucketStart        time.Time `json:"bucket_start"`
	RouteName          string    `json:"route_name"`
	RoutePathPrefix    string    `json:"route_path_prefix"`
	Method             string    `json:"method"`
	RequestCount       int64     `json:"request_count"`
	Status1xxCount     int64     `json:"status_1xx_count"`
	Status2xxCount     int64     `json:"status_2xx_count"`
	Status3xxCount     int64     `json:"status_3xx_count"`
	Status4xxCount     int64     `json:"status_4xx_count"`
	Status5xxCount     int64     `json:"status_5xx_count"`
	RateLimitedCount   int64     `json:"rate_limited_count"`
	TotalDurationMS    int64     `json:"total_duration_ms"`
	MaxDurationMS      int64     `json:"max_duration_ms"`
	TotalResponseBytes int64     `json:"total_response_bytes"`
}

type jsonReport struct {
	GeneratedAt time.Time `json:"generated_at"`
	ProjectID   string    `json:"project_id"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	Items       []row     `json:"items"`
}

func NewGenerator(pool *pgxpool.Pool, directory string) *Generator {
	return &Generator{pool: pool, directory: directory}
}

func (generator *Generator) Generate(ctx context.Context, jobID string, payload platformjobs.ReportPayload) (string, int64, error) {
	if payload.From.IsZero() || payload.To.IsZero() || !payload.From.Before(payload.To) {
		return "", 0, domain.Permanent(ErrInvalidReportRange)
	}
	format := strings.ToLower(strings.TrimSpace(payload.Format))
	if format != "json" && format != "csv" {
		return "", 0, domain.Permanent(ErrInvalidFormat)
	}

	exists, err := generator.projectExists(ctx, payload.ProjectID)
	if err != nil {
		return "", 0, err
	}
	if !exists {
		return "", 0, domain.Permanent(ErrProjectNotFound)
	}

	items, err := generator.loadRows(ctx, payload)
	if err != nil {
		return "", 0, err
	}

	if err := os.MkdirAll(generator.directory, 0o750); err != nil {
		return "", 0, fmt.Errorf("create report directory: %w", err)
	}
	filename := filepath.Join(generator.directory, filepath.Base(jobID)+"."+format)
	temporary, err := os.CreateTemp(generator.directory, ".edgeguard-report-*")
	if err != nil {
		return "", 0, fmt.Errorf("create temporary report: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()

	if format == "json" {
		err = writeJSON(temporary, jsonReport{
			GeneratedAt: time.Now().UTC(),
			ProjectID:   payload.ProjectID,
			From:        payload.From.UTC(),
			To:          payload.To.UTC(),
			Items:       items,
		})
	} else {
		err = writeCSV(temporary, items)
	}
	if err != nil {
		_ = temporary.Close()
		return "", 0, err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return "", 0, fmt.Errorf("sync report: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", 0, fmt.Errorf("close report: %w", err)
	}
	if err := os.Rename(temporaryName, filename); err != nil {
		return "", 0, fmt.Errorf("publish report: %w", err)
	}

	return filename, int64(len(items)), nil
}

func (generator *Generator) projectExists(ctx context.Context, projectID string) (bool, error) {
	var exists bool
	err := generator.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1::uuid)`, projectID).Scan(&exists)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "22P02" {
			return false, domain.Permanent(ErrInvalidProjectID)
		}
		return false, fmt.Errorf("check report project: %w", err)
	}

	return exists, nil
}

func (generator *Generator) loadRows(ctx context.Context, payload platformjobs.ReportPayload) ([]row, error) {
	const query = `
SELECT
    bucket_start,
    route_name,
    route_path_prefix,
    method,
    request_count,
    status_1xx_count,
    status_2xx_count,
    status_3xx_count,
    status_4xx_count,
    status_5xx_count,
    rate_limited_count,
    total_duration_ms,
    max_duration_ms,
    total_response_bytes
FROM gateway_route_stats_hourly
WHERE project_id = $1::uuid
  AND bucket_start >= date_trunc('hour', $2::timestamptz)
  AND bucket_start < CASE
      WHEN $3::timestamptz = date_trunc('hour', $3::timestamptz) THEN $3::timestamptz
      ELSE date_trunc('hour', $3::timestamptz) + interval '1 hour'
  END
ORDER BY bucket_start, route_name, method
`

	rows, err := generator.pool.Query(ctx, query, payload.ProjectID, payload.From, payload.To)
	if err != nil {
		return nil, fmt.Errorf("query report rows: %w", err)
	}
	defer rows.Close()

	result := make([]row, 0)
	for rows.Next() {
		var item row
		if err := rows.Scan(
			&item.BucketStart,
			&item.RouteName,
			&item.RoutePathPrefix,
			&item.Method,
			&item.RequestCount,
			&item.Status1xxCount,
			&item.Status2xxCount,
			&item.Status3xxCount,
			&item.Status4xxCount,
			&item.Status5xxCount,
			&item.RateLimitedCount,
			&item.TotalDurationMS,
			&item.MaxDurationMS,
			&item.TotalResponseBytes,
		); err != nil {
			return nil, fmt.Errorf("scan report row: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report rows: %w", err)
	}

	return result, nil
}

func writeJSON(file *os.File, report jsonReport) error {
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("encode JSON report: %w", err)
	}
	return nil
}

func writeCSV(file *os.File, items []row) error {
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{
		"bucket_start", "route_name", "route_path_prefix", "method", "request_count",
		"status_1xx_count", "status_2xx_count", "status_3xx_count", "status_4xx_count", "status_5xx_count",
		"rate_limited_count", "total_duration_ms", "max_duration_ms", "total_response_bytes",
	}); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	for _, item := range items {
		if err := writer.Write([]string{
			item.BucketStart.UTC().Format(time.RFC3339), item.RouteName, item.RoutePathPrefix, item.Method,
			strconv.FormatInt(item.RequestCount, 10), strconv.FormatInt(item.Status1xxCount, 10),
			strconv.FormatInt(item.Status2xxCount, 10), strconv.FormatInt(item.Status3xxCount, 10),
			strconv.FormatInt(item.Status4xxCount, 10), strconv.FormatInt(item.Status5xxCount, 10),
			strconv.FormatInt(item.RateLimitedCount, 10), strconv.FormatInt(item.TotalDurationMS, 10),
			strconv.FormatInt(item.MaxDurationMS, 10), strconv.FormatInt(item.TotalResponseBytes, 10),
		}); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV report: %w", err)
	}
	return nil
}
