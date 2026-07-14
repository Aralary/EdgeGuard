package observability

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

const defaultReadinessTimeout = 2 * time.Second

type Check struct {
	Name     string
	Run      func(context.Context) error
	Optional bool
}

type Readiness struct {
	service string
	timeout time.Duration
	checks  []Check
}

type checkResponse struct {
	Status string `json:"status"`
}

type readinessResponse struct {
	Status  string                   `json:"status"`
	Service string                   `json:"service"`
	Checks  map[string]checkResponse `json:"checks"`
}

func Optional(check Check) Check {
	check.Optional = true
	return check
}

func NewReadiness(service string, checks ...Check) *Readiness {
	return &Readiness{
		service: strings.TrimSpace(service),
		timeout: defaultReadinessTimeout,
		checks:  append([]Check(nil), checks...),
	}
}

func (r *Readiness) Handler(c *echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), r.timeout)
	defer cancel()

	checks := append([]Check(nil), r.checks...)
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Name < checks[j].Name
	})

	response := readinessResponse{
		Status:  "ready",
		Service: r.service,
		Checks:  make(map[string]checkResponse, len(checks)),
	}
	statusCode := http.StatusOK

	for _, check := range checks {
		name := strings.TrimSpace(check.Name)
		if name == "" || check.Run == nil {
			continue
		}

		status := "ok"
		if err := check.Run(ctx); err != nil {
			status = "failed"
			if check.Optional {
				if response.Status == "ready" {
					response.Status = "degraded"
				}
			} else {
				response.Status = "not_ready"
				statusCode = http.StatusServiceUnavailable
			}
		}
		response.Checks[name] = checkResponse{Status: status}
	}

	return c.JSON(statusCode, response)
}

func HTTPCheck(name string, rawURL string, client *http.Client) Check {
	if client == nil {
		client = &http.Client{Timeout: defaultReadinessTimeout}
	}

	return Check{
		Name: name,
		Run: func(ctx context.Context) error {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(rawURL, "/")+"/ready", nil)
			if err != nil {
				return err
			}
			response, err := client.Do(request)
			if err != nil {
				return err
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("dependency readiness returned status %d", response.StatusCode)
			}
			return nil
		},
	}
}

func Register(e *echo.Echo, metrics *Metrics, readiness *Readiness) {
	e.GET("/metrics", metrics.Handler())
	e.GET("/ready", readiness.Handler)
}
