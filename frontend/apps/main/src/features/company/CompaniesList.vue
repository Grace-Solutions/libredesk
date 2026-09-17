<template>
  <div class="min-h-screen flex flex-col">
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex items-center gap-4 mb-4 w-full">
        <Input
          type="text"
          class="max-w-sm"
          v-model="searchTerm"
          :placeholder="$t('company.searchByName')"
          @input="fetchCompaniesDebounced"
        />

        <Button
          v-if="userStore.can('companies:write')"
          size="sm"
          class="flex items-center h-8 ml-auto"
          @click="showCreateDialog = true"
        >
          {{ $t('company.new') }}
        </Button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col gap-4 w-full">
        <Card v-for="i in perPage" :key="i" class="p-4 flex-shrink-0">
          <div class="flex items-center gap-4">
            <Skeleton class="h-10 w-10 rounded-full" />
            <div class="space-y-2">
              <Skeleton class="h-3 w-[160px]" />
              <Skeleton class="h-3 w-[140px]" />
            </div>
          </div>
        </Card>
      </div>

      <!-- Loaded State -->
      <template v-else>
        <Card
          v-for="company in companies"
          :key="company.id"
          class="p-4 w-full hover:bg-accent/50 cursor-pointer"
          @click="$router.push({ name: 'company-detail', params: { id: company.id } })"
        >
          <div class="flex items-center gap-4">
            <Avatar class="h-10 w-10 border">
              <AvatarFallback class="text-sm font-medium">
                {{ getInitials(company.name) }}
              </AvatarFallback>
            </Avatar>

            <div class="space-y-1 overflow-hidden flex-1">
              <div class="flex items-center gap-2">
                <h4 class="text-sm font-semibold truncate">{{ company.name }}</h4>
                <Badge v-if="company.parent_name" variant="secondary" class="text-xs px-1.5 py-0">
                  {{ company.parent_name }}
                </Badge>
              </div>
              <div class="flex items-center gap-3 text-xs text-muted-foreground">
                <span class="flex items-center gap-1">
                  <UsersIcon size="12" class="flex-shrink-0" />
                  {{ company.contact_count }}
                </span>
                <span v-if="company.child_count" class="flex items-center gap-1">
                  <Building2Icon size="12" class="flex-shrink-0" />
                  {{ company.child_count }}
                </span>
                <span v-if="company.website" class="truncate">{{ company.website }}</span>
              </div>
            </div>
          </div>
        </Card>
        <div v-if="companies.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">{{ $t('company.noCompaniesFound') }}</p>
        </div>
      </template>
    </div>

    <PaginationBar v-model:page="page" v-model:per-page="perPage" :total-pages="totalPages" />
  </div>

  <CreateCompanyDialog v-model:open="showCreateDialog" @created="onCompanyCreated" />
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Card } from '@shared-ui/components/ui/card'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Avatar, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Badge } from '@shared-ui/components/ui/badge'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { UsersIcon, Building2Icon } from 'lucide-vue-next'
import { useDebounceFn } from '@vueuse/core'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import CreateCompanyDialog from './CreateCompanyDialog.vue'
import { useUserStore } from '@main/stores/user'
import api from '@main/api'

const companies = ref([])
const loading = ref(false)
const page = ref(1)
const perPage = ref(15)
const totalPages = ref(0)
const searchTerm = ref('')
const showCreateDialog = ref(false)
const emitter = useEmitter()
const router = useRouter()
const userStore = useUserStore()
let fetchRequestId = 0

const fetchCompaniesDebounced = useDebounceFn(() => {
  if (page.value === 1) {
    fetchCompanies()
  } else {
    page.value = 1
  }
}, 300)

const fetchCompanies = async () => {
  const requestId = ++fetchRequestId
  loading.value = true
  try {
    const response = await api.getCompanies({
      page: page.value,
      page_size: perPage.value,
      q: searchTerm.value
    })
    if (requestId !== fetchRequestId) return
    companies.value = response.data.data.results
    totalPages.value = response.data.data.total_pages
  } catch (error) {
    if (requestId !== fetchRequestId) return
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (requestId === fetchRequestId) loading.value = false
  }
}

const getInitials = (name) =>
  (name || '')
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0])
    .join('')
    .toUpperCase()

const onCompanyCreated = (company) => {
  router.push({ name: 'company-detail', params: { id: company.id } })
}

watch([page, perPage], fetchCompanies)

onMounted(() => {
  fetchCompanies()
})
</script>
