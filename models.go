// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"time"

	"github.com/spacemonkeygo/errors"
)

type APIKey struct {
	UserId string
	Key    string
}

type Project struct {
	Id        int64 `gorm:"primary_key"`
	CreatedAt time.Time
	UserId    string
	Name      string
	Public    bool
}

type Dimension struct {
	Id        int64 `gorm:"primary_key"`
	ProjectId int64
	Name      string
}

type DiffExp struct {
	Id        int64 `gorm:"primary_key"`
	CreatedAt time.Time
	ProjectId int64
	Name      string
}

type DiffExpValue struct {
	DiffExpId   int64
	DimensionId int64
	RankDiff    float64
	AbsRankDiff float64
	SampleValue float64
}

type Control struct {
	Id        int64 `gorm:"primary_key"`
	CreatedAt time.Time
	ProjectId int64
	Name      string
}

type ControlValue struct {
	ControlId   int64
	DimensionId int64
	Value       float64
	Rank        int
}

func (d *Data) CreateDB() error {
	var errs errors.ErrorGroup
	tx := d.db.Begin()

	errs.Add(tx.Exec(`CREATE TABLE
	  api_keys (
  	  user_id character varying(255) NOT NULL,
  	  key character varying(255) NOT NULL
	  );`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_api_keys_user_id ON api_keys(user_id);`).Error)
	errs.Add(tx.Exec(`CREATE UNIQUE INDEX
	  idx_api_keys_key ON api_keys(key);`).Error)

	errs.Add(tx.Exec(`CREATE SEQUENCE projects_id_seq;`).Error)
	errs.Add(tx.Exec(`CREATE TABLE
    projects (
      id bigint NOT NULL DEFAULT nextval('projects_id_seq'),
      created_at timestamp with time zone NOT NULL,
      user_id character varying(255) NOT NULL,
      name character varying(255) NOT NULL,
      public boolean NOT NULL
    );`).Error)
	errs.Add(tx.Exec(`CREATE UNIQUE INDEX
	  idx_projects_user_id_name ON projects(user_id, name);`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_projects_public ON projects(public);`).Error)

	errs.Add(tx.Exec(`CREATE SEQUENCE dimensions_id_seq;`).Error)
	errs.Add(tx.Exec(`CREATE TABLE
    dimensions (
      id bigint NOT NULL DEFAULT nextval('dimensions_id_seq'),
      project_id bigint NOT NULL,
      name character varying(255) NOT NULL
    );`).Error)
	errs.Add(tx.Exec(`CREATE UNIQUE INDEX
	  idx_dimensions_project_id_name ON dimensions(project_id, name);`).Error)

	errs.Add(tx.Exec(`CREATE SEQUENCE diff_exps_id_seq;`).Error)
	errs.Add(tx.Exec(`CREATE TABLE
    diff_exps (
      id bigint NOT NULL DEFAULT nextval('diff_exps_id_seq'),
      created_at timestamp with time zone NOT NULL,
      project_id bigint NOT NULL,
      name character varying(255) NOT NULL
    );`).Error)
	errs.Add(tx.Exec(`CREATE UNIQUE INDEX
	  idx_diff_exps_project_id_name ON diff_exps(project_id, name);`).Error)

	errs.Add(tx.Exec(`CREATE TABLE
    diff_exp_values (
      diff_exp_id bigint NOT NULL,
      dimension_id bigint NOT NULL,
      rank_diff real NOT NULL,
      abs_rank_diff real NOT NULL,
      sample_value real,
      primary key(diff_exp_id, dimension_id)
    );`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_diff_exp_values_diff_exp_id_abs_rank_diff ON
	      diff_exp_values(diff_exp_id, abs_rank_diff);`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_diff_exp_values_diff_exp_id_rank_diff ON
	      diff_exp_values(diff_exp_id, rank_diff);`).Error)

	errs.Add(tx.Exec(`CREATE SEQUENCE controls_id_seq;`).Error)
	errs.Add(tx.Exec(`CREATE TABLE
    controls (
      id bigint NOT NULL DEFAULT nextval('controls_id_seq'),
      created_at timestamp with time zone NOT NULL,
      project_id bigint NOT NULL,
      name character varying(255) NOT NULL
    );`).Error)
	errs.Add(tx.Exec(`CREATE UNIQUE INDEX
	  idx_controls_project_id_name ON controls(project_id, name);`).Error)

	errs.Add(tx.Exec(`CREATE TABLE
    control_values (
      control_id bigint NOT NULL,
      dimension_id bigint NOT NULL,
      rank int NOT NULL,
      value real NOT NULL,
      primary key(control_id, dimension_id)
    );`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_control_values_control_id_rank ON
	      control_values(control_id, rank);`).Error)

	err := errs.Finalize()
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}
