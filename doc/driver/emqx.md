# emqxi — EMQX REST client

Package: `github.com/aarioai/aago/driver/emqxi`

A minimal EMQX v5 REST client with basic authentication.

```go
api := emqxi.NewEmqxAPI("http://127.0.0.1:18083").
    WithBasicAuth("admin", "public").
    WithHttpClient(customClient) // optional; defaults to a 10s timeout

var clients struct {
    Data []struct {
        ClientId string `json:"clientid"`
    } `json:"data"`
}
e, err := api.Request(&clients, http.MethodGet, "/api/v5/clients")
if e != nil {
    // business error reported by EMQX (code + reason)
}
if err != nil {
    // transport / decoding / unexpected status error
}
```

## Behaviour

- The JSON body is decoded into `target` when it is non-nil. Passing the value (not a pointer to
  it) is correct now; earlier versions decoded into `&target` and always failed.
- A non-2xx or non-3xx response that is not an EMQX error document returns a descriptive error
  including the status code and body.
- Requests use a client with `DefaultTimeout` (10s) unless one is supplied.
- EMQX error documents (`{"code": "...", "reason": "..."}`) become `*ae.Error` via
  `NewError(EmqxApiError)`.

> 中文提示：`Request` 的第一个返回值是业务错误（EMQX 返回的 code/reason），
> 第二个是网络或解析错误；默认超时 10 秒。
