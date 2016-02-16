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

type Sample struct {
	Id        int64 `gorm:"primary_key"`
	ControlId int64
	CreatedAt time.Time
	ProjectId int64
	Name      string
}

type SampleValue struct {
	SampleId    int64
	DimensionId int64

	Rank        int
	RankDiff    int
	AbsRankDiff int

	Value        float64
	ValueDiff    float64
	AbsValueDiff float64
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

	errs.Add(tx.Exec(`CREATE SEQUENCE samples_id_seq;`).Error)
	errs.Add(tx.Exec(`CREATE TABLE
    samples (
      id bigint NOT NULL DEFAULT nextval('samples_id_seq'),
      control_id bigint NOT NULL,
      created_at timestamp with time zone NOT NULL,
      project_id bigint NOT NULL,
      name character varying(255) NOT NULL
    );`).Error)
	errs.Add(tx.Exec(`CREATE UNIQUE INDEX
	  idx_samples_project_id_name ON samples(project_id, name);`).Error)

	errs.Add(tx.Exec(`CREATE TABLE
    sample_values (
      sample_id bigint NOT NULL,
      dimension_id bigint NOT NULL,

      rank integer NOT NULL,
      rank_diff integer NOT NULL,
      abs_rank_diff integer NOT NULL,

      value real NOT NULL,
      value_diff real NOT NULL,
      abs_value_diff real NOT NULL,

      primary key(sample_id, dimension_id)
    );`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_sample_values_sample_id_abs_rank_diff ON
	      sample_values(sample_id, abs_rank_diff);`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_sample_values_sample_id_rank_diff ON
	      sample_values(sample_id, rank_diff);`).Error)
	errs.Add(tx.Exec(`CREATE INDEX
	  idx_sample_values_sample_id_abs_value_diff ON
	      sample_values(sample_id, abs_value_diff);`).Error)

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
      rank integer NOT NULL,
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
