package utils_test

import (
	"github.com/dalibo/ldap2pg/internal/utils"
)

func (suite *Suite) TestProductNoLists() {
	r := suite.Require()
	for item := range utils.Product[string]() {
		r.Fail("Got item: %s", item)
	}
}

func (suite *Suite) TestProductOneEmptyList() {
	r := suite.Require()
	for item := range utils.Product(
		[]string{"1", "2"},
		[]string{},
		[]string{"a", "b"},
	) {
		r.Fail("Got item: %s", item)
	}
}

type dumbStruct struct {
	A string
}

func (suite *Suite) TestProductAny() {
	r := suite.Require()

	var combinations [][]any
	s0 := dumbStruct{A: "s0"}
	s1 := dumbStruct{A: "s1"}
	for item := range utils.Product[interface{}](
		[]interface{}{"1", "2", "3"},
		[]interface{}{s0, s1},
	) {
		combinations = append(combinations, item)
	}

	r.Equal(3*2, len(combinations))
	r.Equal([]interface{}{"1", s0}, combinations[0])
	r.Equal([]interface{}{"1", s1}, combinations[1])
	r.Equal([]interface{}{"2", s0}, combinations[2])
	r.Equal([]interface{}{"2", s1}, combinations[3])
	r.Equal([]interface{}{"3", s0}, combinations[4])
	r.Equal([]interface{}{"3", s1}, combinations[5])
}

func (suite *Suite) TestProductString() {
	r := suite.Require()

	var combinations [][]string
	for item := range utils.Product(
		[]string{"1", "2", "3"},
		[]string{"a", "b", "c"},
		[]string{"A", "B"},
	) {
		combinations = append(combinations, item)
	}

	r.Equal(3*3*2, len(combinations))
	r.Equal([]string{"1", "a", "A"}, combinations[0])
	r.Equal([]string{"1", "a", "B"}, combinations[1])
	r.Equal([]string{"1", "b", "A"}, combinations[2])
	r.Equal([]string{"1", "b", "B"}, combinations[3])
	r.Equal([]string{"1", "c", "A"}, combinations[4])
	r.Equal([]string{"1", "c", "B"}, combinations[5])
	r.Equal([]string{"2", "a", "A"}, combinations[6])
	r.Equal([]string{"2", "a", "B"}, combinations[7])
	r.Equal([]string{"2", "b", "A"}, combinations[8])
	r.Equal([]string{"2", "b", "B"}, combinations[9])
	r.Equal([]string{"2", "c", "A"}, combinations[10])
	r.Equal([]string{"2", "c", "B"}, combinations[11])
	r.Equal([]string{"3", "a", "A"}, combinations[12])
	r.Equal([]string{"3", "a", "B"}, combinations[13])
	r.Equal([]string{"3", "b", "A"}, combinations[14])
	r.Equal([]string{"3", "b", "B"}, combinations[15])
	r.Equal([]string{"3", "c", "A"}, combinations[16])
	r.Equal([]string{"3", "c", "B"}, combinations[17])
}

func (suite *Suite) TestProductInt() {
	r := suite.Require()

	var combinations [][]int
	for item := range utils.Product(
		[]int{1, 3},
		[]int{2, 4},
	) {
		combinations = append(combinations, item)
	}

	r.Equal(2*2, len(combinations))
	r.Equal([]int{1, 2}, combinations[0])
	r.Equal([]int{1, 4}, combinations[1])
	r.Equal([]int{3, 2}, combinations[2])
	r.Equal([]int{3, 4}, combinations[3])
}
