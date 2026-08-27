package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/client"
)

func TestGetReviewPollsUntilReady(t *testing.T) {
	orig := reviewPollInterval
	reviewPollInterval = time.Millisecond
	defer func() { reviewPollInterval = orig }()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls < 3 {
			_, _ = w.Write([]byte(`{"code":0,"message":"calculating","data":{"status":"calculating","items":[]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"status":"ready","items":[{"id":"a"}]}}`))
	}))
	defer srv.Close()

	r := &registry{client: client.NewClient(srv.URL, "test-key")}
	result, err := r.getReview(context.Background(), "/api/review/decisions", nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (2 calculating + 1 ready)", calls)
	}
	raw, _ := json.Marshal(result.Content)
	if !strings.Contains(string(raw), "ready") || !strings.Contains(string(raw), "id") {
		t.Fatalf("content = %s, want ready data", string(raw))
	}
}
