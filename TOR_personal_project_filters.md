# TOR: Personal Project Filters (UX)

## Goal
Make filtering inside a project as convenient as Saved Filters are powerful — without forcing navigation to the Projects overview or creating pseudo-projects.

## Problems with current Saved Filters
1. Creation path is outside the project context.
2. Each saved filter becomes a pseudo-project with own views (heavy UX for simple “pin this query”).
3. Ad-hoc project filters are ephemeral (URL/session only).
4. No in-project chips/tabs for quick switching.

## Solution
Introduce **personal project filters**: private, project-scoped filter presets owned by the authenticated user.

### Functional requirements
1. **Save current filter** from the project filter popup (title prompt) without leaving the project.
2. **Chips/tabs** under the project view toolbar for personal filters of that project.
3. **Pin/favorite** personal filters (pinned first; reorder by position).
4. **Apply** a personal filter with one click; active chip highlighted.
5. **Auto-restore** last active personal filter (or last ad-hoc query) per project across sessions/devices.
6. **CRUD**: rename, update query from current filter, delete, pin/unpin.
7. Filters are **private** (owner-only). Link shares cannot read/write them.
8. Must not create pseudo-projects or default views.

### Non-goals (MVP)
- Sharing personal filters with project members.
- Replacing global Saved Filters.
- Kanban bucket sync for personal filters (they only affect the task collection query).

## Data model
Table `user_project_filters`:

| Column | Type | Notes |
|--------|------|-------|
| id | bigint PK | |
| user_id | bigint INDEX | owner |
| project_id | bigint INDEX | real project only (>0) |
| title | varchar(250) | required |
| filter | text | filter query string |
| filter_include_nulls | bool | default true |
| is_pinned | bool | default false |
| position | double | sort order |
| created / updated | datetime | |

Unique suggestion: no hard unique on title (allow duplicates); UI may warn.

### Last-used state
Stored in `user.frontend_settings.projectFilterState`:

```json
{
  "projectFilterState": {
    "<projectId>": {
      "activeFilterId": 12,
      "filter": "done = false",
      "filter_include_nulls": true,
      "s": ""
    }
  }
}
```

`activeFilterId = null` means ad-hoc / none.

## API (v2 only)
- `GET    /api/v2/projects/{project}/user-filters`
- `POST   /api/v2/projects/{project}/user-filters`
- `GET    /api/v2/projects/{project}/user-filters/{filter}`
- `PUT    /api/v2/projects/{project}/user-filters/{filter}`
- `PATCH  /api/v2/projects/{project}/user-filters/{filter}`
- `DELETE /api/v2/projects/{project}/user-filters/{filter}`

Permissions:
- User must have at least read access to the project.
- Only the owner of the filter may mutate/delete it.
- Filters are never visible to other users.

## Frontend UX
1. `FilterPopup` / `Filters.vue`: add secondary button **Save**.
2. New `PersonalFilterBar.vue` next to FilterPopup in List/Kanban/Table (and Gantt if applicable).
3. Chips: title, pin icon, overflow menu (apply / update from current / rename / delete).
4. On project enter: load personal filters + restore from `frontend_settings.projectFilterState`.
5. On apply/save/clear: persist state into frontend settings (debounced).

## Tests
- Backend model + webtests for CRUD/permissions.
- Frontend unit for state helpers; e2e smoke for save → chip → restore.

## Living matrix
| Requirement | Module | Tests |
|-------------|--------|-------|
| Save current filter | `Filters.vue` + `FilterPopup.vue` + `pkg/models/user_project_filters.go` | `TestUserProjectFilter` create (webtests) |
| Personal chips | `PersonalFilterBar.vue` + List/Kanban/Table | manual / future e2e |
| Pinning | `is_pinned` + `position` ordering | `TestUserProjectFilter_ReadAll` |
| Restore last | `frontend_settings.projectFilterState` + `stores/userProjectFilters.ts` | `userProjectFilters.test.ts` |
| Private ownership | `user_project_filters_permissions.go` | webtests forbidden / other-user |
| API v2 routes | `pkg/routes/api/v2/user_project_filters.go` | `TestUserProjectFilter` + PATCH |
| Migration | `pkg/migration/20260730171500.go` | sync via `GetTables()` + fixtures |

## Acceptance
- User can save, pin, switch, and restore personal filters inside a project without opening Projects overview.
- Global Saved Filters remain unchanged.
- Cross-device restore works via user settings API.
