// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("results", `{{ template "header" . }}

<h1>Project: <a href="/project/{{.Page.Project.Id}}">{{.Page.Project.Name}}</a></h1>

<p>Not yet implemented</p>

{{ template "footer" . }}`)
}
