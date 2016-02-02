// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("project", `{{ template "header" . }}

<h1>Project: {{.Page.Project.Name}}</h1>
<p>Created at <i>{{.Page.Project.CreatedAt}}</i></p>
<p>Project is associated with {{ .Page.DimensionCount }} dimensions.</p>

<h2>Search</h2>
<form method="POST" action="/project/{{.Page.Project.Id}}/search">
<div class="row">
<div class="col-md-6">
  <textarea name="up-regulated" class="form-control" rows="3"
      placeholder="up-regulated dimensions (whitespace separated)"></textarea>
  <br/>
</div>
<div class="col-md-6">
  <textarea name="down-regulated" class="form-control" rows="3"
      placeholder="down-regulated dimensions (whitespace separated)"></textarea>
  <br/>
</div>
</div>
<button type="submit" class="btn btn-default">Search</button>
</form>

<div class="row">

<div class="col-md-6">
<h2>Controls</h2>
<ul>
{{ $page := .Page }}
{{ range .Page.Controls }}
<li><a href="/project/{{$page.Project.Id}}/control/{{.Id}}">{{.Name}}</a></li>
{{ end }}

{{ if not .Page.ReadOnly }}
<br/>
<li>Create new:<br/>
  <form method="POST" action="/project/{{.Page.Project.Id}}/control">
  <input type="text" name="name" class="form-control" placeholder="Name"><br/>
  <textarea name="values" class="form-control" rows="5"
      placeholder="dimension value (one dimension per line)"></textarea><br/>
  <button type="submit" class="btn btn-default">Upload</button>
  </form>
</li>
{{ end }}
</ul>

</div>
<div class="col-md-6">

<h2>Differential expressions</h2>
<ul>
{{ $page := .Page }}
{{ range .Page.DiffExps }}
<li><a href="/project/{{$page.Project.Id}}/diffexp/{{.Id}}">{{.Name}}</a></li>
{{ end }}
</ul>

</div>
</div>

{{ template "footer" . }}`)
}
