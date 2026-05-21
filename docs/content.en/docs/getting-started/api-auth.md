---
weight: 20
title: API Authentication
---

# API Authentication

All HTTP endpoints of INFINI Agent require token-based authentication. The token is generated automatically at startup — no manual configuration needed.

## Getting the Token

After the agent starts, the token is printed to the stdout:

```
 --  Login Token  -----------------
|   Token: d7k714kbd4aisf3a7b90    |
 ----------------------------------
```

A new token is generated on every restart.

## Verifying the Token

Use `POST /login` to verify the token and confirm it is valid:

```bash
curl -X POST http://localhost:2900/login \
  -H "Content-Type: application/json" \
  -d '{"token": "d70v0i4bd4atkku9ftlg"}'
```

**Success (200):**

```json
{"acknowledged": true}
```

**Failure (401):**

```json
{"status": 401, "error": {"reason": "invalid login"}}
```

## Calling Other Endpoints

Include the token in the `X-API-TOKEN` header on every request:

```bash
curl http://localhost:2900/elasticsearch/node/_discovery \
  -H "X-API-TOKEN: d70v0i4bd4atkku9ftlg"
```

Requests without the header, or with an incorrect token, receive a `401` response:

```json
{"status": 401, "error": {"reason": "invalid login"}}
```
