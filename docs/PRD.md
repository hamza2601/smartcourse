# SmartCourse product requirements

Status values: Done (built and verified in the code), Partial (built in part or as a stub), Planned (not built yet). Planned items belong to Week 3.

## 1. Overview and goals

SmartCourse is the backend of a course delivery platform. Instructors build and publish courses, students enroll and keep an enrollment history, and administrators manage accounts. The system is split into three independently deployable services (user, course, enrollment) that communicate through a protected internal API and through domain events.

Goals:

- Keep each service autonomous: its own module, its own database schema, its own migrations.
- Never lose or corrupt state: state changes and their events are written together, and a failed publish rolls back cleanly.
- Make enrollment correct under concurrency: no duplicate active enrollments, no seats beyond the limit.
- Move toward asynchronous, event-driven processing, analytics and observability (Week 3).

## 2. Roles and key use cases

| Role | Use cases |
|---|---|
| Admin | Log in; create user accounts with any role; read any enrollment; read any user |
| Instructor | Log in; create and update own courses; add, edit and remove modules and lessons; publish a course and see it return to draft if publishing fails; republish a ready course |
| Student | Log in; browse courses; read the content of ready courses; enroll in a ready course; list own enrollments; cancel an enrollment and free the seat; enroll again after cancelling |

## 3. Functional requirements

| ID | Requirement |
|---|---|
| FR-1 | An admin creates users with the role student, instructor or admin; every route enforces the roles allowed to call it. The first admin is seeded from configuration. |
| FR-2 | Users authenticate with email and password and receive a short-lived access token and a refresh token; refresh tokens rotate on use and a reused token revokes its family. |
| FR-3 | An instructor creates a course (name, description, optional enrollment limit) and can update these fields of a draft or ready course. Any authenticated user can list and read courses. |
| FR-4 | The owning instructor manages ordered modules and lessons of a draft or ready course; non-owners can read the content only once the course is ready. |
| FR-5 | Only students enroll, only in a ready course, only once at a time (no duplicate active enrollment), and only while seats remain under the course's enrollment limit (no limit means unlimited). |
| FR-6 | A student lists their own enrollment history (newest first) and cancels an active enrollment, which releases the seat; a cancelled student can enroll again. Only the student or an admin can read an enrollment. |
| FR-7 | Publishing is a workflow started by the instructor (HTTP 202). The course moves draft or ready to publishing, runs the steps below, and ends ready. A second publish of the same course while one runs is rejected. |
| FR-8 | Publishing validates course metadata: non-empty description, no module without lessons, no lesson with an empty title. |
| FR-9 | Publishing verifies assets: every lesson has non-empty content. |
| FR-10 | Publishing updates the search index. |
| FR-11 | Publishing refreshes the cache. |
| FR-12 | Publishing initialises course analytics. |
| FR-13 | Publishing marks the course ready and records a `CoursePublished` event; ready opens the course for enrollment. |
| FR-14 | A failure in any publishing step must not corrupt state: the course returns to its previous status and a `CoursePublishFailed` event with the reason is recorded. |
| FR-15 | Enrollment is recorded atomically with its seat reservation and its event. |
| FR-16 | Enrollment initialises the student's lesson progress. |
| FR-17 | Enrollment triggers analytics updates and notifications. |
| FR-18 | Enrollment processing is idempotent, applies backpressure under load and recovers from failures. |
| FR-19 | Domain events are written to a per-service outbox in the same transaction as the change and delivered by a relay at least once. |
| FR-20 | Events are published to Kafka and consumed asynchronously by background workers. |
| FR-21 | Analytics: total students, total instructors, courses published, enrollments over time, completion rate, average time to complete, most popular courses, average courses per student, failed events. |

## 4. Non-functional requirements

| ID | Requirement |
|---|---|
| NFR-1 | Security: identity comes only from the verified token; JWT secret and internal key are mandatory (32+ characters) outside dev; `/internal/*` is served on a separate unpublished port and needs a shared key. |
| NFR-2 | Consistency: business writes and their outbox rows commit together; seat reservation is a single conditional update. |
| NFR-3 | Isolation: one Go module and one database schema per service; no cross-schema foreign keys. |
| NFR-4 | Publishing reliability: activities are idempotent and retried with backoff; status changes are guarded updates; the workflow has an overall timeout. |
| NFR-5 | Event delivery is at least once and ordered by creation time within a service. |
| NFR-6 | Observability, logs: structured JSON logs with the service name. |
| NFR-7 | Observability, tracing: OpenTelemetry instrumentation with Jaeger. |
| NFR-8 | Observability, metrics: Prometheus metrics with Grafana dashboards. |
| NFR-9 | Go concurrency: goroutines, WaitGroups, channels, worker pools, mutexes, atomics and context propagation used where they fit. |
| NFR-10 | Testability: business rules are testable without a database; database behaviour is covered by tests that roll back their changes. |
| NFR-11 | Deployability: one command starts the whole stack; migrations run per service; database and Temporal state persist across restarts. |

## 5. Timeline and milestones

| Week | Scope | Status |
|---|---|---|
| 1 | Three services in separate modules; schema per service with Liquibase; JWT authentication, refresh rotation and roles; user, course and enrollment APIs; Docker Compose | Done |
| 2 | Transactional outbox and relay; identity from the token; protected internal API; modules and lessons; status model with `ready` and `publishing`; enrollment rules, cancel and history; Temporal publishing workflow with rollback; course update | Done |
| 3 | Kafka publishing and consumers; background workers; enrollment workflow with progress initialisation; analytics; logs, tracing and metrics; real search index, cache and analytics steps | Not started |

## 6. Traceability

Paths are relative to `smartcourse/`.

| Requirement | Implemented by | Status |
|---|---|---|
| FR-1 | `services/user-service/handlers/user_handler.go`, `services/user-service/services/user_service.go`, `services/user-service/cmd/main.go` (admin seed), `libs/common/auth/middleware.go` (`RequireRole`) | Done |
| FR-2 | `services/user-service/handlers/auth_handler.go`, `services/user-service/services/auth_service.go`, `services/user-service/services/token_service.go` | Done |
| FR-3 | `services/course-service/handlers/course_handler.go`, `services/course-service/services/course_service.go`, `services/course-service/repositories/course_repository.go` | Done |
| FR-4 | `services/course-service/handlers/content_handler.go`, `services/course-service/services/content_service.go`, `services/course-service/repositories/content_repository.go` | Done |
| FR-5 | `services/enrollment-service/services/enrollment_service.go`, `services/enrollment-service/repositories/enrollment_repository.go`, `services/enrollment-service/routes/enrollment_routes.go`, migrations 001 and 003 | Done |
| FR-6 | `services/enrollment-service/services/enrollment_service.go`, `services/enrollment-service/repositories/enrollment_repository.go` | Done |
| FR-7 | `services/course-service/workflows/publish_course.go`, `workflows/publish_starter.go`, `services/course-service/services/course_service.go` (`PublishCourse`), `services/course-service/worker/worker.go` | Done |
| FR-8 | `services/course-service/workflows/publish_activities.go` (`ValidateCourse`) | Done |
| FR-9 | `services/course-service/workflows/publish_activities.go` (`VerifyAssets`) | Done |
| FR-10 | `services/course-service/workflows/publish_activities.go` (`UpdateSearchIndex`, logs only) | Partial |
| FR-11 | `services/course-service/workflows/publish_activities.go` (`RefreshCache`, logs only) | Partial |
| FR-12 | `services/course-service/workflows/publish_activities.go` (`InitAnalytics`, logs only) | Partial |
| FR-13 | `services/course-service/workflows/publish_activities.go` (`FinalizePublishing`), `services/course-service/domain/events.go` | Done |
| FR-14 | `services/course-service/workflows/publish_course.go` (compensation), `workflows/publish_activities.go` (`RevertPublishing`), `domain/events.go` | Done |
| FR-15 | `services/enrollment-service/repositories/enrollment_repository.go` (`Enroll`) | Done |
| FR-16 | Table `lesson_progress` and its model exist; nothing writes to it | Planned (Week 3) |
| FR-17 | None | Planned (Week 3) |
| FR-18 | Duplicate enrollments are prevented by the transaction and a partial unique index (`services/enrollment-service/repositories/enrollment_repository.go`, migration 001); no enrollment workflow, backpressure or recovery | Partial |
| FR-19 | `libs/common/outbox/`, `outbox_events` migrations in every service | Done |
| FR-20 | `Publisher` interface in `libs/common/outbox/relay.go` with a logging implementation only; no Kafka and no workers | Planned (Week 3) |
| FR-21 | None | Planned (Week 3) |
| NFR-1 | `libs/common/auth/`, `services/*/config/jwt_config.go`, `services/*/config/internal_config.go`, `docker-compose.yml` (internal ports not published) | Done |
| NFR-2 | `libs/common/outbox/outbox.go`, `services/enrollment-service/repositories/enrollment_repository.go` | Done |
| NFR-3 | `services/*/go.mod`, `services/*/migrations/`, `services/*/persistence/` | Done |
| NFR-4 | `services/course-service/workflows/` | Done |
| NFR-5 | `libs/common/outbox/relay.go`; delivery goes to the log only until Kafka exists | Partial |
| NFR-6 | `libs/common/logger/logger.go` | Done |
| NFR-7 | None | Planned (Week 3) |
| NFR-8 | None | Planned (Week 3) |
| NFR-9 | Goroutines: outbox relays, Temporal worker and servers (`services/*/cmd/main.go`, `libs/common/httpserver/server.go`). WaitGroup, channels and a bounded set of goroutines: `VerifyAssets`. Mutex: `workflows/publish_starter.go`. Context propagation: all layers. No atomics and no general worker pool yet | Partial |
| NFR-10 | Store interfaces in `services/*/services/`; `*_test.go` files in each module; database tests run only with `SMARTCOURSE_TEST_DSN` | Done |
| NFR-11 | `docker-compose.yml`, `services/*/liquibase/` | Done |
