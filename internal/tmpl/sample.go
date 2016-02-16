// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("sample", `{{ template "header" . }}

<h1>Project: <a href="/project/{{.Page.Project.Id}}">{{.Page.Project.Name}}</a></h1>
<h2>Sample: {{.Page.Sample.Name}}</h2>
<p>Created at <i>{{.Page.Sample.CreatedAt.Format "Jan 02, 2006 15:04 MST"}}</i></p>

<ul class="nav nav-tabs">
  <li role="presentation" class="active">
    <a>Data</a>
  </li>
  <li role="presentation">
    <a href="/project/{{.Page.Project.Id}}/sample/{{.Page.Sample.Id}}/similar">Similar Samples</a>
  </li>
</ul>

<div class="panel panel-default">
  <div class="panel-body">

<table class="table table-striped">
<tr>
  <th>Dimension</th>
  <th>Rank</th>
  <th>Rank difference</th>
  <th>Value</th>
  <th>Value difference</th>
</tr>
{{ $lookup := .Page.Lookup }}
{{ range .Page.Values }}
<tr>
  <td>{{($lookup.LookupName .DimensionId)}}</td>
  <td>{{.Rank}}</td>
  <td>{{.RankDiff}}</td>
  <td>{{.Value}}</td>
  <td>{{.ValueDiff}}</td>
</tr>
{{ end }}
</table>

  </div>
</div>

{{ template "footer" . }}`)
}
