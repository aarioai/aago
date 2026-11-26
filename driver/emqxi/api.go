package emqxi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aarioai/aago/aa/ae"
)

const DefaultTimeout = 10 * time.Second

type EmqxAPI struct {
	Host     string
	username string
	password string
	client   *http.Client
}

func NewEmqxAPI(host string) *EmqxAPI {
	return &EmqxAPI{
		Host:   host,
		client: &http.Client{Timeout: DefaultTimeout},
	}
}

func (s *EmqxAPI) WithBasicAuth(username, password string) *EmqxAPI {
	s.username = username
	s.password = password
	return s
}

// WithHttpClient replaces the default HTTP client (which has a 10s timeout).
func (s *EmqxAPI) WithHttpClient(client *http.Client) *EmqxAPI {
	if client != nil {
		s.client = client
	}
	return s
}

// Request performs an EMQX REST request and decodes the JSON body into target (when target is not nil).
//
// The first return value is a business/API error, the second one is a transport or decoding error.
func (s *EmqxAPI) Request(target any, method, path string) (*ae.Error, error) {
	r, err := http.NewRequest(method, s.Host+path, nil)
	if err != nil {
		return nil, err
	}
	if s.username != "" {
		r.SetBasicAuth(s.username, s.password)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")

	client := s.client
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var emqxError EmqxApiError
	_ = json.Unmarshal(body, &emqxError)
	if e := NewError(emqxError); e != nil {
		return e, nil
	}

	if target != nil {
		if err = json.Unmarshal(body, target); err != nil && resp.StatusCode < http.StatusMultipleChoices {
			return nil, err
		}
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("emqx api %s %s: unexpected status %d: %s", method, path, resp.StatusCode, string(body))
	}
	return nil, nil
}
