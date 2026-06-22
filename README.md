# Store Mind Monorepo

This repository hosts both backend and frontend code.

## Structure

- `backend/`: Go backend (Clean Architecture scaffold)
- `frontend/`: frontend application
- `agents/`: AI agents for orchestration and retrieval

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

```bash
cd services/agent
pip install -r requirements.txt  # 首次
DEEPSEEK_API_KEY=sk-your-key python server.py
```
- 如果没有 API Key，跳过此步骤——Go 后端会自动降级到关键词匹配。

