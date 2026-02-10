<template>
  <div class="space-y-4 max-w-2xl">
    <Button variant="ghost" size="sm" as-child>
      <router-link to="/portal/tickets">{{ t('portal.backToTickets') }}</router-link>
    </Button>
    <h1 class="text-xl font-semibold">{{ t('portal.newTicket') }}</h1>
    <Card>
      <CardContent class="p-6 space-y-4">
        <div class="space-y-2">
          <Label for="subject">{{ t('portal.subject') }}</Label>
          <Input id="subject" v-model.trim="form.subject" :placeholder="t('portal.subjectPlaceholder')" />
        </div>
        <div class="space-y-2">
          <Label for="content">{{ t('portal.message') }}</Label>
          <textarea id="content" v-model="form.content" class="w-full min-h-[120px] rounded-md border bg-background px-3 py-2 text-sm" :placeholder="t('portal.messagePlaceholder')" required />
        </div>
        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
        <Button @click="submit" :disabled="!form.content.trim() || loading">{{ t('portal.submit') }}</Button>
      </CardContent>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const { t } = useI18n()
const router = useRouter()
const form = ref({ subject: '', content: '' })
const error = ref('')
const loading = ref(false)

async function submit () {
  if (!form.value.content.trim()) return
  error.value = ''
  loading.value = true
  try {
    const { data } = await api.portalCreateConversation({ subject: form.value.subject, content: form.value.content })
    router.push({ name: 'portal-ticket-detail', params: { uuid: data.uuid } })
  } catch (e) {
    error.value = e.response?.data?.message || t('globals.messages.errorCreating', { name: t('globals.terms.conversation') })
  } finally {
    loading.value = false
  }
}
</script>
