package ldap2pg

import (
	"os"

	"go.uber.org/zap"
)

var (
	Logger   *zap.SugaredLogger
	LogLevel zap.AtomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
)

func SetupLogging() (err error) {
	config := zap.Config{
		DisableCaller:    true,
		Level:            LogLevel,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	basic, err := config.Build()
	if err != nil {
		return
	}
	_, err = zap.RedirectStdLogAt(basic, zap.DebugLevel)
	if err != nil {
		return
	}
	Logger = basic.Sugar()

	// Early configuration using environment variable, to debug initialization.
	envlevel, found := os.LookupEnv("LDAP2PG_VERBOSITY")
	if !found {
		return
	}
	err = LogLevel.UnmarshalText([]byte(envlevel))

	// Show this debug message only if LDAP2PG_VERBOSITY is set.
	Logger.Debugw("Initializing ldap2pg.", "version", Version)
	return
}
