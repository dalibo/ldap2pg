package config_test

import (
	"flag"
	"log/slog"
	"os"
	"testing"

	"github.com/dalibo/ldap2pg/internal"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		internal.SetLoggingHandler(slog.LevelDebug, false)
	} else {
		internal.SetLoggingHandler(slog.LevelWarn, false)
	}
	os.Exit(m.Run())
}
