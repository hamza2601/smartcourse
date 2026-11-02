#!/bin/sh
# Applies this service's changelogs into its own schema, with Liquibase's
# tracking tables (DATABASECHANGELOG / ...LOCK) in that schema too.
# Connection comes from LIQUIBASE_COMMAND_URL / _USERNAME / _PASSWORD.
set -eu
: "${SERVICE_SCHEMA:?SERVICE_SCHEMA must be set}"

# Liquibase creates its tracking tables before running any changeset and will
# not create a missing schema; creating it via Liquibase itself (execute-sql)
# would leave a lock table in "public". So create it with psql first.
# jdbc:postgresql://host:port/db -> postgresql://host:port/db (a valid psql URI)
PGPASSWORD="${LIQUIBASE_COMMAND_PASSWORD}" psql "${LIQUIBASE_COMMAND_URL#jdbc:}" \
  -U "${LIQUIBASE_COMMAND_USERNAME}" -v ON_ERROR_STOP=1 -q \
  -c "CREATE SCHEMA IF NOT EXISTS \"${SERVICE_SCHEMA}\""

# Relative changelog path + search path: an absolute --changelog-file fails to
# resolve once extra jars (the JDBC driver) are on the classpath.
cd /liquibase/changelog
exec /liquibase/liquibase --headless=true --search-path=. update \
  --changelog-file=db.changelog-master.yaml \
  --default-schema-name="${SERVICE_SCHEMA}" \
  --liquibase-schema-name="${SERVICE_SCHEMA}"
