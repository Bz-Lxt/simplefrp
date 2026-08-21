# 原始需求 Prompt

使用Go语言实现简易版内网穿透工具（类似 FRP/NPS 的核心子集）具体功能：包含一个 Client 端和一个 Server 端。Server 端监听公网端口，Client 端连接内网服务并与 Server 建立长连接（TCP 隧道）。实现流量转发、多路复用（Multiplexing）和基础的连接池管理。
