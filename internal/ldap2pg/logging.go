package ldap2pg

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

const (
	fontReset = "0"
	fontBold  = "1"

	// Colors code from systemctl/journalctl
	fontDebug    = "38;5;245"
	fontInfo     = "1;39"
	fontWarn     = "38;5;185"
	fontError    = "31"
	fontCritical = "1;91"
)

func init() {
	if isatty.IsTerminal(os.Stderr.Fd()) {
		log.SetFormatter(&ConsoleFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			TimestampFormat: time.RFC3339,
		})
	}
	log.SetOutput(os.Stderr)
}

func SetupLogging() (err error) {
	_, debug := os.LookupEnv("DEBUG")
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		// Early configuration using environment variable, to debug initialization.
		envlevel, found := os.LookupEnv("LDAP2PG_VERBOSITY")
		if !found {
			return
		}
		level, err := log.ParseLevel(envlevel)
		if err != nil {
			return fmt.Errorf("Bad LDAP2PG_VERBOSITY value: %s", envlevel)
		}
		log.SetLevel(level)
	}

	// Show this debug message only if LDAP2PG_VERBOSITY is set.
	log.
		WithFields(log.Fields{
			"version": Version,
			"debug":   debug,
		}).
		Debug("Initializing ldap2pg.")

	return
}

type ConsoleFormatter struct{}

func (f *ConsoleFormatter) Format(entry *log.Entry) ([]byte, error) {
	col := 0
	data := make(log.Fields)
	for k, v := range entry.Data {
		data[k] = v
	}

	// Sort extra keys to ensure consistent output.
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Write time (not date)
	s := entry.Time.Format("15:04:05.000")
	fmt.Fprint(b, colorize(s, fontDebug))
	col += len(s)

	// Padding space
	b.WriteByte(' ')
	col = col + 1

	// Write level and message, colored.
	s = fmt.Sprintf("%-5.5s %s", strings.ToUpper(entry.Level.String()), entry.Message)
	fmt.Fprint(b, colorize(s, levelColor(entry.Level.String())))
	col = col + len(s)

	// Pad to align logfmt.
	pad := math.Max(1., float64(64-col))
	fmt.Fprintf(b, "%-*s", int(pad), "")

	// Write extra keys as logfmt.
	for _, k := range keys {
		fmt.Fprintf(b, " %s=", k)
		value, ok := data[k].(string)
		if !ok {
			value = fmt.Sprint(data[k])
		}
		if needsQuoting(value) {
			fmt.Fprintf(b, "%q", value)
		} else {
			fmt.Fprint(b, value)
		}
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func colorize(s interface{}, c string) string {
	return fmt.Sprintf("\x1b[%sm%v\x1b[0m", c, s)
}

func levelColor(levelname string) string {
	switch levelname {
	case "trace":
		return fontDebug
	case "debug":
		return fontDebug
	case "info":
		return fontInfo
	case "warn":
		return fontWarn
	case "error":
		return fontError
	case "fatal":
		return fontBold + ";" + fontError
	case "panic":
		return fontBold + ";" + fontError
	default:
		return fontReset
	}
}

func needsQuoting(s string) bool {
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}
