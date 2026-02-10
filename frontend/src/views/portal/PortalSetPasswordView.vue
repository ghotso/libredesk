<template>
  <div class="min-h-screen flex items-center justify-center bg-background p-4">
    <Card class="w-full max-w-md">
      <CardContent class="p-6 space-y-6">
        <div class="space-y-2 text-center">
          <CardTitle class="text-2xl font-bold">
            {{ t('portal.setPasswordTitle') }}
          </CardTitle>
          <p class="text-muted-foreground text-sm">{{ t('portal.setPasswordDescription') }}</p>
        </div>
        <form @submit.prevent="submit" class="space-y-4">
          <div class="space-y-2">
            <Label for="password">{{ t('globals.terms.password') }}</Label>
            <Input
              id="password"
              v-model="form.password"
              type="password"
              autocomplete="new-password"
              :placeholder="t('auth.enterNewPassword')"
              required
              minlength="8"
            />
          </div>
          <div class="space-y-2">
            <Label for="confirmPassword">{{ t('auth.confirmPassword') }}</Label>
            <Input
              id="confirmPassword"
              v-model="form.confirmPassword"
              type="password"
              autocomplete="new-password"
              :placeholder="t('auth.confirmNewPassword')"
              required
              minlength="8"
            />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ t('auth.setNewPassword') }}
          </Button>
          <p class="text-center text-sm">
            <RouterLink :to="{ name: 'portal-login' }" class="text-primary hover:underline">
              {{ t('portal.signIn') }}
            </RouterLink>
          </p>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { Card, CardContent, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const form = ref({ token: '', password: '', confirmPassword: '' })
const error = ref('')
const loading = ref(false)

onMounted(() => {
  form.value.token = route.query.token || ''
  if (!form.value.token) {
    error.value = t('auth.invalidOrExpiredSession')
  }
})

async function submit () {
  if (!form.value.token) return
  if (form.value.password.length < 8) {
    error.value = t('auth.passwordRequired')
    return
  }
  if (form.value.password !== form.value.confirmPassword) {
    error.value = t('auth.passwordsDoNotMatch')
    return
  }
  error.value = ''
  loading.value = true
  try {
    await api.portalSetPassword({ token: form.value.token, password: form.value.password })
    router.push({ name: 'portal-login' })
  } catch (e) {
    error.value = e.response?.data?.message || t('globals.messages.error')
  } finally {
    loading.value = false
  }
}
</script>
