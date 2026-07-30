<template>
	<XButton
		variant="secondary"
		icon="filter"
		:class="{'has-filters': hasFilters}"
		@click="() => modalOpen = true"
	>
		{{ $t('filters.title') }}
	</XButton>
	<Modal
		:enabled="modalOpen"
		:overflow="true"
		variant="hint-modal"
		:aria-label="$t('filters.title')"
		@close="() => modalOpen = false"
	>
		<Filters
			ref="filtersRef"
			v-model="value"
			:has-title="true"
			class="filter-popup"
			:change-immediately="false"
			:filter-from-view="filterFromView"
			:can-save-personal="Boolean(projectId && projectId > 0)"
			show-close
			@close="modalOpen = false"
			@showResults="showResults"
			@savePersonal="openSaveModal"
		/>
	</Modal>

	<Modal
		:enabled="saveModalOpen"
		variant="hint-modal"
		:aria-label="$t('filters.personal.saveTitle')"
		@close="saveModalOpen = false"
	>
		<Card
			:title="$t('filters.personal.saveTitle')"
			show-close
			@close="saveModalOpen = false"
		>
			<p class="mbe-2">
				{{ $t('filters.personal.saveDescription') }}
			</p>
			<FormField
				id="personal-filter-title"
				v-model="saveTitle"
				v-focus
				:label="$t('filters.attributes.title')"
				type="text"
				:placeholder="$t('filters.personal.titlePlaceholder')"
				:error="saveTitleValid ? null : $t('filters.create.titleRequired')"
			/>
			<label class="is-flex is-align-items-center mbs-2">
				<input
					v-model="savePinned"
					type="checkbox"
					class="mie-2"
				>
				{{ $t('filters.personal.pin') }}
			</label>
			<template #footer>
				<XButton
					variant="primary"
					:loading="saving"
					:disabled="saving || !saveTitle.trim()"
					@click="savePersonalFilter"
				>
					{{ $t('filters.personal.save') }}
				</XButton>
			</template>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {computed, ref, watch, nextTick} from 'vue'
import {useI18n} from 'vue-i18n'

import Filters from '@/components/project/partials/Filters.vue'
import FormField from '@/components/input/FormField.vue'

import {getDefaultTaskFilterParams, type TaskFilterParams} from '@/services/taskCollection'
import {type IProjectView} from '@/modelTypes/IProjectView'
import {type IProject} from '@/modelTypes/IProject'
import {useProjectStore} from '@/stores/projects'
import {useUserProjectFiltersStore} from '@/stores/userProjectFilters'
import {success} from '@/message'

const props = defineProps<{
	modelValue: TaskFilterParams,
	projectId?: IProject['id'],
	viewId?: IProjectView['id'],
}>()

const emit = defineEmits<{
	'update:modelValue': [value: TaskFilterParams]
}>()

const {t} = useI18n()
const projectStore = useProjectStore()
const userProjectFiltersStore = useUserProjectFiltersStore()

const value = ref<TaskFilterParams>(getDefaultTaskFilterParams())
const filtersRef = ref()

watch(
	() => props.modelValue,
	(modelValue: TaskFilterParams) => {
		value.value = modelValue
	},
	{
		immediate: true,
		deep: true,
	},
)

const hasFilters = computed(() => {
	return value.value.filter !== '' ||
		value.value.s !== ''
})

const modalOpen = ref(false)
const saveModalOpen = ref(false)
const saveTitle = ref('')
const savePinned = ref(false)
const saveTitleValid = ref(true)
const saving = ref(false)

// Auto-focus filter input when modal opens
watch(modalOpen, (isOpen) => {
	if (isOpen) {
		nextTick(() => {
			filtersRef.value?.focusFilterInput()
		})
	}
})

function showResults() {
	emit('update:modelValue', {
		...value.value,
		filter: value.value.filter,
		s: value.value.s,
	})
	if (props.projectId && props.projectId > 0) {
		userProjectFiltersStore.setActive(props.projectId, null)
		userProjectFiltersStore.persistState(props.projectId, value.value, null)
	}
	modalOpen.value = false
}

function openSaveModal() {
	saveTitle.value = ''
	savePinned.value = false
	saveTitleValid.value = true
	saveModalOpen.value = true
}

async function savePersonalFilter() {
	if (!props.projectId || props.projectId <= 0) {
		return
	}
	if (!saveTitle.value.trim()) {
		saveTitleValid.value = false
		return
	}

	saving.value = true
	try {
		// Simple search (`s`) is not a filter expression — persist it as a title match.
		let filterExpr = value.value.filter || ''
		if (!filterExpr && value.value.s) {
			const escaped = value.value.s.replace(/\\/g, '\\\\').replace(/'/g, '\\\'')
			filterExpr = `title like '${escaped}'`
		}
		const created = await userProjectFiltersStore.create(props.projectId, {
			title: saveTitle.value.trim(),
			filter: filterExpr,
			filterIncludeNulls: Boolean(value.value.filter_include_nulls),
			isPinned: savePinned.value,
		})
		if (!created) {
			return
		}

		const params = {
			...value.value,
			...userProjectFiltersStore.applyFilter(created),
		}
		emit('update:modelValue', params)
		userProjectFiltersStore.persistState(props.projectId, params, created.id)
		success({message: t('filters.personal.saveSuccess')})
		saveModalOpen.value = false
		modalOpen.value = false
	} finally {
		saving.value = false
	}
}

const filterFromView = computed(() => {
	if (!props.projectId || !props.viewId) {
		return
	}
	
	const project = projectStore.projects[props.projectId]
	if (!project) {
		return
	}
	const view = project.views.find(v => v.id === props.viewId)
	return view?.filter?.filter
})
</script>

<style scoped lang="scss">
.filter-popup {
	margin: 0;

	&.is-open {
		margin: 2rem 0 1rem;
	}
}

$filter-bubble-size: .75rem;
.has-filters {
	position: relative;

	&::after {
		content: '';
		position: absolute;
		inset-block-start: math.div($filter-bubble-size, -2);
		inset-inline-end: math.div($filter-bubble-size, -2);

		inline-size: $filter-bubble-size;
		block-size: $filter-bubble-size;
		border-radius: 100%;
		background: var(--primary);
	}
}
</style>
