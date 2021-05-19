package main

import (
	"github.com/spf13/viper"

	"github.com/gsmcwhirter/go-util/v8/errors"

	"github.com/gsmcwhirter/go-util/v8/cli"
)

func setup(start func(config) error) *cli.Command {
	c := cli.NewCLI(AppName, BuildVersion, BuildSHA, BuildDate, cli.CommandOptions{
		ShortHelp:    "Manage the discord bot",
		Args:         cli.NoArgs,
		SilenceUsage: true,
	})

	var configFile string

	c.Flags().StringVar(&configFile, "config", "./config.toml", "The config file to use")

	c.Flags().String("client_secret_path", "", "The path to the client login file")
	c.Flags().String("client_token_path", "", "The path to the client token file")
	c.Flags().String("postgres_creds_path", "", "The path to the database credentials file")
	c.Flags().String("bugsnag_apikey_path", "", "The path to the bugsnag api key file")
	c.Flags().String("honeycomb_apikey_path", "", "The path to the honeycomb api key file")

	c.Flags().String("log_format", "", "The logger format")
	c.Flags().String("log_level", "", "The minimum log level to show")
	c.Flags().Int("num_workers", 0, "The number of worker goroutines to run")
	c.Flags().String("pprof_hostport", "", "The host and port for the pprof http server to listen on")

	c.SetRunFunc(func(cmd *cli.Command, args []string) (err error) {
		v := viper.New()

		v.SetDefault("pprof_hostport", "127.0.0.1:6060")

		if configFile != "" {
			v.SetConfigFile(configFile)
		} else {
			v.SetConfigName("config")
			v.AddConfigPath(".") // working directory
		}

		v.SetEnvPrefix("EDB")
		v.AutomaticEnv()

		err = v.BindPFlags(cmd.Flags())
		if err != nil {
			return errors.Wrap(err, "could not bind flags to viper")
		}

		err = v.ReadInConfig()
		if err != nil {
			return errors.Wrap(err, "could not read in config file")
		}

		conf := config{}
		err = v.Unmarshal(&conf)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal config into struct")
		}

		err = conf.FillSecrets()
		if err != nil {
			return errors.Wrap(err, "could not fill in secrets into config struct")
		}

		conf.Version = cmd.Version

		return start(conf)
	})

	return c
}
