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

Requests flow through a layered architecture:

```
HTTP Request → Handlers → Services → Repositories → GORM → PostgreSQL
```

- **Handlers** (`internal/handlers/`) — parse HTTP requests, call services, and write JSON responses. No business logic lives here.
- **Services** (`internal/services/`) — business logic and validation (e.g. checking a user's role before letting them create a course).
- **Repositories** (`internal/repositories/`) — the data access layer; all direct database queries live here.
- **GORM** — the ORM layer that translates Go structs and method calls into SQL.
- **PostgreSQL** — the underlying relational database that persists all data.

Each layer only depends on the layer directly below it, which keeps HTTP concerns, business rules, and database queries independent and easy to test in isolation.

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
smartcourse/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Environment/config loading
│   ├── database/                # DB connection + migrations
│   ├── models/                  # User, Course, Enrollment, Progress structs
│   ├── repositories/            # Data access layer (GORM queries)
│   ├── services/                # Business logic and validation
│   ├── handlers/                # HTTP request/response handling (Gin)
│   └── routes/                  # Route registration
├── migrations/                  # Reserved for future SQL migration files
├── docker-compose.yml           # Local PostgreSQL for development
├── .env                         # Local configuration (not committed)
├── .env.example                 # Template for required environment variables
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
2. Move into the Go project directory:
   ```
   cd smartcourse
   ```
3. Create a `.env` file (copy from `.env.example`) with:
   ```
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=password
   DB_NAME=smartcourse_db
   SERVER_PORT=8080
   ```
4. Start PostgreSQL via Docker Compose (run from the project root, where `docker-compose.yml` lives):
   ```
   docker-compose up -d
   ```
5. Run the application:
   ```
   go run cmd/server/main.go
   ```
6. The app listens on `http://localhost:8080`. On startup it automatically runs GORM `AutoMigrate` to create/update the `users`, `courses`, `enrollments`, and `progress` tables.

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

- Stop the Go server with `Ctrl+C`.
- Stop the PostgreSQL container:
  ```
  docker-compose down
  ```
