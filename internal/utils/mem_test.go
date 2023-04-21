package utils_test

import (
	"fmt"
	"testing"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/utils"

	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slog"
)

func ExampleFormatBytes() {
	var value int
	value = 5546875
	fmt.Printf("%dB = %s\n", value, utils.FormatBytes(value))
	value = 4
	fmt.Printf("%dB = %s\n", value, utils.FormatBytes(value))
	value = 900
	fmt.Printf("%dB = %s\n", value, utils.FormatBytes(value))
	// Output:
	// 5546875B = 5.3MiB
	// 4B = 4B
	// 900B = 0.9KiB
}

type MemSuite struct {
	suite.Suite
}

func (suite *MemSuite) TestFormatBytes() {
	r := suite.Require()

	r.Equal("0B", utils.FormatBytes(0))
	r.Equal("1KiB", utils.FormatBytes(999))
}

func TestUtils(t *testing.T) {
	if testing.Verbose() {
		config.SetLoggingHandler(slog.LevelDebug, false)
	} else {
		config.SetLoggingHandler(slog.LevelWarn, false)
	}
	suite.Run(t, new(MemSuite))
}
