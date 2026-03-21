package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/root"
	"github.com/api7/a7/pkg/iostreams"
)

func main() {
	ios := iostreams.System()

	cfg := config.NewFileConfig()

	f := &cmd.Factory{
		IOStreams: ios,
		Config: func() (config.Config, error) {
			return cfg, nil
		},
		HttpClient: func() (*http.Client, error) {
			token := cfg.Token()
			if token == "" {
				return nil, fmt.Errorf("no API token configured; run 'a7 context create' or set A7_TOKEN")
			}
			return api.NewAuthenticatedClient(
				token,
				cfg.TLSSkipVerify(),
				cfg.CACert(),
			), nil
		},
	}

	rootCmd := root.NewCmd(f, cfg)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
