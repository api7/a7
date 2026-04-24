package create

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/iostreams"
)

func TestValidateContext_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/gateway_groups", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total":1,"list":[{"id":"default","name":"default"}]}`))
	}))
	defer srv.Close()

	err := validateContext(config.Context{
		Server: srv.URL,
		Token:  "valid-token",
	})
	require.NoError(t, err)
}

func TestValidateContext_InvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error_msg":"invalid token"}`))
	}))
	defer srv.Close()

	err := validateContext(config.Context{
		Server: srv.URL,
		Token:  "bad-token",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed: invalid token")
}

func TestValidateContext_PermissionDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error_msg":"forbidden"}`))
	}))
	defer srv.Close()

	err := validateContext(config.Context{
		Server: srv.URL,
		Token:  "restricted-token",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestValidateContext_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_msg":"internal error"}`))
	}))
	defer srv.Close()

	err := validateContext(config.Context{
		Server: srv.URL,
		Token:  "some-token",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

func TestValidateContext_UnreachableServer(t *testing.T) {
	err := validateContext(config.Context{
		Server: "https://127.0.0.1:1",
		Token:  "some-token",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot connect to server")
}

func TestValidateContext_GatewayGroupNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/gateway_groups":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total":0,"list":[]}`))
		case "/api/gateway_groups/nonexistent":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_msg":"not found"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	err := validateContext(config.Context{
		Server:       srv.URL,
		Token:        "valid-token",
		GatewayGroup: "nonexistent",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `gateway group "nonexistent" not found`)
}

func TestValidateContext_GatewayGroupExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/gateway_groups":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total":1,"list":[{"id":"default","name":"default"}]}`))
		case "/api/gateway_groups/default":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"default","name":"default"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	err := validateContext(config.Context{
		Server:       srv.URL,
		Token:        "valid-token",
		GatewayGroup: "default",
	})
	require.NoError(t, err)
}

func TestValidateContext_VerifiesAPIKeyHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "expected-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error_msg":"invalid token"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total":0,"list":[]}`))
	}))
	defer srv.Close()

	err := validateContext(config.Context{
		Server: srv.URL,
		Token:  "expected-token",
	})
	require.NoError(t, err)

	err = validateContext(config.Context{
		Server: srv.URL,
		Token:  "wrong-token",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestCreateRun_SkipValidation(t *testing.T) {
	cfgPath := fmt.Sprintf("%s/config.yaml", t.TempDir())
	cfg := config.NewFileConfigWithPath(cfgPath)

	opts := &Options{
		Config: func() (config.Config, error) {
			return cfg, nil
		},
		Name:           "test-ctx",
		Server:         "https://127.0.0.1:1",
		Token:          "fake-token",
		SkipValidation: true,
	}

	ios, _, _, _ := iostreams.Test()
	f := &cmd.Factory{
		IOStreams: ios,
		Config: func() (config.Config, error) {
			return cfg, nil
		},
	}

	err := createRun(opts, f)
	require.NoError(t, err)

	saved, err := cfg.GetContext("test-ctx")
	require.NoError(t, err)
	assert.Equal(t, "https://127.0.0.1:1", saved.Server)
	assert.Equal(t, "fake-token", saved.Token)
}

func TestCreateRun_ValidationFails(t *testing.T) {
	cfgPath := fmt.Sprintf("%s/config.yaml", t.TempDir())
	cfg := config.NewFileConfigWithPath(cfgPath)

	opts := &Options{
		Config: func() (config.Config, error) {
			return cfg, nil
		},
		Name:           "test-ctx",
		Server:         "https://127.0.0.1:1",
		Token:          "fake-token",
		SkipValidation: false,
	}

	ios, _, _, _ := iostreams.Test()
	f := &cmd.Factory{
		IOStreams: ios,
		Config: func() (config.Config, error) {
			return cfg, nil
		},
	}

	err := createRun(opts, f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
	assert.Contains(t, err.Error(), "cannot connect to server")

	_, getErr := cfg.GetContext("test-ctx")
	assert.Error(t, getErr)
}
