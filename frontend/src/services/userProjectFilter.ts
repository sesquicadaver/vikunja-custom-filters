import {AuthenticatedHTTPFactory, apiV2Url} from '@/helpers/fetcher'
import {objectToCamelCase, objectToSnakeCase} from '@/helpers/case'

import type {IUserProjectFilter} from '@/modelTypes/IUserProjectFilter'
import type {IProject} from '@/modelTypes/IProject'

export function parseUserProjectFilter(raw: Record<string, unknown>): IUserProjectFilter {
	const e = objectToCamelCase(raw)
	return {
		id: e.id,
		projectId: e.projectId,
		title: e.title ?? '',
		filter: e.filter ?? '',
		filterIncludeNulls: Boolean(e.filterIncludeNulls),
		isPinned: Boolean(e.isPinned),
		position: e.position ?? 0,
		created: new Date(e.created),
		updated: new Date(e.updated),
		maxPermission: e.maxPermission ?? null,
	}
}

export interface UserProjectFilterListResult {
	items: IUserProjectFilter[]
	total: number
	page: number
	perPage: number
	totalPages: number
}

export interface UserProjectFilterPayload {
	title: string
	filter: string
	filterIncludeNulls?: boolean
	isPinned?: boolean
	position?: number
}

export function useUserProjectFilterService() {
	const http = AuthenticatedHTTPFactory()

	async function getAll(projectId: IProject['id']): Promise<UserProjectFilterListResult> {
		const {data} = await http.get(apiV2Url(`projects/${projectId}/user-filters`))
		return {
			items: (data.items ?? []).map(parseUserProjectFilter),
			total: data.total,
			page: data.page,
			perPage: data.per_page,
			totalPages: data.total_pages,
		}
	}

	async function create(projectId: IProject['id'], body: UserProjectFilterPayload): Promise<IUserProjectFilter> {
		const {data} = await http.post(
			apiV2Url(`projects/${projectId}/user-filters`),
			objectToSnakeCase(body),
		)
		return parseUserProjectFilter(data)
	}

	async function update(
		projectId: IProject['id'],
		filter: Partial<IUserProjectFilter> & {id: number},
	): Promise<IUserProjectFilter> {
		const {data} = await http.put(
			apiV2Url(`projects/${projectId}/user-filters/${filter.id}`),
			objectToSnakeCase(filter),
		)
		return parseUserProjectFilter(data)
	}

	async function remove(projectId: IProject['id'], id: number): Promise<void> {
		await http.delete(apiV2Url(`projects/${projectId}/user-filters/${id}`))
	}

	return {getAll, create, update, remove}
}
