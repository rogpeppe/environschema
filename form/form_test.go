// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package form_test

import (
	"bytes"
	"strings"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/errgo.v1"

	"gopkg.in/juju/environschema.v1"
	"gopkg.in/juju/environschema.v1/form"
)

type formSuite struct {
	testing.OsEnvSuite
}

var _ = gc.Suite(&formSuite{})

var _ form.Filler = form.IOFiller{}

var ioFillerTests = []struct {
	about        string
	form         form.Form
	maxTries     int
	environment  map[string]string
	expectIO     string
	expectResult map[string]interface{}
	expectError  string
}{{
	about: "single field no default",
	form: form.Form{
		Fields: environschema.Fields{
			"A": environschema.Attr{
				Type:        environschema.Tstring,
				Description: "A",
			},
		},
	},
	expectIO: `
	|A: »B
	`,
	expectResult: map[string]interface{}{
		"A": "B",
	},
}, {
	about: "single field with default",
	form: form.Form{
		Fields: environschema.Fields{
			"A": environschema.Attr{
				Type:        environschema.Tstring,
				Description: "A",
				EnvVar:      "A",
			},
		},
	},
	environment: map[string]string{
		"A": "C",
	},
	expectIO: `
	|A [C]: »B
	`,
	expectResult: map[string]interface{}{
		"A": "B",
	},
}, {
	about: "single field with default no input",
	form: form.Form{
		Fields: environschema.Fields{
			"A": environschema.Attr{
				Type:        environschema.Tstring,
				Description: "A",
				EnvVar:      "A",
			},
		},
	},
	environment: map[string]string{
		"A": "C",
	},
	expectIO: `
	|A [C]: »
	`,
	expectResult: map[string]interface{}{
		"A": "C",
	},
}, {
	about: "secret single field with default no input",
	form: form.Form{
		Fields: environschema.Fields{
			"A": environschema.Attr{
				Type:        environschema.Tstring,
				Description: "A",
				EnvVar:      "A",
				Secret:      true,
			},
		},
	},
	environment: map[string]string{
		"A": "password",
	},
	expectIO: `
	|A [********]: »
	`,
	expectResult: map[string]interface{}{
		"A": "password",
	},
}, {
	about: "windows line endings",
	form: form.Form{
		Fields: environschema.Fields{
			"A": environschema.Attr{
				Type:        environschema.Tstring,
				Description: "A",
			},
		},
	},
	expectIO: `
	|A: »B` + "\r" + `
	`,
	expectResult: map[string]interface{}{
		"A": "B",
	},
}, {
	about: "with title",
	form: form.Form{
		Title: "Test Title",
	},
	expectIO: `
	|Test Title
	`,
	expectResult: map[string]interface{}{},
}, {
	about: "title with prompts",
	form: form.Form{
		Title: "Test Title",
		Fields: environschema.Fields{
			"A": environschema.Attr{
				Type:        environschema.Tstring,
				Description: "A",
			},
		},
	},
	expectIO: `
	|Test Title
	|A: »B
	`,
	expectResult: map[string]interface{}{
		"A": "B",
	},
}, {
	about: "correct ordering",
	form: form.Form{
		Fields: environschema.Fields{
			"a1": environschema.Attr{
				Group:       "A",
				Description: "a1",
				Type:        environschema.Tstring,
			},
			"c1": environschema.Attr{
				Group:       "A",
				Description: "c1",
				Type:        environschema.Tstring,
			},
			"b1": environschema.Attr{
				Group:       "A",
				Description: "b1",
				Type:        environschema.Tstring,
				Secret:      true,
			},
			"a2": environschema.Attr{
				Group:       "B",
				Description: "a2",
				Type:        environschema.Tstring,
			},
			"c2": environschema.Attr{
				Group:       "B",
				Description: "c2",
				Type:        environschema.Tstring,
			},
			"b2": environschema.Attr{
				Group:       "B",
				Description: "b2",
				Type:        environschema.Tstring,
				Secret:      true,
			},
		},
	},
	expectIO: `
	|a1: »a1
	|c1: »c1
	|b1: »b1
	|a2: »a2
	|c2: »c2
	|b2: »b2
	`,
	expectResult: map[string]interface{}{
		"a1": "a1",
		"b1": "b1",
		"c1": "c1",
		"a2": "a2",
		"b2": "b2",
		"c2": "c2",
	},
}, {
	about: "string type",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tstring,
			},
			"b": environschema.Attr{
				Description: "b",
				Type:        environschema.Tstring,
				Mandatory:   true,
			},
			"c": environschema.Attr{
				Description: "c",
				Type:        environschema.Tstring,
			},
		},
	},
	expectIO: `
	|a: »
	|b: »
	|c: »something
	`,
	expectResult: map[string]interface{}{
		"b": "",
		"c": "something",
	},
}, {
	about: "bool type",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tbool,
			},
			"b": environschema.Attr{
				Description: "b",
				Type:        environschema.Tbool,
			},
			"c": environschema.Attr{
				Description: "c",
				Type:        environschema.Tbool,
			},
			"d": environschema.Attr{
				Description: "d",
				Type:        environschema.Tbool,
			},
		},
	},
	expectIO: `
	|a: »true
	|b: »false
	|c: »1
	|d: »0
	`,
	expectResult: map[string]interface{}{
		"a": true,
		"b": false,
		"c": true,
		"d": false,
	},
}, {
	about: "int type",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tint,
			},
			"b": environschema.Attr{
				Description: "b",
				Type:        environschema.Tint,
			},
			"c": environschema.Attr{
				Description: "c",
				Type:        environschema.Tint,
			},
		},
	},
	expectIO: `
	|a: »0
	|b: »-1000000
	|c: »1000000
	`,
	expectResult: map[string]interface{}{
		"a": 0,
		"b": -1000000,
		"c": 1000000,
	},
}, {
	about: "attrs type",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tattrs,
			},
			"b": environschema.Attr{
				Description: "b",
				Type:        environschema.Tattrs,
			},
		},
	},
	expectIO: `
	|a: »x=y z= foo=bar
	|b: »
	`,
	expectResult: map[string]interface{}{
		"a": map[string]string{
			"x":   "y",
			"foo": "bar",
			"z":   "",
		},
	},
}, {
	about: "too many bad responses",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tint,
				Mandatory:   true,
			},
		},
	},
	expectIO: `
	|a: »one
	|invalid input: expected number, got string("one")
	|a: »
	|invalid input: expected number, got string("")
	|a: »three
	|invalid input: expected number, got string("three")
	`,
	expectError: `cannot complete form: too many invalid inputs`,
}, {
	about: "too many bad responses with maxtries=1",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tint,
			},
		},
	},
	maxTries: 1,
	expectIO: `
	|a: »one
	|invalid input: expected number, got string("one")
	`,
	expectError: `cannot complete form: too many invalid inputs`,
}, {
	about: "bad then good input",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tint,
			},
		},
	},
	expectIO: `
	|a: »one
	|invalid input: expected number, got string("one")
	|a: »two
	|invalid input: expected number, got string("two")
	|a: »3
	`,
	expectResult: map[string]interface{}{
		"a": 3,
	},
}, {
	about: "empty value entered for optional attribute with no default",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tstring,
			},
			"b": environschema.Attr{
				Description: "b",
				Type:        environschema.Tint,
			},
			"c": environschema.Attr{
				Description: "c",
				Type:        environschema.Tbool,
			},
		},
	},
	expectIO: `
	|a: »
	|b: »
	|c: »
	`,
	expectResult: map[string]interface{}{},
}, {
	about: "unsupported type",
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        "bogus",
			},
		},
	},
	expectError: `invalid field a: invalid type "bogus"`,
}, {
	about: "no interaction is done if any field has an invalid type",
	form: form.Form{
		Title: "some title",
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tstring,
			},
			"b": environschema.Attr{
				Description: "b",
				Type:        "bogus",
			},
		},
	},
	expectError: `invalid field b: invalid type "bogus"`,
}, {
	about: "invalid default value is ignored",
	environment: map[string]string{
		"a": "three",
	},
	expectIO: `
	|warning: invalid default value in $a
	|a: »99
	`,
	form: form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tint,
				EnvVars:     []string{"a"},
			},
		},
	},
	expectResult: map[string]interface{}{
		"a": 99,
	},
}}

func (s *formSuite) TestIOFiller(c *gc.C) {
	for i, test := range ioFillerTests {
		func() {
			c.Logf("%d. %s", i, test.about)
			for k, v := range test.environment {
				defer testing.PatchEnvironment(k, v)()
			}
			ioChecker := newInteractionChecker(c, "»", strings.TrimPrefix(unbeautify(test.expectIO), "\n"))
			ioFiller := form.IOFiller{
				In:       ioChecker,
				Out:      ioChecker,
				MaxTries: test.maxTries,
			}
			result, err := ioFiller.Fill(test.form)
			if test.expectError != "" {
				c.Assert(err, gc.ErrorMatches, test.expectError)
				c.Assert(result, gc.IsNil)
			} else {
				ioChecker.Close()
				c.Assert(err, gc.IsNil)
				c.Assert(result, jc.DeepEquals, test.expectResult)
			}
		}()
	}
}

func (s *formSuite) TestIOFillerReadError(c *gc.C) {
	r := errorReader{}
	var out bytes.Buffer
	ioFiller := form.IOFiller{
		In:  r,
		Out: &out,
	}
	result, err := ioFiller.Fill(form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tstring,
			},
		},
	})
	c.Check(out.String(), gc.Equals, "a: ")
	c.Assert(err, gc.ErrorMatches, `cannot complete form: cannot read input: some read error`)
	c.Assert(result, gc.IsNil)
	// Verify that the cause is masked. Maybe it shouldn't
	// be, but test the code as it is.
	c.Assert(errgo.Cause(err), gc.Not(gc.Equals), errRead)
}

func (s *formSuite) TestIOFillerUnexpectedEOF(c *gc.C) {
	r := strings.NewReader("a")
	var out bytes.Buffer
	ioFiller := form.IOFiller{
		In:  r,
		Out: &out,
	}
	result, err := ioFiller.Fill(form.Form{
		Fields: environschema.Fields{
			"a": environschema.Attr{
				Description: "a",
				Type:        environschema.Tstring,
			},
		},
	})
	c.Check(out.String(), gc.Equals, "a: ")
	c.Assert(err, gc.ErrorMatches, `cannot complete form: cannot read input: unexpected EOF`)
	c.Assert(result, gc.IsNil)
}

func (s *formSuite) TestSortedFields(c *gc.C) {
	fields := environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tstring,
		},
		"c1": environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tstring,
		},
		"b1": environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tstring,
			Secret:      true,
		},
		"a2": environschema.Attr{
			Group:       "B",
			Description: "a2",
			Type:        environschema.Tstring,
		},
		"c2": environschema.Attr{
			Group:       "B",
			Description: "c2",
			Type:        environschema.Tstring,
		},
		"b2": environschema.Attr{
			Group:       "B",
			Description: "b2",
			Type:        environschema.Tstring,
			Secret:      true,
		},
	}
	c.Assert(form.SortedFields(fields), jc.DeepEquals, []form.NamedAttr{{
		Name: "a1",
		Attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tstring,
		}}, {
		Name: "c1",
		Attr: environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tstring,
		}}, {
		Name: "b1",
		Attr: environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tstring,
			Secret:      true,
		}}, {
		Name: "a2",
		Attr: environschema.Attr{
			Group:       "B",
			Description: "a2",
			Type:        environschema.Tstring,
		}}, {
		Name: "c2",
		Attr: environschema.Attr{
			Group:       "B",
			Description: "c2",
			Type:        environschema.Tstring,
		}}, {
		Name: "b2",
		Attr: environschema.Attr{
			Group:       "B",
			Description: "b2",
			Type:        environschema.Tstring,
			Secret:      true,
		},
	}})
}

var errRead = errgo.New("some read error")

type errorReader struct{}

func (r errorReader) Read([]byte) (int, error) {
	return 0, errRead
}

var defaultFromEnvTests = []struct {
	about       string
	environment map[string]string
	attr        environschema.Attr
	expect      string
	expectVar   string
}{{
	about: "no envvars",
}, {
	about: "matching envvar",
	environment: map[string]string{
		"A": "B",
	},
	attr: environschema.Attr{
		EnvVar: "A",
	},
	expect:    "B",
	expectVar: "A",
}, {
	about: "matching envvars",
	environment: map[string]string{
		"B": "C",
	},
	attr: environschema.Attr{
		EnvVar:  "A",
		EnvVars: []string{"B"},
	},
	expect:    "C",
	expectVar: "B",
}, {
	about: "envar takes priority",
	environment: map[string]string{
		"A": "1",
		"B": "2",
	},
	attr: environschema.Attr{
		EnvVar:  "A",
		EnvVars: []string{"B"},
	},
	expect:    "1",
	expectVar: "A",
}}

func (s *formSuite) TestDefaultFromEnv(c *gc.C) {
	for i, test := range defaultFromEnvTests {
		c.Logf("%d. %s", i, test.about)
		func() {
			for k, v := range test.environment {
				defer testing.PatchEnvironment(k, v)()
			}
			result, resultVar := form.DefaultFromEnv(test.attr)
			c.Assert(result, gc.Equals, test.expect)
			c.Assert(resultVar, gc.Equals, test.expectVar)
		}()
	}
}

// indentReplacer deletes tabs and | beautifier characters.
var indentReplacer = strings.NewReplacer("\t", "", "|", "")

// unbeautify strips the leading tabs and | characters that
// we use to make the tests look nicer.
func unbeautify(s string) string {
	return indentReplacer.Replace(s)
}
