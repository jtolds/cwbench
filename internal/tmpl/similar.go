// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("similar", `{{ template "header" . }}

<h1>Project: <a href="/project/{{.Page.Project.Id}}">{{.Page.Project.Name}}</a></h1>
<h2>Sample: {{.Page.Sample.Name}}</h2>
<p>Created at <i>{{.Page.Sample.CreatedAt.Format "Jan 02, 2006 15:04 MST"}}</i></p>

<ul class="nav nav-tabs">
  <li role="presentation">
    <a href="/project/{{.Page.Project.Id}}/sample/{{.Page.Sample.Id}}">Data</a>
  </li>
  <li role="presentation" class="active">
    <a>Similar Samples</a>
  </li>
</ul>

<div class="panel panel-default">
  <div class="panel-body">

  <table class="table table-striped">
  <tr><th>Sample</th><th>Score</th></tr>
  {{ $page := .Page }}
  {{ range .Page.Results }}
  <tr><td>
    <a href="/project/{{$page.Project.Id}}/sample/{{.Id}}/similar?{{safeURL $page.Params}}">{{.Name}}</a>
  </td><td>{{.Score}}</td></tr>
  {{ end }}
  </table>

  </div>
</div>

{{ template "footer" . }}`)
}
