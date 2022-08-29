package ldap2pg

import (
	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger

func SetupLogging() (err error) {
	config := zap.Config{
		DisableCaller:    true,
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
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
