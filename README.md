# Store Mind Monorepo

This repository hosts both backend and frontend code.

## Structure

- `backend/`: Go backend (Clean Architecture scaffold)
- `frontend/`: frontend application

## Backend Quick Start

```bash
cd backend
make test
make smoke
```

If port conflicts happen, run smoke with a custom DB port:

```bash
cd backend
DB_PORT=3308 bash scripts/smoke.sh
```
