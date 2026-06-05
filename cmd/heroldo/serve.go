package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	heroldodiscord "github.com/blackhawk42/heroldo/pkg/heroldo/discord"
	heroldohttp "github.com/blackhawk42/heroldo/pkg/heroldo/http"
	"github.com/blackhawk42/heroldo/pkg/heroldo/registries"
	"github.com/bwmarrin/discordgo"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server and Discord bot",
	RunE:  runServer,
}

func init() {
	serveCmd.Flags().StringP("discord-token", "t", "", "Discord bot token (required)")
	serveCmd.Flags().StringSliceP("channels", "c", nil, "Discord channel IDs (required; comma-separated or repeatable)")
	serveCmd.Flags().IntP("port", "p", 8080, "HTTP server port")
	serveCmd.Flags().IntP("concurrency", "w", 5, "Worker goroutine count")
	serveCmd.Flags().Int64("max-body-size", 50<<20, "Maximum request body size in bytes")
	serveCmd.Flags().Int("shutdown-timeout", 30, "Shutdown timeout in seconds")
}

// runServer starts the HTTP server and the Discord session, creates a
// DiscordSender worker pool, and blocks until SIGINT or SIGTERM is received.
//
// It then performs a graceful shutdown of both the HTTP server and the sender.
func runServer(cmd *cobra.Command, _ []string) error {
	token := v.GetString("discord-token")
	if token == "" {
		return fmt.Errorf("discord-token is required")
	}

	rawChannels := v.Get("channels")
	channelIDs := toStringSlice(rawChannels)
	if len(channelIDs) == 0 {
		return fmt.Errorf("at least one channel is required")
	}

	port := v.GetInt("port")
	concurrency := v.GetInt("concurrency")
	maxBodySize := v.GetInt64("max-body-size")
	shutdownTimeout := time.Duration(v.GetInt("shutdown-timeout")) * time.Second

	listenAddr := ":" + strconv.Itoa(port)

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	if err := session.Open(); err != nil {
		return err
	}
	defer session.Close()

	sender := heroldodiscord.NewDiscordSender(concurrency, session, channelIDs)

	var handler http.Handler
	handler = heroldohttp.RequestHandler(maxBodySize, sender)

	authDBPath := v.GetString("auth-db")
	if authDBPath != "" {
		registry, err := registries.NewBBoltTokenRegistry(authDBPath, nil, 0)
		if err != nil {
			return err
		}
		defer registry.Close()

		handler = heroldohttp.TokenAuthMiddleware(handler, registry)
	}

	handler = heroldohttp.IdMiddleware(handler)

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	if err := sender.Close(shutdownCtx); err != nil {
		slog.Error("Discord sender close error", "error", err)
	}

	return nil
}
