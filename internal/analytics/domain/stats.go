package domain

import "time"

type RouteStatsFilter struct {
	ProjectID string
	From      time.Time
	To        time.Time
	RouteName string
	Method    string
}

type RouteStatsSummary struct {
	RequestCount       int64
	Status1xxCount     int64
	Status2xxCount     int64
	Status3xxCount     int64
	Status4xxCount     int64
	Status5xxCount     int64
	RateLimitedCount   int64
	AverageDurationMS  float64
	MaxDurationMS      int64
	TotalResponseBytes int64
}

type RouteStatsHourly struct {
	BucketStart        time.Time
	RouteName          string
	RoutePathPrefix    string
	Method             string
	RequestCount       int64
	Status1xxCount     int64
	Status2xxCount     int64
	Status3xxCount     int64
	Status4xxCount     int64
	Status5xxCount     int64
	RateLimitedCount   int64
	AverageDurationMS  float64
	MaxDurationMS      int64
	TotalResponseBytes int64
}
