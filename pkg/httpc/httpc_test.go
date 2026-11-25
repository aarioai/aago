package httpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aarioai/aago/pkg/httpc"
)

func TestGetWithSingleHeaderMap(t *testing.T) {
	var gotHeader, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Test")
		gotQuery = r.URL.Query().Get("q")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	code, _, err := httpc.Get(context.Background(), srv.URL, map[string]string{"q": "go"}, map[string]string{"X-Test": "1"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if gotHeader != "1" {
		t.Fatalf("header = %q", gotHeader)
	}
	if gotQuery != "go" {
		t.Fatalf("query = %q", gotQuery)
	}
}

func TestDeleteUsesDeleteMethod(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if _, _, err := httpc.Delete(context.Background(), srv.URL, nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if method != http.MethodDelete {
		t.Fatalf("method = %q", method)
	}
}

func TestGetJsonDecodesTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"aario"}`))
	}))
	defer srv.Close()

	var target struct {
		Name string `json:"name"`
	}
	code, err := httpc.GetJson(context.Background(), &target, srv.URL, nil)
	if err != nil {
		t.Fatalf("GetJson: %v", err)
	}
	if code != http.StatusOK || target.Name != "aario" {
		t.Fatalf("code = %d, target = %+v", code, target)
	}
}

func TestRequestHonorsContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := httpc.Get(ctx, srv.URL, nil); err == nil {
		t.Fatal("expected cancelled context error")
	}
}
