package inspect_test

import (
	"context"

	"github.com/dalibo/ldap2pg/v6/internal/inspect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (suite *Suite) TestQuerierYAML() {
	r := suite.Require()
	var q inspect.Querier[string] = &inspect.YAMLQuery[string]{
		Rows: []string{"adam", "eve"},
	}

	names := make([]string, 0)
	for q.Query(context.TODO(), nil); q.Next(); {
		names = append(names, q.Row())
	}
	r.Nil(q.Err())
	r.Equal(2, len(names))
	r.Equal("adam", names[0])
	r.Equal("eve", names[1])
}

func (suite *Suite) TestQuerierSQL() {
	r := suite.Require()
	// Check implementation by using interface as variable type.
	var q inspect.Querier[string] = &inspect.SQLQuery[string]{
		SQL:   "SELECT",
		RowTo: pgx.RowTo[string],
	}

	c := &MockConn{Rows: []string{"adam", "eve"}}
	names := make([]string, 0)
	for q.Query(context.TODO(), c); q.Next(); {
		names = append(names, q.Row())
	}
	r.Nil(q.Err())
	r.Equal(2, len(names))
	r.Equal("adam", names[0])
	r.Equal("eve", names[1])
}

// MockConn implements inspect.Conn and pgx.Rows to be usable by inspect.SQLQuery.
type MockConn struct {
	Rows []string

	currentIndex int
}

func (c *MockConn) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	c.currentIndex = -1
	return pgx.Rows(c), nil
}

func (c *MockConn) Next() bool {
	c.currentIndex++
	return c.currentIndex < len(c.Rows)
}

func (c *MockConn) Scan(dest ...any) error {
	dest0 := dest[0].(*string)
	*dest0 = c.Rows[c.currentIndex]
	return nil
}

// Unused API.
func (c *MockConn) Values() ([]any, error) {
	return nil, nil
}

func (c *MockConn) Close() {
}

func (c *MockConn) Err() error {
	return nil
}

func (c *MockConn) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("mock")
}

func (c *MockConn) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (c *MockConn) RawValues() [][]byte {
	return nil
}

func (c *MockConn) Conn() *pgx.Conn {
	return nil
}
