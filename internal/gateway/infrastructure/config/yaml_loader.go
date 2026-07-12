package config

import (
	"os"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP   HTTPConfig    `yaml:"http"`
	Routes []RouteConfig `yaml:"routes"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type RouteConfig struct {
	ProjectID    string `yaml:"project_id"`
	Name         string `yaml:"name"`
	PathPrefix   string `yaml:"path_prefix"`
	UpstreamURL  string `yaml:"upstream_url"`
	StripPrefix  bool   `yaml:"strip_prefix"`
	TimeoutMS    int    `yaml:"timeout_ms"`
	AuthRequired bool   `yaml:"auth_required"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}

	return cfg, nil
}

func (c Config) DomainRoutes() []domain.Route {
	routes := make([]domain.Route, 0, len(c.Routes))

	for _, route := range c.Routes {
		timeout := time.Duration(route.TimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 5 * time.Second
		}

		routes = append(routes, domain.Route{
			ProjectID:    route.ProjectID,
			Name:         route.Name,
			PathPrefix:   route.PathPrefix,
			UpstreamURL:  route.UpstreamURL,
			StripPrefix:  route.StripPrefix,
			Timeout:      timeout,
			AuthRequired: route.AuthRequired,
		})
	}

	return routes
}
