package pyfmt_test

import "github.com/dalibo/ldap2pg/internal/pyfmt"

func (suite *Suite) TestParseLiteralOnly() {
	r := suite.Require()
	f, err := pyfmt.Parse("toto")
	r.Nil(err)
	r.Equal(0, len(f.Fields))
	r.Equal(1, len(f.Sections))
	r.Equal("toto", f.Sections[0])
}

func (suite *Suite) TestParseMethod() {
	r := suite.Require()
	f, err := pyfmt.Parse("{member.cn.lower()}")
	r.Nil(err)
	r.Equal(1, len(f.Fields))
	r.Equal(1, len(f.Sections))
	r.Equal("member.cn", f.Fields[0].FieldName)
	r.Equal("lower()", f.Fields[0].Method)
}

func (suite *Suite) TestParseFieldOnly() {
	r := suite.Require()
	f, err := pyfmt.Parse("{member.cn}")
	r.Nil(err)
	r.Equal(1, len(f.Fields))
	r.Equal(1, len(f.Sections))
	r.Equal("member.cn", f.Fields[0].FieldName)
}

func (suite *Suite) TestParseCombination() {
	r := suite.Require()

	f, err := pyfmt.Parse("ext_{member.cn}")
	r.Nil(err)
	r.Equal(2, len(f.Sections))
	r.Equal("ext_", f.Sections[0])
	r.Equal("member.cn", f.Fields[0].FieldName)
}

func (suite *Suite) TestParseEscaped() {
	r := suite.Require()
	f, err := pyfmt.Parse("literal {{toto}} pouet")
	r.Nil(err)
	r.Equal(2, len(f.Sections))
	r.Equal(0, len(f.Fields))
	r.Equal("literal {", f.Sections[0])
	r.Equal("toto} pouet", f.Sections[1])
}

func (suite *Suite) TestParseUnterminatedField() {
	r := suite.Require()
	_, err := pyfmt.Parse("literal{unterminated_field")
	r.Error(err)
}

func (suite *Suite) TestParseConversion() {
	r := suite.Require()
	f, err := pyfmt.Parse("{!r}")
	r.Nil(err)
	r.Equal(1, len(f.Fields))
	r.Equal("", f.Fields[0].FieldName)
	r.Equal("r", f.Fields[0].Conversion)
}

func (suite *Suite) TestParseSpec() {
	r := suite.Require()
	f, err := pyfmt.Parse("{:>30}")
	r.Nil(err)
	r.Equal(1, len(f.Fields))
	r.Equal(&pyfmt.Field{FieldName: "", Conversion: "", FormatSpec: ">30"}, f.Fields[0])
}

func (suite *Suite) TestParseConversionAndSpec() {
	r := suite.Require()

	f, err := pyfmt.Parse("{0!r:>30}")
	r.Nil(err)
	r.Equal(1, len(f.Fields))
	r.Equal(&pyfmt.Field{FieldName: "0", Conversion: "r", FormatSpec: ">30"}, f.Fields[0])
}

func (suite *Suite) TestFormat() {
	r := suite.Require()

	f, err := pyfmt.Parse("ext_{dn.cn}_{member.cn.upper()}")
	r.Nil(err)

	s := f.Format(map[string]string{
		"dn.cn":     "dba",
		"member.cn": "alice",
	})
	r.Equal("ext_dba_ALICE", s)
}
