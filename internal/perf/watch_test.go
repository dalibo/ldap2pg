package perf_test

import (
	"time"

	"github.com/dalibo/ldap2pg/v6/internal/perf"
)

func (suite *Suite) TestStopwatch() {
	r := suite.Require()

	t := perf.StopWatch{}
	t.TimeIt(func() {
		time.Sleep(time.Microsecond)
	})
	r.Less(0*time.Nanosecond, t.Total)
	r.Equal(1, t.Count)
	backup := t.Total

	t.TimeIt(func() {
		time.Sleep(time.Microsecond)
	})
	r.Less(backup, t.Total)
	r.Equal(2, t.Count)
}
