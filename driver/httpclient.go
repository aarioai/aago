package driver

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/pkg/types"
)

const (
	HTTPCLIENT                     = "httpclient"
	DefaultHttpTimeout             = 30 * time.Second
	DefaultHttpDialTimeout         = 10 * time.Second
	DefaultHttpKeepAlive           = 15 * time.Second
	DefaultHttpIdleConnTimeout     = 90 * time.Second
	DefaultHttpTLSHandshakeTimeout = 10 * time.Second
	DefaultHttpMaxIdleConns        = 100
	DefaultHttpMaxIdleConnsPerHost = 32
)

var (
	ErrInvalidHttpCertConfig = errors.New("invalid certificate configuration: format should be: <certFile>,<keyFile>[;<certFile>,<keyFile>...]")
)

type HttpClientConfig struct {

	// http.Client

	// Timeout specifies a time limit for requests made by this
	// Client. The timeout includes connection time, any redirects,
	// and reading the response body. The timer remains
	// running after Get, Head, Post, or Do return and will
	// interrupt reading of the Response.Body.
	//
	// A Timeout of zero means no timeout.
	//
	// The Client cancels requests to the underlying Transport
	// as if the Request's Context ended.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// net.Dialer. The default is no timeout.
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dialTimeout"`

	// DualStack previously enabled RFC 6555 Fast Fallback
	// support, also known as "Happy Eyeballs", in which IPv4 is
	// tried soon if IPv6 appears to be misconfigured and hanging.
	DialDualStack bool `json:"dial_dual_stack" yaml:"dialDualStack"`

	// FallbackDelay specifies the length of time to wait before
	// spawning a RFC 6555 Fast Fallback connection.
	//
	// When dialing "tcp" and both IPv6 and IPv4 addresses are
	// available, the address family of the first resolved address
	// is treated as the primary. After FallbackDelay, a connection
	// attempt to the other address family is started if the primary
	// connection has not yet succeeded.
	//
	// If zero, a default delay of 300ms is used.
	// A negative value disables Fast Fallback support.
	DialFallbackDelay time.Duration `json:"dial_fallback_delay" yaml:"dialFallbackDelay"`

	// KeepAlive specifies the interval between keep-alive
	// probes for an active network connection.
	//
	// KeepAlive is ignored if KeepAliveConfig.Enable is true.
	//
	// If zero, keep-alive probes are sent with a default value
	// (currently 15 seconds), if supported by the protocol and operating
	// system. Network protocols or operating systems that do
	// not support keep-alive ignore this field.
	// If negative, keep-alive probes are disabled.
	DialKeepAlive time.Duration `json:"dial_keep_alive" yaml:"dialKeepAlive"`

	// If Enable is true, keep-alive probes are enabled.
	DialKeepAliveEnable bool `json:"dial_keep_alive_enable" yaml:"dialKeepAliveEnable"`

	// Idle is the time that the connection must be idle before
	// the first keep-alive probe is sent.
	// If zero, a default value of 15 seconds is used.
	DialKeepAliveIdle time.Duration `json:"dial_keep_alive_idle" yaml:"dialKeepAliveIdle"`

	// Interval is the time between keep-alive probes.
	// If zero, a default value of 15 seconds is used.
	DialKeepAliveInterval time.Duration `json:"dial_keep_alive_interval" yaml:"dialKeepAliveInterval"`

	// Count is the maximum number of keep-alive probes that
	// can go unanswered before dropping a connection.
	// If zero, a default value of 9 is used.
	DialKeepAliveCount int `json:"dial_keep_alive_count" yaml:"dialKeepAliveCount"`

	// TLSHandshakeTimeout specifies the maximum amount of time to
	// wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout" yaml:"tlsHandshakeTimeout"`

	// http.Transport
	// DisableKeepAlives, if true, disables HTTP keep-alives and will only use the connection to the server for a single
	// HTTP request.
	DisableKeepAlives bool `json:"disable_keep_alives" yaml:"disableKeepAlives"`

	// DisableCompression, if true, prevents the Transport from
	// requesting compression with an "Accept-Encoding: gzip"
	// request header when the Request contains no existing
	// Accept-Encoding value. If the Transport requests gzip on
	// its own and gets a gzipped response, it's transparently
	// decoded in the Response.Body. However, if the user
	// explicitly requested gzip it is not automatically
	// uncompressed.
	DisableCompression bool `json:"disable_compression" yaml:"disableCompression"`

	// MaxIdleConns controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.
	MaxIdleConns int `json:"max_idle_conns" yaml:"maxIdleConns"`

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle (keep-alive) connections to keep per-host.
	// If zero, http.DefaultMaxIdleConnsPerHost is used.
	MaxIdleConnsPerHost int `json:"max_idle_conns_per_host" yaml:"maxIdleConnsPerHost"`

	// MaxConnsPerHost optionally limits the total number of connections per host, including connections in the dialing,
	// active, and idle states. On limit violation, dials will block. Zero means no limit.
	MaxConnsPerHost int `json:"max_conns_per_host" yaml:"maxConnsPerHost"`

	// IdleConnTimeout is the maximum amount of time an idle (keep-alive) connection will remain idle before closing
	// itself. Zero means no limit.
	IdleConnTimeout time.Duration `json:"idle_conn_timeout" yaml:"idleConnTimeout"`

	// ResponseHeaderTimeout, if non-zero, specifies the amount of time to wait for a server's response headers after fully
	// writing the request (including its body, if any). This time does not include the time to read the response body.
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout" yaml:"responseHeaderTimeout"`

	// ExpectContinueTimeout, if non-zero, specifies the amount of time to wait for a server's first response headers
	// after fully writing the request headers if the request has an "Expect: 100-continue" header.
	// Zero means no timeout and causes the body to be sent immediately, without waiting for the server to approve.
	// This time does not include the time to send the request header.
	ExpectContinueTimeout time.Duration `json:"expect_continue_timeout" yaml:"expectContinueTimeout"`

	// MaxResponseHeaderBytes specifies a limit on how many response bytes are allowed in the server's response header.
	// Zero means to use a default limit.
	MaxResponseHeaderBytes int64 `json:"max_response_header_bytes" yaml:"maxResponseHeaderBytes"`

	// WriteBufferSize specifies the size of the write buffer used
	// when writing to the transport.
	// If zero, a default (currently 4KB) is used.
	WriteBufferSize int `json:"write_buffer_size" yaml:"writeBufferSize"`

	// ReadBufferSize specifies the size of the read buffer used
	// when reading from the transport.
	// If zero, a default (currently 4KB) is used.
	ReadBufferSize int `json:"read_buffer_size" yaml:"readBufferSize"`

	// ForceAttemptHTTP2 controls whether HTTP/2 is enabled when a non-zero
	// Dial, DialTLS, or DialContext func or TLSClientConfig is provided.
	// By default, use of any those fields conservatively disables HTTP/2.
	// To use a custom dialer or TLS config and still attempt HTTP/2 upgrades, set this to true.
	ForceAttemptHTTP2 bool `json:"force_attempt_http2" yaml:"forceAttemptHTTP2"`

	// tls.Config

	// NextProtos is a list of supported application level protocols, in
	// order of preference. If both peers support ALPN, the selected
	// protocol will be one from this list, and the connection will fail
	// if there is no mutually supported protocol. If NextProtos is empty
	// or the peer doesn't support ALPN, the connection will succeed and
	// ConnectionState.NegotiatedProtocol will be empty.
	TLSNextProtos []string `json:"tls_next_protos" yaml:"tlsNextProtos"`

	TLSServerName                     string                   `json:"tls_server_name" yaml:"tlsServerName"`
	TLSClientAuth                     tls.ClientAuthType       `json:"tls_client_auth" yaml:"tlsClientAuth"`
	TLSInsecureSkipVerify             bool                     `json:"tls_insecure_skip_verify" yaml:"tlsInsecureSkipVerify"`
	TLSCipherSuites                   []uint16                 `json:"tls_cipher_suites" yaml:"tlsCipherSuites"`
	TLSSessionTicketsDisabled         bool                     `json:"tls_session_tickets_disabled" yaml:"tlsSessionTicketsDisabled"`
	TLSMinVersion                     uint16                   `json:"tls_min_version" yaml:"tlsMinVersion"`
	TLSMaxVersion                     uint16                   `json:"tls_max_version" yaml:"tlsMaxVersion"`
	TLSCurvePreferences               []tls.CurveID            `json:"tls_curve_preferences" yaml:"tlsCurvePreferences"`
	TLSDynamicRecordSizingDisabled    bool                     `json:"tls_dynamic_record_sizing_disabled" yaml:"tlsDynamicRecordSizingDisabled"`
	TLSRenegotiation                  tls.RenegotiationSupport `json:"tls_renegotiation" yaml:"tlsRenegotiation"`
	TLSEncryptedClientHelloConfigList []byte                   `json:"tls_encrypted_client_hello_config_list" yaml:"tlsEncryptedClientHelloConfigList"`

	// tls.Certificate  tls.LoadX509KeyPair(certFile, keyFile)
	TLSCertificateFilePairs [][2]string `json:"tls_certificate_file_pairs" yaml:"tlsCertificateFilePairs"`
}

func (c HttpClientConfig) Dialer() *net.Dialer {
	return &net.Dialer{
		Timeout: c.DialTimeout,
		//Deadline: c.DialDeadline,
		//LocalAddr:     nil,
		//DualStack:     false,
		FallbackDelay: c.DialFallbackDelay,
		KeepAlive:     c.DialKeepAlive,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   c.DialKeepAliveEnable,
			Idle:     c.DialKeepAliveIdle,
			Interval: c.DialKeepAliveInterval,
			Count:    c.DialKeepAliveCount,
		},
		//Resolver:       nil,
		//Cancel:         nil,
		//Control:        nil,
		//ControlContext: nil,
	}
}

func (c HttpClientConfig) TLSCertificates() ([]tls.Certificate, error) {
	if len(c.TLSCertificateFilePairs) == 0 {
		return nil, nil
	}
	certs := make([]tls.Certificate, len(c.TLSCertificateFilePairs))
	for i, pair := range c.TLSCertificateFilePairs {
		cert, err := tls.LoadX509KeyPair(pair[0], pair[1])
		if err != nil {
			return nil, err
		}
		certs[i] = cert
	}
	return certs, nil
}

func (c HttpClientConfig) TLSClientConfig() (*tls.Config, error) {
	certs, err := c.TLSCertificates()
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:                   certs,
		NextProtos:                     c.TLSNextProtos,
		ServerName:                     c.TLSServerName,
		ClientAuth:                     c.TLSClientAuth,
		InsecureSkipVerify:             c.TLSInsecureSkipVerify,
		CipherSuites:                   c.TLSCipherSuites,
		SessionTicketsDisabled:         c.TLSSessionTicketsDisabled,
		MinVersion:                     c.TLSMinVersion,
		MaxVersion:                     c.TLSMaxVersion,
		CurvePreferences:               c.TLSCurvePreferences,
		DynamicRecordSizingDisabled:    c.TLSDynamicRecordSizingDisabled,
		Renegotiation:                  c.TLSRenegotiation,
		EncryptedClientHelloConfigList: c.TLSEncryptedClientHelloConfigList,
	}, nil
}

func (c HttpClientConfig) Transport() (*http.Transport, error) {
	tlsConfig, err := c.TLSClientConfig()
	if err != nil {
		return nil, err
	}
	return &http.Transport{
		DialContext:            c.Dialer().DialContext,
		TLSHandshakeTimeout:    c.TLSHandshakeTimeout,
		TLSClientConfig:        tlsConfig,
		DisableKeepAlives:      c.DisableKeepAlives,
		DisableCompression:     c.DisableCompression,
		MaxIdleConns:           c.MaxIdleConns,
		MaxIdleConnsPerHost:    c.MaxIdleConnsPerHost,
		MaxConnsPerHost:        c.MaxConnsPerHost,
		IdleConnTimeout:        c.IdleConnTimeout,
		ResponseHeaderTimeout:  c.ResponseHeaderTimeout,
		ExpectContinueTimeout:  c.ExpectContinueTimeout,
		MaxResponseHeaderBytes: c.MaxResponseHeaderBytes,
		ForceAttemptHTTP2:      c.ForceAttemptHTTP2,
		WriteBufferSize:        c.WriteBufferSize,
		ReadBufferSize:         c.ReadBufferSize,
	}, nil
}

func NewHttpClient(app *aa.App, section string) (*http.Client, *ae.Error) {
	c, err := ParseHttpClientConfig(app, section)
	if err != nil {
		return nil, newConfigError(HTTPCLIENT, section, err)
	}

	transport, err := c.Transport()
	if err != nil {
		return nil, newConfigError(HTTPCLIENT, section, err)
	}

	return &http.Client{
		Transport:     transport,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       c.Timeout,
	}, nil
}

func ParseHttpClientConfig(app *aa.App, section string) (HttpClientConfig, error) {
	timeout, err1 := getSectionCfgDurationWithDefault(app, HTTPCLIENT, section, DefaultHttpTimeout, "timeout", "Timeout")
	dialTimeout, err2 := getSectionCfgDurationWithDefault(app, HTTPCLIENT, section, DefaultHttpDialTimeout, "dialTimeout", "DialTimeout", "dial_timeout")
	dialDualStack, err3 := getSectionCfgBoolWithFallback(app, HTTPCLIENT, section, "dialDualStack", "DialDualStack", "dial_dual_stack")
	dialFallbackDelay, err4 := getSectionCfgDurationWithFallback(app, HTTPCLIENT, section, "dialFallbackDelay", "DialFallbackDelay", "dial_fallback_delay")
	dialKeepAlive, err5 := getSectionCfgDurationWithDefault(app, HTTPCLIENT, section, DefaultHttpKeepAlive, "dialKeepAlive", "DialKeepAlive", "dial_keep_alive")
	dialKeepAliveEnable, err6 := getSectionCfgBoolWithFallback(app, HTTPCLIENT, section, "dialKeepAliveEnable", "DialKeepAliveEnable", "dial_keep_alive_enable")
	dialKeepAliveIdle, err7 := getSectionCfgDurationWithFallback(app, HTTPCLIENT, section, "dialKeepAliveIdle", "DialKeepAliveIdle", "dial_keep_alive_idle")
	dialKeepAliveInterval, err8 := getSectionCfgDurationWithFallback(app, HTTPCLIENT, section, "dialKeepAliveInterval", "DialKeepAliveInterval", "dial_keep_alive_interval")
	dialKeepAliveCount, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "dial_keep_alive_count")

	disableKeepAlives, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "disable_keep_alives")
	disableCompression, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "disable_compression")
	maxIdleConns, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "max_idle_conns")
	maxIdleConnsPerHost, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "max_idle_conns_per_host")
	maxConnsPerHost, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "max_conns_per_host")
	idleConnTimeout, err9 := getSectionCfgDurationWithDefault(app, HTTPCLIENT, section, DefaultHttpIdleConnTimeout, "idleConnTimeout", "IdleConnTimeout", "idle_conn_timeout")
	responseHeaderTimeout, err10 := getSectionCfgDurationWithFallback(app, HTTPCLIENT, section, "responseHeaderTimeout", "ResponseHeaderTimeout", "response_header_timeout")
	expectContinueTimeout, err11 := getSectionCfgDurationWithFallback(app, HTTPCLIENT, section, "expectContinueTimeout", "ExpectContinueTimeout", "expect_continue_timeout")
	maxResponseHeaderBytes, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "max_response_header_bytes")
	writeBufferSize, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "write_buffer_size")
	readBufferSize, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "read_buffer_size")
	forceAttemptHTTP2, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "force_attempt_http2")

	tlsHandshakeTimeout, err12 := getSectionCfgDurationWithDefault(app, HTTPCLIENT, section, DefaultHttpTLSHandshakeTimeout, "tlsHandshakeTimeout", "TlsHandshakeTimeout", "tls_handshake_timeout")
	tlsNextProtos, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_next_protos")
	tlsServerName, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_server_name")
	tlsClientAuth, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_client_auth")
	tlsInsecureSkipVerify, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_insecure_skip_verify")
	tlsCipherSuites, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_cipher_suites")
	tlsSessionTicketsDisabled, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_session_tickets_disabled")
	tlsMinVersion, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_min_version")
	tlsMaxVersion, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_max_version")
	tlsCurvePreferences, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_curve_preferences")
	tlsDynamicRecordSizingDisabled, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_dynamic_record_sizing_disabled")
	tlsRenegotiation, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_renegotiation")
	tlsEncryptedClientHelloConfigList, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_encrypted_client_hello_config_list")
	tlsCertificateFilePairs, _ := tryGetSectionCfg(app, HTTPCLIENT, section, "tls_certificate_file_pairs")

	if err := ae.FirstError(err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12); err != nil {
		return HttpClientConfig{}, err
	}

	dialKeepAliveCountN, _ := types.ParseInt(dialKeepAliveCount)
	maxIdleConnsN, _ := types.ParseInt(maxIdleConns)
	maxIdleConnsPerHostN, _ := types.ParseInt(maxIdleConnsPerHost)
	if maxIdleConns == "" {
		maxIdleConnsN = DefaultHttpMaxIdleConns
	}
	if maxIdleConnsPerHost == "" {
		maxIdleConnsPerHostN = DefaultHttpMaxIdleConnsPerHost
	}
	maxConnsPerHostN, _ := types.ParseInt(maxConnsPerHost)
	maxResponseHeaderBytesN, _ := types.ParseInt64(maxResponseHeaderBytes)
	writeBufferSizeN, _ := types.ParseInt(writeBufferSize)
	readBufferSizeN, _ := types.ParseInt(readBufferSize)
	tlsClientAuthN, _ := types.ParseInt(tlsClientAuth)

	tlsCurvePreferencesN := parseUint16s(tlsCurvePreferences, ",")
	var tlsCurveIDs []tls.CurveID
	if len(tlsCurvePreferences) > 0 {
		tlsCurveIDs = make([]tls.CurveID, len(tlsCurvePreferencesN))
		for i, tlsCurveID := range tlsCurvePreferencesN {
			tlsCurveIDs[i] = tls.CurveID(tlsCurveID)
		}
	}

	tlsRenegotiationN, _ := types.ParseInt(tlsRenegotiation)
	var tlsEncryptedClientHelloConfigListN []byte
	tlsEncryptedClientHelloConfigList = strings.Trim(tlsEncryptedClientHelloConfigList, " ")
	if tlsEncryptedClientHelloConfigList != "" {
		tlsEncryptedClientHelloConfigListN = []byte(tlsEncryptedClientHelloConfigList)
	}
	//   /a/df/a.crt,/a/df/a.key;/a/df/b.crt,/a/df/b.key
	var tlsCertFilePairs [][2]string
	tlsCertFiles := parseStrings(tlsCertificateFilePairs, ";")
	if len(tlsCertFiles) > 0 {
		tlsCertFilePairs = make([][2]string, 0, len(tlsCertFiles))
		for _, tcf := range tlsCertFiles {
			pair := parseStrings(tcf, ",")
			if len(pair) == 0 {
				continue
			}
			if len(pair) != 2 {
				return HttpClientConfig{}, ErrInvalidHttpCertConfig
			}
			tlsCertFilePairs = append(tlsCertFilePairs, [2]string{pair[0], pair[1]})
		}
	}

	return HttpClientConfig{
		Timeout:                           timeout,
		DialTimeout:                       dialTimeout,
		DialDualStack:                     dialDualStack,
		DialFallbackDelay:                 dialFallbackDelay,
		DialKeepAlive:                     dialKeepAlive,
		DialKeepAliveEnable:               dialKeepAliveEnable,
		DialKeepAliveIdle:                 dialKeepAliveIdle,
		DialKeepAliveInterval:             dialKeepAliveInterval,
		DialKeepAliveCount:                dialKeepAliveCountN,
		DisableKeepAlives:                 types.ToBool(disableKeepAlives),
		DisableCompression:                types.ToBool(disableCompression),
		MaxIdleConns:                      maxIdleConnsN,
		MaxIdleConnsPerHost:               maxIdleConnsPerHostN,
		MaxConnsPerHost:                   maxConnsPerHostN,
		IdleConnTimeout:                   idleConnTimeout,
		ResponseHeaderTimeout:             responseHeaderTimeout,
		ExpectContinueTimeout:             expectContinueTimeout,
		MaxResponseHeaderBytes:            maxResponseHeaderBytesN,
		WriteBufferSize:                   writeBufferSizeN,
		ReadBufferSize:                    readBufferSizeN,
		ForceAttemptHTTP2:                 types.ToBool(forceAttemptHTTP2),
		TLSHandshakeTimeout:               tlsHandshakeTimeout,
		TLSNextProtos:                     parseStrings(tlsNextProtos, ","),
		TLSServerName:                     tlsServerName,
		TLSClientAuth:                     tls.ClientAuthType(tlsClientAuthN),
		TLSInsecureSkipVerify:             types.ToBool(tlsInsecureSkipVerify),
		TLSCipherSuites:                   parseUint16s(tlsCipherSuites, ","),
		TLSSessionTicketsDisabled:         types.ToBool(tlsSessionTicketsDisabled),
		TLSMinVersion:                     types.ToUint16(tlsMinVersion),
		TLSMaxVersion:                     types.ToUint16(tlsMaxVersion),
		TLSCurvePreferences:               tlsCurveIDs,
		TLSDynamicRecordSizingDisabled:    types.ToBool(tlsDynamicRecordSizingDisabled),
		TLSRenegotiation:                  tls.RenegotiationSupport(tlsRenegotiationN),
		TLSEncryptedClientHelloConfigList: tlsEncryptedClientHelloConfigListN,
		TLSCertificateFilePairs:           tlsCertFilePairs,
	}, nil
}

func NewHttpError(err error) *ae.Error {
	return ae.NewErr(err)
}
