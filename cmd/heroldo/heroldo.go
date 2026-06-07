// This files implements the main command line of the program.
package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Viper configuration
var v *viper.Viper

// initConfig loads configuration from a config file, environment variables,
// and command-line flags using viper.
//
// It reads "heroldo" config files from the current directory and the OS-specific
// config directory, then applies environment overrides and flag bindings.
func initConfig(cmd *cobra.Command) error {
	v = viper.New()

	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		osConfigDir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("error while getting user config dir: %w", err)
		}
		osConfigPath := filepath.Join(osConfigDir, "heroldo")

		v.SetConfigName("heroldo")
		v.AddConfigPath(".")
		v.AddConfigPath(osConfigPath)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	v.SetEnvPrefix("HEROLDO")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.BindPFlags(cmd.Flags())
	v.BindPFlags(cmd.Root().PersistentFlags())

	return nil
}

// closeLogging will be set to something that closes the underlying file handle
// if a logging file is used. Otherwise, it will be nil.
var closeLogging func()

// initLogging initializes logging, including log level and opening files.
func initLogging() error {
	verboseOutput := v.GetBool("verbose")

	var loggerLevel = slog.LevelInfo
	if verboseOutput {
		loggerLevel = slog.LevelDebug
	}

	var loggingOutput io.Writer = os.Stderr
	if logPath := v.GetString("log-file"); logPath != "" {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0664)
		if err != nil {
			return fmt.Errorf("while attempting to open log file %s: %w", logPath, err)
		}

		closeLogging = func() { f.Close() }
		loggingOutput = f
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(loggingOutput, &slog.HandlerOptions{
		Level: loggerLevel,
	})))

	return nil
}

// main sets up the root cobra command with all required flags and executes it.
//
// It configures structured logging to stderr before handling CLI arguments.
func main() {
	rootCmd := &cobra.Command{
		Use:   "heroldo",
		Short: "Discord bot that relays HTTP multipart form data to Discord channels",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			err := initConfig(cmd)
			if err != nil {
				return err
			}

			err = initLogging()
			if err != nil {
				return err
			}

			configFileUsed := v.ConfigFileUsed()
			if configFileUsed == "" {
				slog.Debug("no config file used")
			} else {
				slog.Debug("config file used", "config_file", configFileUsed)
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringP("config", "f", "", "Path to custom config file")
	rootCmd.PersistentFlags().String("auth-db", "", "Path to bbolt authentication database (optional; enables token auth)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable debug messages")
	rootCmd.PersistentFlags().String("log-file", "", "Path to a custom log file. Leave empty for stderr.")

	rootCmd.AddCommand(serveCmd, tokensCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	if closeLogging != nil {
		closeLogging()
	}
}
