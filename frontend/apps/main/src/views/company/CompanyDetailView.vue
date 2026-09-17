<template>
  <CompanyDetail>
    <div class="flex flex-col mx-auto items-start">
      <div class="mb-6" v-if="userStore.can('companies:read')">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>

      <div
        v-if="company"
        class="flex justify-center space-y-4 w-full"
        :class="{ 'loading-fade': formLoading }"
      >
        <div class="flex flex-col w-full mt-12">
          <div class="flex flex-col space-y-2">
            <div class="flex gap-2 justify-start items-center">
              <h2 class="text-xl font-semibold text-foreground">{{ company.name }}</h2>
              <RouterLink
                v-if="company.parent_id"
                :to="{ name: 'company-detail', params: { id: company.parent_id } }"
              >
                <Badge variant="secondary" class="cursor-pointer">{{ company.parent_name }}</Badge>
              </RouterLink>
              <DropdownMenu v-if="userStore.can('companies:delete')">
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" class="h-7 w-7">
                    <MoreVerticalIcon class="h-4 w-4" />
                    <span class="sr-only">{{ t('globals.terms.openMenu') }}</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" class="w-[200px]">
                  <DropdownMenuItem
                    class="text-destructive cursor-pointer"
                    @click="showDeleteConfirmation = true"
                  >
                    <Trash2Icon class="mr-2" size="15" />
                    {{ t('company.deleteCompany') }}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
              <CalendarIcon size="14" class="flex-shrink-0" />
              {{ $t('globals.terms.createdOn') }}
              {{ company.created_at ? format(new Date(company.created_at), 'PPP') : 'N/A' }}
            </div>
          </div>

          <div class="mt-12 space-y-10">
            <CompanyForm
              :formLoading="formLoading"
              :onSubmit="onSubmit"
              :excludeIDs="[company.id]"
            />

            <!-- Sub-companies -->
            <div v-if="children.length" class="space-y-3">
              <h3 class="text-sm font-semibold">{{ t('company.subCompanies') }}</h3>
              <Card
                v-for="child in children"
                :key="child.id"
                class="p-3 w-full hover:bg-accent/50 cursor-pointer"
                @click="$router.push({ name: 'company-detail', params: { id: child.id } })"
              >
                <div class="flex items-center justify-between gap-4">
                  <span class="text-sm truncate">{{ child.name }}</span>
                  <span class="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
                    <UsersIcon size="12" />
                    {{ child.contact_count }}
                  </span>
                </div>
              </Card>
            </div>

            <!-- Contacts at this company -->
            <div v-if="canListContacts" class="space-y-3">
              <h3 class="text-sm font-semibold">
                {{ t('globals.terms.contact', 2) }}
                <span class="text-muted-foreground font-normal">({{ company.contact_count }})</span>
              </h3>
              <Card
                v-for="contact in contacts"
                :key="contact.id"
                class="p-3 w-full hover:bg-accent/50 cursor-pointer"
                @click="$router.push({ name: 'contact-detail', params: { id: contact.id } })"
              >
                <div class="flex items-center gap-3">
                  <Avatar class="h-8 w-8 border">
                    <AvatarImage :src="contact.avatar_url || ''" />
                    <AvatarFallback class="text-xs font-medium">
                      {{ getInitials(contact.first_name, contact.last_name) }}
                    </AvatarFallback>
                  </Avatar>
                  <div class="space-y-0.5 overflow-hidden">
                    <p class="text-sm truncate">
                      {{ contact.first_name }} {{ contact.last_name }}
                    </p>
                    <p class="text-xs text-muted-foreground truncate">{{ contact.email }}</p>
                  </div>
                </div>
              </Card>
              <p v-if="!contacts.length" class="text-sm text-muted-foreground">
                {{ t('contact.noContactsFound') }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <Spinner v-if="formLoading" />

      <AlertDialog
        :open="showDeleteConfirmation"
        @update:open="(v) => (showDeleteConfirmation = v)"
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{{ t('company.deleteCompany') }}</AlertDialogTitle>
            <AlertDialogDescription>{{ t('company.deleteConfirm') }}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" @click="confirmDelete">
              {{ t('globals.messages.delete') }}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  </CompanyDetail>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { format } from 'date-fns'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Card } from '@shared-ui/components/ui/card'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem
} from '@shared-ui/components/ui/dropdown-menu'
import { useUserStore } from '@/stores/user'
import { CalendarIcon, Trash2Icon, MoreVerticalIcon, UsersIcon } from 'lucide-vue-next'
import CompanyDetail from '@/layouts/company/CompanyDetail.vue'
import api from '@/api'
import CompanyForm from '@/features/company/CompanyForm.vue'
import { createFormSchema } from '@/features/company/formSchema.js'
import { toCompanyPayload } from '@/features/company/payload.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Spinner } from '@shared-ui/components/ui/spinner'

const { t } = useI18n()
const emitter = useEmitter()
const route = useRoute()
const router = useRouter()
const formLoading = ref(false)
const company = ref(null)
const children = ref([])
const contacts = ref([])
const showDeleteConfirmation = ref(false)
const userStore = useUserStore()

// Listing a company's contacts goes through the contacts API, which has its own permission.
const canListContacts = computed(() => userStore.can('contacts:read_all'))

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const breadcrumbLinks = [
  { path: 'companies', label: t('globals.terms.company', 2) },
  { path: '', label: t('company.editCompany') }
]

// The route stays mounted when navigating between companies, so refetch on id change.
watch(() => route.params.id, fetchCompany, { immediate: true })

async function fetchCompany() {
  const id = route.params.id
  if (!id) return
  formLoading.value = true
  try {
    const { data } = await api.getCompany(id)
    company.value = data.data
    form.setValues(
      {
        ...data.data,
        parent_id: data.data.parent_id ? String(data.data.parent_id) : undefined
      },
      false
    )
    await Promise.all([fetchChildren(id), canListContacts.value ? fetchContacts(id) : null])
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
}

async function fetchChildren(id) {
  try {
    const { data } = await api.getCompanies({ parent_id: id, page: 1, page_size: 50 })
    children.value = data.data.results
  } catch (err) {
    showError(err)
  }
}

async function fetchContacts(id) {
  try {
    const { data } = await api.getContacts({
      page: 1,
      page_size: 50,
      filters: JSON.stringify([
        { model: 'users', field: 'company_id', operator: 'equals', value: String(id) }
      ])
    })
    contacts.value = data.data.results
  } catch (err) {
    showError(err)
  }
}

const getInitials = (firstName, lastName) =>
  `${firstName?.[0] || ''}${lastName?.[0] || ''}`.toUpperCase()

async function confirmDelete() {
  showDeleteConfirmation.value = false
  try {
    formLoading.value = true
    await api.deleteCompany(company.value.id)
    emitToast(t('globals.messages.deletedSuccessfully'))
    router.push({ name: 'companies' })
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await api.updateCompany(company.value.id, toCompanyPayload(values))
    await fetchCompany()
    emitToast(t('globals.messages.savedSuccessfully'))
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
})

function emitToast(description) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description })
}

function showError(err) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(err).message
  })
}
</script>
