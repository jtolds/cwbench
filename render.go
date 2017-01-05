// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/jtolds/cwbench/internal/tmpl"
	"golang.org/x/net/context"
	"gopkg.in/webhelp.v1/whcompat"
	"gopkg.in/webhelp.v1/wherr"
	"gopkg.in/webhelp.v1/whfatal"
	"gopkg.in/webhelp.v1/whmux"
	"gopkg.in/webhelp.v1/whredir"
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
			ctx := whcompat.Context(req)
			user := LoadUser(ctx)
			tmpl, page, err := logic(ctx, req, user)
			if err != nil {
				whfatal.Error(err)
			}
			t := r.Templates.Lookup(tmpl)
			if t == nil {
				whfatal.Error(wherr.InternalServerError.New(
					"no template %#v registered", tmpl))
			}
			w.Header().Set("Content-Type", "text/html")
			err = t.Execute(w, PageCtx{
				User:      user,
				LogoutURL: oauth2.LogoutURL("/"),
				Page:      page})
			if err != nil {
				whfatal.Error(err)
			}
		})
}

type Handler func(w http.ResponseWriter, req *http.Request, user *UserInfo)

func (r Renderer) Process(logic Handler) http.Handler {
	return whmux.ExactPath(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			logic(w, req, LoadUser(whcompat.Context(req)))
		}))
}

var (
	ProjectRedirector = whredir.RedirectHandlerFunc(
		func(r *http.Request) string {
			return fmt.Sprintf("/project/%d", projectId.MustGet(whcompat.Context(r)))
		})
	ControlRedirector = whredir.RedirectHandlerFunc(
		func(r *http.Request) string {
			ctx := whcompat.Context(r)
			return fmt.Sprintf("/project/%d/control/%d",
				projectId.MustGet(ctx), controlId.MustGet(ctx))
		})
)
