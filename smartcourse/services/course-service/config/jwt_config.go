package config

import (
	"fmt"
	"time"
)

const (
	devJWTSecret        = "dev-secret-key"
	minProdJWTSecretLen = 32
)

type JWTConfig struct {
	SecretKey          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

func loadJWTConfig(env string) (JWTConfig, error) {
	secret := getEnv("JWT_SECRET", "")
	if secret == "" {
		if env != "dev" {
			return JWTConfig{}, fmt.Errorf("JWT_SECRET must be set when ENV is not \"dev\"")
		}
		secret = devJWTSecret
	}
	if env != "dev" && len(secret) < minProdJWTSecretLen {
		return JWTConfig{}, fmt.Errorf("JWT_SECRET must be at least %d characters", minProdJWTSecretLen)
	}

	return JWTConfig{
		SecretKey:          secret,
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}, nil
}
