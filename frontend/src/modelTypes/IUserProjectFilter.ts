import type {IProject} from './IProject'

export interface IUserProjectFilter {
	id: number
	projectId: IProject['id']
	title: string
	filter: string
	filterIncludeNulls: boolean
	isPinned: boolean
	position: number
	created: Date
	updated: Date
	maxPermission?: number | null
}

export interface IProjectFilterStateEntry {
	activeFilterId: number | null
	filter: string
	filterIncludeNulls: boolean
	s: string
}

export type ProjectFilterStateMap = Record<string, IProjectFilterStateEntry>
