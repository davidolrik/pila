package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var Config *viper.Viper

var globalFlagsToConfigKey = map[string]string{
	"config-path": "config_path",
	"verbose":     "verbose",
}

func InitializeConfig(cmd *cobra.Command) ([]string, error) {
	Config = viper.New()

	// Set config path from user input
	configPath, err := cmd.Parent().Flags().GetString("config-path")
	if err != nil {
		panic("Unable to determine config path")
	}
	Config.AddConfigPath(configPath)

	// Set config name
	Config.SetConfigName("config")
	Config.SetConfigType("toml")

	// Set defaults
	Config.SetDefault("verbose", 0)
	Config.SetDefault("logger.level", "debug")
	Config.SetDefault("logger.path", configPath)

	// Setup env reading
	Config.SetEnvPrefix("pila")

	// Load config file
	if err := Config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found - create config path and write config with defaults
			err := os.MkdirAll(configPath, 0o755)
			if err != nil {
				panic(err)
			}
			Config.SafeWriteConfig()
		} else {
			// Config file was found but another error occurred
			panic(err)
		}
	}

	// In order to get environment variables mapped into config sections, we need to replace . with _
	Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	Config.AutomaticEnv() // read in environment variables that match

	// Bind the current command's flags to viper
	if cmd != nil {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			// Is this a global flag
			configKey, ok := globalFlagsToConfigKey[f.Name]
			if !ok {
				return
			}

			// Apply the viper config value to the flag when the flag is not set and viper has a value
			if !f.Changed && Config.IsSet(configKey) {
				cmd.Flags().Set(f.Name, fmt.Sprintf("%v", Config.Get(configKey)))
			} else {
				Config.Set(configKey, fmt.Sprintf("%v", f.Value))
			}
		})
	}

	return []string{}, nil
}
