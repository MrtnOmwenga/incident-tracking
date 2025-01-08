# Docker Development Environment Setup

## Overview
This repository uses Docker and Docker Compose to run a full-stack application with:
- Frontend (Node.js/Vite)
- Backend (Go)
- PostgreSQL database
- Database migrations

## Prerequisites
- Docker
- Docker Compose
- `.env` file with required variables:
  ```
  DB_USER=
  DB_PASSWORD=
  DB_NAME=
  ```

## Quick Start
1. Create `.env` file with required variables
2. Run migrations:
   ```bash
   docker compose --profile migrate up migration -d
   ```
3. Start services:
   ```bash
   docker compose up
   ```

## Architecture Details

### Service Configuration

#### Frontend (`frontend` service)
- Built from Node.js 22 Alpine image
- Runs on port 3000 (mapped from container port 5173)
- Volume mounts for hot reloading:
  - `./frontend:/app`: Source code
  - `/app/node_modules`: Persisted node_modules
- Depends on backend service
- Runs npm install, seeds data, and starts dev server
- Uses Vite for development

#### Backend (`backend` service)
- Built from Go 1.23 Alpine image
- Runs on port 8080
- Volume mounts:
  - `./backend:/app`: Source code
  - `/app/go/pkg/mod`: Go modules cache
- Healthcheck configured for startup validation
- Installs Wire for dependency injection
- Compiles and runs the application

#### Database (`postgres` service)
- PostgreSQL 15 Alpine image
- Persistent volume for data storage
- Healthcheck for readiness validation
- Standard port 5432 exposed

#### Migrations (`migration` service)
- Uses official migrate/migrate image
- Runs in separate profile for controlled execution
- Depends on postgres service
- Automatically applies migrations from ./backend/migrations

### Key Design Decisions

1. **Volume Mounts**
   - Development-optimized with hot reloading
   - Persistent volumes for dependencies and DB data
   - Separate bind mounts for source code

2. **Health Checks**
   - Ensures proper startup order
   - Configurable retry and timeout parameters
   - Critical for CI/CD reliability

3. **Dependency Management**
   - Dependencies installed at runtime for development
   - Wire tool installation included in backend
   - Node modules isolated in named volume

AWS access portal URL: https://d-9067c3daf6.awsapps.com/start, Username: admin, One-time password: u!2Hs)r29ry^%UDNVWj&3uXu1UCvZSkhEa3^NWRTXkmV5o8X>*c4Vh.r-w.7s4