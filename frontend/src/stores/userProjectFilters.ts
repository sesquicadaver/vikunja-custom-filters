import {defineStore} from 'pinia'
import {ref} from 'vue'
import {useDebounceFn} from '@vueuse/core'

import {useUserProjectFilterService} from '@/services/userProjectFilter'
import type {IUserProjectFilter, IProjectFilterStateEntry, ProjectFilterStateMap} from '@/modelTypes/IUserProjectFilter'
import type {IFrontendSettings} from '@/modelTypes/IUserSettings'
import type {IProject} from '@/modelTypes/IProject'
import type {TaskFilterParams} from '@/services/taskCollection'
import {useAuthStore} from '@/stores/auth'
import {error} from '@/message'

export const useUserProjectFiltersStore = defineStore('userProjectFilters', () => {
	const service = useUserProjectFilterService()
	const byProject = ref<Record<number, IUserProjectFilter[]>>({})
	const activeByProject = ref<Record<number, number | null>>({})
	const loadingByProject = ref<Record<number, boolean>>({})

	function filtersFor(projectId: IProject['id']): IUserProjectFilter[] {
		return byProject.value[projectId] ?? []
	}

	function activeIdFor(projectId: IProject['id']): number | null {
		return activeByProject.value[projectId] ?? null
	}

	async function load(projectId: IProject['id']): Promise<IUserProjectFilter[]> {
		loadingByProject.value[projectId] = true
		try {
			const result = await service.getAll(projectId)
			byProject.value[projectId] = result.items
			return result.items
		} catch (e) {
			error(e)
			return []
		} finally {
			loadingByProject.value[projectId] = false
		}
	}

	async function create(
		projectId: IProject['id'],
		payload: {
			title: string
			filter: string
			filterIncludeNulls: boolean
			isPinned?: boolean
		},
	): Promise<IUserProjectFilter | null> {
		try {
			const created = await service.create(projectId, {
				title: payload.title,
				filter: payload.filter,
				filterIncludeNulls: payload.filterIncludeNulls,
				isPinned: payload.isPinned ?? false,
			})
			await load(projectId)
			activeByProject.value[projectId] = created.id
			return created
		} catch (e) {
			error(e)
			return null
		}
	}

	async function update(
		projectId: IProject['id'],
		filter: Partial<IUserProjectFilter> & {id: number},
	): Promise<IUserProjectFilter | null> {
		try {
			const updated = await service.update(projectId, filter)
			await load(projectId)
			return updated
		} catch (e) {
			error(e)
			return null
		}
	}

	async function remove(projectId: IProject['id'], id: number): Promise<boolean> {
		try {
			await service.remove(projectId, id)
			if (activeByProject.value[projectId] === id) {
				activeByProject.value[projectId] = null
			}
			await load(projectId)
			return true
		} catch (e) {
			error(e)
			return false
		}
	}

	async function togglePin(projectId: IProject['id'], filter: IUserProjectFilter): Promise<IUserProjectFilter | null> {
		return update(projectId, {
			...filter,
			isPinned: !filter.isPinned,
		})
	}

	function setActive(projectId: IProject['id'], id: number | null) {
		activeByProject.value[projectId] = id
	}

	function applyFilter(filter: IUserProjectFilter): TaskFilterParams {
		return {
			sort_by: [],
			order_by: [],
			filter: filter.filter,
			filter_include_nulls: filter.filterIncludeNulls,
			s: '',
		}
	}

	function getStoredState(projectId: IProject['id']): IProjectFilterStateEntry | null {
		const authStore = useAuthStore()
		const map = authStore.settings?.frontendSettings?.projectFilterState as ProjectFilterStateMap | undefined
		if (!map) {
			return null
		}
		return map[String(projectId)] ?? null
	}

	const persistStateDebounced = useDebounceFn(async (
		projectId: IProject['id'],
		entry: IProjectFilterStateEntry,
	) => {
		const authStore = useAuthStore()
		const current = (authStore.settings.frontendSettings.projectFilterState ?? {}) as ProjectFilterStateMap
		const next: ProjectFilterStateMap = {
			...current,
			[String(projectId)]: entry,
		}
		const frontendSettings: IFrontendSettings = {
			...authStore.settings.frontendSettings,
			quickAddDefaultReminders: [...(authStore.settings.frontendSettings.quickAddDefaultReminders ?? [])],
			projectFilterState: next,
		}
		await authStore.saveUserSettings({
			settings: {
				...authStore.settings,
				frontendSettings,
			},
			showMessage: false,
		})
	}, 400)

	function persistState(projectId: IProject['id'], params: TaskFilterParams, activeFilterId: number | null) {
		activeByProject.value[projectId] = activeFilterId
		void persistStateDebounced(projectId, {
			activeFilterId,
			filter: params.filter ?? '',
			filterIncludeNulls: Boolean(params.filter_include_nulls),
			s: params.s ?? '',
		})
	}

	/**
	 * Restores last-used filter when the URL has no ad-hoc filter/search.
	 * Returns params to apply, or null when nothing should change.
	 */
	function restoreIfNeeded(
		projectId: IProject['id'],
		current: TaskFilterParams,
	): {params: TaskFilterParams, activeFilterId: number | null} | null {
		const hasUrlFilter = Boolean(current.filter) || Boolean(current.s)
		if (hasUrlFilter) {
			const match = filtersFor(projectId).find(f =>
				f.filter === current.filter &&
				f.filterIncludeNulls === Boolean(current.filter_include_nulls),
			)
			activeByProject.value[projectId] = match?.id ?? null
			return null
		}

		const stored = getStoredState(projectId)
		if (!stored) {
			return null
		}

		if (stored.activeFilterId) {
			const preset = filtersFor(projectId).find(f => f.id === stored.activeFilterId)
			if (preset) {
				activeByProject.value[projectId] = preset.id
				return {
					params: applyFilter(preset),
					activeFilterId: preset.id,
				}
			}
		}

		if (!stored.filter && !stored.s) {
			return null
		}

		activeByProject.value[projectId] = null
		return {
			params: {
				...current,
				filter: stored.filter,
				filter_include_nulls: stored.filterIncludeNulls,
				s: stored.s,
			},
			activeFilterId: null,
		}
	}

	return {
		byProject,
		activeByProject,
		loadingByProject,
		filtersFor,
		activeIdFor,
		load,
		create,
		update,
		remove,
		togglePin,
		setActive,
		applyFilter,
		getStoredState,
		persistState,
		restoreIfNeeded,
	}
})
