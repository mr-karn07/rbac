package config

import (
	"os"
)

type Config struct {
	OpenSearchAddresses []string
	Index               string
	ModelPath           string
}

func LoadConfig() *Config {
	return &Config{
		OpenSearchAddresses: []string{getEnv("OPENSEARCH_ADDRESSES", "http://localhost:9200")},
		Index:               getEnv("OPENSEARCH_INDEX", "casbin_policies"),
		ModelPath:           getEnv("MODEL_PATH", "rbac_model.conf"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
