// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("results", `{{ template "header" . }}

<h1>Project: <a href="/project/{{.Page.Project.Id}}">{{.Page.Project.Name}}</a></h1>

<h2>Search results</h2>

<table class="table table-striped">
<tr><th>Sample</th><th>Score</th></tr>
{{ $page := .Page }}
{{ range .Page.Results }}
<tr><td>
  <a href="/project/{{$page.Project.Id}}/sample/{{.Id}}">{{.Name}}</a>
</td><td>{{.Score}}</td></tr>
{{ end }}
</table>

{{ template "footer" . }}`)
}
