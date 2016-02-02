// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("apikeys", `{{ template "header" . }}

<h1>API keys</h1>
<ul>
{{ range .Page.Keys }}
<li><code>{{.Key}}</code></li>
{{ end }}

<br/>
<li>
  <form method="POST">
    <button type="submit" class="btn btn-default">Create New</button>
  </form>
</li>
</ul>

{{ template "footer" . }}`)
}
