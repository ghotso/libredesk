<template>
  <div class="min-h-screen flex flex-col bg-background">
    <header class="border-b px-4 py-3 flex items-center justify-between">
      <div class="flex items-center gap-4">
        <router-link to="/portal/tickets" class="font-semibold text-foreground hover:underline">
          {{ siteName }}
        </router-link>
        <nav class="flex gap-3">
          <router-link
            to="/portal/tickets"
            class="text-sm text-muted-foreground hover:text-foreground"
            active-class="text-foreground font-medium"
          >
            {{ t('portal.myTickets') }}
          </router-link>
          <router-link
            to="/portal/tickets/new"
            class="text-sm text-muted-foreground hover:text-foreground"
          >
            {{ t('portal.newTicket') }}
          </router-link>
        </nav>
      </div>
      <Button variant="ghost" size="sm" @click="logout">
        {{ t('portal.logout') }}
      </Button>
    </header>
    <main class="flex-1 p-4">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppSettingsStore } from '@/stores/appSettings'
import api from '@/api'
import { Button } from '@/components/ui/button'

const { t } = useI18n()
const router = useRouter()
const appSettingsStore = useAppSettingsStore()

const siteName = computed(() => appSettingsStore.public_config?.['app.site_name'] || 'Portal')

async function logout () {
  try {
    await api.portalLogout()
  } catch (_) {}
  router.push({ name: 'portal-login' })
}
</script>
