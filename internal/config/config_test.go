package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFileConfig(t *testing.T) (*FileConfig, string) {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	return NewFileConfigWithPath(path), path
}

func TestNewFileConfig_LazyLoading(t *testing.T) {
	cfg, path := newTestFileConfig(t)

	assert.False(t, cfg.loaded, "config should not be loaded before first access")

	assert.Equal(t, "", cfg.BaseURL())
	assert.Equal(t, "", cfg.Token())
	assert.Equal(t, "", cfg.GatewayGroup())
	assert.Equal(t, "", cfg.CurrentContext())
	assert.Empty(t, cfg.Contexts())

	assert.True(t, cfg.loaded, "config should load lazily on first access")

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "missing file should not be created by lazy load reads")
}

func TestAddContext(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	ctx := Context{Name: "dev", Server: "https://dev.example.com"}
	require.NoError(t, cfg.AddContext(ctx))

	contexts := cfg.Contexts()
	require.Len(t, contexts, 1)
	assert.Equal(t, ctx, contexts[0])
}

func TestAddContext_Duplicate(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{Name: "dev", Server: "https://dev-1.example.com"}))
	err := cfg.AddContext(Context{Name: "dev", Server: "https://dev-2.example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddContext_AutoSetCurrent(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{Name: "dev", Server: "https://dev.example.com"}))
	assert.Equal(t, "dev", cfg.CurrentContext())
}

func TestRemoveContext(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{Name: "dev", Server: "https://dev.example.com"}))
	require.NoError(t, cfg.AddContext(Context{Name: "prod", Server: "https://prod.example.com"}))

	require.NoError(t, cfg.RemoveContext("dev"))

	contexts := cfg.Contexts()
	require.Len(t, contexts, 1)
	assert.Equal(t, "prod", contexts[0].Name)
	_, err := cfg.GetContext("dev")
	require.Error(t, err)
}

func TestRemoveContext_NotFound(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	err := cfg.RemoveContext("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveContext_UpdatesCurrent(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{Name: "dev", Server: "https://dev.example.com"}))
	require.NoError(t, cfg.AddContext(Context{Name: "prod", Server: "https://prod.example.com"}))

	assert.Equal(t, "dev", cfg.CurrentContext())
	require.NoError(t, cfg.RemoveContext("dev"))
	assert.Equal(t, "prod", cfg.CurrentContext())
}

func TestSetCurrentContext(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{Name: "dev", Server: "https://dev.example.com"}))
	require.NoError(t, cfg.AddContext(Context{Name: "prod", Server: "https://prod.example.com"}))

	require.NoError(t, cfg.SetCurrentContext("prod"))
	assert.Equal(t, "prod", cfg.CurrentContext())
}

func TestSetCurrentContext_NotFound(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{Name: "dev", Server: "https://dev.example.com"}))

	err := cfg.SetCurrentContext("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetContext(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	want := Context{
		Name:          "dev",
		Server:        "https://dev.example.com",
		Token:         "token-dev",
		GatewayGroup:  "group-dev",
		TLSSkipVerify: true,
		CACert:        "/tmp/dev-ca.crt",
	}
	require.NoError(t, cfg.AddContext(want))

	got, err := cfg.GetContext("dev")
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestGetContext_NotFound(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	_, err := cfg.GetContext("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOverridePrecedence(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{
		Name:         "dev",
		Server:       "https://context.example.com",
		Token:        "context-token",
		GatewayGroup: "context-group",
	}))

	assert.Equal(t, "https://context.example.com", cfg.BaseURL())
	assert.Equal(t, "context-token", cfg.Token())
	assert.Equal(t, "context-group", cfg.GatewayGroup())

	cfg.SetServerOverride("https://override.example.com")
	cfg.SetTokenOverride("override-token")
	cfg.SetGatewayGroupOverride("override-group")

	assert.Equal(t, "https://override.example.com", cfg.BaseURL())
	assert.Equal(t, "override-token", cfg.Token())
	assert.Equal(t, "override-group", cfg.GatewayGroup())
}

func TestSaveAndReload(t *testing.T) {
	cfg, path := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{
		Name:         "dev",
		Server:       "https://dev.example.com",
		Token:        "dev-token",
		GatewayGroup: "dev-group",
	}))
	require.NoError(t, cfg.AddContext(Context{
		Name:          "prod",
		Server:        "https://prod.example.com",
		Token:         "prod-token",
		GatewayGroup:  "prod-group",
		TLSSkipVerify: true,
		CACert:        "/tmp/prod-ca.crt",
	}))
	require.NoError(t, cfg.SetCurrentContext("prod"))
	require.NoError(t, cfg.Save())

	reloaded := NewFileConfigWithPath(path)

	assert.Equal(t, "prod", reloaded.CurrentContext())
	contexts := reloaded.Contexts()
	require.Len(t, contexts, 2)

	ctxNames := []string{contexts[0].Name, contexts[1].Name}
	assert.ElementsMatch(t, []string{"dev", "prod"}, ctxNames)

	assert.Equal(t, "https://prod.example.com", reloaded.BaseURL())
	assert.Equal(t, "prod-token", reloaded.Token())
	assert.Equal(t, "prod-group", reloaded.GatewayGroup())
	assert.True(t, reloaded.TLSSkipVerify())
	assert.Equal(t, "/tmp/prod-ca.crt", reloaded.CACert())
}

func TestBaseURL_Empty(t *testing.T) {
	cfg, _ := newTestFileConfig(t)
	assert.Equal(t, "", cfg.BaseURL())
}

func TestTLSSkipVerify(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{
		Name:          "dev",
		Server:        "https://dev.example.com",
		TLSSkipVerify: true,
	}))

	assert.True(t, cfg.TLSSkipVerify())
}

func TestThreadSafety(t *testing.T) {
	cfg, _ := newTestFileConfig(t)

	require.NoError(t, cfg.AddContext(Context{
		Name:         "ctx1",
		Server:       "https://ctx1.example.com",
		Token:        "token-1",
		GatewayGroup: "group-1",
	}))
	require.NoError(t, cfg.AddContext(Context{
		Name:         "ctx2",
		Server:       "https://ctx2.example.com",
		Token:        "token-2",
		GatewayGroup: "group-2",
	}))

	errCh := make(chan error, 64)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				_ = cfg.BaseURL()
				_ = cfg.Token()
				_ = cfg.GatewayGroup()
				_ = cfg.TLSSkipVerify()
				_ = cfg.CACert()
				_ = cfg.CurrentContext()
				_ = cfg.Contexts()

				if _, err := cfg.GetContext("ctx1"); err != nil {
					errCh <- err
					return
				}

				target := "ctx1"
				if (i+j)%2 == 0 {
					target = "ctx2"
				}
				if err := cfg.SetCurrentContext(target); err != nil {
					errCh <- err
					return
				}

				cfg.SetServerOverride(fmt.Sprintf("https://override-%d-%d.example.com", i, j))
				cfg.SetTokenOverride(fmt.Sprintf("override-token-%d-%d", i, j))
				cfg.SetGatewayGroupOverride(fmt.Sprintf("override-group-%d-%d", i, j))
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	assert.Contains(t, []string{"ctx1", "ctx2"}, cfg.CurrentContext())
	assert.NotEmpty(t, cfg.BaseURL())
	assert.NotEmpty(t, cfg.Token())
	assert.NotEmpty(t, cfg.GatewayGroup())
}
