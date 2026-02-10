<template>
  <div class="min-h-screen flex items-center justify-center bg-background p-4">
    <Card class="w-full max-w-md">
      <CardContent class="p-6 space-y-6">
        <div class="space-y-2 text-center">
          <CardTitle class="text-2xl font-bold">
            {{ siteName }}
          </CardTitle>
          <p class="text-muted-foreground text-sm">{{ t('portal.signIn') }}</p>
        </div>
        <form @submit.prevent="login" class="space-y-4">
          <div class="space-y-2">
            <Label for="email">{{ t('globals.terms.email') }}</Label>
            <Input
              id="email"
              v-model.trim="form.email"
              type="email"
              autocomplete="email"
              :placeholder="t('auth.enterEmail')"
              required
            />
          </div>
          <div class="space-y-2">
            <Label for="password">{{ t('globals.terms.password') }}</Label>
            <Input
              id="password"
              v-model="form.password"
              type="password"
              autocomplete="current-password"
              :placeholder="t('auth.enterPassword')"
              required
            />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ t('portal.signIn') }}
          </Button>
          <p class="text-center text-sm">
            <RouterLink :to="{ name: 'portal-forgot-password' }" class="text-primary hover:underline">
              {{ t('portal.forgotPassword') }}
            </RouterLink>
          </p>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppSettingsStore } from '@/stores/appSettings'
import api from '@/api'
import { Card, CardContent, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'

const { t } = useI18n()
const router = useRouter()
const appSettingsStore = useAppSettingsStore()

const siteName = computed(() => appSettingsStore.public_config?.['app.site_name'] || 'Portal')

const form = ref({ email: '', password: '' })
const error = ref('')
const loading = ref(false)

async function login () {
  error.value = ''
  loading.value = true
  try {
    await api.portalLogin(form.value)
    router.push({ name: 'portal-tickets' })
  } catch (e) {
    error.value = e.response?.data?.message || t('user.invalidEmailPassword')
  } finally {
    loading.value = false
  }
}
</script>
