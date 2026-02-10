<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <div v-else class="space-y-8">
    <form @submit.prevent="onSubmit" class="space-y-6 max-w-md">
      <FormField v-slot="{ field }" name="name">
        <FormItem>
          <FormLabel>{{ t('admin.organizations.name') }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="field" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
      <FormField v-slot="{ field }" name="description">
        <FormItem>
          <FormLabel>{{ t('admin.organizations.description') }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="field" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
      <Button type="submit" :disabled="formLoading">{{ t('globals.messages.save') }}</Button>
    </form>

    <div>
      <h3 class="text-lg font-medium mb-3">{{ t('admin.organizations.domains') }}</h3>
      <p class="text-sm text-muted-foreground mb-3">{{ t('admin.organizations.domainsHelp') }}</p>
      <div class="flex gap-2 mb-3 flex-wrap items-end">
        <Input
          v-model="newDomain"
          type="text"
          :placeholder="t('admin.organizations.domainPlaceholder')"
          class="max-w-xs"
          @keydown.enter.prevent="addDomain"
        />
        <Button type="button" @click="addDomain" :disabled="!newDomainTrimmed">{{ t('globals.messages.add') }}</Button>
      </div>
      <ul v-if="domains.length === 0" class="text-muted-foreground text-sm">{{ t('admin.organizations.noDomains') }}</ul>
      <ul v-else class="space-y-2">
        <li
          v-for="d in domains"
          :key="d.domain"
          class="flex items-center justify-between gap-4 rounded border p-3"
        >
          <span class="font-mono text-sm">{{ d.domain }}</span>
          <Button type="button" variant="ghost" size="sm" @click="removeDomain(d.domain)">{{ t('admin.organizations.delete') }}</Button>
        </li>
      </ul>
    </div>

    <div>
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-lg font-medium">{{ t('admin.organizations.members') }}</h3>
        <Button variant="outline" size="sm" @click="assignContactModalOpen = true">
          {{ t('admin.organizations.assignContact') }}
        </Button>
      </div>
      <div v-if="members.length === 0" class="rounded-lg border border-dashed p-8 text-center text-muted-foreground">
        <p class="text-sm">{{ t('admin.organizations.noMembers') }}</p>
        <Button variant="outline" class="mt-3" @click="assignContactModalOpen = true">
          {{ t('admin.organizations.assignContact') }}
        </Button>
      </div>
      <ul v-else class="space-y-2">
        <li
          v-for="m in members"
          :key="m.contact_id"
          class="flex items-center justify-between gap-4 rounded border p-3"
        >
          <router-link
            :to="{ name: 'contact-detail', params: { id: String(m.contact_id) } }"
            class="font-medium text-primary hover:underline"
          >
            {{ m.contact_first_name }} {{ m.contact_last_name }} ({{ m.contact_email || '—' }})
          </router-link>
          <Button type="button" variant="ghost" size="sm" @click="removeMember(m.contact_id)">{{ t('admin.organizations.remove') }}</Button>
        </li>
      </ul>

      <!-- Assign Contact modal -->
      <Dialog :open="assignContactModalOpen" @update:open="assignContactModalOpen = $event">
        <DialogContent class="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{{ t('admin.organizations.assignContact') }}</DialogTitle>
            <DialogDescription>
              {{ t('admin.organizations.assignContactHelp') }}
            </DialogDescription>
          </DialogHeader>
          <div class="space-y-4 py-4">
            <div class="space-y-2">
              <Label>{{ t('admin.organizations.searchContacts') }}</Label>
              <Input
                v-model="contactSearchQuery"
                type="text"
                placeholder="Search by email..."
                @input="onContactSearchInput"
              />
            </div>
            <div class="flex gap-2">
              <RadioGroup v-model="assignContactMode" class="flex gap-4">
                <label class="flex items-center gap-2 cursor-pointer">
                  <RadioGroupItem value="existing" />
                  <span class="text-sm">{{ t('admin.organizations.addExistingContact') }}</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <RadioGroupItem value="new" />
                  <span class="text-sm">{{ t('admin.organizations.createNewContact') }}</span>
                </label>
              </RadioGroup>
            </div>
            <div v-if="assignContactMode === 'existing'" class="max-h-48 overflow-auto space-y-1 border rounded p-2">
              <div
                v-for="c in contactSearchResults"
                :key="c.id"
                class="flex items-center justify-between py-2 px-2 rounded hover:bg-muted"
              >
                <span class="text-sm">{{ c.first_name }} {{ c.last_name }} ({{ c.email || '—' }})</span>
                <Button
                  type="button"
                  size="sm"
                  @click="addContactAsMember(c)"
                  :disabled="members.some(m => m.contact_id === c.id)"
                >
                  {{ t('admin.organizations.assignButton') }}
                </Button>
              </div>
              <p v-if="contactSearchQuery.trim().length < 2" class="text-sm text-muted-foreground py-2">{{ t('admin.organizations.typeToSearchContacts') }}</p>
              <p v-else-if="contactSearchResults.length === 0" class="text-sm text-muted-foreground py-2">{{ t('admin.organizations.noContactsFound') }}</p>
            </div>
            <div v-if="assignContactMode === 'new'" class="space-y-2">
              <Label>{{ t('globals.terms.firstName') }}</Label>
              <Input v-model="newContactFirstName" type="text" />
              <Label>{{ t('globals.terms.lastName') }}</Label>
              <Input v-model="newContactLastName" type="text" />
              <Label>{{ t('globals.terms.email') }}</Label>
              <Input v-model="newContactEmail" type="email" />
              <Button
                type="button"
                @click="createContactAndAssign"
                :disabled="!newContactEmail?.trim() || !newContactFirstName?.trim()"
              >
                {{ t('admin.organizations.createNewContact') }} & {{ t('admin.organizations.assignButton') }}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, nextTick, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { useI18n } from 'vue-i18n'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'
import api from '@/api'

const { t } = useI18n()
const props = defineProps({ id: { type: [String, Number], required: true } })
const emitter = useEmitter()
const org = ref(null)
const members = ref([])
const formLoading = ref(false)
const isLoading = ref(true)
const contactSearchQuery = ref('')
const contactSearchResults = ref([])
const assignContactModalOpen = ref(false)
const assignContactMode = ref('existing')
const newContactFirstName = ref('')
const newContactLastName = ref('')
const newContactEmail = ref('')
const domains = ref([])
const newDomain = ref('')
let searchDebounce = null

const newDomainTrimmed = computed(() => (newDomain.value || '').trim())

const breadcrumbLinks = computed(() => [
  { path: 'organization-list', label: t('globals.terms.organization', 2) },
  { path: '', label: org.value?.name ?? t('admin.organizations.edit') }
])

function onContactSearchInput() {
  clearTimeout(searchDebounce)
  searchDebounce = setTimeout(doContactSearch, 300)
}

function doContactSearch() {
  const q = contactSearchQuery.value?.trim()
  if (!q || q.length < 2) {
    contactSearchResults.value = []
    return
  }
  api.searchContacts({ query: q }).then((res) => {
    const data = res.data?.data ?? []
    contactSearchResults.value = data.filter((u) => u.type === 'contact').slice(0, 20)
  }).catch(() => { contactSearchResults.value = [] })
}

const schema = toTypedSchema(
  z.object({
    name: z.string().min(1, { message: t('globals.messages.required') }),
    description: z.string().optional()
  })
)
const form = useForm({ validationSchema: schema })

async function loadOrganization() {
  try {
    const res = await api.getOrganization(Number(props.id))
    org.value = res.data.data
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

async function loadMembers() {
  try {
    const res = await api.getOrganizationMembers(Number(props.id))
    members.value = res.data.data ?? []
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

async function loadDomains() {
  try {
    const res = await api.getOrganizationDomains(Number(props.id))
    domains.value = res?.data?.data ?? []
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

function addDomain() {
  const domain = newDomainTrimmed.value
  if (!domain) return
  api
    .addOrganizationDomain(Number(props.id), { domain })
    .then(() => {
      loadDomains()
      newDomain.value = ''
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.createdSuccessfully', { name: t('admin.organizations.domain') }) })
    })
    .catch((e) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message }))
}

function removeDomain(domain) {
  api
    .removeOrganizationDomain(Number(props.id), domain)
    .then(() => {
      loadDomains()
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.deletedSuccessfully', { name: t('admin.organizations.domain') }) })
    })
    .catch((e) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message }))
}

function addContactAsMember(contact) {
  if (members.value.some((m) => m.contact_id === contact.id)) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: 'Contact already in organization' })
    return
  }
  api
    .addOrganizationMember(Number(props.id), { contact_id: contact.id, share_tickets_by_default: false })
    .then(() => {
      loadMembers()
      assignContactModalOpen.value = false
      contactSearchResults.value = contactSearchResults.value.filter((c) => c.id !== contact.id)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.createdSuccessfully', { name: 'Member' }) })
    })
    .catch((e) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message }))
}

async function createContactAndAssign() {
  const email = newContactEmail.value?.trim()
  const firstName = newContactFirstName.value?.trim()
  const lastName = newContactLastName.value?.trim()
  if (!email || !firstName) return
  try {
    const res = await api.createContact({
      email,
      first_name: firstName,
      last_name: lastName || '',
      organization_id: Number(props.id),
      share_tickets_by_default: false
    })
    const contact = res?.data?.data
    if (contact?.id) {
      await loadMembers()
      assignContactModalOpen.value = false
      newContactFirstName.value = ''
      newContactLastName.value = ''
      newContactEmail.value = ''
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.createdSuccessfully', { name: t('globals.terms.contact') }) })
    }
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  }
}

function removeMember(contactId) {
  api
    .removeOrganizationMember(Number(props.id), contactId)
    .then(() => {
      loadMembers()
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.deletedSuccessfully', { name: 'Member' }) })
    })
    .catch((e) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message }))
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await api.updateOrganization(org.value.id, { name: values.name, description: values.description ?? '' })
    org.value = { ...org.value, name: values.name, description: values.description }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.updatedSuccessfully', { name: t('globals.terms.organization') }) })
  } catch (e) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(e).message })
  } finally {
    formLoading.value = false
  }
})

watch(assignContactModalOpen, (open) => {
  if (open) {
    contactSearchQuery.value = ''
    contactSearchResults.value = []
    assignContactMode.value = 'existing'
    newContactFirstName.value = ''
    newContactLastName.value = ''
    newContactEmail.value = ''
  }
})

onMounted(async () => {
  isLoading.value = true
  await Promise.all([loadOrganization(), loadMembers(), loadDomains()])
  isLoading.value = false
  await nextTick()
  if (org.value) {
    form.setValues({
      name: org.value.name ?? '',
      description: org.value.description ?? ''
    })
  }
})
</script>
