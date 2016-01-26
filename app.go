// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jtolds/webhelp"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/context"
)

var (
	sqlitePath = flag.String("db", "./db.db", "")
)

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
	err = a.db.Where(
		"user_id = ? AND id = ?", user.Id, projectId.Get(ctx)).First(&proj).Error
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}
	var dims []Dimension
	err = a.db.Where("project_id = ?", proj.Id).Find(&dims).Error
	if err != nil {
		return "", nil, err
	}
	return "project", map[string]interface{}{
		"Project":    proj,
		"Dimensions": dims}, nil
}

func (a *App) NewProject(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	tx := txWrapper{DB: a.db.Begin()}
	defer tx.Rollback()
	proj := Project{UserId: user.Id, Name: req.FormValue("name")}
	err := tx.Create(&proj).Error
	if err != nil {
		return err
	}
	added := map[string]bool{}
	for _, dim := range strings.Fields(req.FormValue("dimensions")) {
		if added[dim] {
			continue
		}
		added[dim] = true
		err := tx.Create(&Dimension{ProjectId: proj.Id, Name: dim}).Error
		if err != nil {
			return err
		}
	}
	tx.Commit()
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d", proj.Id))
}

func (a *App) UpdateProject(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	return webhelp.Redirect(w, req, req.RequestURI)
}
