# SimpleFrp HTTP API

所有时间字段对外展示为北京时间 `yyyy-MM-dd HH:mm:ss`。成功响应使用 `{ "data": ... }`，错误使用 `{ "error": { "code", "message" } }`。

## 错误码

| code | HTTP | 含义 |
|---|---|---|
| `not_found` | 404 | 路径不存在 |
| `not_ready` | 503 | 隧道未建立 |
| `invalid_json` | 400 | 请求体无法解析 |
| `internal_error` | 500 | 未预期错误 |

## Server 状态 API（容器内 `:9090`，宿主机 `42818`）

### `GET /api/v1/health`

**响应 200**

```json
{
  "data": {
    "status": "ok",
    "role": "server",
    "time": "2026-08-20 14:00:00"
  }
}
```

### `GET /api/v1/status`

**响应 200**

```json
{
  "data": {
    "role": "server",
    "client_connected": true,
    "client_id": "edge-01",
    "session_id": "a1b2c3d4",
    "active_streams": 2,
    "bytes_up": 1024,
    "bytes_down": 4096,
    "visitors_total": 15,
    "visitors_rejected": 0,
    "time": "2026-08-20 14:00:00"
  }
}
```

## Client 健康 API（容器内 `:9091`，不对外）

### `GET /api/v1/health`

```json
{
  "data": {
    "status": "ok",
    "role": "client",
    "connected": true,
    "time": "2026-08-20 14:00:00"
  }
}
```

### `GET /api/v1/status`

```json
{
  "data": {
    "role": "client",
    "connected": true,
    "client_id": "edge-01",
    "reconnects": 0,
    "pool": {
      "idle": 4,
      "active": 1,
      "dialed": 12,
      "reused": 8,
      "rejected": 0,
      "expired": 1
    },
    "time": "2026-08-20 14:00:00"
  }
}
```

**503 示例（隧道未就绪）**

```json
{
  "error": {
    "code": "not_ready",
    "message": "control tunnel is not connected"
  }
}
```

## Demo 内网服务（容器内 `:8080`）

### `GET /api/v1/health`

```json
{
  "data": {
    "status": "ok",
    "role": "demo",
    "node_id": "intranet-alpha-7",
    "time": "2026-08-20 14:00:00"
  }
}
```

### `GET /api/v1/identity`

访客经隧道访问本接口即可证明穿透成功。`node_id` 必须等于环境变量 `SIMPLEFRP_NODE_ID`。

```json
{
  "data": {
    "node_id": "intranet-alpha-7",
    "hostname": "demo",
    "hits": 3,
    "remote_addr": "172.18.0.4:53122",
    "time": "2026-08-20 14:00:01",
    "time_rfc3339": "2026-08-20T14:00:01+08:00"
  }
}
```
