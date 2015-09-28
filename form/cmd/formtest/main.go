// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/juju/environschema.v1"
	"gopkg.in/juju/environschema.v1/form"
)

func main() {
	f := &form.PromptingFiller{}
	fmt.Println(`formtest:
This is a simple interactive test program for environschema forms.
Expect the prompts to be as follows:

Name:
e-mail (user@example.com):
PIN (****):
Password:

The entered values will be displayed at the end.
`)
	os.Setenv("PIN", "1234")
	os.Setenv("EMAIL", "user@example.com")
	r, err := f.Fill(environschema.Fields{
		"name": environschema.Attr{
			Description: "Name",
			Type:        environschema.Tstring,
		},
		"email": environschema.Attr{
			Description: "e-mail",
			Type:        environschema.Tstring,
			EnvVar:      "EMAIL",
		},
		"password": environschema.Attr{
			Description: "Password",
			Type:        environschema.Tstring,
			Secret:      true,
		},
		"pin": environschema.Attr{
			Description: "PIN",
			Type:        environschema.Tint,
			EnvVar:      "PIN",
			Secret:      true,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}
