package ldap2pg

import (
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
	return
}
