<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="portal_enabled">
      <FormItem class="flex flex-row items-center justify-between rounded-lg border p-4">
        <div class="space-y-0.5">
          <FormLabel>{{ t('admin.general.portalEnabled') }}</FormLabel>
          <FormDescription>{{ t('admin.general.portalEnabled.description') }}</FormDescription>
        </div>
        <FormControl>
          <Switch v-bind="componentField" :checked="componentField.modelValue" @update:checked="(v) => componentField['onUpdate:modelValue'](v)" />
        </FormControl>
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="portal_default_inbox_id">
      <FormItem>
        <FormLabel>{{ t('admin.general.portalDefaultInbox') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="t('admin.general.portalDefaultInbox.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem :value="0">{{ t('admin.general.portalDefaultInbox.none') }}</SelectItem>
                <SelectItem v-for="inbox in inboxes" :key="inbox.id" :value="inbox.id">
                  {{ inbox.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ t('admin.general.portalDefaultInbox.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :disabled="formLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { watch, ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createPortalFormSchema } from './portalFormSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const emitter = useEmitter()
const { t } = useI18n()
const inboxes = ref([])
const formLoading = ref(false)
const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: ''
  }
})

const submitLabel = props.submitLabel || t('globals.messages.save')
const form = useForm({
  validationSchema: toTypedSchema(createPortalFormSchema(t))
})

onMounted(() => {
  fetchInboxes()
})

const fetchInboxes = async () => {
  try {
    const response = await api.getInboxes()
    inboxes.value = response.data?.data ?? []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await props.submitForm(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('admin.portal')
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues || {}).length === 0) return
    form.setValues(newValues)
  },
  { deep: true }
)
</script>
