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
	Id        int64 `gorm:"primary_key"`
	CreatedAt time.Time
	UserId    string `sql:"index"`
	Name      string
	Public    bool
}

type Dimension struct {
	Id        int64 `gorm:"primary_key"`
	ProjectId int64 `sql:"index"`
	Name      string
}

type DiffExp struct {
	Id        int64 `gorm:"primary_key"`
	CreatedAt time.Time
	ProjectId int64 `sql:"index"`
	Name      string
}

type DiffExpValue struct {
	DiffExpId   int64
	DimensionId int64
	Diff        int
}

type Control struct {
	Id        int64 `gorm:"primary_key"`
	CreatedAt time.Time
	ProjectId int64 `sql:"index"`
	Name      string
}

type ControlValue struct {
	ControlId   int64
	DimensionId int64
	Rank        int
}

func (a *App) Migrate() error {
	var errs errors.ErrorGroup
	tx := a.db.Begin()
	// why oh why does gorm break with composite primary keys?!
	errs.Add(tx.Exec(`CREATE TABLE "diff_exp_values" (` +
		`"diff_exp_id" bigint,` +
		`"dimension_id" bigint,` +
		`"diff" int, ` +
		`primary key("diff_exp_id", "dimension_id"));`).Error)
	errs.Add(tx.Exec(`CREATE TABLE "control_values" (` +
		`"control_id" bigint,` +
		`"dimension_id" bigint,` +
		`"diff" int, ` +
		`primary key("control_id", "dimension_id"));`).Error)
	tx.Commit()
	errs.Add(a.db.AutoMigrate(&Project{}).Error)
	errs.Add(a.db.AutoMigrate(&Dimension{}).Error)
	errs.Add(a.db.AutoMigrate(&DiffExp{}).Error)
	errs.Add(a.db.AutoMigrate(&DiffExpValue{}).Error)
	errs.Add(a.db.AutoMigrate(&Control{}).Error)
	errs.Add(a.db.AutoMigrate(&ControlValue{}).Error)
	return errs.Finalize()
}
