// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type userProjectFilter20260730171500 struct {
	ID                 int64     `xorm:"autoincr not null unique pk"`
	UserID             int64     `xorm:"bigint not null index"`
	ProjectID          int64     `xorm:"bigint not null index"`
	Title              string    `xorm:"varchar(250) not null"`
	Filter             string    `xorm:"text not null"`
	FilterIncludeNulls bool      `xorm:"not null default true"`
	IsPinned           bool      `xorm:"not null default false"`
	Position           float64   `xorm:"double not null default 0"`
	Created            time.Time `xorm:"created not null"`
	Updated            time.Time `xorm:"updated not null"`
}

func (userProjectFilter20260730171500) TableName() string {
	return "user_project_filters"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260730171500",
		Description: "Add user_project_filters table for personal in-project filter presets",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync(userProjectFilter20260730171500{}) //nolint:forbidigo // brand-new table, nothing to drop
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(userProjectFilter20260730171500{})
		},
	})
}
