// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("projects", `{{ template "header" . }}

<h1>Your Projects</h1>
<ul>
{{ range .Page.Projects }}
<li><a href="/project/{{.Id}}">{{.Name}}</a>{{if .Public}} (public){{end}}</li>
{{ end }}

<br/>
<li>Create new:<br/>
  <form method="POST" action="/project/">
  <input type="text" name="name" class="form-control" placeholder="Name"><br/>
  <textarea name="dimensions" class="form-control" rows="5"
      placeholder="Dimensions (whitespace-separated)"></textarea><br/>
  <button type="submit" class="btn btn-default">Create</button>
  </form>
</li>
</ul>

{{ template "footer" . }}`)
}
