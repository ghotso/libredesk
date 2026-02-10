<template>
  <AdminPageWithHelp>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <GeneralSettingForm
          v-if="hasLoaded"
          :key="formKey"
          :submitForm="submitForm"
          :initial-values="initialValues"
        />
        <Spinner v-if="isLoading" />
      </div>
    </template>
    <template #help>
      <p>General settings for your support desk like timezone, working hours, etc.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Spinner } from '@/components/ui/spinner'
import GeneralSettingForm from '@/features/admin/general/GeneralSettingForm.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { useAppSettingsStore } from '@/stores/appSettings'
import api from '@/api'

const initialValues = ref({})
const isLoading = ref(true)
const hasLoaded = ref(false)
const formKey = ref(0)
const settingsStore = useAppSettingsStore()

onMounted(async () => {
  try {
    const response = await api.getSettings('general')
    const data = response?.data?.data ?? response?.data ?? {}
    settingsStore.setSettings(data)
    initialValues.value = Object.keys(data).reduce((acc, key) => {
      const newKey = key.replace(/^app\./, '')
      acc[newKey] = data[key]
      return acc
    }, {})
    formKey.value = Object.keys(initialValues.value).length
  } catch (_) {
    initialValues.value = {}
  } finally {
    isLoading.value = false
    hasLoaded.value = true
  }
})

const submitForm = async (values) => {
  // Prepend keys with `app.`
  const updatedValues = Object.fromEntries(
    Object.entries(values).map(([key, value]) => [`app.${key}`, value])
  )
  await api.updateSettings('general', updatedValues)
}
</script>
