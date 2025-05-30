package perf_test

import (
	"fmt"

	"github.com/dalibo/ldap2pg/v6/internal/perf"
)

func ExampleFormatBytes() {
	var value int
	value = 5546875
	fmt.Printf("%dB = %s\n", value, perf.FormatBytes(value))
	value = 4
	fmt.Printf("%dB = %s\n", value, perf.FormatBytes(value))
	value = 900
	fmt.Printf("%dB = %s\n", value, perf.FormatBytes(value))
	// Output:
	// 5546875B = 5.3MiB
	// 4B = 4B
	// 900B = 0.9KiB
}

func (suite *Suite) TestFormatBytes() {
	r := suite.Require()

	r.Equal("0B", perf.FormatBytes(0))
	r.Equal("1KiB", perf.FormatBytes(999))
}
