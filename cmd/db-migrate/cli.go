package main

import (
	"github.com/spf13/viper"

	"github.com/gsmcwhirter/go-util/v8/cli"
	"github.com/gsmcwhirter/go-util/v8/errors"
)

func setup(start func(config) error) *cli.Command {
	c := cli.NewCLI(AppName, BuildVersion, BuildSHA, BuildDate, cli.CommandOptions{
		ShortHelp: "Manage the discord bot",
		Args:      cli.NoArgs,
	})

	var configFile string

	c.Flags().StringVar(&configFile, "config", "./config.toml", "The config file to use")
	c.Flags().String("pg", "", "The postgres connection string")

	c.SetRunFunc(func(cmd *cli.Command, args []string) (err error) {
		v := viper.New()

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

		return start(conf)
	})

	return c
}
