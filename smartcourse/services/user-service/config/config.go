package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ServerPort string

	Env           string
	JWT           JWTConfig
	AdminEmail    string
	AdminPassword string
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	env := getEnv("ENV", "production")
	jwtCfg, err := loadJWTConfig(env)
	if err != nil {
		return nil, err
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "smartcourse_db"),
		ServerPort: getEnv("SERVER_PORT", "8080"),

		Env:           env,
		JWT:           jwtCfg,
		AdminEmail:    getEnv("ADMIN_EMAIL", ""),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
	}, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
