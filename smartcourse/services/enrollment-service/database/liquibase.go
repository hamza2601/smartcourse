package database

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"smartcourse/services/enrollment-service/config"
)

const (
	liquibasePostgresContainer = "smartcourse-postgres"
	liquibaseImage             = "smartcourse-liquibase-enrollment:local"
)

// RunLiquibaseMigrations applies this service's changelogs when running
// outside docker-compose: it builds the service's Liquibase image (driver +
// migrate.sh, which owns the schema/tracking-table setup) and runs it against
// the smartcourse-postgres container.
func RunLiquibaseMigrations(cfg *config.Config) error {
	// In docker-compose a dedicated liquibase-enrollment job applies the changelogs
	// (service containers have no docker CLI to run it from here).
	if os.Getenv("SKIP_LIQUIBASE") == "true" {
		return nil
	}

	liquibaseDir, err := filepath.Abs("liquibase")
	if err != nil {
		return fmt.Errorf("failed to resolve liquibase directory: %w", err)
	}
	migrationsDir, err := filepath.Abs("migrations")
	if err != nil {
		return fmt.Errorf("failed to resolve migrations directory: %w", err)
	}

	buildCmd := exec.Command("docker", "build", "-t", liquibaseImage, liquibaseDir)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build liquibase image: %w", err)
	}

	runCmd := exec.Command("docker", "run", "--rm",
		"--network", "container:"+liquibasePostgresContainer,
		"-v", migrationsDir+":/liquibase/changelog:ro",
		"-e", fmt.Sprintf("LIQUIBASE_COMMAND_URL=jdbc:postgresql://localhost:%s/%s", cfg.DBPort, cfg.DBName),
		"-e", "LIQUIBASE_COMMAND_USERNAME="+cfg.DBUser,
		"-e", "LIQUIBASE_COMMAND_PASSWORD="+cfg.DBPassword,
		liquibaseImage,
	)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("liquibase migration failed: %w", err)
	}
	return nil
}
