// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("control", `{{ template "header" . }}

<h1>Project: <a href="/project/{{.Page.Project.Id}}">{{.Page.Project.Name}}</a></h1>
<h2>Control: {{.Page.Control.Name}}</h2>
<p>Created at <i>{{.Page.Control.CreatedAt}}</i></p>

<div class="row">
<div class="col-md-6">

<h3>Expression ranks</h3>
<table class="table table-striped">
<tr><th>Dimension</th><th>Rank</th></tr>
{{ $lookup := .Page.Lookup }}
{{ range .Page.Values }}
<tr><td>{{(index $lookup .DimensionId)}}</td><td>{{.Rank}}</td></tr>
{{ end }}
</table>

</div><div class="col-md-6">

{{ if not .Page.ReadOnly }}
<h3>Upload a new sample</h3>

<form method="POST" action="/project/{{.Page.Project.Id}}/control/{{.Page.Control.Id}}/sample">
<input type="text" name="name" class="form-control" placeholder="Name"><br/>
<textarea name="values" class="form-control" rows="5"
    placeholder="dimension value (one dimension per line)"></textarea><br/>
<button type="submit" class="btn btn-default">Upload</button>
</form>
{{ end }}

</div>
</div>

{{ template "footer" . }}`)
}
