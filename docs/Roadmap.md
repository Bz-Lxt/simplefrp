# SimpleFrp 路线图

> 规模：预估 3k–5k LoC（< 10,000，无需 MVP/V1/V2 强制分界）
> 日期：2026-08-20

## Phase Order Decision

**UI-First（默认）**

理由：验证面是内网 Demo 静态站点，组件结构不依赖隧道数据模型；先冻结视觉契约，再把真实 TCP 隧道接到同一套 `/api/v1/identity` 接口上。

## 目录结构

```
backend/                 Go：server / client / demo
frontend-user/          内网 Demo 站点（经隧道访问的验证 UI）
frontend-admin/         不适用（需求明确不做 Dashboard）
frontend-mp/            不适用（非微信小程序）
tests/                  API smoke + E2E
docs/                   需求 / 路线图 / 设计 / QA / 审计
docker-compose.yml      随机端口（开发阶段）
```

## 任务清单

### Phase 1 — 架构骨架
- [x] Git 初始化与 `.gitignore`
- [x] `docs/Roadmap.md` 与相位顺序决策
- [x] `docker-compose.yml` 随机端口骨架
- [x] 模块目录落位

### Phase 2 — UI（内网 Demo）
- [x] `docs/DesignSpec.md`
- [x] `frontend-user` React/Vite/Tailwind 站点
- [x] 身份探测、隧道示意、错误 Toast、北京时间展示

### Phase 3 — 逻辑
- [x] 帧编解码与握手协议
- [x] yamux 多路复用会话
- [x] 连接池（上限 / 空闲 TTL / 预拨号）
- [x] 双向转发
- [x] Server：控制口 + 访接口 + 状态 API
- [x] Client：长连接、断线重连、本地池
- [x] Demo HTTP 服务托管前端产物
- [x] 多阶段跨平台 Dockerfile

### Phase 4 — QA
- [x] 核心包单元测试
- [x] `tests/api_smoke.py`（Mock/离线，¥0）
- [x] `tests/e2e_flow.spec.ts`
- [x] `docs/QA_Record.md`

### Phase 5 — 审计
- [x] `docs/AuditReport.md`
- [x] Knowledge Harvest

## 端口（开发阶段，随机）

| 服务 | 容器内 | 宿主机 |
|---|---|---|
| Server 访客（验证入口） | 8080 | 42817 |
| Server 状态 API | 9090 | 42818 |
| Server 控制面 | 7000 | 42819 |
| Demo 直连（对照） | 8080 | 42820 |
