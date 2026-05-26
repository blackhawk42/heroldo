package heroldo

import (
	"io"
)

type File struct {
	Content     io.Reader
	ContentType string
	Name        string
	Spoiler     bool
}

func (f *File) CompleteName() string {
	if f.Spoiler {
		return "SPOILER_" + f.Name
	} else {
		return f.Name
	}
}

type Request struct {
	RequestID string
	Text      string
	Files     []*File
}
