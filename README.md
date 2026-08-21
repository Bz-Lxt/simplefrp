# SimpleFrp

Go 实现的简易内网穿透（FRP/NPS 核心子集）：Server 监听公网，Client 与内网服务及 Server 保持 TCP 长连接，在单条隧道上做 yamux 多路复用，并维护本地连接池。

## 1. 如何启动

```bash
docker compose up --build -d
```

浏览器打开 http://127.0.0.1:42817 （访客端口，流量经过隧道）。

## 2. 使用说明

Server 在公网侧监听控制口与访客户口。Client 使用 token 注册后，外部访问访客户口的 TCP 流量经 yamux 流转发到内网 Demo。断开控制连接后 Client 会在 2s 退避内重连。

默认 token：`simplefrp-dev-token`。可用环境变量 `SIMPLEFRP_TOKEN` 覆盖。

## 3. 服务列表及API说明

| 入口 | 地址 | 说明 |
|---|---|---|
| 访客（验证主入口） | http://127.0.0.1:42817 | 经隧道访问内网 Demo |
| Server 状态 API | http://127.0.0.1:42818 | `/api/v1/health` `/api/v1/status` |
| 控制面 | 127.0.0.1:42819 | Client 接入（TCP） |
| Demo 直连（对照） | http://127.0.0.1:42820 | 不经隧道 |

完整契约见 `docs/API.md`。

## 4. 测试账号

无用户系统。隧道口令：`simplefrp-dev-token`。Client ID：`edge-01`。内网节点：`intranet-alpha-7`。

## 5. 题目内容

使用 Go 实现简易版内网穿透工具（类似 FRP/NPS 的核心子集）。包含 Client 与 Server。Server 监听公网端口，Client 连接内网服务并与 Server 建立 TCP 长连接。实现流量转发、多路复用与基础连接池管理。

## 6. 项目结构

```
backend/            Server / Client / Demo（Go）
frontend-user/     内网 Demo 站点
tests/             API smoke（容器内 pytest）
docs/              需求、设计、QA、审计
docker-compose.yml
```

## 7. API 模拟与切换指南

本项目没有第三方按量 API，因此不存在 Mock Provider。所有转发路径均为真实 TCP：访客 → Server → yamux → Client → 内网服务。测试在 Docker 网络内打真实链路，预期费用 ¥0。若将来接入 TLS 或外部鉴权，应在本段补充 real/mock 开关，不得用假转发静默替换隧道。
