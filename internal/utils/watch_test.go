package utils_test

import (
	"time"

	"github.com/dalibo/ldap2pg/internal/utils"
)

func (suite *Suite) TestStopwatch() {
	r := suite.Require()

	t := utils.StopWatch{}
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
