<template>
  <div class="space-y-4 max-w-3xl">
    <Button variant="ghost" size="sm" as-child>
      <router-link to="/portal/tickets">{{ t('portal.backToTickets') }}</router-link>
    </Button>
    <div v-if="loading" class="text-muted-foreground">{{ t('globals.messages.loading') }}</div>
    <template v-else-if="conversation">
      <Card>
        <CardHeader>
          <CardTitle class="text-lg">#{{ conversation.reference_number }} – {{ conversation.subject || '–' }}</CardTitle>
          <p class="text-sm text-muted-foreground">{{ t('portal.status') }}: {{ conversation.status }}</p>
        </CardHeader>
      </Card>
      <div class="space-y-2">
        <h2 class="font-medium">{{ t('portal.messages') }}</h2>
        <div class="space-y-3 border rounded-lg p-4 bg-muted/30">
          <div v-for="m in messages" :key="m.uuid" class="flex flex-col gap-1">
            <span class="text-xs text-muted-foreground">{{ m.sender_type }} · {{ formatDate(m.created_at) }}</span>
            <p class="text-sm whitespace-pre-wrap">{{ m.content }}</p>
          </div>
        </div>
      </div>
      <Card v-if="conversation.status_id !== 4">
        <CardContent class="p-4 space-y-3">
          <Label>{{ t('portal.reply') }}</Label>
          <textarea v-model="replyText" class="w-full min-h-[80px] rounded-md border bg-background px-3 py-2 text-sm" :placeholder="t('portal.replyPlaceholder')" />
          <div class="flex gap-2">
            <Button @click="sendReply" :disabled="!replyText.trim() || sending">{{ t('portal.sendReply') }}</Button>
            <Button variant="outline" @click="showCloseDialog = true">{{ t('portal.closeTicket') }}</Button>
          </div>
        </CardContent>
      </Card>
      <Dialog v-model:open="showCloseDialog">
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{{ t('portal.closeTicket') }}</DialogTitle>
            <DialogDescription>{{ t('portal.closeTicketDescription') }}</DialogDescription>
          </DialogHeader>
          <div class="space-y-2 py-2">
            <Label>{{ t('portal.closingComment') }}</Label>
            <textarea v-model="closeComment" class="w-full min-h-[60px] rounded-md border bg-background px-3 py-2 text-sm" required />
          </div>
          <DialogFooter>
            <Button variant="outline" @click="showCloseDialog = false">{{ t('globals.terms.cancel') }}</Button>
            <Button @click="closeTicket" :disabled="!closeComment.trim() || closing">{{ t('portal.closeTicket') }}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'

const { t } = useI18n()
const route = useRoute()
const uuid = computed(() => route.params.uuid)
const conversation = ref(null)
const messages = ref([])
const loading = ref(true)
const replyText = ref('')
const sending = ref(false)
const showCloseDialog = ref(false)
const closeComment = ref('')
const closing = ref(false)

function formatDate (v) {
  if (!v) return '–'
  return new Date(v).toLocaleString()
}

onMounted(async () => {
  try {
    const { data } = await api.portalGetConversation(uuid.value)
    conversation.value = data.conversation
    messages.value = data.messages || []
  } catch (_) {
    conversation.value = null
  } finally {
    loading.value = false
  }
})

async function sendReply () {
  if (!replyText.value.trim()) return
  sending.value = true
  try {
    await api.portalSendMessage(uuid.value, { message: replyText.value })
    messages.value.push({ content: replyText.value, sender_type: 'contact', created_at: new Date().toISOString() })
    replyText.value = ''
  } finally {
    sending.value = false
  }
}

async function closeTicket () {
  if (!closeComment.value.trim()) return
  closing.value = true
  try {
    await api.portalCloseConversation(uuid.value, { comment: closeComment.value })
    if (conversation.value) {
      conversation.value.status_id = 4
      conversation.value.status = conversation.value.status || 'Closed'
    }
    showCloseDialog.value = false
    closeComment.value = ''
  } finally {
    closing.value = false
  }
}
</script>
