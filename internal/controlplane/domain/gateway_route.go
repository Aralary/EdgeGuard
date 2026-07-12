package domain

// GatewayRoute is a read model used to publish the active routing
// configuration from Control Plane to Gateway.
type GatewayRoute struct {
	Name        string
	PathPrefix  string
	UpstreamURL string
	StripPrefix bool
	TimeoutMS   int
}
