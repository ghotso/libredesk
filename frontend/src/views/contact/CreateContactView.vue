<template>
  <div class="max-w-2xl space-y-6">
    <CustomBreadcrumb :links="breadcrumbLinks" />
    <h1 class="text-xl font-semibold">{{ t('contact.newContact') }}</h1>
    <form @submit.prevent="onSubmit" class="space-y-6">
      <div class="flex flex-wrap gap-6">
        <div class="flex-1 min-w-[200px]">
          <FormField v-slot="{ componentField }" name="first_name">
            <FormItem>
              <FormLabel>{{ t('globals.terms.firstName') }}</FormLabel>
              <FormControl>
                <Input v-bind="componentField" type="text" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
        <div class="flex-1 min-w-[200px]">
          <FormField v-slot="{ componentField }" name="last_name">
            <FormItem>
              <FormLabel>{{ t('globals.terms.lastName') }}</FormLabel>
              <FormControl>
                <Input v-bind="componentField" type="text" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
      </div>
      <FormField v-slot="{ componentField }" name="email">
        <FormItem>
          <FormLabel>{{ t('globals.terms.email') }}</FormLabel>
          <FormControl>
            <Input v-bind="componentField" type="email" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
      <div class="flex flex-wrap gap-6">
        <div class="flex-1 min-w-[120px]">
          <FormField v-slot="{ componentField }" name="phone_number_country_code">
            <FormItem>
              <FormLabel>{{ t('globals.terms.phoneNumber') }}</FormLabel>
              <FormControl>
                <Select v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue :placeholder="t('globals.messages.select', { name: '' })" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="c in countryOptions" :key="c.value" :value="c.value">
                      {{ c.label }} ({{ c.calling_code }})
                    </SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
            </FormItem>
          </FormField>
        </div>
        <div class="flex-[2] min-w-[180px]">
          <FormField v-slot="{ componentField }" name="phone_number">
            <FormItem class="pt-8">
              <FormControl>
                <Input v-bind="componentField" type="tel" inputmode="numeric" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
      </div>

      <div class="border rounded-lg p-4 space-y-4">
        <FormLabel class="text-base">{{ t('globals.terms.organization') }}</FormLabel>
        <div class="flex flex-col gap-3">
          <label class="flex items-center gap-2">
            <input type="radio" v-model="orgMode" value="none" class="rounded" />
            <span>{{ t('contact.noOrganization') }}</span>
          </label>
          <template v-if="canManageOrganizations">
            <label class="flex items-center gap-2">
              <input type="radio" v-model="orgMode" value="existing" class="rounded" />
              <span>{{ t('contact.addToExistingOrganization') }}</span>
            </label>
          </template>
          <div v-if="orgMode === 'existing'" class="ml-6">
            <Select v-model="form.organization_id">
              <SelectTrigger class="w-full max-w-sm">
                <SelectValue :placeholder="t('globals.messages.select', { name: t('globals.terms.organization') })" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="org in organizations" :key="org.id" :value="org.id">
                  {{ org.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <label class="flex items-center gap-2">
            <input type="radio" v-model="orgMode" value="new" class="rounded" />
            <span>{{ t('contact.createNewOrganization') }}</span>
          </label>
          <div v-if="orgMode === 'new'" class="ml-6">
            <Input v-model="form.create_organization_name" type="text" :placeholder="t('admin.organizations.name')" class="max-w-sm" />
          </div>
        </div>
      </div>

      <div class="flex gap-2">
        <Button type="submit" :disabled="formLoading">{{ t('globals.messages.create', { name: t('globals.terms.contact') }) }}</Button>
        <Button type="button" variant="outline" @click="router.push({ name: 'contacts' })">{{ t('globals.terms.cancel') }}</Button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { useI18n } from 'vue-i18n'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import countries from '@/constants/countries.js'

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const userStore = useUserStore()
const canManageOrganizations = computed(() => userStore.can('organizations:manage'))
const organizations = ref([])
const orgMode = ref('none')
const formLoading = ref(false)

const breadcrumbLinks = [
  { path: { name: 'contacts' }, label: t('globals.terms.contact', 2) },
  { path: '', label: t('contact.newContact') }
]

const form = reactive({
  organization_id: null,
  create_organization_name: '',
  share_tickets_by_default: false
})

const schema = toTypedSchema(z.object({
  first_name: z.string().min(1, { message: t('globals.messages.required') }),
  last_name: z.string().optional(),
  email: z.string().min(1).email(t('globals.messages.invalidEmailAddress')),
  phone_number: z.string().optional(),
  phone_number_country_code: z.string().optional().nullable()
}))

const { handleSubmit } = useForm({ validationSchema: schema })

const countryOptions = countries.map((c) => ({
  label: c.name,
  value: c.iso_2,
  calling_code: c.calling_code
}))

onMounted(async () => {
  if (!userStore.can('organizations:manage')) return
  try {
    const res = await api.getOrganizations()
    organizations.value = res.data?.data ?? []
  } catch (_) {
    organizations.value = []
  }
})

watch(orgMode, (mode) => {
  if (mode === 'none') {
    form.organization_id = null
    form.create_organization_name = ''
  }
})

const onSubmit = handleSubmit(async (values) => {
  formLoading.value = true
  try {
    const payload = {
      email: values.email,
      first_name: values.first_name,
      last_name: values.last_name || '',
      phone_number: values.phone_number || '',
      phone_number_country_code: values.phone_number_country_code || '',
      share_tickets_by_default: form.share_tickets_by_default
    }
    if (orgMode.value === 'existing' && form.organization_id) {
      payload.organization_id = form.organization_id
    }
    if (orgMode.value === 'new' && form.create_organization_name?.trim()) {
      payload.create_organization_name = form.create_organization_name.trim()
    }
    const res = await api.createContact(payload)
    const contact = res.data?.data ?? res.data
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', { name: t('globals.terms.contact') })
    })
    router.push({ name: 'contact-detail', params: { id: contact.id } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})
</script>
