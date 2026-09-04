package mcp

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/client"
	"github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer builds the MCP server and registers all tools. The server exposes
// two tool families against ops-be read-only endpoints:
//
//	decision review  -> /api/v1/reporting/*
//	business metrics -> /api/v1/reporting/*
func NewServer(opsBEBaseURL, opsAPIKey string) *server.MCPServer {
	s := server.NewMCPServer("flowmax-ops-mcp", version.Version)
	r := &registry{client: client.NewClient(opsBEBaseURL, opsAPIKey)}
	registerReviewTools(s, r)
	registerOpsTools(s, r)
	return s
}

type registry struct {
	client *client.Client
}

// get performs a GET and returns the pretty-printed data payload as a tool
// result. Errors are reported inside the result (not as handler errors).
func (r *registry) get(ctx context.Context, path string, query url.Values) (*mcp.CallToolResult, error) {
	data, err := r.client.Get(ctx, path, query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return r.render(data), nil
}

// getReview is like get, but cross-trader review endpoints return 202
// "calculating" until a background snapshot is ready. It polls until the result
// leaves "calculating" (or a timeout), so callers always see the final data.
const maxReviewAttempts = 10

// reviewPollInterval is a package var so tests can shorten it.
var reviewPollInterval = 3 * time.Second

func (r *registry) getReview(ctx context.Context, path string, query url.Values) (*mcp.CallToolResult, error) {
	for attempt := 0; attempt < maxReviewAttempts; attempt++ {
		data, err := r.client.Get(ctx, path, query)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var status struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(data, &status) != nil || status.Status != "calculating" {
			return r.render(data), nil
		}
		time.Sleep(reviewPollInterval)
	}
	return mcp.NewToolResultError("review query did not become ready within timeout"), nil
}

func (r *registry) render(data json.RawMessage) *mcp.CallToolResult {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return mcp.NewToolResultError("decode data: " + err.Error())
	}
	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("encode data: " + err.Error())
	}
	return mcp.NewToolResultText(string(pretty))
}

func setStr(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func setInt(q url.Values, key string, value int) {
	q.Set(key, strconv.Itoa(value))
}

func registerReviewTools(s *server.MCPServer, r *registry) {
	s.AddTool(mcp.NewTool("list_pm_agents",
		mcp.WithDescription("列出所有 PM 交易员目录，含决策次数与最近决策时间。"),
		mcp.WithString("q", mcp.Description("搜索关键词（名称/邮箱）")),
		mcp.WithString("exchange", mcp.Description("交易所筛选")),
		mcp.WithString("symbol", mcp.Description("交易对筛选")),
		mcp.WithBoolean("is_active", mcp.Description("是否活跃，默认 true")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页条数，默认 20，最大 100")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		setStr(q, "q", req.GetString("q", ""))
		setStr(q, "exchange", req.GetString("exchange", ""))
		setStr(q, "symbol", req.GetString("symbol", ""))
		q.Set("is_active", strconv.FormatBool(req.GetBool("is_active", true)))
		setInt(q, "page", req.GetInt("page", 1))
		setInt(q, "page_size", req.GetInt("page_size", 20))
		return r.getReview(ctx, "/api/v1/reporting/pm-agents", q)
	})

	s.AddTool(mcp.NewTool("search_decisions",
		mcp.WithDescription("跨交易员检索决策列表，可按时间窗、交易员、状态筛选。"),
		mcp.WithString("start", mcp.Description("起始时间 RFC3339，如 2026-08-01T00:00:00Z")),
		mcp.WithString("end", mcp.Description("结束时间 RFC3339")),
		mcp.WithString("pm_id", mcp.Description("限定某个交易员（可选）")),
		mcp.WithString("status", mcp.Description("状态过滤：created/skipped/execution_failed")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页条数，默认 20，最大 100")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		setStr(q, "start", req.GetString("start", ""))
		setStr(q, "end", req.GetString("end", ""))
		setStr(q, "pm_id", req.GetString("pm_id", ""))
		setStr(q, "status", req.GetString("status", ""))
		setInt(q, "page", req.GetInt("page", 1))
		setInt(q, "page_size", req.GetInt("page_size", 20))
		return r.getReview(ctx, "/api/v1/reporting/decisions", q)
	})

	s.AddTool(mcp.NewTool("get_round",
		mcp.WithDescription("获取单轮完整复盘：原始决策 + 事件时间线 + 订单/仓位/权益。"),
		mcp.WithString("pm_id", mcp.Required(), mcp.Description("交易员 ID")),
		mcp.WithString("round_id", mcp.Required(), mcp.Description("轮次 ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pmID := req.GetString("pm_id", "")
		roundID := req.GetString("round_id", "")
		if pmID == "" || roundID == "" {
			return mcp.NewToolResultError("pm_id 和 round_id 必填"), nil
		}
		path := "/api/v1/reporting/pm-agents/" + url.PathEscape(pmID) + "/rounds/" + url.PathEscape(roundID)
		return r.get(ctx, path, nil)
	})

	s.AddTool(mcp.NewTool("list_orders",
		mcp.WithDescription("某交易员的订单流水（时间窗，分页）。"),
		mcp.WithString("pm_id", mcp.Required(), mcp.Description("交易员 ID")),
		mcp.WithString("start", mcp.Description("起始时间 RFC3339")),
		mcp.WithString("end", mcp.Description("结束时间 RFC3339")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页条数，默认 100，最大 1000")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pmID := req.GetString("pm_id", "")
		if pmID == "" {
			return mcp.NewToolResultError("pm_id 必填"), nil
		}
		q := url.Values{}
		setStr(q, "start", req.GetString("start", ""))
		setStr(q, "end", req.GetString("end", ""))
		setInt(q, "page", req.GetInt("page", 1))
		setInt(q, "page_size", req.GetInt("page_size", 100))
		return r.get(ctx, "/api/v1/reporting/pm-agents/"+url.PathEscape(pmID)+"/orders", q)
	})

	s.AddTool(mcp.NewTool("list_positions",
		mcp.WithDescription("某交易员的仓位流水（时间窗，分页）。"),
		mcp.WithString("pm_id", mcp.Required(), mcp.Description("交易员 ID")),
		mcp.WithString("start", mcp.Description("起始时间 RFC3339")),
		mcp.WithString("end", mcp.Description("结束时间 RFC3339")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页条数，默认 100，最大 1000")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pmID := req.GetString("pm_id", "")
		if pmID == "" {
			return mcp.NewToolResultError("pm_id 必填"), nil
		}
		q := url.Values{}
		setStr(q, "start", req.GetString("start", ""))
		setStr(q, "end", req.GetString("end", ""))
		setInt(q, "page", req.GetInt("page", 1))
		setInt(q, "page_size", req.GetInt("page_size", 100))
		return r.get(ctx, "/api/v1/reporting/pm-agents/"+url.PathEscape(pmID)+"/positions", q)
	})

	s.AddTool(mcp.NewTool("get_equity_curve",
		mcp.WithDescription("某交易员的账户权益快照序列（时间窗，分页），用于画权益曲线。"),
		mcp.WithString("pm_id", mcp.Required(), mcp.Description("交易员 ID")),
		mcp.WithString("start", mcp.Description("起始时间 RFC3339")),
		mcp.WithString("end", mcp.Description("结束时间 RFC3339")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页条数，默认 100，最大 1000")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pmID := req.GetString("pm_id", "")
		if pmID == "" {
			return mcp.NewToolResultError("pm_id 必填"), nil
		}
		q := url.Values{}
		setStr(q, "start", req.GetString("start", ""))
		setStr(q, "end", req.GetString("end", ""))
		setInt(q, "page", req.GetInt("page", 1))
		setInt(q, "page_size", req.GetInt("page_size", 100))
		return r.get(ctx, "/api/v1/reporting/pm-agents/"+url.PathEscape(pmID)+"/equity", q)
	})
}

func registerOpsTools(s *server.MCPServer, r *registry) {
	s.AddTool(mcp.NewTool("ops_overview",
		mcp.WithDescription("平台经营概览：总用户、新增用户、总/活跃 Agent、昨日决策、邀请码使用率、最近用户。"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return r.get(ctx, "/api/v1/reporting/overview", nil)
	})

	s.AddTool(mcp.NewTool("list_users",
		mcp.WithDescription("用户列表（搜索 + 分页）。"),
		mcp.WithString("q", mcp.Description("搜索关键词")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页条数，默认 20")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		setStr(q, "q", req.GetString("q", ""))
		setInt(q, "page", req.GetInt("page", 1))
		setInt(q, "page_size", req.GetInt("page_size", 20))
		return r.get(ctx, "/api/v1/reporting/users", q)
	})

	s.AddTool(mcp.NewTool("list_invite_codes",
		mcp.WithDescription("邀请码列表（状态筛选 + 分页）。"),
		mcp.WithString("status", mcp.Description("状态筛选：used/unused/expired")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页条数，默认 20")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		setStr(q, "status", req.GetString("status", ""))
		setInt(q, "page", req.GetInt("page", 1))
		setInt(q, "page_size", req.GetInt("page_size", 20))
		return r.get(ctx, "/api/v1/reporting/invite-codes", q)
	})

	s.AddTool(mcp.NewTool("credit_summary",
		mcp.WithDescription("Credit 按类型汇总（recharge/deduction/bonus/grant/expiry），可选时间窗。"),
		mcp.WithString("start", mcp.Description("起始时间 RFC3339")),
		mcp.WithString("end", mcp.Description("结束时间 RFC3339")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		setStr(q, "start", req.GetString("start", ""))
		setStr(q, "end", req.GetString("end", ""))
		return r.get(ctx, "/api/v1/reporting/credits/summary", q)
	})

	s.AddTool(mcp.NewTool("pm_agent_stats",
		mcp.WithDescription("交易员绩效统计：净盈亏/收益率/胜率分布与排行榜。"),
		mcp.WithString("period_unit", mcp.Description("周期单位：day/week/month（默认 week）")),
		mcp.WithString("account_type", mcp.Description("账户类型：all/mock/real（默认 all）")),
		mcp.WithString("fork", mcp.Description("fork 筛选：all/original/fork（默认 all）")),
		mcp.WithString("currency", mcp.Description("计价货币，默认 USDT")),
		mcp.WithBoolean("include_groups", mcp.Description("是否返回 per-metric 分布明细（raw_samples/bins/kde，体积大，默认 false）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		setStr(q, "period_unit", req.GetString("period_unit", "week"))
		setStr(q, "account_type", req.GetString("account_type", "all"))
		setStr(q, "fork", req.GetString("fork", "all"))
		setStr(q, "currency", req.GetString("currency", "USDT"))
		if req.GetBool("include_groups", false) {
			q.Set("include_groups", "true")
		}
		return r.get(ctx, "/api/v1/reporting/pm-statistics", q)
	})
}
