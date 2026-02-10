<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
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
</template>

<script setup>
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { useI18n } from 'vue-i18n'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const { t } = useI18n()
const formLoading = ref(false)
const router = useRouter()
const emitter = useEmitter()
const breadcrumbLinks = [
  { path: 'organization-list', label: t('globals.terms.organization', 2) },
  { path: '', label: t('admin.organizations.newOrganization') }
]

const schema = toTypedSchema(
  z.object({
    name: z.string().min(1, { message: t('globals.messages.required') }),
    description: z.string().optional()
  })
)
const form = useForm({ validationSchema: schema })

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    const res = await api.createOrganization({ name: values.name, description: values.description || '' })
    const created = res?.data?.data
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.createdSuccessfully', { name: t('globals.terms.organization') }) })
    if (created?.id) {
      router.push({ name: 'organization-detail', params: { id: String(created.id) } })
    } else {
      router.push({ name: 'organization-list' })
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  } finally {
    formLoading.value = false
  }
})
</script>
