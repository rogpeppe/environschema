// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// Package environschema implements a way to specify
// configuration attributes for Juju environments.
package environschema // import "gopkg.in/juju/environschema.v1"

import (
	"fmt"
	"github.com/juju/schema"
	"strings"
)

// What to do about reading content from paths?
// Could just have a load of client-side special cases.

// Fields holds a map from attribute name to
// information about that attribute.
type Fields map[string]Attr

type Attr struct {
	// Description holds a human-readable description
	// of the attribute.
	Description string `json:"description"`

	// Type holds the type of the attribute value.
	Type FieldType `json:"type"`

	// Mandatory specifies whether the attribute
	// must be specified. If Default is also specified,
	// this field is redundant, as the attribute will be
	// filled in with the default value if missing.
	Mandatory bool `json:"mandatory,omitempty"`

	// Secret specifies whether the attribute should be
	// considered secret.
	Secret bool `json:"is-secret,omitempty"`

	// EnvVar holds the environment variable
	// that will be used to obtain the default value
	// if it isn't specified.
	EnvVar string `json:"env-var,omitempty"`

	// Default holds the default value of the attribute
	// when not specified.
	Default interface{} `json:"default,omitempty"`

	// Example holds an example value for the attribute
	// that can be used to produce a plausible-looking
	// entry for the attribute without necessarily using
	// it as a default value.
	Example interface{} `json:"example,omitempty"`

	// Values holds the set of all possible values of the attribute.
	Values []interface{} `json:"values,omitempty"`
}

// FieldType describes the type of an attribute value.
type FieldType string

// The following constants are the possible type values.
const (
	Tstring FieldType = "string"
	Tbool   FieldType = "bool"
	Tint    FieldType = "int"
)

var checkers = map[FieldType]schema.Checker{
	Tstring: schema.String(),
	Tbool:   schema.Bool(),
	Tint:    schema.ForceInt(),
}

// Alternative possibilities to ValidationSchema to bear in mind for
// the future:
// func (s Fields) Checker() schema.Checker
// func (s Fields) Validate(value map[string]interface{}) (v map[string] interface{}, extra []string, err error)

// ValidationSchema returns values suitable for passing to
// schema.FieldMap to create a schema.Checker that will validate the given fields.
// It will return an error if the fields are invalid.
func (s Fields) ValidationSchema() (schema.Fields, schema.Defaults, error) {
	fields := make(schema.Fields)
	defaults := make(schema.Defaults)
	for name, attr := range s {
		path := []string{name}
		checker := checkers[attr.Type]
		if checker == nil {
			return nil, nil, fmt.Errorf("%sinvalid type %q", pathPrefix(path), attr.Type)
		}
		if attr.Values != nil {
			var err error
			checker, err = oneOfValues(checker, attr.Values, path)
			if err != nil {
				return nil, nil, err
			}
		}
		fields[name] = checker
		if attr.Default == nil {
			if !attr.Mandatory {
				defaults[name] = schema.Omit
			}
			continue
		}
		var err error
		defaults[name], err = checker.Coerce(attr.Default, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("%sinvalid default value: %v", pathPrefix(path), err.Error())
		}
	}
	return fields, defaults, nil
}

// oneOfValues returns a checker that coerces its value
// using the supplied checker, then checks that the
// resulting value is equal to one of the given values.
func oneOfValues(checker schema.Checker, values []interface{}, path []string) (schema.Checker, error) {
	cvalues := make([]interface{}, len(values))
	for i, v := range values {
		cv, err := checker.Coerce(v, nil)
		if err != nil {
			return nil, fmt.Errorf("%sinvalid enumerated value: %v", pathPrefix(path), err)
		}
		cvalues[i] = cv
	}
	return oneOfValuesC{
		vals:    cvalues,
		checker: checker,
	}, nil
}

type oneOfValuesC struct {
	vals    []interface{}
	checker schema.Checker
}

// Coerce implements schema.Checker.Coerce.
func (c oneOfValuesC) Coerce(v interface{}, path []string) (interface{}, error) {
	v, err := c.checker.Coerce(v, path)
	if err != nil {
		return v, err
	}
	for _, allow := range c.vals {
		if allow == v {
			return v, nil
		}
	}
	return nil, fmt.Errorf("%sexpected one of %v, got %v", pathPrefix(path), c.vals, v)
}

// pathPrefix returns an error message prefix holding
// the concatenation of the path elements. If path
// starts with a ".", the dot is omitted.
func pathPrefix(path []string) string {
	if len(path) == 0 {
		return ""
	}
	if path[0] == "." {
		return strings.Join(path[1:], "") + ": "
	}
	return strings.Join(path, "") + ": "
}

// ExampleYAML returns the fields formatted as a YAML
// example, with non-mandatory fields commented out,
// like the providers do currently.
func (s Fields) ExampleYAML() []byte {
	panic("unimplemented")
}
