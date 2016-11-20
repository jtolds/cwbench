// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/jtolds/cwbench/internal/tmpl"
	"github.com/jtolds/webhelp"
	"golang.org/x/net/context"
)

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
	return &Renderer{Templates: tmpl.Templates}, nil
}

func (r Renderer) Render(logic Logic) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			user := LoadUser(ctx)
			tmpl, page, err := logic(ctx, req, user)
			if err != nil {
				webhelp.FatalError(err)
			}
			t := r.Templates.Lookup(tmpl)
			if t == nil {
				webhelp.FatalError(webhelp.ErrInternalServerError.New(
					"no template %#v registered", tmpl))
			}
			w.Header().Set("Content-Type", "text/html")
			err = t.Execute(w, PageCtx{
				User:      user,
				LogoutURL: oauth2.LogoutURL("/"),
				Page:      page})
			if err != nil {
				webhelp.FatalError(err)
			}
		})
}

type Handler func(w http.ResponseWriter, req *http.Request, user *UserInfo)

func (r Renderer) Process(logic Handler) http.Handler {
	return webhelp.ExactPath(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			logic(w, req, LoadUser(req.Context()))
		}))
}

var (
	ProjectRedirector = webhelp.RedirectHandlerFunc(
		func(r *http.Request) string {
			return fmt.Sprintf("/project/%d", projectId.MustGet(r.Context()))
		})
	ControlRedirector = webhelp.RedirectHandlerFunc(
		func(r *http.Request) string {
			ctx := r.Context()
			return fmt.Sprintf("/project/%d/control/%d",
				projectId.MustGet(ctx), controlId.MustGet(ctx))
		})
)
