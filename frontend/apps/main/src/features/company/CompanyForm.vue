<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <div class="flex flex-wrap gap-6">
      <div class="flex-1">
        <FormField v-slot="{ componentField }" name="name">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.name') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div class="flex-1">
        <FormField v-slot="{ componentField }" name="website">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('company.website') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div class="flex flex-wrap gap-6">
      <div class="flex-1">
        <FormField v-slot="{ componentField, handleChange }" name="parent_id">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('company.parentCompany') }}</FormLabel>
            <FormControl>
              <SelectCompanyCombobox
                :model-value="componentField.modelValue"
                :exclude-ids="excludeIDs"
                @update:model-value="handleChange"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div class="flex flex-col flex-1">
        <PhoneNumberInput />
      </div>
    </div>

    <div class="flex flex-wrap gap-6">
      <div class="flex-1">
        <FormField v-slot="{ componentField }" name="country">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.country') }}</FormLabel>
            <FormControl>
              <ComboBox
                v-bind="componentField"
                :items="countryOptions"
                :placeholder="t('globals.terms.select')"
              >
                <template #item="{ item }">
                  <div class="flex items-center gap-2">
                    <span v-if="item.emoji">{{ item.emoji }}</span>
                    <span class="text-sm">{{ item.label }}</span>
                  </div>
                </template>

                <template #selected="{ selected }">
                  <div class="flex items-center gap-1">
                    <span v-if="selected" class="text-lg">{{ selected.emoji }}</span>
                    <span v-if="selected" class="text-sm">{{ selected.label }}</span>
                  </div>
                </template>
              </ComboBox>
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
      <div class="flex-1"></div>
    </div>

    <FormField v-slot="{ componentField }" name="description">
      <FormItem class="flex flex-col">
        <FormLabel class="flex items-center">{{ t('globals.terms.description') }}</FormLabel>
        <FormControl><Textarea v-bind="componentField" rows="3" /></FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div v-if="userStore.can('companies:write')">
      <Button type="submit" :isLoading="formLoading" :disabled="formLoading">
        {{ submitLabel || t('company.updateCompany') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import {
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Button } from '@shared-ui/components/ui/button'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import PhoneNumberInput from '@shared-ui/components/PhoneNumberInput.vue'
import SelectCompanyCombobox from '@main/components/combobox/SelectCompanyCombobox.vue'
import { countryOptions } from '@shared-ui/constants/countries.js'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@main/stores/user'

defineProps({
  formLoading: Boolean,
  onSubmit: Function,
  submitLabel: String,
  // The company being edited cannot be offered as its own parent.
  excludeIDs: {
    type: Array,
    default: () => []
  }
})

const { t } = useI18n()
const userStore = useUserStore()
</script>
