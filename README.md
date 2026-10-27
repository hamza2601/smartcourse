# SmartCourse

## 1. Project Overview

SmartCourse is a backend API for a course delivery platform. It supports:

- **User management** — registering students, instructors, and admins
- **Course management** — creating and listing courses owned by instructors
- **Enrollments** — students enrolling in courses
- **Progress tracking** — tracking how far a student has progressed through a course

The backend is built with **Go**, using **Gin** for HTTP routing, **GORM** as the ORM, and **PostgreSQL** as the data store.

## 2. Tech Stack

- Go 1.25+
- [Gin](https://github.com/gin-gonic/gin) — HTTP web framework
- [GORM](https://gorm.io/) — ORM for PostgreSQL
- [PostgreSQL](https://www.postgresql.org/) — relational database
- [Docker Compose](https://docs.docker.com/compose/) — local PostgreSQL instance for development

## 3. Architecture

SmartCourse is a distributed course management platform built as true microservices:

- **User Service** (port 8081): Authentication, user management, admin seeding
- **Course Service** (port 8082): Course CRUD, validates instructors via User Service
- **Enrollment Service** (port 8083): Enrollment management, validates courses and students via inter-service APIs
- **PostgreSQL**: Single database with schema-per-service isolation (user_service, course_service, enrollment_service schemas)
- **Temporal**: Workflow orchestration server for Week 2 (course publishing pipeline)
- **Liquibase**: Versioned schema migrations, run once at startup

Services communicate via HTTP internal APIs (/internal/* endpoints) over the Docker Compose network.

Inside each service, requests flow through the same layers:

```
HTTP Request → Handlers → Services → Repositories → GORM → PostgreSQL
```

## 4. Database Schema

Four tables back the application:

### `users`

| Column     | Type      | Notes                              |
|------------|-----------|-------------------------------------|
| id         | uint      | Primary key                        |
| name       | string    | Required                           |
| email      | string    | Required, unique                   |
| role       | string    | `student`, `instructor`, or `admin`|
| created_at | timestamp | Set automatically by GORM          |
| updated_at | timestamp | Set automatically by GORM          |

### `courses`

| Column        | Type      | Notes                              |
|---------------|-----------|-------------------------------------|
| id            | uint      | Primary key                        |
| name          | string    | Required                           |
| instructor_id | uint      | Foreign key → `users.id`           |
| status        | string    | `draft` or `published` (default: `draft`) |
| created_at    | timestamp | Set automatically by GORM          |
| updated_at    | timestamp | Set automatically by GORM          |

### `enrollments`

| Column     | Type      | Notes                                        |
|------------|-----------|-----------------------------------------------|
| id         | uint      | Primary key                                  |
| student_id | uint      | Foreign key → `users.id`                     |
| course_id  | uint      | Foreign key → `courses.id`                   |
| created_at | timestamp | Set automatically by GORM                    |

`(student_id, course_id)` has a unique constraint — a student can only enroll in a given course once.

### `progress`

| Column            | Type      | Notes                                        |
|-------------------|-----------|-----------------------------------------------|
| student_id        | uint      | Foreign key → `users.id`, part of composite key |
| course_id         | uint      | Foreign key → `courses.id`, part of composite key |
| completed_lessons | int       | Default: `0`                                 |
| last_accessed     | timestamp | Last time the student accessed the course     |

`(student_id, course_id)` is a composite unique key — one progress record per student per course.

## 5. Project Structure

```
.
├── docker-compose.yml           # Postgres, Liquibase, the three services, Temporal
├── README.md
└── smartcourse/                 # Go module
    ├── cmd/
    │   ├── user-service/        # main.go + Dockerfile (port 8081)
    │   ├── course-service/      # main.go + Dockerfile (port 8082)
    │   └── enrollment-service/  # main.go + Dockerfile (port 8083)
    ├── internal/
    │   ├── config/              # Environment/config loading
    │   ├── database/            # DB connection
    │   ├── domain/              # Business entities (no DB concerns)
    │   ├── persistence/         # GORM models
    │   ├── mappers/             # domain <-> persistence conversion
    │   ├── errors/              # Domain error types
    │   ├── repositories/        # Data access layer (GORM queries)
    │   ├── services/            # Business logic, inter-service HTTP calls
    │   ├── handlers/            # HTTP request/response handling (Gin)
    │   ├── middleware/          # JWT auth + role checks
    │   ├── models/              # Outbox event models
    │   └── routes/              # Per-service route registration
    ├── liquibase/               # Dockerfile: Liquibase + PostgreSQL driver
    ├── migrations/              # Liquibase changelogs (schema source of truth)
    ├── .env.example             # Template for required environment variables
    └── go.mod
```

## 6. Setup Instructions

### Prerequisites

- Go 1.25+
- Docker (and Docker Compose)
- Git

### Steps

1. Clone the repository:
   ```
   git clone <repo-url>
   ```
2. Move into the repository root (the directory containing `docker-compose.yml`).
3. Build and start everything:
   ```
   docker-compose up --build
   ```
4. This starts PostgreSQL, runs the Liquibase migrations once, then starts the three services (User on `http://localhost:8081`, Course on `http://localhost:8082`, Enrollment on `http://localhost:8083`) and Temporal on port `7233`.

To run a service outside Docker, copy `smartcourse/.env.example` to `smartcourse/.env` and fill in the values.

## 7. API Endpoints

### `POST /api/v1/users` — Register a user

Request body:
```json
{
  "name": "John",
  "email": "john@example.com",
  "role": "instructor"
}
```

Response: `201 Created`
```json
{
  "id": 1,
  "name": "John",
  "email": "john@example.com",
  "role": "instructor",
  "created_at": "2026-09-30T10:00:00Z",
  "updated_at": "2026-09-30T10:00:00Z"
}
```

### `GET /api/v1/users/:id` — Get a user

Response: `200 OK` with the user object (same shape as above).

### `POST /api/v1/courses` — Create a course

Request body:
```json
{
  "name": "Go Backend",
  "instructor_id": 1
}
```

Response: `201 Created`
```json
{
  "id": 1,
  "name": "Go Backend",
  "instructor_id": 1,
  "status": "draft",
  "created_at": "2026-09-30T10:05:00Z",
  "updated_at": "2026-09-30T10:05:00Z"
}
```

### `GET /api/v1/courses/:id` — Get a course

Response: `200 OK` with the course object (same shape as above).

### `GET /api/v1/courses` — List all courses

Response: `200 OK`
```json
[
  {
    "id": 1,
    "name": "Go Backend",
    "instructor_id": 1,
    "status": "draft",
    "created_at": "2026-09-30T10:05:00Z",
    "updated_at": "2026-09-30T10:05:00Z"
  }
]
```

## 8. Business Rules

- A user's email must be unique across the system.
- A user's role must be one of: `student`, `instructor`, or `admin`.
- A course's `instructor_id` must reference an existing user, and that user's role must be `instructor` — otherwise course creation fails.
- A student can only enroll once per course (unique constraint on `student_id` + `course_id` in `enrollments`).
- A progress record is unique per student per course (composite key on `student_id` + `course_id` in `progress`).

## 9. Testing

You can exercise the API with `curl` or a tool like Postman.

Register a user:
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John","email":"john@example.com","role":"instructor"}'
```

Fetch that user:
```bash
curl http://localhost:8080/api/v1/users/1
```

Create a course for that instructor:
```bash
curl -X POST http://localhost:8080/api/v1/courses \
  -H "Content-Type: application/json" \
  -d '{"name":"Go Backend","instructor_id":1}'
```

Fetch a course:
```bash
curl http://localhost:8080/api/v1/courses/1
```

List all courses:
```bash
curl http://localhost:8080/api/v1/courses
```

## 10. Stopping the App

- Stop the stack with `Ctrl+C` in the terminal running `docker-compose up`, or from another terminal:
  ```
  docker-compose down
  ```
