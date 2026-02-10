<template>
  <div class="w-full space-y-6 pb-8">
    <div class="flex items-center justify-between mb-4">
      <span class="text-xl font-semibold text-gray-900 dark:text-foreground">
        {{ t('globals.terms.organization', 2) }}
      </span>
      <Button
        v-if="canManageOrganizations"
        variant="outline"
        size="sm"
        @click="assignModalOpen = true"
      >
        {{ t('admin.organizations.assignToOrganization') }}
      </Button>
    </div>

    <div class="h-20 flex items-center" v-if="isLoading">
      <Spinner />
    </div>

    <template v-else>
      <div v-if="memberships.length === 0" class="rounded-lg border border-dashed p-8 text-center text-muted-foreground">
        <p class="text-sm">{{ t('admin.organizations.noOrganizationsAssigned') }}</p>
        <Button
          v-if="canManageOrganizations"
          variant="outline"
          class="mt-3"
          @click="assignModalOpen = true"
        >
          {{ t('admin.organizations.assignToOrganization') }}
        </Button>
      </div>
      <ul v-else class="space-y-2">
        <li
          v-for="m in memberships"
          :key="m.organization_id"
          class="flex items-center justify-between gap-4 rounded border p-3"
        >
          <router-link
            :to="{ name: 'organization-detail', params: { id: String(m.organization_id) } }"
            class="font-medium text-primary hover:underline"
          >
            {{ m.organization_name }}
          </router-link>
          <Button
            v-if="canManageOrganizations"
            type="button"
            variant="ghost"
            size="sm"
            @click="removeFromOrg(m.organization_id)"
          >
            {{ t('admin.organizations.remove') }}
          </Button>
        </li>
      </ul>
    </template>

    <!-- Assign to Organization modal -->
    <Dialog :open="assignModalOpen" @update:open="assignModalOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('admin.organizations.assignToOrganization') }}</DialogTitle>
          <DialogDescription>
            {{ t('admin.organizations.assignToOrganizationHelp') }}
          </DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-4">
          <div class="space-y-2">
            <Label>{{ t('admin.organizations.searchOrCreate') }}</Label>
            <Input
              v-model="assignSearchQuery"
              type="text"
              :placeholder="t('admin.organizations.searchByName')"
              @input="onAssignSearchInput"
            />
          </div>
          <div class="flex gap-2">
            <RadioGroup v-model="assignMode" class="flex gap-4">
              <label class="flex items-center gap-2 cursor-pointer">
                <RadioGroupItem value="existing" />
                <span class="text-sm">{{ t('contact.addToExistingOrganization') }}</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <RadioGroupItem value="new" />
                <span class="text-sm">{{ t('contact.createNewOrganization') }}</span>
              </label>
            </RadioGroup>
          </div>
          <div v-if="assignMode === 'existing'" class="max-h-48 overflow-auto space-y-1 border rounded p-2">
            <div
              v-for="org in assignFilteredOrgs"
              :key="org.id"
              class="flex items-center justify-between py-2 px-2 rounded hover:bg-muted"
            >
              <span class="text-sm">{{ org.name }}</span>
              <Button type="button" size="sm" @click="assignToOrg(org.id)">{{ t('admin.organizations.assignButton') }}</Button>
            </div>
            <p v-if="assignFilteredOrgs.length === 0" class="text-sm text-muted-foreground py-2">{{ t('admin.organizations.noOrganizationsFound') }}</p>
          </div>
          <div v-if="assignMode === 'new'" class="space-y-2">
            <Label>{{ t('admin.organizations.name') }}</Label>
            <Input v-model="assignNewOrgName" type="text" :placeholder="t('admin.organizations.name')" />
            <Button
              type="button"
              @click="createAndAssignOrg"
              :disabled="!assignNewOrgName?.trim()"
            >
              {{ t('contact.createNewOrganization') }} & {{ t('globals.messages.assign') }}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const props = defineProps({
  contactId: { type: Number, required: true },
  initialMemberships: { type: Array, default: null },
  initialAllOrganizations: { type: Array, default: null }
})

const { t } = useI18n()
const userStore = useUserStore()
const emitter = useEmitter()

const isLoading = ref(true)
const memberships = ref([])
const allOrganizations = ref([])
const assignModalOpen = ref(false)
const assignSearchQuery = ref('')
const assignMode = ref('existing')
const assignNewOrgName = ref('')
let assignSearchDebounce = null

const canManageOrganizations = computed(() => userStore.can('organizations:manage'))

const orgsNotIn = computed(() => {
  const inIds = new Set(memberships.value.map((m) => m.organization_id))
  return allOrganizations.value.filter((o) => !inIds.has(o.id))
})

const assignFilteredOrgs = computed(() => {
  const q = assignSearchQuery.value?.trim().toLowerCase()
  let list = orgsNotIn.value
  if (q) {
    list = list.filter((o) => (o.name || '').toLowerCase().includes(q))
  }
  return list.slice(0, 20)
})

async function fetchMemberships() {
  try {
    const res = await api.getContactOrganizations(props.contactId)
    memberships.value = res?.data?.data ?? []
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

async function fetchOrganizations() {
  if (!canManageOrganizations.value) return
  try {
    const res = await api.getOrganizations()
    allOrganizations.value = res?.data?.data ?? []
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

function onAssignSearchInput() {
  clearTimeout(assignSearchDebounce)
  assignSearchDebounce = setTimeout(() => {}, 200)
}

function removeFromOrg(organizationId) {
  api
    .removeOrganizationMember(organizationId, props.contactId)
    .then(() => {
      memberships.value = memberships.value.filter((m) => m.organization_id !== organizationId)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.deletedSuccessfully', { name: 'Member' }) })
    })
    .catch((e) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message }))
}

async function assignToOrg(orgId) {
  try {
    await api.addOrganizationMember(orgId, { contact_id: props.contactId, share_tickets_by_default: false })
    await fetchMemberships()
    assignModalOpen.value = false
    assignSearchQuery.value = ''
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.updatedSuccessfully', { name: t('globals.terms.organization') }) })
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

async function createAndAssignOrg() {
  const name = assignNewOrgName.value?.trim()
  if (!name) return
  try {
    const res = await api.createOrganization({ name, description: '' })
    const id = res?.data?.data?.id
    if (id) {
      await assignToOrg(id)
      assignNewOrgName.value = ''
    }
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

onMounted(async () => {
  if (props.initialMemberships != null) {
    memberships.value = props.initialMemberships
    if (props.initialAllOrganizations != null) {
      allOrganizations.value = props.initialAllOrganizations
    } else if (canManageOrganizations.value) {
      await fetchOrganizations()
    }
    isLoading.value = false
    return
  }
  isLoading.value = true
  await Promise.all([fetchMemberships(), fetchOrganizations()])
  isLoading.value = false
})

watch(() => props.contactId, () => {
  if (props.initialMemberships == null) fetchMemberships()
})

watch(assignModalOpen, (open) => {
  if (open) {
    assignSearchQuery.value = ''
    assignMode.value = 'existing'
    assignNewOrgName.value = ''
  }
})
</script>
