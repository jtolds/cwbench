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

func (r Renderer) Render(logic Logic) webhelp.Handler {
	return webhelp.HandlerFunc(
		func(ctx context.Context, w webhelp.ResponseWriter,
			req *http.Request) error {
			user := LoadUser(ctx)
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
		})
}

type Handler func(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error

func (r Renderer) Process(logic Handler) webhelp.Handler {
	return webhelp.ExactPath(webhelp.HandlerFunc(func(ctx context.Context,
		w webhelp.ResponseWriter, req *http.Request) error {
		return logic(ctx, w, req, LoadUser(ctx))
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
