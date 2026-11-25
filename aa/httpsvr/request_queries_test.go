package httpsvr

import (
	"net/http"
	"testing"
)

func TestQueriesMergeInjectedOverUser(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/users?page=1&q=go", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-From", "client")
	c := &Context{
		r:               req,
		injectedQueries: map[string]any{"page": "9", "uid": "3"},
		injectedHeaders: map[string]any{"X-From": "injected", "X-Debug": "1"},
		paramsInjected:  true,
	}

	got := c.Queries()
	if got["page"] != "9" || got["q"] != "go" || got["uid"] != "3" {
		t.Fatalf("Queries() = %#v", got)
	}
	if _, ok := got["X-From"]; ok {
		t.Fatalf("Queries() included header %#v", got)
	}

	headers := c.Headers()
	if headers["X-From"] != "injected" || headers["X-Debug"] != "1" {
		t.Fatalf("Headers() = %#v", headers)
	}
	if _, ok := headers["page"]; ok {
		t.Fatalf("Headers() included query %#v", headers)
	}
}

func TestQueriesWithoutInjection(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/users?page=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &Context{r: req, paramsInjected: true}
	got := c.Queries()
	if got["page"] != "2" || len(got) != 1 {
		t.Fatalf("Queries() = %#v", got)
	}
}

func TestQueryUint8RejectsOutOfRange(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/?n=300", nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &Context{r: req}
	if _, e := c.QueryUint8("n"); e == nil {
		t.Fatal("QueryUint8(300) should be out of range")
	}

	req, err = http.NewRequest(http.MethodGet, "http://example.test/?n=255", nil)
	if err != nil {
		t.Fatal(err)
	}
	c = &Context{r: req}
	if v, e := c.QueryUint8("n"); e != nil || v != 255 {
		t.Fatalf("QueryUint8(255) = %d, %v", v, e)
	}
}

func TestHeaderParamReadsHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Token", "abc")
	c := &Context{r: req}
	v, e := c.HeaderParam("X-Token", false)
	if e != nil {
		t.Fatalf("HeaderParam: %v", e)
	}
	if v.String() != "abc" {
		t.Fatalf("HeaderParam = %q", v.String())
	}
	if s := c.HeaderValue("X-Token"); s == nil || s.String() != "abc" {
		t.Fatalf("HeaderValue = %v", s)
	}
}

func TestURLQueriesCacheInvalidatedByResetRequest(t *testing.T) {
	first, err := http.NewRequest(http.MethodGet, "http://example.test/?page=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &Context{r: first}
	if got := c.URLParam("page"); got != "1" {
		t.Fatalf("URLParam = %q", got)
	}
	second, err := http.NewRequest(http.MethodGet, "http://example.test/?page=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	c.ResetRequest(second)
	if got := c.URLParam("page"); got != "2" {
		t.Fatalf("URLParam after ResetRequest = %q", got)
	}
}
