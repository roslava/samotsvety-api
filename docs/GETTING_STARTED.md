# Samotsvety API — Getting Started

## Prerequisites

- Go 1.26.4+
- Docker & Docker Compose
- `golang-migrate` (install with `make dev-install`)

## Quick Start

### 1. Install Development Tools
```bash
make dev-install
```

### 2. Start PostgreSQL (Docker)
```bash
make db-up
```
This will:
- Start PostgreSQL 16 in a Docker container
- Create `samotsvety` database
- Wait for health check (~10 seconds)

### 3. Apply Migrations
```bash
make migrate-up
```
This will create the `minerals` table and indexes.

### 4. Run the Server
```bash
make run
```

The server starts at `http://localhost:8080`

Test with:
```bash
curl http://localhost:8080/health
```

## Environment Setup

- `.env` file is created from `.env.example`
- Database credentials: postgres/postgres@localhost:5432/samotsvety
- All settings are in `.env`

## Useful Commands

```bash
make build           # Build binary
make test            # Run all tests
make clean           # Clean build artifacts
make db-down         # Stop PostgreSQL
make db-reset        # Remove PostgreSQL and data (⚠️ destructive)
make migrate-down    # Rollback migrations
make deps            # Download Go dependencies
```

## Stopping Everything

```bash
make db-down
```

## Troubleshooting

**PostgreSQL connection failed:**
- Ensure Docker is running: `docker ps`
- Check logs: `docker-compose logs postgres`
- Wait for health check (may take 10-15s)

**Migrations not applied:**
- Ensure DB is running: `make db-up`
- Check migration status: `migrate -path migrations -database "$(DB_URL)" version`

**Port 5432 already in use:**
- Stop other PostgreSQL: `docker ps` and check containers
- Or change `DB_PORT` in `.env`

---

## Architecture

```
cmd/server/          ← Application entry point
internal/
  ├── domain/        ← Data models (Mineral, etc.)
  ├── repository/    ← Database abstraction
  ├── handler/       ← HTTP handlers
  ├── config/        ← Configuration
  └── server/        ← Server setup
migrations/          ← SQL migrations
seeds/               ← Sample data
```

## Next Steps

1. ✅ Phase 0-1: Domain models & in-memory repository (DONE)
2. ⏳ Phase 2: PostgreSQL integration
3. ⏳ Phase 3: HTTP handlers
4. ⏳ Phase 4-8: Features, docs, finalization
