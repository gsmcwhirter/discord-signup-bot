package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
)

type config struct {
	Version                        string  `mapstructure:"-"`
	DisableSends                   bool    `mapstructure:"disable_sends"`
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

	ClientSecretPath    string `mapstructure:"client_secret_path"`
	ClientTokenPath     string `mapstructure:"client_token_path"`
	PostgresCredsPath   string `mapstructure:"postgres_creds_path"`
	BugsnagAPIKeyPath   string `mapstructure:"bugsnag_apikey_path"`
	HoneycombAPIKeyPath string `mapstructure:"honeycomb_apikey_path"`

	ClientID        string `mapstructure:"-"`
	ClientSecret    string `mapstructure:"-"`
	ClientToken     string `mapstructure:"-"`
	BugsnagAPIKey   string `mapstructure:"-"`
	HoneycombAPIKey string `mapstructure:"-"`
	PgDetails       string `mapstructure:"-"`
}

func (c *config) FillSecrets() error {
	var data []byte
	var dataStr string
	var err error

	if data, err = readFile(c.ClientSecretPath); err != nil {
		return errors.Wrap(err, "could not read client secret path", "path", c.ClientSecretPath)
	}
	dataStr = strings.TrimSpace(string(data))
	parts := strings.SplitN(dataStr, ":", 2)
	if len(parts) < 2 {
		return errors.New("malformed client secret")
	}
	c.ClientID = parts[0]
	c.ClientSecret = parts[1]

	if data, err = readFile(c.ClientTokenPath); err != nil {
		return errors.Wrap(err, "could not read client token path", "path", c.ClientTokenPath)
	}
	c.ClientToken = strings.TrimSpace(string(data))

	if data, err = readFile(c.BugsnagAPIKeyPath); err != nil {
		return errors.Wrap(err, "could not read bugsnag apikey path", "path", c.BugsnagAPIKeyPath)
	}
	c.BugsnagAPIKey = strings.TrimSpace(string(data))

	if data, err = readFile(c.HoneycombAPIKeyPath); err != nil {
		return errors.Wrap(err, "could not read honeycomb apikey path", "path", c.HoneycombAPIKeyPath)
	}
	c.HoneycombAPIKeyPath = strings.TrimSpace(string(data))

	if data, err = readFile(c.PostgresCredsPath); err != nil {
		return errors.Wrap(err, "could not read postgres creds path", "path", c.PostgresCredsPath)
	}
	dataStr = strings.TrimSpace(string(data))
	parts = strings.SplitN(dataStr, ":", 2)
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

func readFile(fname string) ([]byte, error) {
	fh, err := os.Open(fname)
	if err != nil {
		return nil, errors.Wrap(err, "could not open file")
	}
	defer deferutil.CheckDefer(fh.Close)

	return ioutil.ReadAll(fh)
}
