// This files implements the main command line of the program.
package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var v *viper.Viper

// initConfig loads configuration from a config file, environment variables,
// and command-line flags using viper.
//
// It reads "heroldo" config files from the current directory and the OS-specific
// config directory, then applies environment overrides and flag bindings.
func initConfig(cmd *cobra.Command) {
	v = viper.New()

	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		osConfigDir, err := os.UserConfigDir()
		if err != nil {
			slog.Warn("error while getting user config dir", "error", err)
		}
		osConfigPath := filepath.Join(osConfigDir, "heroldo")
		slog.Debug("os-dependent config path set", "os_dependent_config_path", osConfigPath)

		v.SetConfigName("heroldo")
		v.AddConfigPath(".")
		v.AddConfigPath(osConfigPath)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			slog.Warn("error reading config file", "error", err)
		}
	}

	v.SetEnvPrefix("HEROLDO")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.BindPFlags(cmd.Flags())
	v.BindPFlags(cmd.Root().PersistentFlags())

	verboseOutput := v.GetBool("verbose")
	if verboseOutput {
		logLvl.Set(slog.LevelDebug)
	}

	configFileUsed := v.ConfigFileUsed()
	if configFileUsed == "" {
		slog.Debug("no config file used")
	} else {
		slog.Debug("config file used", "config_file", configFileUsed)
	}
}

// toStringSlice normalises the channels value across different viper sources:
// CLI ([]string), config file list ([]any), and env var (comma-separated string).
func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		ids := make([]string, 0, len(val))
		for _, c := range val {
			if s, ok := c.(string); ok {
				ids = append(ids, s)
			}
		}
		return ids
	case string:
		ids := strings.Split(val, ",")
		for i := range ids {
			ids[i] = strings.TrimSpace(ids[i])
		}
		return ids
	default:
		return nil
	}
}

// logLvl is used to dynamically change the log level
var logLvl = new(slog.LevelVar)

// main sets up the root cobra command with all required flags and executes it.
//
// It configures structured logging to stderr before handling CLI arguments.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLvl,
	})))

	rootCmd := &cobra.Command{
		Use:   "heroldo",
		Short: "Discord bot that relays HTTP multipart form data to Discord channels",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			initConfig(cmd)
			return nil
		},
	}

	rootCmd.PersistentFlags().StringP("config", "f", "", "Path to custom config file")
	rootCmd.PersistentFlags().String("auth-db", "", "Path to bbolt authentication database (optional; enables token auth)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable debug messages")

	rootCmd.AddCommand(serveCmd, tokensCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
