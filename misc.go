// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"github.com/jinzhu/gorm"
)

type txWrapper struct {
	*gorm.DB
}

func (tx *txWrapper) Rollback() {
	if tx.DB != nil {
		tx.DB.Rollback()
		tx.DB = nil
	}
}

func (tx *txWrapper) Commit() {
	if tx.DB != nil {
		tx.DB.Commit()
		tx.DB = nil
	}
}
