package httpc

import (
	"context"
	"io"
	"net/http"
)

func Request(ctx context.Context, method, url string, params map[string]string, data io.Reader, headers ...map[string]string) (int, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, url, data)
	if err != nil {
		return 0, nil, err
	}
	AddHeaders(req, headers)
	SetQueries(req, params)

	return requestBody(req)
}

func Delete(ctx context.Context, url string, params map[string]string, headers ...map[string]string) (int, []byte, error) {
	return Request(ctx, http.MethodDelete, url, params, nil, headers...)
}

func Get(ctx context.Context, url string, params map[string]string, headers ...map[string]string) (int, []byte, error) {
	return Request(ctx, http.MethodGet, url, params, nil, headers...)
}

func Patch(ctx context.Context, url string, data io.Reader, headers ...map[string]string) (int, []byte, error) {
	return Request(ctx, http.MethodPatch, url, nil, data, headers...)
}

func Post(ctx context.Context, url string, data io.Reader, headers ...map[string]string) (int, []byte, error) {
	return Request(ctx, http.MethodPost, url, nil, data, headers...)
}

func Put(ctx context.Context, url string, data io.Reader, headers ...map[string]string) (int, []byte, error) {
	return Request(ctx, http.MethodPut, url, nil, data, headers...)
}
