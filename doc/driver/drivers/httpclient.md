# HTTP Client

Package: `github.com/aarioai/aago/driver` (file `httpclient.go`)

## Configuration

```yaml
httpc:
  DEFAULT:
    timeout: 30s
    dial_timeout: 10s
    dial_keep_alive: 15s
    max_idle_conns: 100
    max_idle_conns_per_host: 32
    idle_conn_timeout: 90s
    tls_handshake_timeout: 10s
    force_attempt_http2: true
  api:
    timeout: 5s
    tls_insecure_skip_verify: false
    tls_certificate_file_pairs: /certs/a.crt,/certs/a.key
```

Defaults applied when a key is missing:

| Key                       | Default |
|---------------------------|---------|
| `timeout`                 | `30s`   |
| `dial_timeout`            | `10s`   |
| `dial_keep_alive`         | `15s`   |
| `idle_conn_timeout`       | `90s`   |
| `tls_handshake_timeout`   | `10s`   |
| `max_idle_conns`          | `100`   |
| `max_idle_conns_per_host` | `32`    |

Other supported keys map 1:1 to `net.Dialer.KeepAliveConfig`, `http.Transport` and
`tls.Config` fields (`dial_keep_alive_enable`, `response_header_timeout`,
`expect_continue_timeout`, `write_buffer_size`, `read_buffer_size`, `tls_min_version`,
`tls_max_version`, `tls_cipher_suites`, `tls_curve_preferences`, `tls_renegotiation`, …).
Keys may use snake_case, camelCase, or PascalCase; section values fall back to `DEFAULT`.

## API

```go
client, e := driver.NewHttpClient(app, "api") // *http.Client
cfg, err := driver.ParseHttpClientConfig(app, "api")

cfg.Dialer()        // *net.Dialer
cfg.TLSCertificates()
cfg.TLSClientConfig() // *tls.Config
cfg.Transport()       // *http.Transport
```

The client is **not** pooled: HTTP clients are cheap and `http.Transport` already pools
connections. Keep one client per target and reuse it.

> 中文提示：`*http.Client` 不需要池化，但它们内部的 `Transport` 会复用连接，
> 因此应当每个目标复用一个 client，而不是每次请求都新建。

## Notes

- `dial_dual_stack` is parsed for backward compatibility but is a no-op since Go 1.16.
- `tls_certificate_file_pairs` accepts `;` separated pairs of `cert,key`; a malformed pair
  returns `driver.ErrInvalidHttpCertConfig`.
- `NewHttpError(err)` wraps errors as `*ae.Error`.

> 中文提示：`dial_dual_stack` 自 Go 1.16 起已无效果，仅为兼容旧配置保留。
