package database

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"smartcourse/internal/config"
)

const liquibasePostgresContainer = "smartcourse-postgres"
const liquibaseImage = "smartcourse-liquibase:local"

// RunLiquibaseMigrations builds a Liquibase image with the PostgreSQL JDBC
// driver baked in (the upstream liquibase/liquibase image ships without it)
// and runs it against the running smartcourse-postgres container, applying
// every changelog under migrations/.
func RunLiquibaseMigrations(cfg *config.Config) error {
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

	jdbcURL := fmt.Sprintf("jdbc:postgresql://localhost:%s/%s", cfg.DBPort, cfg.DBName)

	runCmd := exec.Command("docker", "run", "--rm",
		"--network", "container:"+liquibasePostgresContainer,
		"-v", migrationsDir+":/liquibase/changelog",
		liquibaseImage,
		// Relative path: lets the entrypoint cd into /liquibase/changelog
		// and resolves the changelog from the filesystem, not the classpath.
		"--changelog-file=db.changelog-master.yaml",
		"--url="+jdbcURL,
		"--username="+cfg.DBUser,
		"--password="+cfg.DBPassword,
		"update",
	)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("liquibase migration failed: %w", err)
	}
	return nil
}
