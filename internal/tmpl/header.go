// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package tmpl

func init() {
	register("header", `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">

    <title>JT's Connectivity Workbench!</title>

    <link rel="stylesheet"
      href="https://cdnjs.cloudflare.com/ajax/libs/bootswatch/3.3.6/lumen/bootstrap.css">
    <link rel="stylesheet"
      href="https://cdnjs.cloudflare.com/ajax/libs/bootswatch/3.3.6/lumen/bootstrap.min.css">

    <style>
      body { padding-top: 70px; }
    </style>
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.2/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->
  </head>
  <body>
    <nav class="navbar navbar-inverse navbar-fixed-top">
      <div class="container-fluid">
        <div class="navbar-header">
          <button type="button" class="navbar-toggle collapsed"
              data-toggle="collapse" data-target="#navbar"
              aria-expanded="false" aria-controls="navbar">
            <span class="sr-only">Toggle navigation</span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
          </button>
          <a class="navbar-brand" href="/">JT's Connectivity Workbench!</a>
        </div>
        <div id="navbar" class="navbar-collapse collapse">
          <ul class="nav navbar-nav navbar-left">
            <li><a href="/">Your projects</a></li>
          </ul>
          <ul class="nav navbar-nav navbar-right">
            <li class="dropdown">
              <a href="#" class="dropdown-toggle" data-toggle="dropdown">
                <img src="{{.User.Picture}}"
                    style="max-width: 25px; max-height: 25px; margin: 0; padding: 0;"
                    class="img-rounded">
                 {{ if (ne .User.Name "") }}{{.User.Name}}{{ else if (ne .User.Email "") }}{{.User.Email}}{{else}}User <code>{{.User.Id}}</code>{{ end }}
                 <span class="caret"></span></a>
              <ul class="dropdown-menu" role="menu">
                <li><a href="/account/apikeys">API keys</a></li>
                <li><a href="{{.LogoutURL}}">Log Out</a></li>
              </ul>
            </li>
          </ul>
        </div>
      </div>
    </nav>
    <div class="container main">`)
}
