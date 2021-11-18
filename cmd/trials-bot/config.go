package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/errors"
)

var ErrMissingSecret = errors.New("missing secret")

type config struct {
	Version                        string  `mapstructure:"-"`
	DisableSends                   bool    `mapstructure:"disable_sends"`
	DisableInteractionSends        bool    `mapstructure:"disable_interaction_sends"`
	BotName                        string  `mapstructure:"bot_name"`
	BotPresence                    string  `mapstructure:"bot_presence"`
	ClientURL                      string  `mapstructure:"client_url"`
	LogFormat                      string  `mapstructure:"log_format"`
	LogLevel                       string  `mapstructure:"log_level"`
	PProfHostPort                  string  `mapstructure:"pprof_hostport"`
	NumWorkers                     int     `mapstructure:"num_workers"`
	TraceProbability               float64 `mapstructure:"trace_probability"`
	PrometheusNamespace            string  `mapstructure:"prometheus_namespace"`
	PrometheusHostPort             string  `mapstructure:"prometheus_hostport"`
	BugsnagReleaseStage            string  `mapstructure:"bugsnag_release_stage"`
	HoneycombDataset               string  `mapstructure:"honeycomb_dataset"`
	PostgresHost                   string  `mapstructure:"postgres_host"`
	PostgresPort                   int     `mapstructure:"postgres_port"`
	PostgresSSLMode                string  `mapstructure:"postgres_sslmode"`
	PostgresDatabase               string  `mapstructure:"postgres_database"`
	PostgresStatementCacheCapacity int     `mapstructure:"postgres_statement_cache_capacity"`
	PostgresStatementCacehMode     string  `mapstructure:"postgres_statement_cache_mode"`
	PostgresMinPoolSize            int32   `mapstructure:"postgres_min_pool_size"`
	PostgresMaxPoolSize            int32   `mapstructure:"postgres_max_pool_size"`

	ClientSecretVar    string `mapstructure:"client_secret_var"`
	ClientTokenVar     string `mapstructure:"client_token_var"`
	PostgresCredsVar   string `mapstructure:"postgres_creds_var"`
	BugsnagAPIKeyVar   string `mapstructure:"bugsnag_apikey_var"`
	HoneycombAPIKeyVar string `mapstructure:"honeycomb_apikey_var"`

	ClientID        string `mapstructure:"-"`
	ClientSecret    string `mapstructure:"-"`
	ClientToken     string `mapstructure:"-"`
	BugsnagAPIKey   string `mapstructure:"-"`
	HoneycombAPIKey string `mapstructure:"-"`
	PgDetails       string `mapstructure:"-"`
}

func (c *config) FillSecrets() error {
	var data string
	var ok bool

	if data, ok = os.LookupEnv(c.ClientSecretVar); !ok {
		return errors.Wrap(ErrMissingSecret, "could not read client secret", "var", c.ClientSecretVar)
	}
	data = strings.TrimSpace(data)
	parts := strings.SplitN(data, ":", 2)
	if len(parts) < 2 {
		return errors.New("malformed client secret")
	}
	c.ClientID = parts[0]
	c.ClientSecret = parts[1]

	if data, ok = os.LookupEnv(c.ClientTokenVar); !ok {
		return errors.Wrap(ErrMissingSecret, "could not read client token", "var", c.ClientTokenVar)
	}
	c.ClientToken = strings.TrimSpace(data)

	if data, ok = os.LookupEnv(c.BugsnagAPIKeyVar); !ok {
		return errors.Wrap(ErrMissingSecret, "could not read bugsnag apikey", "var", c.BugsnagAPIKeyVar)
	}
	c.BugsnagAPIKey = strings.TrimSpace(data)

	if data, ok = os.LookupEnv(c.HoneycombAPIKeyVar); !ok {
		return errors.Wrap(ErrMissingSecret, "could not read honeycomb apikey", "var", c.HoneycombAPIKeyVar)
	}
	c.HoneycombAPIKey = strings.TrimSpace(data)

	if data, ok = os.LookupEnv(c.PostgresCredsVar); !ok {
		return errors.Wrap(ErrMissingSecret, "could not read postgres creds", "var", c.PostgresCredsVar)
	}
	data = strings.TrimSpace(data)
	parts = strings.SplitN(data, ":", 2)
	if len(parts) < 2 {
		return errors.New("malformed postgres secret")
	}
	c.PgDetails = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s&statement_cache_capacity=%d&statement_cache_mode=%s",
		parts[0],
		parts[1],
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDatabase,
		c.PostgresSSLMode,
		c.PostgresStatementCacheCapacity,
		c.PostgresStatementCacehMode)

	return nil
}

// func readFile(fname string) ([]byte, error) {
// 	fh, err := os.Open(fname)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "could not open file")
// 	}
// 	defer deferutil.CheckDefer(fh.Close)

// 	return ioutil.ReadAll(fh)
// }
