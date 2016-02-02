// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("diffexp", `{{ template "header" . }}

<h1>Project: <a href="/project/{{.Page.Project.Id}}">{{.Page.Project.Name}}</a></h1>
<h2>Differential expression: {{.Page.DiffExp.Name}}</h2>
<p>Created at <i>{{.Page.DiffExp.CreatedAt}}</i></p>

<table class="table table-striped">
<tr><th>Dimension</th><th>Rank difference</th></tr>
{{ $lookup := .Page.Lookup }}
{{ range .Page.Values }}
<tr><td>{{(index $lookup .DimensionId)}}</td><td>{{.Diff}}</td></tr>
{{ end }}
</table>

{{ template "footer" . }}`)
}
