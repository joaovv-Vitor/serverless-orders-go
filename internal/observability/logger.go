package observability

import (
	"io"
	"log/slog"
)

// NewJSONLogger creates the structured logger shared by all Lambda functions.
func NewJSONLogger(writer io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attribute slog.Attr) slog.Attr {
			if attribute.Key == slog.MessageKey {
				attribute.Key = "message"
			}
			return attribute
		},
	}))
}
