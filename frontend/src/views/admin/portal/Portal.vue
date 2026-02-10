<template>
  <AdminPageWithHelp>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <PortalSettingForm
          :submitForm="submitForm"
          :initial-values="initialValues"
        />
        <Spinner v-if="isLoading" />
      </div>
    </template>
    <template #help>
      <p>Enable the customer portal so contacts can sign in and view their tickets. Set the default inbox for new portal tickets and optionally enable organizations for ticket sharing.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Spinner } from '@/components/ui/spinner'
import PortalSettingForm from '@/features/admin/portal/PortalSettingForm.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { useAppSettingsStore } from '@/stores/appSettings'
import api from '@/api'

const initialValues = ref({})
const isLoading = ref(false)
const settingsStore = useAppSettingsStore()

onMounted(async () => {
  isLoading.value = true
  await settingsStore.fetchSettings('general')
  const data = settingsStore.settings
  isLoading.value = false
  const portalKeys = ['app.portal_enabled', 'app.portal_default_inbox_id']
  initialValues.value = portalKeys.reduce((acc, key) => {
    if (data[key] !== undefined) {
      const newKey = key.replace(/^app\./, '')
      acc[newKey] = data[key]
    }
    return acc
  }, {})
})

const submitForm = async (values) => {
  const updatedValues = Object.fromEntries(
    Object.entries(values).map(([key, value]) => [`app.${key}`, value])
  )
  await api.updateSettings('general', updatedValues)
}
</script>
