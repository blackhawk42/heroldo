package main

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/r/blackhawk42/heroldo/pkg/heroldo"
	"github.com/r/blackhawk42/heroldo/pkg/set"
)

type DiscordSender struct {
	channels       set.Set[string]
	discordSession *discordgo.Session
	wg             *sync.WaitGroup
	requests       chan heroldo.Request
	closeOnce      sync.Once
}

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

func (ds *DiscordSender) Channels() []string {
	return slices.Collect(ds.channels.Members())
}

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
