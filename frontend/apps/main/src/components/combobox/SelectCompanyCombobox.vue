<template>
  <SelectComboBox
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="searchCompanies"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useCompanyStore } from '@/stores/company'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: undefined
  },
  placeholder: {
    type: String,
    default: ''
  },
  // Companies a picker must not offer, e.g. a company cannot be its own parent.
  excludeIDs: {
    type: Array,
    default: () => []
  }
})

const { t } = useI18n()
const companyStore = useCompanyStore()

const placeholderText = computed(() => props.placeholder || t('placeholders.selectCompany'))

const excluded = computed(() => new Set(props.excludeIDs.map(String)))

const items = computed(() =>
  companyStore.companyOptions.filter((option) => !excluded.value.has(option.value))
)

// The exclusion has to apply to remote results too, not just the cached first page.
const searchCompanies = async (query) =>
  (await companyStore.searchCompanyOptions(query)).filter(
    (option) => !excluded.value.has(option.value)
  )

onMounted(companyStore.fetchCompanies)

// The selected company may sit outside the first page, so pin it to keep its label.
watch(
  () => props.modelValue,
  (value) => companyStore.ensureCompanyIDs([value]),
  { immediate: true }
)
</script>
