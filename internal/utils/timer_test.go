package utils_test

import (
	"time"

	"github.com/dalibo/ldap2pg/internal/utils"
)

func (suite *Suite) TestTimer() {
	r := suite.Require()

	t := utils.Timer{}
	t.TimeIt(func() {
		time.Sleep(time.Microsecond)
	})
	r.Less(0*time.Nanosecond, t.Total)
	backup := t.Total

	t.TimeIt(func() {
		time.Sleep(time.Microsecond)
	})
	r.Less(backup, t.Total)
}
