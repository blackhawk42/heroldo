// Package heroldo defines the core types used by the heroldo Discord bot.
package heroldo

import "github.com/r/blackhawk42/heroldo/pkg/set"

// File represents a file attached to a Request.
type File struct {
	// Content holds the raw bytes of the file.
	Content []byte

	// ContentType is the MIME type of the file (e.g. "image/png").
	ContentType string

	// Name is the original filename.
	Name string

	// Spoiler marks whether the file should be hidden behind a Discord spoiler tag.
	Spoiler bool
}

// CompleteName returns the filename, prefixed with "SPOILER_" when the file is
// marked as a spoiler.
func (f *File) CompleteName() string {
	if f.Spoiler {
		return "SPOILER_" + f.Name
	} else {
		return f.Name
	}
}

// Request represents content to be posted to one or more Discord channels.
type Request struct {
	// RequestID is a unique identifier generated for each incoming request.
	RequestID string

	// Text is the optional message text accompanying the files.
	Text string

	// Files holds the files attached to the request.
	Files []*File

	// Channels is the set of target Discord channel IDs.
	//
	// This should be a subset of the configured channels for the main Discord bot.
	// Channels listed here but not configured will simply be ignored
	//
	// When empty, the request is broadcasted to all configured channels.
	Channels set.Set[string]
}
