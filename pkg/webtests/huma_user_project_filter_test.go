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

package webtests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserProjectFilter(t *testing.T) {
	owned := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/projects/1/user-filters",
		idParam:  "filter",
		t:        t,
	}
	require.NoError(t, owned.ensureEnv())

	forbidden := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/projects/2/user-filters",
		idParam:  "filter",
		t:        t,
		e:        owned.e,
	}

	readShared := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/projects/9/user-filters",
		idParam:  "filter",
		t:        t,
		e:        owned.e,
	}

	otherUser := webHandlerTestV2{
		user:     &testuser2,
		basePath: "/api/v2/projects/1/user-filters",
		idParam:  "filter",
		t:        t,
		e:        owned.e,
	}

	t.Run("ReadAll", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := owned.testReadAllWithUser(nil, nil)
			require.NoError(t, err)

			var env struct {
				Items []struct {
					ID        int64  `json:"id"`
					ProjectID int64  `json:"project_id"`
					Title     string `json:"title"`
					IsPinned  bool   `json:"is_pinned"`
				} `json:"items"`
				Total int64 `json:"total"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			assert.Len(t, env.Items, 2)
			assert.Equal(t, int64(2), env.Total)
			assert.True(t, env.Items[0].IsPinned)
			assert.Equal(t, "Open tasks", env.Items[0].Title)
			for _, item := range env.Items {
				assert.Equal(t, int64(1), item.ProjectID)
			}
			assert.NotContains(t, rec.Body.String(), `"title":"Other user filter"`)
		})
		t.Run("Read-only share can list own filters", func(t *testing.T) {
			rec, err := readShared.testReadAllWithUser(nil, nil)
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"title":"Shared project filter"`)
		})
		t.Run("Forbidden project", func(t *testing.T) {
			_, err := forbidden.testReadAllWithUser(nil, nil)
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
	})

	t.Run("ReadOne", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := owned.testReadOneWithUser(nil, map[string]string{"filter": "1"})
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"title":"Open tasks"`)
			assert.Contains(t, rec.Body.String(), `"max_permission":`)
			assert.NotEmpty(t, rec.Result().Header.Get("ETag"))
		})
		t.Run("Other user's filter", func(t *testing.T) {
			_, err := owned.testReadOneWithUser(nil, map[string]string{"filter": "3"})
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
		t.Run("Wrong project path", func(t *testing.T) {
			_, err := owned.testReadOneWithUser(nil, map[string]string{"filter": "4"})
			require.Error(t, err)
			assert.Equal(t, http.StatusNotFound, getHTTPErrorCode(err))
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := owned.testCreateWithUser(nil, nil, `{
				"title": "Created via API",
				"filter": "done = false && priority >= 3",
				"filter_include_nulls": true,
				"is_pinned": true
			}`)
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"title":"Created via API"`)
			assert.Contains(t, rec.Body.String(), `"project_id":1`)
			assert.Contains(t, rec.Body.String(), `"is_pinned":true`)
		})
		t.Run("Invalid filter", func(t *testing.T) {
			_, err := owned.testCreateWithUser(nil, nil, `{
				"title": "bad",
				"filter": "not_a_field = 1"
			}`)
			require.Error(t, err)
		})
		t.Run("Read share can create personal filter", func(t *testing.T) {
			rec, err := readShared.testCreateWithUser(nil, nil, `{
				"title": "On shared project",
				"filter": "done = false"
			}`)
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"project_id":9`)
		})
		t.Run("Forbidden project", func(t *testing.T) {
			_, err := forbidden.testCreateWithUser(nil, nil, `{
				"title": "nope",
				"filter": "done = false"
			}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := owned.testUpdateWithUser(nil, map[string]string{"filter": "2"}, `{
				"title": "High priority updated",
				"filter": "priority >= 5",
				"filter_include_nulls": true,
				"is_pinned": true,
				"position": 10
			}`)
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"title":"High priority updated"`)
			assert.Contains(t, rec.Body.String(), `"is_pinned":true`)
		})
		t.Run("Other user cannot update", func(t *testing.T) {
			_, err := otherUser.testUpdateWithUser(nil, map[string]string{"filter": "1"}, `{
				"title": "hijack",
				"filter": "done = false",
				"filter_include_nulls": true,
				"is_pinned": false,
				"position": 1
			}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			_, err := owned.testDeleteWithUser(nil, map[string]string{"filter": "2"})
			require.NoError(t, err)
			_, err = owned.testReadOneWithUser(nil, map[string]string{"filter": "2"})
			require.Error(t, err)
			assert.Equal(t, http.StatusNotFound, getHTTPErrorCode(err))
		})
		t.Run("Other user cannot delete", func(t *testing.T) {
			_, err := otherUser.testDeleteWithUser(nil, map[string]string{"filter": "1"})
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
	})
}

func TestUserProjectFilter_PATCHMergePatch(t *testing.T) {
	h := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/projects/1/user-filters",
		idParam:  "filter",
		t:        t,
	}
	require.NoError(t, h.ensureEnv())

	token := humaTokenFor(t, &testuser1)
	rec := humaRequest(t, h.e, http.MethodPatch, "/api/v2/projects/1/user-filters/1", `{"is_pinned":false}`, token, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"is_pinned":false`)
	assert.Contains(t, rec.Body.String(), `"title":"Open tasks"`)
}
