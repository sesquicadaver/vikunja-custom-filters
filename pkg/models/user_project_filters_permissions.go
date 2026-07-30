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
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

func (upf *UserProjectFilter) ensureProjectReadable(s *xorm.Session, auth web.Auth) (bool, error) {
	if _, is := auth.(*LinkSharing); is {
		return false, ErrUserProjectFilterNotAvailableForLinkShare{LinkShareID: auth.GetID()}
	}
	if upf.ProjectID <= 0 {
		return false, ErrUserProjectFilterInvalidProject{ProjectID: upf.ProjectID}
	}
	canRead, _, err := (&Project{ID: upf.ProjectID}).CanRead(s, auth)
	return canRead, err
}

// canDoFilter verifies ownership and optional project path scoping.
// When populate is true the caller's struct is replaced with the DB row
// (safe for read/delete). When false only permission is checked so update
// payloads are not overwritten.
func (upf *UserProjectFilter) canDoFilter(s *xorm.Session, auth web.Auth, populate bool) (bool, error) {
	if _, is := auth.(*LinkSharing); is {
		return false, ErrUserProjectFilterNotAvailableForLinkShare{LinkShareID: auth.GetID()}
	}

	requestedProjectID := upf.ProjectID
	loaded, err := GetUserProjectFilterSimpleByID(s, upf.ID)
	if err != nil {
		return false, err
	}
	if requestedProjectID != 0 && loaded.ProjectID != requestedProjectID {
		return false, ErrUserProjectFilterDoesNotExist{UserProjectFilterID: upf.ID}
	}
	if loaded.UserID != auth.GetID() {
		return false, nil
	}

	canRead, err := loaded.ensureProjectReadable(s, auth)
	if err != nil || !canRead {
		return false, err
	}

	if populate {
		*upf = *loaded
	}
	return true, nil
}

// CanCreate checks whether the user can create a personal filter for the project.
func (upf *UserProjectFilter) CanCreate(s *xorm.Session, auth web.Auth) (bool, error) {
	return upf.ensureProjectReadable(s, auth)
}

// CanRead checks whether the user can read this personal filter.
func (upf *UserProjectFilter) CanRead(s *xorm.Session, auth web.Auth) (bool, int, error) {
	can, err := upf.canDoFilter(s, auth, true)
	return can, int(PermissionAdmin), err
}

// CanUpdate checks whether the user can update this personal filter.
func (upf *UserProjectFilter) CanUpdate(s *xorm.Session, auth web.Auth) (bool, error) {
	// Avoid overwriting the incoming update payload (same pattern as SavedFilter).
	check := &UserProjectFilter{ID: upf.ID, ProjectID: upf.ProjectID}
	return check.canDoFilter(s, auth, false)
}

// CanDelete checks whether the user can delete this personal filter.
func (upf *UserProjectFilter) CanDelete(s *xorm.Session, auth web.Auth) (bool, error) {
	return upf.canDoFilter(s, auth, true)
}
