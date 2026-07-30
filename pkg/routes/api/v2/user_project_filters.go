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

package apiv2

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

type userProjectFilterListBody struct {
	Body Paginated[*models.UserProjectFilter]
}

// RegisterUserProjectFilterRoutes wires personal project-filter CRUD onto the Huma API.
func RegisterUserProjectFilterRoutes(api huma.API) {
	tags := []string{"user_project_filters"}

	Register(api, huma.Operation{
		OperationID: "user-project-filters-list",
		Summary:     "List personal filters for a project",
		Description: "Returns the authenticated user's personal filter presets for the given project. Requires read access to the project. Filters are never shared with other users.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/user-filters",
		Tags:        tags,
	}, userProjectFiltersList)

	Register(api, huma.Operation{
		OperationID: "user-project-filters-read",
		Summary:     "Get a personal project filter",
		Description: "Returns one personal filter. The filter must belong to the project in the path and to the authenticated user. Sends an ETag; pass it as If-None-Match on a later read to get a 304 Not Modified.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/user-filters/{filter}",
		Tags:        tags,
	}, userProjectFiltersRead)

	Register(api, huma.Operation{
		OperationID: "user-project-filters-create",
		Summary:     "Create a personal project filter",
		Description: "Creates a personal filter preset for the authenticated user in the given project. The parent project is taken from the URL. Requires read access to the project.",
		Method:      http.MethodPost,
		Path:        "/projects/{project}/user-filters",
		Tags:        tags,
	}, userProjectFiltersCreate)

	Register(api, huma.Operation{
		OperationID: "user-project-filters-update",
		Summary:     "Update a personal project filter",
		Description: "Replaces a personal filter's fields. Only the owner may update it. Use PATCH for a partial update.",
		Method:      http.MethodPut,
		Path:        "/projects/{project}/user-filters/{filter}",
		Tags:        tags,
	}, userProjectFiltersUpdate)

	Register(api, huma.Operation{
		OperationID: "user-project-filters-delete",
		Summary:     "Delete a personal project filter",
		Description: "Deletes a personal filter. Only the owner may delete it.",
		Method:      http.MethodDelete,
		Path:        "/projects/{project}/user-filters/{filter}",
		Tags:        tags,
	}, userProjectFiltersDelete)
}

func init() { AddRouteRegistrar(RegisterUserProjectFilterRoutes) }

func userProjectFiltersList(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ListParams
}) (*userProjectFilterListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.UserProjectFilter{ProjectID: in.ProjectID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.UserProjectFilter)
	if !ok {
		return nil, fmt.Errorf("userProjectFilters.ReadAll returned unexpected type %T (expected []*models.UserProjectFilter)", result)
	}
	return &userProjectFilterListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type userProjectFilterReadBody struct {
	models.UserProjectFilter
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The maximum permission the requesting user has on this personal filter (0=read, 1=read/write, 2=admin). Filters are owner-only, so this is always 2 for a successful read."`
}

func userProjectFiltersRead(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ID        int64 `path:"filter"`
	conditional.Params
}) (*singleReadBody[userProjectFilterReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	filter := &models.UserProjectFilter{ID: in.ID, ProjectID: in.ProjectID}
	maxPermission, err := handler.DoReadOne(ctx, filter, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &userProjectFilterReadBody{UserProjectFilter: *filter, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, filter.Updated, maxPermission)
}

func userProjectFiltersCreate(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	Body      models.UserProjectFilter
}) (*singleBody[models.UserProjectFilter], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	in.Body.ProjectID = in.ProjectID // URL wins over body
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.UserProjectFilter]{Body: &in.Body}, nil
}

// Body matches the read shape so AutoPatch's GET→PUT echo of max_permission validates.
func userProjectFiltersUpdate(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ID        int64 `path:"filter"`
	Body      userProjectFilterReadBody
}) (*singleBody[models.UserProjectFilter], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	filter := &in.Body.UserProjectFilter
	filter.ID = in.ID               // URL wins over body
	filter.ProjectID = in.ProjectID // parent from the path scopes the update
	if err := handler.DoUpdate(ctx, filter, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.UserProjectFilter]{Body: filter}, nil
}

func userProjectFiltersDelete(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ID        int64 `path:"filter"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.UserProjectFilter{ID: in.ID, ProjectID: in.ProjectID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
