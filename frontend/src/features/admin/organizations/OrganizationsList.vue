<template>
  <div class="w-full h-full flex flex-col min-h-0">
    <!-- Main Content Area (scrollable list) -->
    <div class="flex-1 min-h-0 overflow-auto flex flex-col gap-4 pb-4">
      <div class="flex items-center justify-between gap-4">
        <div class="flex items-center gap-4">
          <Input
            type="text"
            v-model="searchTerm"
            :placeholder="t('admin.organizations.searchByName')"
            @input="onSearchInput"
          />
          <Popover>
            <PopoverTrigger>
              <Button variant="outline" size="sm" class="flex items-center h-8">
                <ArrowDownWideNarrow size="18" class="text-muted-foreground cursor-pointer" />
              </Button>
            </PopoverTrigger>
            <PopoverContent class="w-[200px] p-4 flex flex-col gap-4">
              <Select v-model="orderBy" @update:model-value="applySort">
                <SelectTrigger class="h-8 w-full">
                  <SelectValue :placeholder="orderBy" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="name_asc">{{ t('admin.organizations.orderByNameAsc') }}</SelectItem>
                  <SelectItem value="name_desc">{{ t('admin.organizations.orderByNameDesc') }}</SelectItem>
                  <SelectItem value="updated_desc">{{ t('admin.organizations.orderByUpdatedDesc') }}</SelectItem>
                  <SelectItem value="updated_asc">{{ t('admin.organizations.orderByUpdatedAsc') }}</SelectItem>
                </SelectContent>
              </Select>
            </PopoverContent>
          </Popover>
        </div>
        <Button asChild>
          <router-link :to="{ name: 'new-organization' }">
            {{ t('admin.organizations.newOrganization') }}
          </router-link>
        </Button>
      </div>

      <div v-if="loading" class="flex flex-col gap-4 w-full">
        <Card v-for="i in Math.min(perPage, 5)" :key="i" class="p-4 flex-shrink-0">
          <div class="flex items-center gap-4">
            <Skeleton class="h-10 w-10 rounded-full" />
            <div class="space-y-2">
              <Skeleton class="h-3 w-[160px]" />
              <Skeleton class="h-3 w-[140px]" />
            </div>
          </div>
        </Card>
      </div>

      <template v-else>
        <Card
          v-for="org in paginatedOrganizations"
          :key="org.id"
          class="p-4 w-full hover:bg-accent/50 cursor-pointer"
          @click="$router.push({ name: 'organization-detail', params: { id: org.id } })"
        >
          <div class="flex items-center gap-4">
            <div
              class="h-10 w-10 rounded-full border flex items-center justify-center bg-muted text-sm font-medium shrink-0"
            >
              {{ getInitials(org.name) }}
            </div>
            <div class="space-y-1 overflow-hidden min-w-0">
              <h4 class="text-sm font-semibold truncate">
                {{ org.name }}
              </h4>
              <p class="text-xs text-muted-foreground truncate">
                {{ org.description || '—' }}
              </p>
            </div>
          </div>
        </Card>
        <div v-if="paginatedOrganizations.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">{{ t('admin.organizations.noOrganizationsFound') }}</p>
        </div>
      </template>
    </div>

    <div class="flex-shrink-0 bg-background p-4 border-t">
      <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
        <div class="flex items-center gap-2">
          <span class="text-sm text-muted-foreground"> Page {{ page }} of {{ totalPages }} </span>
          <Select v-model="perPage" @update:model-value="handlePerPageChange">
            <SelectTrigger class="h-8 w-[70px]">
              <SelectValue :placeholder="perPage" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="15">15</SelectItem>
              <SelectItem :value="30">30</SelectItem>
              <SelectItem :value="50">50</SelectItem>
              <SelectItem :value="100">100</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Pagination>
          <PaginationList class="flex items-center gap-1">
            <PaginationListItem>
              <PaginationFirst
                :class="{ 'cursor-not-allowed opacity-50': page === 1 }"
                @click.prevent="page > 1 ? goToPage(1) : null"
              />
            </PaginationListItem>
            <PaginationListItem>
              <PaginationPrev
                :class="{ 'cursor-not-allowed opacity-50': page === 1 }"
                @click.prevent="page > 1 ? goToPage(page - 1) : null"
              />
            </PaginationListItem>
            <template v-for="pageNumber in visiblePages" :key="pageNumber">
              <PaginationListItem v-if="pageNumber === '...'">
                <PaginationEllipsis />
              </PaginationListItem>
              <PaginationListItem v-else>
                <Button
                  :is-active="pageNumber === page"
                  @click.prevent="goToPage(pageNumber)"
                  :variant="pageNumber === page ? 'default' : 'outline'"
                >
                  {{ pageNumber }}
                </Button>
              </PaginationListItem>
            </template>
            <PaginationListItem>
              <PaginationNext
                :class="{ 'cursor-not-allowed opacity-50': page === totalPages }"
                @click.prevent="page < totalPages ? goToPage(page + 1) : null"
              />
            </PaginationListItem>
            <PaginationListItem>
              <PaginationLast
                :class="{ 'cursor-not-allowed opacity-50': page === totalPages }"
                @click.prevent="page < totalPages ? goToPage(totalPages) : null"
              />
            </PaginationListItem>
          </PaginationList>
        </Pagination>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Pagination,
  PaginationEllipsis,
  PaginationFirst,
  PaginationLast,
  PaginationList,
  PaginationListItem,
  PaginationNext,
  PaginationPrev
} from '@/components/ui/pagination'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { ArrowDownWideNarrow } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { getVisiblePages } from '@/utils/pagination'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()

const allOrganizations = ref([])
const loading = ref(false)
const searchTerm = ref('')
const orderBy = ref('updated_desc')
const page = ref(1)
const perPage = ref(15)

const filteredOrganizations = computed(() => {
  let list = [...allOrganizations.value]
  const q = searchTerm.value?.trim().toLowerCase()
  if (q) {
    list = list.filter(
      (o) =>
        o.name?.toLowerCase().includes(q) ||
        (o.description && o.description.toLowerCase().includes(q))
    )
  }
  if (orderBy.value === 'name_asc') list.sort((a, b) => (a.name || '').localeCompare(b.name || ''))
  else if (orderBy.value === 'name_desc') list.sort((a, b) => (b.name || '').localeCompare(a.name || ''))
  else if (orderBy.value === 'updated_asc') list.sort((a, b) => new Date(a.updated_at) - new Date(b.updated_at))
  else list.sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at))
  return list
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredOrganizations.value.length / perPage.value)))

const paginatedOrganizations = computed(() => {
  const list = filteredOrganizations.value
  const start = (page.value - 1) * perPage.value
  return list.slice(start, start + perPage.value)
})

const visiblePages = computed(() => getVisiblePages(page.value, totalPages.value))

const onSearchInput = useDebounceFn(() => {
  page.value = 1
}, 300)

function applySort() {
  page.value = 1
}

function getInitials(name) {
  if (!name || !name.trim()) return '?'
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase().slice(0, 2)
  return name.slice(0, 2).toUpperCase()
}

function goToPage(p) {
  page.value = p
}

function handlePerPageChange(val) {
  perPage.value = val
  page.value = 1
}

async function fetchOrganizations() {
  loading.value = true
  try {
    const res = await api.getOrganizations()
    allOrganizations.value = res?.data?.data ?? []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchOrganizations()
})

watch(
  () => filteredOrganizations.value.length,
  () => {
    if (page.value > totalPages.value) page.value = Math.max(1, totalPages.value)
  }
)
</script>
