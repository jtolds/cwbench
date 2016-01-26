// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/jinzhu/gorm"
	"github.com/jtolds/webhelp"
	"github.com/jtolds/webhelp/sessions"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/context"
)

var (
	listenAddr   = flag.String("addr", ":8080", "address to listen on")
	cookieSecret = flag.String("cookie_secret", "abcdef0123456789",
		"the secret for securing cookie information")
	sqlitePath = flag.String("db", "./db.db", "")

	projectId = webhelp.NewIntArgMux()
)

// data model:
// * user
// * project
// * genes
// * controls
// * perturbagens
// * differential expressions (w/ranks)

type Project struct {
	gorm.Model

	UserId string `sql:"index"`
	Name   string
}

type App struct {
	db gorm.DB
}

func NewApp() (*App, error) {
	db, err := gorm.Open("sqlite3", *sqlitePath)
	if err != nil {
		return nil, err
	}
	return &App{db: db}, nil
}

func (a *App) Close() error { return a.db.Close() }

func (a *App) Migrate() error {
	return a.db.AutoMigrate(&Project{}).Error
}

func (a *App) ProjectList(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{},
	err error) {
	var projects []*Project
	err = a.db.Where("user_id = ?", user.Id).Find(&projects).Error
	if err != nil {
		return "", nil, err
	}
	return "projects", map[string]interface{}{"Projects": projects}, nil
}

func (a *App) Project(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	var proj Project
	err = a.db.First(&proj, projectId.Get(ctx)).Error
	if err != nil {
		return "", nil, err
	}
	return "project", map[string]interface{}{"Project": proj}, nil
}

func (a *App) NewProject(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	tx := a.db.Begin()
	defer tx.Rollback()
	proj := Project{UserId: user.Id, Name: req.FormValue("name")}
	err := tx.Create(&proj).Error
	if err != nil {
		return err
	}
	tx.Commit()
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d", proj.ID))
}

func (a *App) UpdateProject(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	return webhelp.Redirect(w, req, req.RequestURI)
}

func main() {
	flag.Parse()
	loadOAuth2()
	secret, err := hex.DecodeString(*cookieSecret)
	if err != nil {
		panic(err)
	}

	renderer, err := NewRenderer()
	if err != nil {
		panic(err)
	}

	app, err := NewApp()
	if err != nil {
		panic(err)
	}
	defer app.Close()

	switch flag.Arg(0) {
	case "migrate":
		err := app.Migrate()
		if err != nil {
			panic(err)
		}
	case "serve":
		panic(webhelp.ListenAndServe(*listenAddr, webhelp.LoggingHandler(
			sessions.HandlerWithStore(sessions.NewCookieStore(secret),
				webhelp.DirMux{
					"": oauth2.LoginRequired(renderer.Render(app.ProjectList)),
					"project": projectId.ShiftIf(webhelp.MethodMux{
						"GET":  renderer.Render(app.Project),
						"POST": renderer.Process(app.UpdateProject),
					}, webhelp.ExactPath(webhelp.MethodMux{
						"GET":  webhelp.RedirectHandler("/"),
						"POST": renderer.Process(app.NewProject),
					})),
					"auth": oauth2}))))
	default:
		fmt.Printf("Usage: %s <serve|migrate>\n", os.Args[0])
	}
}
