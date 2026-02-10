<template>
  <div class="min-h-screen flex items-center justify-center bg-background p-4">
    <Card class="w-full max-w-md">
      <CardContent class="p-6 space-y-6">
        <div class="space-y-2 text-center">
          <CardTitle class="text-2xl font-bold">
            {{ t('portal.forgotPasswordTitle') }}
          </CardTitle>
          <p class="text-muted-foreground text-sm">{{ t('portal.forgotPasswordDescription') }}</p>
        </div>
        <form @submit.prevent="submit" class="space-y-4">
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
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <p v-if="success" class="text-sm text-green-600 dark:text-green-400">{{ success }}</p>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ t('portal.forgotPasswordTitle') }}
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
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { Card, CardContent, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'

const { t } = useI18n()
const form = ref({ email: '' })
const error = ref('')
const success = ref('')
const loading = ref(false)

async function submit () {
  error.value = ''
  success.value = ''
  loading.value = true
  try {
    await api.portalForgotPassword({ email: form.value.email })
    success.value = t('portal.forgotPasswordSuccess')
  } catch (e) {
    error.value = e.response?.data?.message || t('globals.messages.error')
  } finally {
    loading.value = false
  }
}
</script>
