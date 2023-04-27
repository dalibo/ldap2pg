package utils_test

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/utils"
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

func (suite *Suite) TestFormatBytes() {
	r := suite.Require()

	r.Equal("0B", utils.FormatBytes(0))
	r.Equal("1KiB", utils.FormatBytes(999))
}
