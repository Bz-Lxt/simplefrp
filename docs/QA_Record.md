# QA Record

## Round 1 — 2026-08-20 14:17 CST
**Cost**: ¥0
**Mode**: Mock/offline（无外部计费 API）
**Environment**: `docker compose exec -T qa pytest tests/api_smoke.py`

**Go unit tests** (`backend`, host then confirmed via golang image): PASS
- protocol 帧编解码 / handshake
- pool 上限拒绝 / 空闲 TTL
- forward 双向拷贝
- yamux 32 路回显
- backoff 上限

**Smoke**:
```
...F...
FAILED tests/api_smoke.py::test_demo_page_through_tunnel
assert '内网节点观察窗' in '<!doctype html>...(SPA shell only)...'
6 passed, 1 failed
```

**判定**: 隧道与 identity API 已通。失败原因是 SPA 文案在 JS bundle 中，服务端 HTML 只有 title。属测试断言过严，不改产品代码。

**拟定修复（锁定）**: 断言改为 `SimpleFrp` + `id="root"` + `.js`。

---

## Round 2 — 2026-08-20 14:18 CST
**Cost**: ¥0

复跑时 2 失败：
```
FAILED test_demo_page_through_tunnel - ConnectionError: Remote end closed connection without response
FAILED test_concurrent_visitors - ConnectionError: Remote end closed connection without response
```

**根因（锁定）**: Client 连接池预拨号 TCP 后，Demo `http.Server.ReadHeaderTimeout=10s` 关闭空闲已 accept 连接；池仍按 IdleTTL=30s 复用死连接，访客得到空响应。

**拟定修复（锁定，本轮不得改方案）**:
1. Demo 取消 ReadHeaderTimeout/ReadTimeout 硬限制，IdleTimeout=120s
2. `pool.Get`/`popIdle` 对空闲连接做 1ms 探测读，EOF 丢弃并计入 expired

---

## Round 3 — 2026-08-20 14:20 CST
**Cost**: ¥0
**Command**: `docker compose exec -T qa pytest tests/api_smoke.py -q`

```
.......  7 passed in 0.30s
```

间隔 12s 后再跑（回归死连接）：
```
.......  7 passed in 0.19s
```

并发改为 100 后再跑：
```
.......  7 passed in 2.78s
```

**结果**: PASS。进入 Phase 5。
