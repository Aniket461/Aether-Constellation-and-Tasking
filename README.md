# Aether Constellation and Tasking

Aether is a twelve-satellite synthetic-aperture-radar constellation. This repository simulates the fleet and the tasking portal that orders pictures from it.

The work is split in two.

**Constellation** owns the fleet. It stores the planes, classes, imaging modes, and satellites, propagates each orbit, and finds the earliest free collection for a point. When an order is placed it stores a task and moves that task through the satellite's work: scheduled, imaging, downlink, processing, and done. Cancelled and failed tasks stay in the table and do not block the satellite. A done task keeps its imaging second, so that second is not offered again.

**Tasking** owns the order. The portal asks constellation whether a collection is feasible, then places an order. The order stores the collection that was booked and the last status constellation reported. Once a second, tasking asks about orders that are still open and copies the status back. The portal does not decide the status from its own clock.

The imaging interval is the only time a satellite is busy. Downlink and processing happen after imaging ends and do not extend that interval.

## Layout

| Path | What it is |
| --- | --- |
| `constellation/` | Fleet catalog, orbit math, acquisitions, and tasks |
| `tasking/backend/` | Feasibility and orders |
| `tasking/frontend/` | The tasking portal |
| `database.md` | Both databases, with example rows |
| `constellation.md` | How a collection is chosen |
| `constellation/openapi.yaml` | Constellation API |
| `tasking/backend/openapi.yaml` | Tasking API |

## Tech stack

- Go 1.27.3 for both APIs
- PostgreSQL 17, one server and two databases: `constellation` and `tasking`
- [golang-migrate](https://github.com/golang-migrate/migrate) for schema changes. Each API migrates its own database on startup
- [go-satellite](https://github.com/joshuaferrara/go-satellite) for SGP4 propagation
- Angular 22 and Cesium for the portal

## Run with Docker Compose

From the repository root:

```bash
docker compose up --build
```

| Service | URL |
| --- | --- |
| Portal | http://localhost:4200 |
| Tasking API | http://localhost:8080 |
| Constellation API | http://localhost:8081 |
| Postgres | localhost:5432 |

Postgres uses user `aether`, password `aether`, and a default database named `aether` so the health check can connect. The APIs create `constellation` and `tasking` if those databases are missing.

The portal sends `/api/orders` and `/api/feasibility` to tasking. Every other `/api` request goes to constellation.

## Run locally

Start only Postgres from Compose, then run the two APIs and the portal on the host. This is the usual way to work on the code.

```bash
docker compose up -d postgres
```

Constellation listens on 8081, because both APIs default to 8080:

```bash
cd constellation
ADDR=:8081 \
DATABASE_URL='postgres://aether:aether@127.0.0.1:5432/constellation?sslmode=disable' \
go run ./cmd/api
```

Tasking listens on 8080 and calls constellation:

```bash
cd tasking/backend
ADDR=:8080 \
DATABASE_URL='postgres://aether:aether@127.0.0.1:5432/tasking?sslmode=disable' \
CONSTELLATION_URL='http://127.0.0.1:8081' \
go run ./cmd/api
```

The portal:

```bash
cd tasking/frontend
npm install
npx ng serve --host 127.0.0.1 --port 4200
```

Open http://127.0.0.1:4200. The dev server proxies `/api` the same way the Compose portal does.

## Task timing

After imaging ends, a task stays in downlink and then processing before it becomes done. Both waits are read once when constellation starts.

| Variable | Default | Meaning |
| --- | --- | --- |
| `TASK_DOWNLINK_DELAY` | `15s` | Time in downlink after imaging ends |
| `TASK_PROCESSING_DELAY` | `20s` | Time in processing after downlink |

Use Go duration values such as `15s`, `2m`, or `0s`. `0s` skips that phase. A value that does not parse, or a negative value, stops constellation from starting. A task already marked done stays done.

## Tests

The Go tests start a throwaway Postgres container, so Docker has to be running.

```bash
cd constellation && go test ./...
cd tasking/backend && go test ./...
```
