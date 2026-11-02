package config

import "fmt"

const (
	devInternalAPIKey     = "dev-internal-api-key"
	minProdInternalKeyLen = 32
)

// loadInternalAPIKey reads the shared service-to-service key that guards
// /internal/*. Every service must use the same value.
func loadInternalAPIKey(env string) (string, error) {
	key := getEnv("INTERNAL_API_KEY", "")
	if key == "" {
		if env != "dev" {
			return "", fmt.Errorf("INTERNAL_API_KEY must be set when ENV is not \"dev\"")
		}
		key = devInternalAPIKey
	}
	if env != "dev" && len(key) < minProdInternalKeyLen {
		return "", fmt.Errorf("INTERNAL_API_KEY must be at least %d characters", minProdInternalKeyLen)
	}
	return key, nil
}
