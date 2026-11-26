package driver_test

import (
	"strings"
	"testing"
	"time"

	"github.com/aarioai/aago/driver"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongodbClientOptionsKeepsCredentialsOutOfURI(t *testing.T) {
	o := &driver.MongodbOptions{
		Protocol: "mongodb",
		Hosts:    "localhost:27017",
		DB:       "test",
		Credential: &driver.MongodbCredential{
			Username:   "alice",
			Password:   "s3cret",
			AuthSource: "admin",
		},
	}
	opts := o.ClientOptions()
	uri := opts.GetURI()
	if strings.Contains(uri, "s3cret") || strings.Contains(uri, "@") {
		t.Errorf("mongodb URI must not embed credentials, got %q", uri)
	}
	if !strings.HasPrefix(uri, "mongodb://localhost:27017") {
		t.Errorf("mongodb URI = %q", uri)
	}
	if opts.Auth == nil {
		t.Fatal("credentials were not applied")
	}
	if opts.Auth.Username != "alice" || opts.Auth.Password != "s3cret" || opts.Auth.AuthSource != "admin" {
		t.Errorf("auth = %#v", opts.Auth)
	}
}

func TestMongodbConnectionOptionsLoadTo(t *testing.T) {
	co := &driver.MongodbConnectionOptions{
		ConnectTimeout:         2 * time.Second,
		Timeout:                5 * time.Second,
		ServerSelectionTimeout: 3 * time.Second,
		TLS:                    true,
		WriteConcern:           "2",
		WriteConcernJournal:    true,
	}
	opts := options.Client()
	co.LoadTo(opts)

	if opts.ConnectTimeout == nil || *opts.ConnectTimeout != 2*time.Second {
		t.Errorf("ConnectTimeout = %v", opts.ConnectTimeout)
	}
	if opts.Timeout == nil || *opts.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want the configured operation timeout", opts.Timeout)
	}
	if opts.ServerSelectionTimeout == nil || *opts.ServerSelectionTimeout != 3*time.Second {
		t.Errorf("ServerSelectionTimeout = %v", opts.ServerSelectionTimeout)
	}
	if opts.TLSConfig == nil {
		t.Error("TLSConfig was not applied")
	}
	if opts.WriteConcern == nil || opts.WriteConcern.W != 2 {
		t.Errorf("WriteConcern = %#v, want numeric w=2", opts.WriteConcern)
	}
}
