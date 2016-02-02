// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("control", `{{ template "header" . }}

<h1>Project: <a href="/project/{{.Page.Project.Id}}">{{.Page.Project.Name}}</a></h1>
<h2>Control: {{.Page.Control.Name}}</h2>
<p>Created at <i>{{.Page.Control.CreatedAt.Format "Jan 02, 2006 15:04 MST"}}</i></p>

<ul class="nav nav-tabs" role="tablist">
  <li role="presentation" class="active">
    <a href="#ranks" aria-controls="ranks" role="tab" data-toggle="tab">Expression Ranks</a>
  </li>
{{ if not .Page.ReadOnly }}
  <li role="presentation">
    <a href="#newsample" aria-controls="newsample" role="tab"
      data-toggle="tab">Upload new sample</a>
  </li>
{{ end }}
</ul>

<div class="panel panel-default">
  <div class="panel-body">

<div class="tab-content">
  <div role="tabpanel" id="ranks" class="tab-pane fade in active">

<table class="table table-striped">
<tr><th>Dimension</th><th>Rank</th></tr>
{{ $lookup := .Page.Lookup }}
{{ range .Page.Values }}
<tr><td>{{(index $lookup .DimensionId)}}</td><td>{{.Rank}}</td></tr>
{{ end }}
</table>

  </div>
{{ if not .Page.ReadOnly }}
  <div role="tabpanel" id="newsample" class="tab-pane fade">

<form method="POST" action="/project/{{.Page.Project.Id}}/control/{{.Page.Control.Id}}/sample">
<input type="text" name="name" class="form-control" placeholder="Name"><br/>
<textarea name="values" class="form-control" rows="5"
    placeholder="dimension value (one dimension per line)"></textarea><br/>
<button type="submit" class="btn btn-default">Upload</button>
</form>

  </div>
{{ end }}

</div>

  </div>
</div>

{{ template "footer" . }}`)
}
