// This file implements a worker-pool Discord sender that dequeues requests
// from a buffered channel and posts them to the configured Discord channels.
package discord

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/blackhawk42/heroldo/pkg/heroldo"
	"github.com/blackhawk42/heroldo/pkg/set"
	"github.com/bwmarrin/discordgo"
)

// DiscordSender manages a pool of worker goroutines that read from a buffered
// channel and send requests to the configured Discord channels.
//
// For graceful shutdown, the Close method should be used.
type DiscordSender struct {
	channels       set.Set[string]
	discordSession *discordgo.Session
	wg             *sync.WaitGroup
	requests       chan heroldo.Request
	closeOnce      sync.Once
}

// NewDiscordSender creates a DiscordSender with the given concurrency level,
// Discord session, and list of allowed channel IDs.
//
// It starts concurrency worker goroutines that process incoming requests.
func NewDiscordSender(concurrency int, session *discordgo.Session, channels []string) *DiscordSender {
	if concurrency < 1 {
		concurrency = 1
	}

	ds := &DiscordSender{
		channels:       set.NewSet(channels...),
		discordSession: session,
		wg:             new(sync.WaitGroup),
		requests:       make(chan heroldo.Request, concurrency),
	}

	for range concurrency {
		ds.wg.Add(1)
		go ds.sendWorker()
	}

	return ds
}

// Channels returns the configured Discord channel IDs as a slice.
func (ds *DiscordSender) Channels() []string {
	return slices.Collect(ds.channels.Members())
}

// Send queues a request for delivery.
//
// It returns the list of channels the request will be actually sent to, or an error if
// the context is cancelled.
func (ds *DiscordSender) Send(ctx context.Context, request heroldo.Request) ([]string, error) {
	var channels []string
	if request.Channels.Len() == 0 {
		channels = ds.Channels()
	} else {
		channels = slices.Collect(ds.channels.Intersection(request.Channels).Members())
	}

	select {
	case ds.requests <- request:
		return channels, nil
	case <-ctx.Done():
		return channels, ctx.Err()
	}
}

// Close signals all workers to shut down and waits for them to finish or
// for the context to expire.
func (ds *DiscordSender) Close(ctx context.Context) error {
	ds.closeOnce.Do(func() {
		close(ds.requests)
	})

	done := make(chan struct{})
	go func() {
		ds.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// sendWorker is a long-running goroutine that dequeues requests from the
// buffered channel and posts them to the appropriate Discord channels.
func (ds *DiscordSender) sendWorker() {
	defer ds.wg.Done()

	workerLogger := slog.With()

	for request := range ds.requests {
		requestLogger := workerLogger.With("request_id", request.RequestID)

		for ch := range ds.channels.Members() {
			if request.Channels.Len() != 0 && !request.Channels.Contains(ch) {
				requestLogger.Debug("skipping channel, as it's not in request list", "channel", ch)
				continue
			}

			messageSend := &discordgo.MessageSend{
				Content: request.Text,
			}

			for _, f := range request.Files {
				messageSend.Files = append(messageSend.Files, &discordgo.File{
					Name:        f.CompleteName(),
					ContentType: f.ContentType,
					Reader:      bytes.NewReader(f.Content),
				})
			}

			_, err := ds.discordSession.ChannelMessageSendComplex(ch, messageSend)
			if err != nil {
				requestLogger.Error("while sending message to discord", "channel", ch, "error", err)
				continue
			}
			requestLogger.Info("message sent", "channel", ch)
		}
	}
}
