<template>
  <div class="space-y-4">
    <h1 class="text-xl font-semibold">{{ t('portal.myTickets') }}</h1>
    <div v-if="loading" class="text-muted-foreground">{{ t('globals.messages.loading') }}</div>
    <div v-else-if="list.length === 0" class="text-muted-foreground py-8 text-center">
      {{ t('portal.noTickets') }}
    </div>
    <div v-else class="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ t('portal.reference') }}</TableHead>
            <TableHead>{{ t('portal.subject') }}</TableHead>
            <TableHead>{{ t('portal.status') }}</TableHead>
            <TableHead>{{ t('portal.lastUpdate') }}</TableHead>
            <TableHead></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="c in list" :key="c.uuid">
            <TableCell>{{ c.reference_number }}</TableCell>
            <TableCell>{{ c.subject || '–' }}</TableCell>
            <TableCell>{{ c.status }}</TableCell>
            <TableCell>{{ formatDate(c.last_message_at) }}</TableCell>
            <TableCell>
              <Button variant="ghost" size="sm" as-child>
                <router-link :to="{ name: 'portal-ticket-detail', params: { uuid: c.uuid } }">
                  {{ t('portal.view') }}
                </router-link>
              </Button>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import Table from '@/components/ui/table/Table.vue'
import TableBody from '@/components/ui/table/TableBody.vue'
import TableCell from '@/components/ui/table/TableCell.vue'
import TableHead from '@/components/ui/table/TableHead.vue'
import TableHeader from '@/components/ui/table/TableHeader.vue'
import TableRow from '@/components/ui/table/TableRow.vue'
import { Button } from '@/components/ui/button'

const { t } = useI18n()
const list = ref([])
const loading = ref(true)

function formatDate (v) {
  if (!v) return '–'
  return new Date(v).toLocaleString()
}

onMounted(async () => {
  try {
    const { data } = await api.portalGetConversations({ page: 1, page_size: 50 })
    list.value = data.conversations || []
  } catch (_) {
    list.value = []
  } finally {
    loading.value = false
  }
})
</script>
