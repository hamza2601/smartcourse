FROM liquibase/liquibase:latest

# psql creates the service schema before Liquibase runs (see migrate.sh).
USER root
RUN apt-get update && apt-get install -y --no-install-recommends postgresql-client
USER liquibase

ADD --chmod=644 https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.4/postgresql-42.7.4.jar /liquibase/lib/postgresql.jar
COPY --chmod=755 migrate.sh /liquibase/migrate.sh
ENV SERVICE_SCHEMA=enrollment_service
ENTRYPOINT ["/liquibase/migrate.sh"]
