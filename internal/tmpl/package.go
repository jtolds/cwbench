// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

import (
	"html/template"
)

var (
	Templates = template.New("root").Funcs(
		template.FuncMap{
			"makepair": makePair,
			"safeURL":  func(val string) template.URL { return template.URL(val) }})
)

type Pair struct {
	First, Second interface{}
}

func makePair(first, second interface{}) Pair {
	return Pair{First: first, Second: second}
}

func register(name, tmpl string) {
	template.Must(Templates.New(name).Parse(tmpl))
}
