# Smoke Test

Run one-command smoke test:

```bash
bash scripts/smoke.sh
```

What it does:

- Starts MySQL with Docker Compose
- Applies migrations and seed data
- Starts backend server
- Verifies `/healthz`
- Verifies FAQ search endpoint
- Prints `SMOKE PASS` on success
