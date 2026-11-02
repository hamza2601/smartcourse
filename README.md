# SmartCourse

SmartCourse is the backend for a course delivery platform: instructors create and publish courses, and students enroll in them. It is built as three independent Go services that share one PostgreSQL server but never share tables.

## Tech stack

- Go 1.26, [Gin](https://github.com/gin-gonic/gin), [GORM](https://gorm.io/)
- PostgreSQL 15, one schema per service
- [Liquibase](https://www.liquibase.com/) for versioned migrations, one changelog set per service
- JWT access and refresh tokens for authentication
- Transactional outbox for domain events
- [Temporal](https://temporal.io/) dev server, reserved for the Week 2 course-publishing workflow
- Docker Compose for local development

## Architecture

| Service | Public API | Internal API | Owns schema | Responsibilities |
|---|---|---|---|---|
| user-service | `8081` | `9081` | `user_service` | Login, refresh tokens, user management, admin seeding |
| course-service | `8082` | `9082` | `course_service` | Course creation and publishing |
| enrollment-service | `8083` | `9083` | `enrollment_service` | Enrollments, seat limits |

- **Database per service, logically.** Each service reads and writes only its own schema. There are no foreign keys across schemas. An ID that points into another service, such as `courses.instructor_id` or `lesson_progress.lesson_id`, is stored as a plain UUID.
- **Identity comes from the token.** User, instructor and student IDs are read from the verified JWT and never from request bodies; an `instructor_id` or `student_id` in a request body is ignored. Roles are enforced on the route: creating a user needs `admin`, and creating or publishing a course needs `instructor`.
- **Service-to-service calls.** enrollment-service calls course-service's `GET /internal/courses/:id` to check that a course exists, is published, and to read its `enrollment_limit`. No other service calls another.
- **Two listeners per service.** The public API listens on `PORT`. The `/internal/*` routes listen only on `INTERNAL_PORT`, and every request there must carry an `X-Internal-Token` header equal to the shared `INTERNAL_API_KEY`, compared in constant time.
  - In Compose only the public ports are published to the host. Internal ports are reachable only on the Compose network.
  - **An API gateway must never route `/internal/*` or the internal ports.**
- **Transactional outbox.** Every state change writes its domain event to the service's own `outbox_events` table in the same database transaction. If the business write fails, no event is stored, and vice versa.
  - A relay goroutine in each service polls unprocessed rows with `FOR UPDATE SKIP LOCKED`, so multiple replicas are safe. It publishes the events in `created_at` order and then sets `processed_at`.
  - Publishing currently means logging the event through a `Publisher` interface; a Kafka or Temporal publisher can replace it without other changes.
  - Delivery is at-least-once.

| Event | Emitted by | Written in the same transaction as |
|---|---|---|
| `UserRegistered` | user-service | the user insert |
| `CourseCreated` | course-service | the course insert |
| `CoursePublished` | course-service | the `draft` → `published` status change |
| `EnrollmentCreated` | enrollment-service | the enrollment insert and the seat reservation |

## Database schemas

Each service's Liquibase changelogs live in `services/<service>/migrations`, are numbered from `001` per service, and create exactly these tables. Each service's Liquibase tracking tables (`databasechangelog`, `databasechangeloglock`) live in its own schema too.

### `user_service`

| Table | Columns / constraints |
|---|---|
| `users` | `id` PK, `email` UNIQUE, `name`, `role`, `password_hash` (bcrypt), timestamps |
| `refresh_tokens` | `id` PK, `user_id`, `token_hash` UNIQUE (SHA-256), `family_id`, `revoked_at`, `expires_at`, `created_at`; index on (`user_id`, `family_id`) |
| `outbox_events` | see below |

### `course_service`

| Table | Columns / constraints |
|---|---|
| `courses` | `id` PK, `name`, `description`, `instructor_id` (UUID, no FK), `status` CHECK in (`draft`, `published`, `archived`), `enrollment_limit` nullable (NULL = unlimited) CHECK `> 0`, timestamps |
| `modules` | `id` PK, `course_id` FK → `courses` (cascade), `title`, `position`; UNIQUE (`course_id`, `position`) |
| `lessons` | `id` PK, `module_id` FK → `modules` (cascade), `title`, `content`, `position`; UNIQUE (`module_id`, `position`) |
| `outbox_events` | see below |

### `enrollment_service`

| Table | Columns / constraints |
|---|---|
| `enrollments` | `id` PK, `student_id`, `course_id` (UUIDs, no FK), `status`, timestamps; partial UNIQUE (`student_id`, `course_id`) WHERE `status = 'active'` |
| `lesson_progress` | `id` PK, `enrollment_id` FK → `enrollments` (cascade), `lesson_id` (UUID, no FK), `status` CHECK in (`not_started`, `in_progress`, `completed`), `completed_at`; UNIQUE (`enrollment_id`, `lesson_id`) |
| `course_seats` | `course_id` PK, `capacity` (copied from the course's `enrollment_limit`; NULL = unlimited), `taken`; CHECK (`taken >= 0 AND taken <= capacity`) |
| `outbox_events` | see below |

### `outbox_events` (identical in every schema)

`id` UUID PK, `aggregate_type`, `aggregate_id` UUID, `event_type`, `payload` JSONB, `created_at`, `processed_at` NULL; partial index on `created_at` WHERE `processed_at IS NULL`.

### How enrollment works

All of the following steps run in one transaction:

1. The seat row for the course is created, or its capacity is refreshed from the course's current `enrollment_limit`. A limit lowered below the seats already taken makes the course full rather than violating the CHECK.
2. A second active enrollment for the same student and course is rejected with 409.
3. A seat is reserved atomically: `UPDATE course_seats SET taken = taken + 1 WHERE course_id = ? AND (capacity IS NULL OR taken < capacity)`. Zero rows updated means the course is full, which returns 409.
4. The enrollment and its `EnrollmentCreated` outbox row are inserted.

If any step fails, nothing is kept: no seat, no enrollment and no event.

## Running locally

Prerequisites: Docker with Compose. Go 1.26+ is needed only to run tests or a service outside Docker.

```bash
docker compose up --build        # from the repository root
docker compose down              # stop
docker compose down -v           # stop and delete the database volume
```

On startup:

1. Postgres becomes healthy.
2. Three one-shot jobs (`liquibase-user`, `liquibase-course`, `liquibase-enrollment`) each create their service's schema if it doesn't exist and apply only that service's changelogs.
3. Each service waits for its own job to finish successfully, then starts its public listener, internal listener and outbox relay.

user-service seeds an admin account, `admin@smartcourse.local` / `admin-password-123` by default; override it with `ADMIN_EMAIL` and `ADMIN_PASSWORD`. Shutdown on SIGTERM is graceful: listeners drain and the relay finishes its current batch.

### Quick tour

```bash
# 1. Log in as the seeded admin
ADMIN=$(curl -s -X POST localhost:8081/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@smartcourse.local","password":"admin-password-123"}' | jq -r .access_token)

# 2. Create an instructor and a student
curl -s -X POST localhost:8081/api/v1/users -H "Authorization: Bearer $ADMIN" -H 'Content-Type: application/json' \
  -d '{"name":"Ivy","email":"ivy@example.com","role":"instructor","password":"password-123"}'
curl -s -X POST localhost:8081/api/v1/users -H "Authorization: Bearer $ADMIN" -H 'Content-Type: application/json' \
  -d '{"name":"Sam","email":"sam@example.com","role":"student","password":"password-123"}'

INSTR=$(curl -s -X POST localhost:8081/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"ivy@example.com","password":"password-123"}' | jq -r .access_token)
STUDENT=$(curl -s -X POST localhost:8081/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"sam@example.com","password":"password-123"}' | jq -r .access_token)

# 3. Create and publish a course (the instructor is taken from the token)
COURSE=$(curl -s -X POST localhost:8082/api/v1/courses -H "Authorization: Bearer $INSTR" -H 'Content-Type: application/json' \
  -d '{"name":"Go Microservices","description":"Intro","enrollment_limit":30}' | jq -r .id)
curl -s -X POST localhost:8082/api/v1/courses/$COURSE/publish -H "Authorization: Bearer $INSTR"

# 4. Enroll (the student is taken from the token; enrolling twice returns 409)
curl -s -X POST localhost:8083/api/v1/enrollments -H "Authorization: Bearer $STUDENT" -H 'Content-Type: application/json' \
  -d "{\"course_id\":\"$COURSE\"}"
```

## API

All `/api/v1/*` routes require `Authorization: Bearer <access token>`.

| Service | Method and path | Who | Notes |
|---|---|---|---|
| user | `POST /auth/login` | anyone | `{email, password}` → `{access_token, refresh_token}` |
| user | `POST /auth/refresh` | refresh token | Rotates the refresh token; reusing an old one revokes its whole family |
| user | `POST /api/v1/users` | admin | `{name, email, role, password}` (password ≥ 8 chars) |
| user | `GET /api/v1/users/:id` | authenticated | |
| course | `POST /api/v1/courses` | instructor | `{name, description?, enrollment_limit?}` → draft course owned by the caller |
| course | `POST /api/v1/courses/:id/publish` | the course's instructor | Draft → published; 403 for anyone else, 409 if not a draft |
| course | `GET /api/v1/courses`, `GET /api/v1/courses/:id` | authenticated | |
| enrollment | `POST /api/v1/enrollments` | authenticated | `{course_id}`. 409 if already enrolled, the course is full, or the course isn't published |
| enrollment | `GET /api/v1/enrollments/:id` | authenticated | |

Internal routes, reachable on `INTERNAL_PORT` only and requiring `X-Internal-Token`:

- user-service: `GET /internal/users/:id`
- course-service: `GET /internal/courses/:id`
- enrollment-service: `GET /internal/enrollments/:id`

## Configuration

Each service reads env vars, or a `.env` file in its directory; see each `services/<service>/.env.example`.

| Variable | Default | Notes |
|---|---|---|
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | `localhost`, `5432`, `postgres`, `password`, `smartcourse_db` | |
| `PORT` / `INTERNAL_PORT` | `8081`/`9081`, `8082`/`9082`, `8083`/`9083` | Public and internal listeners |
| `ENV` | `production` | `dev` allows the defaults below for `JWT_SECRET` and `INTERNAL_API_KEY` |
| `JWT_SECRET` | dev only: `dev-secret-key` | Required (32+ chars) outside dev; must be identical in every service |
| `INTERNAL_API_KEY` | dev only: `dev-internal-api-key` | Required (32+ chars) outside dev; must be identical in every service |
| `COURSE_SERVICE_URL` | `http://course-service:9082` | enrollment-service only; course-service's internal listener |
| `OUTBOX_POLL_INTERVAL`, `OUTBOX_BATCH_SIZE` | `1s`, `100` | Outbox relay tuning |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | unset | user-service only; seeds an admin at startup if both are set |
| `SKIP_LIQUIBASE` | unset | `true` in Compose, where the Liquibase jobs run the migrations |

## Project structure

```
.
├── docker-compose.yml          # postgres, liquibase-{user,course,enrollment}, three services, temporal
├── README.md
└── smartcourse/
    ├── libs/common/            # shared Go module (no service code)
    │   ├── auth/               # JWT middleware, RequireRole, InternalAuth, UserIDFrom
    │   ├── errors/             # typed domain errors
    │   ├── httpclient/         # service-to-service client (sends X-Internal-Token)
    │   ├── httpserver/         # runs public + internal listeners with graceful shutdown
    │   ├── logger/             # structured JSON logger
    │   └── outbox/             # Event, Insert(tx, …), Relay, Publisher
    └── services/
        └── <service>/          # independent Go module, imports only itself and libs/common
            ├── cmd/main.go
            ├── config/ domain/ persistence/ mappers/ repositories/ services/ handlers/ routes/ database/
            ├── migrations/     # this service's Liquibase changelogs (001, 002, …)
            ├── liquibase/      # Liquibase image: JDBC driver + migrate.sh
            ├── Dockerfile      # build context is smartcourse/
            └── .env.example
```

## Testing

Each module is tested on its own:

```bash
cd smartcourse/libs/common           && go test ./...
cd smartcourse/services/user-service && go test ./...   # likewise course-service, enrollment-service
```

Tests that need PostgreSQL skip themselves unless `SMARTCOURSE_TEST_DSN` is set. These cover the outbox relay, the transaction rollbacks and the seat reservation. Point the variable at the Compose database to run them:

```bash
export SMARTCOURSE_TEST_DSN="host=localhost port=5432 user=postgres password=password dbname=smartcourse_db sslmode=disable"
```

Repository tests run inside a transaction that is rolled back. Outbox tests use a throwaway schema that is dropped afterwards. Neither leaves data behind.
