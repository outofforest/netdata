package infra

// Config stores configuration
type Config struct {
	// NATSAddresses contains addresses of NATS cluster
	NATSAddresses []string

	// VerboseLogging turns on verbose logging
	VerboseLogging bool
}
