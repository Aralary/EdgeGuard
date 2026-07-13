package httpdelivery

import "time"

type errorResponse struct {
	Error string `json:"error"`
}

type summaryResponse struct {
	ProjectID          string  `json:"project_id"`
	From               string  `json:"from"`
	To                 string  `json:"to"`
	RouteName          string  `json:"route_name,omitempty"`
	Method             string  `json:"method,omitempty"`
	RequestCount       int64   `json:"request_count"`
	Status1xxCount     int64   `json:"status_1xx_count"`
	Status2xxCount     int64   `json:"status_2xx_count"`
	Status3xxCount     int64   `json:"status_3xx_count"`
	Status4xxCount     int64   `json:"status_4xx_count"`
	Status5xxCount     int64   `json:"status_5xx_count"`
	RateLimitedCount   int64   `json:"rate_limited_count"`
	AverageDurationMS  float64 `json:"average_duration_ms"`
	MaxDurationMS      int64   `json:"max_duration_ms"`
	TotalResponseBytes int64   `json:"total_response_bytes"`
}

type hourlyItemResponse struct {
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
	AverageDurationMS  float64   `json:"average_duration_ms"`
	MaxDurationMS      int64     `json:"max_duration_ms"`
	TotalResponseBytes int64     `json:"total_response_bytes"`
}

type hourlyResponse struct {
	ProjectID string               `json:"project_id"`
	From      string               `json:"from"`
	To        string               `json:"to"`
	RouteName string               `json:"route_name,omitempty"`
	Method    string               `json:"method,omitempty"`
	Items     []hourlyItemResponse `json:"items"`
}
