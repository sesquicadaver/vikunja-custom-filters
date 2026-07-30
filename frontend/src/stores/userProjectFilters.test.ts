import {describe, expect, it, beforeEach, vi} from 'vitest'
import {setActivePinia, createPinia} from 'pinia'

import {useUserProjectFiltersStore} from './userProjectFilters'
import type {IUserProjectFilter} from '@/modelTypes/IUserProjectFilter'

vi.mock('@/services/userProjectFilter', () => ({
	useUserProjectFilterService: () => ({
		getAll: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		remove: vi.fn(),
	}),
}))

vi.mock('@/stores/auth', () => ({
	useAuthStore: () => ({
		settings: {
			frontendSettings: {
				projectFilterState: {
					'1': {
						activeFilterId: 10,
						filter: 'done = false',
						filterIncludeNulls: true,
						s: '',
					},
				},
			},
		},
		saveUserSettings: vi.fn(),
	}),
}))

vi.mock('@/message', () => ({
	error: vi.fn(),
}))

describe('useUserProjectFiltersStore', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
	})

	it('applyFilter maps preset fields into TaskFilterParams', () => {
		const store = useUserProjectFiltersStore()
		const preset: IUserProjectFilter = {
			id: 10,
			projectId: 1,
			title: 'Open',
			filter: 'done = false',
			filterIncludeNulls: true,
			isPinned: true,
			position: 100,
			created: new Date(),
			updated: new Date(),
		}
		expect(store.applyFilter(preset)).toEqual({
			sort_by: [],
			order_by: [],
			filter: 'done = false',
			filter_include_nulls: true,
			s: '',
		})
	})

	it('restoreIfNeeded restores stored personal filter when URL is empty', () => {
		const store = useUserProjectFiltersStore()
		store.byProject[1] = [{
			id: 10,
			projectId: 1,
			title: 'Open',
			filter: 'done = false',
			filterIncludeNulls: true,
			isPinned: true,
			position: 100,
			created: new Date(),
			updated: new Date(),
		}]

		const restored = store.restoreIfNeeded(1, {
			sort_by: [],
			order_by: [],
			filter: '',
			filter_include_nulls: false,
			s: '',
		})

		expect(restored).toEqual({
			params: {
				sort_by: [],
				order_by: [],
				filter: 'done = false',
				filter_include_nulls: true,
				s: '',
			},
			activeFilterId: 10,
		})
		expect(store.activeIdFor(1)).toBe(10)
	})

	it('restoreIfNeeded does not override an existing URL filter', () => {
		const store = useUserProjectFiltersStore()
		store.byProject[1] = [{
			id: 10,
			projectId: 1,
			title: 'Open',
			filter: 'done = false',
			filterIncludeNulls: true,
			isPinned: true,
			position: 100,
			created: new Date(),
			updated: new Date(),
		}]

		const restored = store.restoreIfNeeded(1, {
			sort_by: [],
			order_by: [],
			filter: 'priority >= 4',
			filter_include_nulls: false,
			s: '',
		})

		expect(restored).toBeNull()
		expect(store.activeIdFor(1)).toBeNull()
	})
})
