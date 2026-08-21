# Design Spec — SimpleFrp Demo Node

> 审美方向：**海缆登陆站 / 声纳甲板**（Cable Landing Station）
> 不是通用仪表盘，而是一间压暗的机房观察窗：磷光绿、警戒琥珀、钢青底。

## 1. 概念

用户经公网访客端口看到的，应是「内网节点本体」，而不是管理后台。
页面证明三件事：这台机器在内网、流量穿过隧道、当前北京时间是活的。

记忆点：中央声纳环上有一条缓慢爬行的光点，代表一条 yamux 逻辑流。

## 2. 色彩

| Token | Hex | 用途 |
|---|---|---|
| `--void` | `#070B10` | 页面底 |
| `--steel` | `#121A24` | 卡片 |
| `--steel-2` | `#1B2633` | 抬升面 |
| `--line` | `#2A3A4A` | 分割 |
| `--phosphor` | `#7CFFB2` | 在线 / 主强调 |
| `--phosphor-dim` | `#1E4D38` | 强调底 |
| `--amber` | `#F5B942` | 警示 / 等待 |
| `--fog` | `#C5D0DA` | 主文字 |
| `--mist` | `#7A8B9A` | 次文字 |
| `--danger` | `#FF6B6B` | 错误 Toast |

禁止紫色渐变白底、禁止 Inter/Roboto 作为主字体。

## 3. 字体

- Display：`Syne`（标题、节点名）
- Body：`Figtree`（说明文字）
- Mono：`Fragment Mono`（时间、地址、计数）

## 4. 布局

全宽 `w-full`，无页面级 `max-w-*`。

```
┌─────────────────────────────────────────────────────┐
│ 顶栏：SIMPLEFRP · NODE  /  北京时间时钟  /  状态芯片 │
├──────────────────┬──────────────────────────────────┤
│ 身份铭牌         │ 声纳隧道示意（动画光点）           │
│ node_id/hostname │ Client ──mux── Server ──visitor──│
├──────────────────┴──────────────────────────────────┤
│ 探测记录表（最近请求，等宽数字）                      │
└─────────────────────────────────────────────────────┘
```

断点：768px 双列变单列；480px 顶栏堆叠、表格横向滚动。

## 5. 组件

- **StatusChip**：Online / Degraded / Offline，磷光呼吸灯
- **IdentityPlate**：node_id、hostname、hits
- **SonarDeck**：CSS 同心圆 + 一条贝塞尔光缆
- **ProbeTable**：时间 / 延迟 / 路径，等宽
- **Toast**：手动关闭 + 5s 自动消失（禁止 alert）

## 6. 交互

- 进入：标题、芯片、声纳依次 stagger 显现（CSS animation-delay）
- 悬停：卡片边框由 `--line` 过渡到 `--phosphor`
- 「重新探测」：真实请求 `/api/v1/identity`，按钮 loading 态
- 每 3s 静默刷新身份；失败弹出 Toast

## 7. 时间

所有可见时间：`yyyy-MM-dd HH:mm:ss`，时区 GMT+8。
内部传输可用 RFC3339。
