package shared

import (
	"log/slog"
)

// LogClose calls closer and logs any error via slog. Intended for use in
// deferred cleanup where the error cannot be propagated, e.g.:
//
//	defer shared.LogClose(rows.Close, "could not close rows")
func LogClose(closer func() error, msg string) {
	if err := closer(); err != nil {
		slog.Default().Error(msg, "error", err)
	}
}
