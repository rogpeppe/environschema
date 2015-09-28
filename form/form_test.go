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

type prompt struct {
	name string
	attr environschema.Attr
}

var ioPrompterTests = []struct {
	about        string
	prompts      []prompt
	input        string
	environment  map[string]string
	expect       []string
	expectOutput string
	expectError  string
}{{
	about: "single field no default",
	prompts: []prompt{{
		attr: environschema.Attr{
			Description: "A",
		},
	}},
	input: "B\n",
	expect: []string{
		"B",
	},
	expectOutput: "A: ",
}, {
	about: "single field with default",
	prompts: []prompt{{
		attr: environschema.Attr{
			Description: "A",
			EnvVar:      "A",
		},
	}},
	environment: map[string]string{
		"A": "C",
	},
	input: "B\n",
	expect: []string{
		"B",
	},
	expectOutput: "A (C): ",
}, {
	about: "single field with default no input",
	prompts: []prompt{{
		attr: environschema.Attr{
			Description: "A",
			EnvVar:      "A",
		},
	}},
	environment: map[string]string{
		"A": "C",
	},
	input: "\n",
	expect: []string{
		"C",
	},
	expectOutput: "A (C): ",
}, {
	about: "secret single field with default no input",
	prompts: []prompt{{
		attr: environschema.Attr{
			Description: "A",
			EnvVar:      "A",
			Secret:      true,
		},
	}},
	environment: map[string]string{
		"A": "C",
	},
	input: "\n",
	expect: []string{
		"C",
	},
	expectOutput: "A (*): ",
}, {
	about: "input error",
	prompts: []prompt{{
		attr: environschema.Attr{
			Description: "A",
		},
	}},
	input:        "",
	expectOutput: "A: ",
	expectError:  "cannot read input: unexpected EOF",
}, {
	about: "windows line endings",
	prompts: []prompt{{
		attr: environschema.Attr{
			Description: "A",
		},
	}},
	input: "A\r\n",
	expect: []string{
		"A",
	},
	expectOutput: "A: ",
}}

func (s *formSuite) TestIOPrompter(c *gc.C) {
	for i, test := range ioPrompterTests {
		func() {
			c.Logf("%d. %s", i, test.about)
			for k, v := range test.environment {
				defer testing.PatchEnvironment(k, v)()
			}
			outBuf := new(bytes.Buffer)
			prompter := form.IOPrompter{
				In:  strings.NewReader(test.input),
				Out: outBuf,
			}
			for j, p := range test.prompts {
				result, err := prompter.Prompt(p.name, p.attr)
				if err != nil {
					c.Assert(err, gc.ErrorMatches, test.expectError)
					c.Assert(outBuf.String(), gc.Equals, test.expectOutput)
					return
				}
				c.Assert(result, gc.Equals, test.expect[j])
			}
			c.Assert(test.expectError, gc.Equals, "", gc.Commentf("did not reveive expected error"))
			c.Assert(outBuf.String(), gc.Equals, test.expectOutput)
		}()
	}
}

var promptingFillerTests = []struct {
	about         string
	schema        environschema.Fields
	maxTries      int
	responses     []response
	expectPrompts []prompt
	expectResult  map[string]interface{}
	expectError   string
}{{
	about: "correct ordering",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tstring,
		},
		"b1": environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tstring,
			Secret:      true,
		},
		"c1": environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tstring,
		},
		"a2": environschema.Attr{
			Group:       "B",
			Description: "a2",
			Type:        environschema.Tstring,
		},
		"b2": environschema.Attr{
			Group:       "B",
			Description: "b2",
			Type:        environschema.Tstring,
			Secret:      true,
		},
		"c2": environschema.Attr{
			Group:       "B",
			Description: "c2",
			Type:        environschema.Tstring,
		},
	},
	responses: []response{{
		data: "a1",
	}, {
		data: "c1",
	}, {
		data: "b1",
	}, {
		data: "a2",
	}, {
		data: "c2",
	}, {
		data: "b2",
	}},
	expectResult: map[string]interface{}{
		"a1": "a1",
		"b1": "b1",
		"c1": "c1",
		"a2": "a2",
		"b2": "b2",
		"c2": "c2",
	},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tstring,
		},
	}, {
		name: "c1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tstring,
		},
	}, {
		name: "b1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tstring,
			Secret:      true,
		},
	}, {
		name: "a2",
		attr: environschema.Attr{
			Group:       "B",
			Description: "a2",
			Type:        environschema.Tstring,
		},
	}, {
		name: "c2",
		attr: environschema.Attr{
			Group:       "B",
			Description: "c2",
			Type:        environschema.Tstring,
		},
	}, {
		name: "b2",
		attr: environschema.Attr{
			Group:       "B",
			Description: "b2",
			Type:        environschema.Tstring,
			Secret:      true,
		},
	}},
}, {
	about: "bool type",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tbool,
		},
		"b1": environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tbool,
		},
	},
	responses: []response{{
		data: "true",
	}, {
		data: "false",
	}},
	expectResult: map[string]interface{}{
		"a1": true,
		"b1": false,
	},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tbool,
		},
	}, {
		name: "b1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tbool,
		},
	}},
}, {
	about: "int type",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
		"b1": environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tint,
		},
		"c1": environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tint,
		},
	},
	responses: []response{{
		data: "0",
	}, {
		data: "-1000000",
	}, {
		data: "1000000",
	}},
	expectResult: map[string]interface{}{
		"a1": int64(0),
		"b1": int64(-1000000),
		"c1": int64(1000000),
	},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}, {
		name: "b1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tint,
		},
	}, {
		name: "c1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tint,
		},
	}},
}, {
	about: "too many bad responses",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	},
	responses: []response{{
		data: "one",
	}, {
		data: "two",
	}, {
		data: "three",
	}},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}, {
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}, {
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}},
	expectError: "cannot complete form: too many invalid inputs",
}, {
	about: "too many bad responses fewer tries",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	},
	maxTries: 1,
	responses: []response{{
		data: "one",
	}},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}},
	expectError: "cannot complete form: too many invalid inputs",
}, {
	about: "bad then good input",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	},
	responses: []response{{
		data: "one",
	}, {
		data: "two",
	}, {
		data: "3",
	}},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}, {
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}, {
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}},
	expectResult: map[string]interface{}{
		"a1": int64(3),
	},
}, {
	about: "prompt error",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tstring,
		},
	},
	responses: []response{{
		err: errgo.New("test error"),
	}},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tstring,
		},
	}},
	expectError: "cannot complete form: cannot get input: test error",
}, {
	about: "unsupported type",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tattrs,
		},
	},
	responses: []response{{
		data: "A",
	}},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tattrs,
		},
	}},
	expectError: `cannot complete form: unsupported attribute type "attrs"`,
}, {
	about: "bool type with bool from prompt",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tbool,
		},
		"b1": environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tbool,
		},
	},
	responses: []response{{
		data: true,
	}, {
		data: false,
	}},
	expectResult: map[string]interface{}{
		"a1": true,
		"b1": false,
	},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tbool,
		},
	}, {
		name: "b1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tbool,
		},
	}},
}, {
	about: "int type with int from prompt",
	schema: environschema.Fields{
		"a1": environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
		"b1": environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tint,
		},
		"c1": environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tint,
		},
	},
	responses: []response{{
		data: 0,
	}, {
		data: -1000000,
	}, {
		data: 1000000,
	}},
	expectResult: map[string]interface{}{
		"a1": int64(0),
		"b1": int64(-1000000),
		"c1": int64(1000000),
	},
	expectPrompts: []prompt{{
		name: "a1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "a1",
			Type:        environschema.Tint,
		},
	}, {
		name: "b1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "b1",
			Type:        environschema.Tint,
		},
	}, {
		name: "c1",
		attr: environschema.Attr{
			Group:       "A",
			Description: "c1",
			Type:        environschema.Tint,
		},
	}},
}}

func (s *formSuite) TestPromptingFiller(c *gc.C) {
	for i, test := range promptingFillerTests {
		c.Logf("%d. %s", i, test.about)
		p := &testPrompter{
			responses: test.responses,
		}
		f := form.PromptingFiller{
			Prompter: p,
			MaxTries: test.maxTries,
		}
		result, err := f.Fill(test.schema)
		if test.expectError != "" {
			c.Assert(err, gc.ErrorMatches, test.expectError)
		} else {
			c.Assert(err, gc.IsNil)
		}
		c.Assert(result, jc.DeepEquals, test.expectResult)
		c.Assert(p.prompts, jc.DeepEquals, test.expectPrompts)
	}
}

type response struct {
	data interface{}
	err  error
}

type testPrompter struct {
	prompts   []prompt
	responses []response
}

func (p *testPrompter) Prompt(name string, attr environschema.Attr) (interface{}, error) {
	i := len(p.prompts)
	r := p.responses[i]
	p.prompts = append(p.prompts, prompt{
		name: name,
		attr: attr,
	})
	return r.data, r.err
}

var defaultFromEnvTests = []struct {
	about       string
	environment map[string]string
	attr        environschema.Attr
	expect      string
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
	expect: "B",
}, {
	about: "matching envvars",
	environment: map[string]string{
		"B": "C",
	},
	attr: environschema.Attr{
		EnvVar:  "A",
		EnvVars: []string{"B"},
	},
	expect: "C",
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
	expect: "1",
}}

func (s *formSuite) TestDefaultFromEnv(c *gc.C) {
	for i, test := range defaultFromEnvTests {
		c.Logf("%d. %s", i, test.about)
		func() {
			for k, v := range test.environment {
				defer testing.PatchEnvironment(k, v)()
			}
			result := form.DefaultFromEnv(test.attr)
			c.Assert(result, gc.Equals, test.expect)
		}()
	}
}
