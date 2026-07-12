package domain

import (
	"strings"
	"time"
)

type Route struct {
	ProjectID    string
	Name         string
	PathPrefix   string
	UpstreamURL  string
	StripPrefix  bool
	Timeout      time.Duration
	AuthRequired bool
}

func (r Route) Matches(path string) bool {
	if r.PathPrefix == "/" {
		return true
	}

	prefix := strings.TrimRight(r.PathPrefix, "/")

	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func (r Route) UpstreamPath(originalPath string) string {
	if !r.StripPrefix {
		return originalPath
	}

	prefix := strings.TrimRight(r.PathPrefix, "/")
	path := strings.TrimPrefix(originalPath, prefix)

	if path == "" {
		return "/"
	}

	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}

	return path
}
