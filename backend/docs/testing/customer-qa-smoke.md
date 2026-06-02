# Customer QA Smoke

## Automated

```bash
cd backend
DB_PORT=3308 GO_BIN=/opt/homebrew/opt/go@1.24/bin/go bash scripts/smoke.sh
```

Expected:

- script exits with code `0`
- output ends with `SMOKE PASS`

## Manual Cases

### 1. Product location

Request:

```bash
curl -sS http://127.0.0.1:18080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"可乐在哪里"}'
```

Expected:

- `intent` is `product_location`
- `answer` contains `饮料区` and `B-02`

### 2. Inventory

Request:

```bash
curl -sS http://127.0.0.1:18080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"可乐还有吗"}'
```

Expected:

- `intent` is `inventory`
- `answer` contains `系统显示`

### 3. Promotion

Request:

```bash
curl -sS http://127.0.0.1:18080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"今天有什么优惠"}'
```

Expected:

- `intent` is `promotion`
- `answer` contains `饮料第二件半价`

### 4. FAQ

Request:

```bash
curl -sS http://127.0.0.1:18080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"怎么付款"}'
```

Expected:

- `intent` is `faq`
- `answer` contains `微信` or `支付宝`
