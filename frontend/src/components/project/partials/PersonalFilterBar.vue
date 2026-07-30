<template>
	<div
		v-if="filters.length > 0"
		class="personal-filter-bar"
		role="tablist"
		:aria-label="$t('filters.personal.barLabel')"
	>
		<button
			v-for="filter in filters"
			:key="filter.id"
			type="button"
			class="personal-filter-chip"
			:class="{
				'is-active': filter.id === activeId,
				'is-pinned': filter.isPinned,
			}"
			role="tab"
			:aria-selected="filter.id === activeId"
			@click="apply(filter)"
		>
			<Icon
				v-if="filter.isPinned"
				:icon="['fas', 'star']"
				class="pin-icon"
			/>
			<span class="chip-title">{{ filter.title }}</span>
			<Dropdown
				class="chip-menu"
				:trigger-label="$t('filters.personal.actions')"
			>
				<template #trigger="{toggleOpen}">
					<BaseButton
						class="chip-menu-trigger"
						:aria-label="$t('filters.personal.actions')"
						@click.prevent.stop="toggleOpen()"
					>
						<Icon icon="ellipsis-h" />
					</BaseButton>
				</template>
				<DropdownItem
					:icon="filter.isPinned ? 'star' : ['far', 'star']"
					@click.stop="pin(filter)"
				>
					{{ filter.isPinned ? $t('filters.personal.unpin') : $t('filters.personal.pin') }}
				</DropdownItem>
				<DropdownItem
					icon="sync"
					@click.stop="updateFromCurrent(filter)"
				>
					{{ $t('filters.personal.updateFromCurrent') }}
				</DropdownItem>
				<DropdownItem
					icon="pen"
					@click.stop="startRename(filter)"
				>
					{{ $t('filters.personal.rename') }}
				</DropdownItem>
				<DropdownItem
					icon="trash-alt"
					class="has-text-danger"
					@click.stop="remove(filter)"
				>
					{{ $t('filters.personal.delete') }}
				</DropdownItem>
			</Dropdown>
		</button>

		<Modal
			:enabled="renameOpen"
			variant="hint-modal"
			:aria-label="$t('filters.personal.rename')"
			@close="renameOpen = false"
		>
			<Card
				:title="$t('filters.personal.rename')"
				show-close
				@close="renameOpen = false"
			>
				<FormField
					id="personal-filter-rename"
					v-model="renameTitle"
					v-focus
					:label="$t('filters.attributes.title')"
					type="text"
					:placeholder="$t('filters.personal.titlePlaceholder')"
				/>
				<template #footer>
					<XButton
						variant="primary"
						:disabled="!renameTitle.trim()"
						@click="confirmRename"
					>
						{{ $t('misc.save') }}
					</XButton>
				</template>
			</Card>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'

import Dropdown from '@/components/misc/Dropdown.vue'
import DropdownItem from '@/components/misc/DropdownItem.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import FormField from '@/components/input/FormField.vue'

import type {IProject} from '@/modelTypes/IProject'
import type {IUserProjectFilter} from '@/modelTypes/IUserProjectFilter'
import type {TaskFilterParams} from '@/services/taskCollection'
import {useUserProjectFiltersStore} from '@/stores/userProjectFilters'
import {success} from '@/message'

const props = defineProps<{
	projectId: IProject['id']
	modelValue: TaskFilterParams
}>()

const emit = defineEmits<{
	'update:modelValue': [value: TaskFilterParams]
}>()

const {t} = useI18n()
const store = useUserProjectFiltersStore()

const filters = computed(() => store.filtersFor(props.projectId))
const activeId = computed(() => store.activeIdFor(props.projectId))

const renameOpen = ref(false)
const renameTitle = ref('')
const renameTarget = ref<IUserProjectFilter | null>(null)

watch(
	() => props.projectId,
	async (projectId) => {
		if (!projectId || projectId <= 0) {
			return
		}
		await store.load(projectId)
		const restored = store.restoreIfNeeded(projectId, props.modelValue)
		if (restored) {
			emit('update:modelValue', {
				...props.modelValue,
				...restored.params,
			})
			store.persistState(projectId, restored.params, restored.activeFilterId)
		}
	},
	{immediate: true},
)

watch(
	() => [props.modelValue.filter, props.modelValue.s, props.modelValue.filter_include_nulls] as const,
	() => {
		if (!props.projectId || props.projectId <= 0) {
			return
		}
		const match = filters.value.find(f =>
			f.filter === (props.modelValue.filter ?? '') &&
			f.filterIncludeNulls === Boolean(props.modelValue.filter_include_nulls) &&
			!props.modelValue.s,
		)
		const resolvedActive = match?.id ?? null
		if (resolvedActive !== activeId.value) {
			store.setActive(props.projectId, resolvedActive)
		}
		store.persistState(props.projectId, props.modelValue, resolvedActive)
	},
)

function apply(filter: IUserProjectFilter) {
	const params = {
		...props.modelValue,
		...store.applyFilter(filter),
	}
	store.setActive(props.projectId, filter.id)
	emit('update:modelValue', params)
	store.persistState(props.projectId, params, filter.id)
}

async function pin(filter: IUserProjectFilter) {
	const updated = await store.togglePin(props.projectId, filter)
	if (updated) {
		success({message: t('filters.personal.updateSuccess')})
	}
}

async function updateFromCurrent(filter: IUserProjectFilter) {
	const updated = await store.update(props.projectId, {
		...filter,
		filter: props.modelValue.filter ?? '',
		filterIncludeNulls: Boolean(props.modelValue.filter_include_nulls),
	})
	if (updated) {
		success({message: t('filters.personal.updateSuccess')})
	}
}

function startRename(filter: IUserProjectFilter) {
	renameTarget.value = filter
	renameTitle.value = filter.title
	renameOpen.value = true
}

async function confirmRename() {
	if (!renameTarget.value || !renameTitle.value.trim()) {
		return
	}
	const updated = await store.update(props.projectId, {
		...renameTarget.value,
		title: renameTitle.value.trim(),
	})
	if (updated) {
		success({message: t('filters.personal.updateSuccess')})
		renameOpen.value = false
	}
}

async function remove(filter: IUserProjectFilter) {
	const message = t('filters.personal.deleteConfirm', {title: filter.title})
	if (!window.confirm(message)) {
		return
	}
	const wasActive = activeId.value === filter.id
	const deleted = await store.remove(props.projectId, filter.id)
	if (!deleted) {
		return
	}
	success({message: t('filters.personal.deleteSuccess')})
	if (wasActive) {
		const cleared = {
			...props.modelValue,
			filter: '',
			s: '',
		}
		emit('update:modelValue', cleared)
		store.persistState(props.projectId, cleared, null)
	}
}
</script>

<style scoped lang="scss">
.personal-filter-bar {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .35rem;
	max-inline-size: min(40rem, 55vw);
}

.personal-filter-chip {
	display: inline-flex;
	align-items: center;
	gap: .3rem;
	max-inline-size: 14rem;
	padding: .25rem .45rem .25rem .65rem;
	border: 1px solid var(--grey-200);
	border-radius: 999px;
	background: var(--white);
	color: var(--text);
	font-size: .85rem;
	line-height: 1.2;
	cursor: pointer;

	&:hover {
		border-color: var(--primary);
	}

	&.is-active {
		background: color-mix(in srgb, var(--primary) 16%, var(--white));
		border-color: var(--primary);
		font-weight: 600;
	}

	.pin-icon {
		font-size: .75rem;
		color: var(--primary);
	}

	.chip-title {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
}

.chip-menu-trigger {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	inline-size: 1.25rem;
	block-size: 1.25rem;
	border-radius: 50%;
	color: var(--grey-500);

	&:hover {
		background: var(--grey-100);
		color: var(--text);
	}
}
</style>
