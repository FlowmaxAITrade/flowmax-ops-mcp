package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetReturnsData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Ops-Key"); got != "test-key" {
			t.Fatalf("X-Ops-Key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"total_users":42}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	data, err := c.Get(context.Background(), "/api/v1/reporting/overview", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"total_users":42}` {
		t.Fatalf("data = %s", string(data))
	}
}

func TestGetReturnsErrorOnNonZeroCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":-1,"message":"invalid_time_window"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	_, err := c.Get(context.Background(), "/api/v1/reporting/credits/summary", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
}
