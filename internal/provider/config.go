package provider

// NetcupConfig holds Netcup specific configuration.
type NetcupConfig struct {
	CustomerNumber string
	ApiKey         string
	ApiPassword    string
}

// AwsRoute53Config holds AWS Route53 specific configuration.
type AwsRoute53Config struct {
	// Configuration for AWS Route53 can be added here if needed.
	// Currently, it uses the default AWS SDK configuration chain.
}
