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

package models

import (
	"time"

	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// UserProjectFilter is a personal, project-scoped filter preset.
// Unlike SavedFilter it does not create a pseudo-project and is only visible to its owner.
type UserProjectFilter struct {
	// The unique numeric id of this personal project filter
	ID int64 `xorm:"autoincr not null unique pk" json:"id" param:"userfilter" readOnly:"true" doc:"The unique, numeric id of this personal project filter."`
	// The owning user id
	UserID int64 `xorm:"bigint not null INDEX" json:"-"`
	// The project this filter belongs to
	ProjectID int64 `xorm:"bigint not null INDEX" json:"project_id" param:"project" doc:"The project this personal filter belongs to."`
	// The title of the filter
	Title string `xorm:"varchar(250) not null" json:"title" valid:"required,runelength(1|250)" minLength:"1" maxLength:"250" doc:"The title of the filter."`
	// The filter query string
	Filter string `xorm:"text not null" json:"filter" doc:"The filter query string applied when this preset is active."`
	// Whether tasks with empty values should be included
	FilterIncludeNulls bool `xorm:"not null default true" json:"filter_include_nulls" doc:"If true, tasks with empty values for filtered fields are included."`
	// Whether the filter is pinned for quick access
	IsPinned bool `xorm:"not null default false" json:"is_pinned" doc:"If true, the filter is pinned and shown first in the personal filter bar."`
	// Sort position among the owner's filters for this project
	Position float64 `xorm:"double not null default 0" json:"position" doc:"Sort position among the owner's personal filters for this project."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this filter was created."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when this filter was last updated."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for personal project filters.
func (upf *UserProjectFilter) TableName() string {
	return "user_project_filters"
}

// GetUserProjectFilterSimpleByID loads a filter by id without permission checks.
func GetUserProjectFilterSimpleByID(s *xorm.Session, id int64) (upf *UserProjectFilter, err error) {
	upf = &UserProjectFilter{}
	exists, err := s.Where("id = ?", id).Get(upf)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserProjectFilterDoesNotExist{UserProjectFilterID: id}
	}
	return
}

func (upf *UserProjectFilter) validateFilterQuery() error {
	_, err := getTaskFiltersFromFilterString(upf.Filter, "")
	return err
}

func (upf *UserProjectFilter) nextPosition(s *xorm.Session) (float64, error) {
	last := &UserProjectFilter{}
	exists, err := s.
		Where("user_id = ? AND project_id = ?", upf.UserID, upf.ProjectID).
		OrderBy("position DESC").
		Limit(1).
		Get(last)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 100, nil
	}
	return last.Position + 100, nil
}

// Create creates a personal project filter for the authenticated user.
func (upf *UserProjectFilter) Create(s *xorm.Session, auth web.Auth) (err error) {
	if upf.ProjectID <= 0 {
		return ErrUserProjectFilterInvalidProject{ProjectID: upf.ProjectID}
	}
	if err = upf.validateFilterQuery(); err != nil {
		return err
	}

	upf.UserID = auth.GetID()
	upf.ID = 0
	if upf.Position == 0 {
		upf.Position, err = upf.nextPosition(s)
		if err != nil {
			return err
		}
	}

	_, err = s.Insert(upf)
	return err
}

// ReadOne returns one personal project filter.
// CanRead already populates the struct from the database.
func (upf *UserProjectFilter) ReadOne(_ *xorm.Session, _ web.Auth) error {
	return nil
}

// ReadAll lists personal project filters for the given project owned by the caller.
func (upf *UserProjectFilter) ReadAll(s *xorm.Session, auth web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, totalItems int64, err error) {
	if _, is := auth.(*LinkSharing); is {
		return nil, 0, 0, ErrUserProjectFilterNotAvailableForLinkShare{LinkShareID: auth.GetID()}
	}
	if upf.ProjectID <= 0 {
		return nil, 0, 0, ErrUserProjectFilterInvalidProject{ProjectID: upf.ProjectID}
	}

	canRead, _, err := (&Project{ID: upf.ProjectID}).CanRead(s, auth)
	if err != nil {
		return nil, 0, 0, err
	}
	if !canRead {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	limit, start := getLimitFromPageIndex(page, perPage)
	where := "user_id = ? AND project_id = ?"
	args := []interface{}{auth.GetID(), upf.ProjectID}
	if search != "" {
		where += " AND title LIKE ?"
		args = append(args, "%"+search+"%")
	}

	totalItems, err = s.Where(where, args...).Count(&UserProjectFilter{})
	if err != nil {
		return nil, 0, 0, err
	}

	filters := make([]*UserProjectFilter, 0)
	q := s.Where(where, args...).OrderBy("is_pinned DESC, position ASC, id ASC")
	if limit > 0 {
		err = q.Limit(limit, start).Find(&filters)
	} else {
		err = q.Find(&filters)
	}
	if err != nil {
		return nil, 0, 0, err
	}

	return filters, len(filters), totalItems, nil
}

// Update updates an existing personal project filter.
func (upf *UserProjectFilter) Update(s *xorm.Session, _ web.Auth) error {
	orig, err := GetUserProjectFilterSimpleByID(s, upf.ID)
	if err != nil {
		return err
	}

	upf.UserID = orig.UserID
	upf.ProjectID = orig.ProjectID
	if err = upf.validateFilterQuery(); err != nil {
		return err
	}

	_, err = s.
		Where("id = ?", upf.ID).
		Cols(
			"title",
			"filter",
			"filter_include_nulls",
			"is_pinned",
			"position",
		).
		Update(upf)
	return err
}

// Delete removes a personal project filter.
func (upf *UserProjectFilter) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.Where("id = ?", upf.ID).Delete(&UserProjectFilter{})
	return err
}
