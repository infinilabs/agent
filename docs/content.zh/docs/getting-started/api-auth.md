---
weight: 20
title: API 认证
---

# API 认证

INFINI Agent 的所有 HTTP 接口均需要 Token 认证。Token 在进程启动时自动生成，无需手动配置。

## 获取 Token

Agent 启动后，Token 会以 `info` 级别打印在日志中：

```
[INF] [auth_token.go] [agent] api token: d70v0i4bd4atkku9ftlg
```

每次重启进程都会生成新的 Token。

## 登录验证

通过 `POST /login` 接口验证 Token，成功后可确认 Token 有效：

```bash
curl -X POST http://localhost:2900/login \
  -H "Content-Type: application/json" \
  -d '{"token": "d70v0i4bd4atkku9ftlg"}'
```

**成功响应（200）：**

```json
{"acknowledged": true}
```

**失败响应（401）：**

```json
{"status": 401, "error": {"reason": "invalid login"}}
```

## 调用其他接口

获取 Token 后，在所有请求的 `X-API-TOKEN` 请求头中携带该 Token：

```bash
curl http://localhost:2900/elasticsearch/node/_discovery \
  -H "X-API-TOKEN: d70v0i4bd4atkku9ftlg"
```

未携带或 Token 不正确时，接口返回 `401`：

```json
{"status": 401, "error": {"reason": "invalid login"}}
```
