# Incident Tracking System Documentation

## Table of Contents
- [Overview](#overview)
- [Project Structure](#project-structure)
- [Setup and Installation](#setup-and-installation)
- [Architecture](#architecture)
- [API Endpoints](#api-endpoints)
- [Development Guidelines](#development-guidelines)
- [Database](#database)

## Overview
The Incident Tracking System is a Go-based backend service that manages and tracks incidents. It provides RESTful APIs for creating, updating, and managing incidents along with their associated images and comments.

## Project Structure
```
.
├── cmd/
│   └── main.go           # Application entry point
├── pkg/
│   ├── containers/       # Dependency injection setup
│   │   ├── container.go  # Main container definitions
│   │   └── wire.go      # Wire dependency injection
│   ├── controllers/      # HTTP request handlers
│   ├── repositories/     # Database operations
│   ├── middleware/       # HTTP middleware
│   ├── services/         # Business logic
│   └── models/          # Data models
├── scripts/
│   └── migrate.sh       # Database migration script
├── go.mod               # Go module definition
└── go.sum               # Go module checksums
```

## Setup and Installation

### Prerequisites
- Go 1.21 or higher
- PostgreSQL database
- Wire (dependency injection tool)

### Initial Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd <project-directory>
   ```

2. **Install dependencies**
   ```bash
   go install
   go get
   ```

3. **Install Wire CLI**
   ```bash
   go install github.com/google/wire/cmd/wire@latest
   ```

4. **Generate Wire code**
   ```bash
   wire ./pkg/containers
   ```
   This command generates dependency injection code based on the providers defined in your application. Wire automatically creates the necessary code to wire up your application's components (repositories, services, controllers) based on their dependencies.

5. **Run database migrations**
   ```bash
   ./scripts/migrate.sh
   ```

6. **Start the application**
   ```bash
   cd cmd
   go run main.go
   ```

## Architecture

### Dependency Injection
The project uses the Wire framework for dependency injection, following these principles:
- Clear separation of concerns
- Testable components
- Maintainable dependency graph

Key components:
```go
type AppContainer struct {
    Controllers   *controllers.ControllerContainer
    Services      *services.ServiceContainer
    Repositories  *repositories.RepositoryContainer
    Router        *mux.Router
}
```

### Layers

1. **Controllers Layer**
   - Handles HTTP requests/responses
   - Input validation
   - Route handling
   - HTTP status codes

2. **Services Layer**
   - Business logic
   - Data transformation
   - Cross-cutting concerns

3. **Repository Layer**
   - Database operations
   - Data persistence
   - Query handling

## API Endpoints

### Incidents

```
GET    /api/v1/incidents           # Get all incidents
POST   /api/v1/incidents           # Create new incident
GET    /api/v1/incidents/{id}      # Get incident by ID
PUT    /api/v1/incidents/{id}      # Update incident
DELETE /api/v1/incidents/{id}      # Delete incident
```

### Images

```
GET    /api/v1/incidents/{id}/images      # Get images for incident
POST   /api/v1/incidents/{id}/images      # Upload image
DELETE /api/v1/incidents/{id}/images/{id} # Delete image
```

### Request/Response Examples

#### Create Incident
```json
POST /api/v1/incidents
{
    "title": "System Outage",
    "description": "Main database is down",
    "severity": "high",
    "status": "open"
}
```

#### Response
```json
{
    "id": "uuid-string",
    "title": "System Outage",
    "description": "Main database is down",
    "severity": "high",
    "status": "open",
    "created_at": "2024-01-06T10:00:00Z",
    "updated_at": "2024-01-06T10:00:00Z"
}
```

## Development Guidelines

### Adding New Features

1. **Create Models**
   - Define GORM models in `pkg/models`
   - Include proper validation tags

2. **Implement Repository**
   - Create repository in `pkg/repositories`
   - Implement CRUD operations

3. **Create Service**
   - Add business logic in `pkg/services`
   - Implement validation

4. **Add Controller**
   - Create endpoints in `pkg/controllers`
   - Handle HTTP operations

5. **Update Wire**
   - Add providers to container
   - Regenerate wire code

### Validation
The system uses `go-playground/validator` for request validation:

```go
type IncidentDTO struct {
    Title       string   `validate:"required,min=3,max=255"`
    Description string   `validate:"required,min=10"`
    Severity    Severity `validate:"required,oneof=high medium low"`
    Status      Status   `validate:"required"`
}
```

### Error Handling
Consistent error handling across the application:
```go
if err != nil {
    return nil, fmt.Errorf("failed to create incident: %w", err)
}
```

## Database

### Models

#### Incident
```go
type Incident struct {
    ID          uuid.UUID  `gorm:"type:uuid"`
    Title       string     `gorm:"size:255;not null"`
    Description string     `gorm:"type:text;not null"`
    Status      Status     `gorm:"type:\"Status\""`
    Severity    Severity   `gorm:"type:\"Severity\""`
    Images      []Image    `gorm:"foreignKey:IncidentID"`
    Comments    []Comment  `gorm:"foreignKey:IncidentID"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Migrations
Run migrations using the provided script:
```bash
./scripts/migrate.sh
```

## Testing

### Running Tests
```bash
go test ./...
```

### Writing Tests
Example test structure:
```go
func TestIncidentService_CreateIncident(t *testing.T) {
    // Test implementation
}
```

## Deployment

### Building
```bash
go build -o incident-tracker ./cmd/main.go
```

### Environment Variables
```
DB_HOST=localhost
DB_PORT=5432
DB_NAME=incidents
DB_USER=postgres
DB_PASSWORD=password
```

## Contributing
1. Fork the repository
2. Create a feature branch
3. Commit changes
4. Push to branch
5. Create Pull Request