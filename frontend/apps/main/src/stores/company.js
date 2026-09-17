import { computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { createRemoteLookup } from '@/utils/remote-lookup'
import api from '@/api'

export const useCompanyStore = defineStore('companies', () => {
    const emitter = useEmitter()
    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const lookup = createRemoteLookup({
        fetchRows: (params) => api.getCompaniesCompact(params),
        onError: showFetchError
    })

    const companies = lookup.rows
    const companyOptions = computed(() =>
        companies.value.map((company) => ({
            label: company.name,
            value: String(company.id)
        }))
    )
    const searchCompanyOptions = async (query) =>
        (await lookup.search(query)).map((company) => ({
            label: company.name,
            value: String(company.id)
        }))

    return {
        companies,
        companyOptions,
        fetchCompanies: lookup.fetchFirstPage,
        searchCompanyOptions,
        ensureCompanyIDs: lookup.ensureIDs
    }
})
