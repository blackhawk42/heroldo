package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var v *viper.Viper

func initConfig(cmd *cobra.Command) {
	v = viper.New()

	osConfigPath, err := os.UserConfigDir()
	if err != nil {
		slog.Warn("error while getting user config dir", "error", err)
	}
	osConfigPath = filepath.Join(osConfigPath, "heroldo")
	slog.Debug("os-dependent config path set", "os_dependent_config_path", osConfigPath)

	v.SetConfigName("heroldo")
	v.AddConfigPath(".")
	v.AddConfigPath(osConfigPath)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			slog.Warn("error reading config file", "error", err)
		}
	}

	v.SetEnvPrefix("HEROLDO")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.BindPFlags(cmd.Flags())
}

func runServer(cmd *cobra.Command, _ []string) error {
	token := v.GetString("token")
	if token == "" {
		return fmt.Errorf("token is required")
	}

	rawChannels := v.Get("channels")
	channelIDs := toStringSlice(rawChannels)
	if len(channelIDs) == 0 {
		return fmt.Errorf("at least one channel is required")
	}

	port := v.GetInt("port")
	concurrency := v.GetInt("concurrency")
	maxBodySize := v.GetInt64("max-body-size")

	listenAddr := ":" + strconv.Itoa(port)

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	if err := session.Open(); err != nil {
		return err
	}
	defer session.Close()

	sender := NewDiscordSender(concurrency, session, channelIDs)
	defer sender.Close()

	handler := RequestHandler(maxBodySize, sender)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}

	go func() {
		slog.Info("starting HTTP server", "addr", listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")

	if err := server.Shutdown(context.Background()); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	return nil
}

// toStringSlice normalises the "channels" value across different sources:
// CLI ([]string), config file list ([]any), and env var/comma-separated string.
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

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	rootCmd := &cobra.Command{
		Use:   "heroldo",
		Short: "Discord bot that relays HTTP multipart form data to Discord channels",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			initConfig(cmd)
			return nil
		},
		RunE: runServer,
	}

	rootCmd.Flags().StringP("token", "t", "", "Discord bot token")
	rootCmd.Flags().StringSliceP("channels", "c", nil, "Discord channel IDs (comma-separated or repeatable)")
	rootCmd.Flags().IntP("port", "p", 8080, "HTTP server port")
	rootCmd.Flags().IntP("concurrency", "w", 5, "Worker goroutine count")
	rootCmd.Flags().Int64("max-body-size", 50<<20, "Maximum request body size in bytes")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
