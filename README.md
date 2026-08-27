# flowmax-ops-mcp

Flowmax 内部「经营驾驶舱 + 复盘」的 MCP Server。让老板/管理员用 Claude Code（或任意 MCP 客户端）以自然语言调取两类数据：

1. **决策复盘** —— 所有 PM 交易员的决策、订单、仓位、权益，用于复盘。
2. **经营驾驶舱** —— 平台经营数据：用户、Agent、Credit 消耗/发放、邀请码。

本服务是 ops-be（`ai-trading-ops-be`）的薄封装，只读、走 HTTP，不直接访问数据库。

## 工具列表

**复盘决策（`/api/review/*`）**

| tool | 说明 |
|---|---|
| `list_traders` | PM 交易员目录，含决策次数与最近决策时间 |
| `search_decisions` | 跨交易员检索决策列表（时间窗/交易员/状态） |
| `get_round` | 单轮完整复盘：原始决策 + 事件时间线 + 订单/仓位/权益 |
| `list_orders` | 某交易员的订单流水 |
| `list_positions` | 某交易员的仓位流水 |
| `get_equity_curve` | 某交易员的账户权益快照序列 |

**经营驾驶舱（`/api/ops/*`）**

| tool | 说明 |
|---|---|
| `ops_overview` | 总用户/新增用户/总与活跃 Agent/昨日决策/邀请码使用率 |
| `list_users` | 用户列表 |
| `list_invite_codes` | 邀请码列表 |
| `credit_summary` | Credit 按类型汇总（recharge/deduction/bonus/grant/expiry） |
| `trader_stats` | 交易员绩效统计（净盈亏/收益率/胜率/排行榜） |

## 配置

环境变量：

| 变量 | 说明 |
|---|---|
| `OPS_BE_BASE_URL` | ops-be 服务地址，如 `https://<ops-be-host>` |
| `OPS_API_KEY` | ops-be 的只读 key（对应服务端 `OPS_API_KEY`，经 `X-Ops-Key` 头传递） |

> 两个变量缺一不可，缺失时启动即报错。

## 安装与接入 Claude Code

### 方式一：`go install`

```bash
GOPROXY=https://goproxy.cn,direct go install github.com/FlowmaxAITrade/flowmax-ops-mcp/cmd/flowmax-ops-mcp@latest
```

### 方式二：下载预编译二进制

见仓库 Releases，选择对应平台（darwin/linux × amd64/arm64）下载解压。

### 接入 Claude Code

```bash
claude mcp add flowmax-ops --scope user \
  --env OPS_BE_BASE_URL=<占位> \
  --env OPS_API_KEY=<占位> \
  -- /path/to/flowmax-ops-mcp
```

之后在任意目录的 Claude Code 里即可使用上述工具。

## 开发

```bash
go build ./...
go vet ./...
go test ./...

# 本地跑（stdio，配合 MCP 客户端）
OPS_BE_BASE_URL=http://127.0.0.1:8080 OPS_API_KEY=xxx go run ./cmd/flowmax-ops-mcp
```

## 版本管理

版本号的**单一来源是 git tag**：打 `v*.*.*` tag 触发 GoReleaser 构建，`-ldflags -X` 把 tag、commit、构建时间注入二进制：

- MCP 协议 `initialize` 握手报告的版本 = tag（如 `0.1.0`）；
- `flowmax-ops-mcp --version` 打印完整构建身份：
  `flowmax-ops-mcp 0.1.0 (commit=abc1234, built=2026-08-27T10:00:00Z)`；
- 未注入 ldflags 的构建（本地 / `go install`）回退读取 Go build info，显示模块伪版本或 `dev`，并附带 commit hash（含 `+dirty` 标记，便于定位）。

规则：每次对外发布必须先打 tag，tag 采用语义化版本 `vMAJOR.MINOR.PATCH`；不手动改 `internal/version/version.go` 的默认值。

## 版本发布

打 `v*.*.*` tag 会触发 GitHub Actions 用 GoReleaser 构建 darwin/linux 的 amd64/arm64 二进制并发布到 Releases。
