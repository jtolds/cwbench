// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/jtolds/webhelp"
	"golang.org/x/net/context"
)

var (
	tmplPath = flag.String("templates", "./tmpl", "path to templates")
)

type Pair struct {
	First, Second interface{}
}

func makePair(first, second interface{}) Pair {
	return Pair{First: first, Second: second}
}

func LoadTemplates() (rv *template.Template, err error) {
	rv = template.New("root").Funcs(template.FuncMap{
		"makepair": makePair})
	files, err := ioutil.ReadDir(*tmplPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".tmpl") {
			continue
		}
		content, err := ioutil.ReadFile(filepath.Join(*tmplPath, file.Name()))
		if err != nil {
			return nil, err
		}
		_, err = rv.New(strings.TrimSuffix(file.Name(), ".tmpl")).Parse(
			string(content))
		if err != nil {
			return nil, err
		}
	}
	return rv, nil
}

type PageCtx struct {
	User      *UserInfo
	LogoutURL string
	Page      map[string]interface{}
}

type Logic func(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{},
	err error)

type Renderer struct {
	Templates *template.Template
}

func NewRenderer() (*Renderer, error) {
	templates, err := LoadTemplates()
	if err != nil {
		return nil, err
	}
	return &Renderer{Templates: templates}, nil
}

func (r Renderer) Render(logic Logic) webhelp.Handler {
	return webhelp.Exact(webhelp.HandlerFunc(
		func(ctx context.Context, w webhelp.ResponseWriter,
			req *http.Request) error {
			user, err := LoadUser(ctx)
			if err != nil {
				return err
			}
			tmpl, page, err := logic(ctx, req, user)
			if err != nil {
				return err
			}
			t := r.Templates.Lookup(tmpl)
			if t == nil {
				return webhelp.ErrInternalServerError.New("no template %#v registered", tmpl)
			}
			w.Header().Set("Content-Type", "text/html")
			return t.Execute(w, PageCtx{
				User:      user,
				LogoutURL: oauth2.LogoutURL("/"),
				Page:      page})
		}))
}

type Handler func(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error

func (r Renderer) Process(logic Handler) webhelp.Handler {
	return webhelp.ExactPath(webhelp.HandlerFunc(func(ctx context.Context,
		w webhelp.ResponseWriter, req *http.Request) error {
		user, err := LoadUser(ctx)
		if err != nil {
			return err
		}
		return logic(ctx, w, req, user)
	}))
}

var (
	ProjectRedirector = webhelp.RedirectHandlerFunc(
		func(ctx context.Context, r *http.Request) string {
			return fmt.Sprintf("/project/%d", projectId.Get(ctx))
		})
	ControlRedirector = webhelp.RedirectHandlerFunc(
		func(ctx context.Context, r *http.Request) string {
			return fmt.Sprintf("/project/%d/control/%d",
				projectId.Get(ctx), controlId.Get(ctx))
		})
)
