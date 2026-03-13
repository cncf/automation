package main

// LandscapeConfig holds configuration for the landscape being served.
type LandscapeConfig struct {
	Name        string // e.g. "CNCF"
	Description string // e.g. "Cloud Native Computing Foundation"
}

func defaultConfig() LandscapeConfig {
	return LandscapeConfig{
		Name:        "CNCF",
		Description: "Cloud Native Computing Foundation",
	}
}
