// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// Package form provides ways to fill out forms corresponding to
// environschema schemas.
package form

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/juju/schema"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/errgo.v1"
	"gopkg.in/juju/environschema.v1"
)

// PromptingFiller fills out a set of environschema fields by prompting
// the user for each field in sequence.
type PromptingFiller struct {
	// The Prompter to use to get the responses. If this is nil then
	// DefaultPrompter will be used.
	Prompter Prompter

	// MaxTries is the number of times to attempt to get a valid
	// response when prompting. If this is 0 then the default of 3
	// attempts will be used.
	MaxTries int
}

// Fill processes fields by first sorting them and then prompting for the
// value of each one in turn.
//
// The fields are sorted by first by group name. Those in the same group
// are sorted so that secret fields come after non-secret ones, finally
// the fields are sorted by description.
//
// Each field will be prompted for, then the returned value will be
// validated against the field's type. If the returned value does not
// validate correctly it will be prompted again up to MaxTries before
// giving up.
func (f *PromptingFiller) Fill(fields environschema.Fields) (map[string]interface{}, error) {
	fs := make(fieldSlice, 0, len(fields))
	for k, v := range fields {
		fs = append(fs, field{
			name:  k,
			attrs: v,
		})
	}
	sort.Sort(fs)
	form := make(map[string]interface{}, len(fields))
	for _, field := range fs {
		var err error
		form[field.name], err = f.prompt(field.name, field.attrs)
		if err != nil {
			return nil, errgo.Notef(err, "cannot complete form")
		}
	}
	return form, nil
}

type field struct {
	name  string
	attrs environschema.Attr
}

type fieldSlice []field

func (s fieldSlice) Len() int {
	return len(s)
}

func (s fieldSlice) Less(i, j int) bool {
	a1 := s[i].attrs
	a2 := s[j].attrs
	if a1.Group != a2.Group {
		return a1.Group < a2.Group
	}
	if a1.Secret != a2.Secret {
		return a2.Secret
	}
	return a1.Description < a2.Description
}

func (s fieldSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (f *PromptingFiller) prompt(name string, attr environschema.Attr) (interface{}, error) {
	prompter := f.Prompter
	if prompter == nil {
		prompter = DefaultPrompter
	}
	tries := f.MaxTries
	if tries == 0 {
		tries = 3
	}
	for i := 0; i < tries; i++ {
		val, err := prompter.Prompt(name, attr)
		if err != nil {
			return nil, errgo.Notef(err, "cannot get input")
		}
		switch attr.Type {
		case environschema.Tbool:
			b, err := schema.Bool().Coerce(val, nil)
			if err == nil {
				return b, nil
			}
		case environschema.Tint:
			i, err := schema.Int().Coerce(val, nil)
			if err == nil {
				return i, nil
			}
		case environschema.Tstring:
			i, err := schema.String().Coerce(val, nil)
			if err == nil {
				return i, nil
			}
		default:
			return nil, errgo.Newf("unsupported attribute type %q", attr.Type)
		}
	}
	return nil, errgo.New("too many invalid inputs")
}

// Prompt prompts for the value of the given named attribute.
// It is always acceptable for an implementation to return a string,
// which will be coerced to the appropriate type if possible.
type Prompter interface {
	Prompt(name string, attr environschema.Attr) (interface{}, error)
}

// DefaultPrompter is the default Prompter used by a PromptingFiller when
// Prompter has not been set.
var DefaultPrompter Prompter = IOPrompter{
	In:  os.Stdin,
	Out: os.Stderr,
}

// IOPrompter is a Prompter based around an io.Reader and io.Writer.
// It uses Out to print prompts to and reads responses from In.
type IOPrompter struct {
	In  io.Reader
	Out io.Writer
}

// Prompt implements Prompter. The prompt is generated from attr based on
// Description field. DefaultFromEnv is used to attempt to find a default
// value for the attribute. If attr.Secret is true then any default value
// will be obscured with *s and if In is a terminal local echo will be
// turned off. Input is read from In until the first \n. If the entered
// value is "" and there is a default then the default will be returned.
func (p IOPrompter) Prompt(name string, attr environschema.Attr) (interface{}, error) {
	prompt := attr.Description
	def := DefaultFromEnv(attr)
	def1 := def
	if def1 != "" {
		if attr.Secret {
			def1 = strings.Repeat("*", len(def))
		}
		prompt = fmt.Sprintf("%s (%s)", attr.Description, def1)
	}
	_, err := fmt.Fprintf(p.Out, "%s: ", prompt)
	if err != nil {
		return "", errgo.Notef(err, "cannot write prompt")
	}
	input, err := readLine(p.Out, p.In, attr.Secret)
	if err != nil {
		return "", errgo.Notef(err, "cannot read input")
	}
	if len(input) == 0 {
		return def, nil
	}
	return string(input), nil
}

func readLine(w io.Writer, r io.Reader, secret bool) ([]byte, error) {
	if f, ok := r.(*os.File); ok && secret && terminal.IsTerminal(int(f.Fd())) {
		defer w.Write([]byte{'\n'})
		return terminal.ReadPassword(int(f.Fd()))
	}
	var input []byte
	for {
		var buf [1]byte
		n, err := r.Read(buf[:])
		if n == 1 {
			if buf[0] == '\n' {
				break
			}
			input = append(input, buf[0])
		}
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, errgo.Mask(err)
		}
	}
	if len(input) > 0 && input[len(input)-1] == '\r' {
		input = input[:len(input)-1]
	}
	return input, nil
}

// DefaultFromEnv atttempts to derive a default value from environment
// variables. The environment varialbles specified in attr will be
// checked in order and the first non-empty value found is returned. If
// no non-empty values are found then "" is returned.
func DefaultFromEnv(attr environschema.Attr) string {
	if attr.EnvVar != "" {
		if env := os.Getenv(attr.EnvVar); env != "" {
			return env
		}
	}
	for _, envVar := range attr.EnvVars {
		if env := os.Getenv(envVar); env != "" {
			return env
		}
	}
	return ""
}
