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
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserProjectFilter_Create(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		upf := &UserProjectFilter{
			ProjectID:          1,
			Title:              "My filter",
			Filter:             "done = false",
			FilterIncludeNulls: true,
			IsPinned:           true,
		}
		u := &user.User{ID: 1}
		err := upf.Create(s, u)
		require.NoError(t, err)
		assert.Equal(t, u.ID, upf.UserID)
		assert.NotZero(t, upf.ID)
		assert.Greater(t, upf.Position, float64(0))
		err = s.Commit()
		require.NoError(t, err)
		db.AssertExists(t, "user_project_filters", map[string]interface{}{
			"id":         upf.ID,
			"user_id":    1,
			"project_id": 1,
			"title":      "My filter",
			"is_pinned":  true,
		}, false)
	})

	t.Run("invalid filter string", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		upf := &UserProjectFilter{
			ProjectID: 1,
			Title:     "bad",
			Filter:    "foo = value",
		}
		err := upf.Create(s, &user.User{ID: 1})
		require.Error(t, err)
		db.AssertMissing(t, "user_project_filters", map[string]interface{}{
			"title": "bad",
		})
	})

	t.Run("pseudo project rejected", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		upf := &UserProjectFilter{
			ProjectID: -2,
			Title:     "nope",
			Filter:    "done = false",
		}
		err := upf.Create(s, &user.User{ID: 1})
		require.Error(t, err)
		assert.True(t, IsErrUserProjectFilterInvalidProject(err))
	})
}

func TestUserProjectFilter_ReadAll(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	upf := &UserProjectFilter{ProjectID: 1}
	result, count, total, err := upf.ReadAll(s, &user.User{ID: 1}, "", 0, 0)
	require.NoError(t, err)
	filters := result.([]*UserProjectFilter)
	assert.Equal(t, 2, count)
	assert.Equal(t, int64(2), total)
	assert.Len(t, filters, 2)
	// pinned first
	assert.True(t, filters[0].IsPinned)
	assert.Equal(t, "Open tasks", filters[0].Title)
	// other user's filter must not appear
	for _, f := range filters {
		assert.Equal(t, int64(1), f.UserID)
	}
}

func TestUserProjectFilter_UpdateAndDelete(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	upf := &UserProjectFilter{
		ID:                 2,
		Title:              "Updated",
		Filter:             "priority = 5",
		FilterIncludeNulls: true,
		IsPinned:           true,
		Position:           50,
	}
	err := upf.Update(s, &user.User{ID: 1})
	require.NoError(t, err)
	err = s.Commit()
	require.NoError(t, err)
	db.AssertExists(t, "user_project_filters", map[string]interface{}{
		"id":        2,
		"title":     "Updated",
		"is_pinned": true,
	}, false)

	s2 := db.NewSession()
	defer s2.Close()
	del := &UserProjectFilter{ID: 2}
	err = del.Delete(s2, &user.User{ID: 1})
	require.NoError(t, err)
	err = s2.Commit()
	require.NoError(t, err)
	db.AssertMissing(t, "user_project_filters", map[string]interface{}{
		"id": 2,
	})
}

func TestUserProjectFilter_Permissions(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	t.Run("owner can read", func(t *testing.T) {
		upf := &UserProjectFilter{ID: 1, ProjectID: 1}
		can, perm, err := upf.CanRead(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.Equal(t, int(PermissionAdmin), perm)
		assert.Equal(t, "Open tasks", upf.Title)
	})

	t.Run("non-owner cannot read", func(t *testing.T) {
		upf := &UserProjectFilter{ID: 1, ProjectID: 1}
		can, _, err := upf.CanRead(s, &user.User{ID: 2})
		require.NoError(t, err)
		assert.False(t, can)
	})

	t.Run("wrong project path", func(t *testing.T) {
		upf := &UserProjectFilter{ID: 1, ProjectID: 9}
		can, _, err := upf.CanRead(s, &user.User{ID: 1})
		require.Error(t, err)
		assert.False(t, can)
		assert.True(t, IsErrUserProjectFilterDoesNotExist(err))
	})

	t.Run("link share blocked", func(t *testing.T) {
		upf := &UserProjectFilter{ProjectID: 1, Title: "x", Filter: "done = false"}
		can, err := upf.CanCreate(s, &LinkSharing{ID: 1})
		require.Error(t, err)
		assert.False(t, can)
		assert.True(t, IsErrUserProjectFilterNotAvailableForLinkShare(err))
	})

	t.Run("update does not wipe payload", func(t *testing.T) {
		upf := &UserProjectFilter{
			ID:        1,
			ProjectID: 1,
			Title:     "Should stay",
			Filter:    "done = false",
		}
		can, err := upf.CanUpdate(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.Equal(t, "Should stay", upf.Title)
	})
}
