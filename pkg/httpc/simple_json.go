package httpc

import (
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
)

var (
	JsonUnmarshal = json.Unmarshal
)

func requestJsonBody(req *http.Request, target interface{}) (int, error) {
	code, data, err := requestBody(req)
	if err != nil {
		return code, err
	}
	if target == nil {
		return code, nil
	}
	return code, JsonUnmarshal(data, target)
}

func addJsonHeaders(req *http.Request, headers []map[string]string) {
	SetHeaders(req, headers)
	SetNxHeader(req, "Content-Type", "application/json")
	SetNxHeader(req, "Accept", "application/json")
}

// RequestJson
// Example:
// - RequestJson("HEAD", url, nil)
// - RequestJson("TRACE", url, nil)
// - RequestJson("CONNECT", url, nil)
func RequestJson(ctx context.Context, method, url string, params map[string]string, headers ...map[string]string) (*http.Response, error) {
	req, err := newJSONRequest(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	addJsonHeaders(req, headers)
	SetQueries(req, params)
	return request(req)
}

func RequestJsonBody(ctx context.Context, target interface{}, method, url string, params map[string]string, data io.Reader, headers ...map[string]string) (int, error) {
	req, err := newJSONRequest(ctx, method, url, data)
	if err != nil {
		return 0, err
	}
	addJsonHeaders(req, headers)
	SetQueries(req, params)
	return requestJsonBody(req, target)
}

func newJSONRequest(ctx context.Context, method, url string, data io.Reader) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return http.NewRequestWithContext(ctx, method, url, data)
}

func DeleteJson(ctx context.Context, target interface{}, url string, params map[string]string, headers ...map[string]string) (int, error) {
	return RequestJsonBody(ctx, target, http.MethodDelete, url, params, nil, headers...)
}

func GetJson(ctx context.Context, target interface{}, url string, params map[string]string, headers ...map[string]string) (int, error) {
	return RequestJsonBody(ctx, target, http.MethodGet, url, params, nil, headers...)
}

func PatchJson(ctx context.Context, target interface{}, url string, data io.Reader, headers ...map[string]string) (int, error) {
	return RequestJsonBody(ctx, target, http.MethodPatch, url, nil, data, headers...)
}

func PostJson(ctx context.Context, target interface{}, url string, data io.Reader, headers ...map[string]string) (int, error) {
	return RequestJsonBody(ctx, target, http.MethodPost, url, nil, data, headers...)
}

func PutJson(ctx context.Context, target interface{}, url string, data io.Reader, headers ...map[string]string) (int, error) {
	return RequestJsonBody(ctx, target, http.MethodPut, url, nil, data, headers...)
}
