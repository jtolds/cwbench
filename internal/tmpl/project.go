// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("project", `{{ template "header" . }}

<h1>Project: {{.Page.Project.Name}}</h1>
<p>Created at <i>{{.Page.Project.CreatedAt.Format "Jan 02, 2006 15:04 MST"}}</i></p>
<p>Project is associated with {{ .Page.DimensionCount }} dimensions.</p>

<h2>Search</h2>

<ul class="nav nav-tabs" role="tablist">
  <li role="presentation" class="active">
    <a href="#topk" aria-controls="topk" role="tab" data-toggle="tab">Top-k</a>
  </li>
  <li role="presentation">
    <a href="#kolmogorov" aria-controls="kolmogorov" role="tab"
      data-toggle="tab">Kolmogorov-Smirnov</a>
  </li>
  <li role="presentation">
    <a href="#kbarcoding" aria-controls="kbarcoding" role="tab"
      data-toggle="tab">k-Barcoding</a>
  </li>
</ul>

<div class="panel panel-default">
  <div class="panel-body">

<div class="tab-content">
  <div role="tabpanel" id="topk" class="tab-pane fade in active">

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
<div class="row">
<div class="col-md-12 form-inline" style="text-align:right;">
  <div style="display:inline-block; text-align:left; padding-right:20px;">
  <div class="radio">
    <label>
      <input type="radio" name="topk-type" value="rankdiff" checked>
      Score based on rank difference
    </label>
  </div><br/>
  <div class="radio">
    <label>
      <input type="radio" name="topk-type" value="valdiff">
      Score based on value difference
    </label>
  </div>
  </div>

  <div class="form-group">
    <label for="topkInput"><strong>k = </strong></label>
    <input type="number" name="k" class="form-control" id="topkInput"
      value="25" />
  </div>
  <input type="hidden" name="search-type" value="topk" />
  <button type="submit" class="btn btn-default">Search</button>
</div>
</div>
</form>

  </div>
  <div role="tabpanel" id="kolmogorov" class="tab-pane fade">

<!--
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
<div class="row">
<div class="col-md-12 form-inline" style="text-align:right;">
  <input type="hidden" name="search-type" value="kolmogorov" />
  <button type="submit" class="btn btn-default">Search</button>
</div>
</div>
</form>
-->
Not yet implemented

  </div>
  <div role="tabpanel" id="kbarcoding" class="tab-pane fade">
  Not yet implemented
  </div>
</div>

  </div>
</div>

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
      placeholder="<dimension> <value> (one dimension per line)"></textarea><br/>
  <button type="submit" class="btn btn-default">Upload</button>
  </form>
</li>
{{ end }}
</ul>

</div>
<div class="col-md-6">

<h2>Samples</h2>

<ul>
{{ $page := .Page }}
{{ range .Page.Samples }}
<li><a href="/project/{{$page.Project.Id}}/sample/{{.Id}}">{{.Name}}</a></li>
{{ end }}
</ul>

</div>
</div>

{{ template "footer" . }}`)
}
