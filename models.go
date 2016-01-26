// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"time"

	"github.com/spacemonkeygo/errors"
)

// data model:
// * user
// * project
// * genes
// * controls
// * perturbagens
// * differential expressions (w/ranks)

type Project struct {
	Id        uint64 `gorm:"primary_key"`
	CreatedAt time.Time
	UserId    string `sql:"index"`
	Name      string
}

type Dimension struct {
	ProjectId uint64 `sql:"index"`
	Name      string
}

func (a *App) Migrate() error {
	var errs errors.ErrorGroup
	errs.Add(a.db.AutoMigrate(&Project{}).Error)
	errs.Add(a.db.AutoMigrate(&Dimension{}).Error)
	return errs.Finalize()
}
