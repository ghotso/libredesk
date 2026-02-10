<template>
  <div class="space-y-2">
    <div class="flex justify-between items-start">
      <Avatar class="size-20">
        <AvatarImage :src="conversation?.contact?.avatar_url || ''" />
        <AvatarFallback>
          {{ conversation?.contact?.first_name?.toUpperCase().substring(0, 2) }}
        </AvatarFallback>
      </Avatar>
      <Button
        variant="ghost"
        size="icon"
        class="h-7 w-7"
        @click="emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)"
      >
        <ViewVerticalIcon />
      </Button>
    </div>

    <div class="h-6 flex items-center gap-2">
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-24 h-4" />
      </span>
      <span v-else>
        {{ conversation?.contact?.first_name + ' ' + conversation?.contact?.last_name }}
      </span>
      <ExternalLink
        v-if="!conversationStore.conversation.loading && userStore.can('contacts:read')"
        size="16"
        class="text-muted-foreground cursor-pointer flex-shrink-0"
        @click="$router.push({ name: 'contact-detail', params: { id: conversation?.contact_id } })"
      />
    </div>
    <div
      v-if="!conversationStore.conversation.loading && displayOrganization"
      class="text-sm text-muted-foreground flex gap-2 items-center"
    >
      <Building2 size="16" class="flex-shrink-0" />
      <a
        :href="organizationLink"
        target="_blank"
        rel="noopener noreferrer"
        class="text-primary hover:underline break-all"
      >
        {{ displayOrganization.organization_name }}
      </a>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Mail size="16" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else class="break-all">
        {{ conversation?.contact?.email }}
      </span>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Phone size="16" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else>
        {{ phoneNumber }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ViewVerticalIcon } from '@radix-icons/vue'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Mail, Phone, ExternalLink, Building2 } from 'lucide-vue-next'
import countries from '@/constants/countries.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@/components/ui/skeleton'
import { useUserStore } from '@/stores/user'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const router = useRouter()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()
const userStore = useUserStore()

const contactOrganization = ref(null)

// Use org from conversation when present (from getConversation), so the link appears with no extra request
const contactOrganizationFromConversation = computed(() => {
  const c = conversation.value
  const id = c?.contact_organization_id
  const name = c?.contact_organization_name
  if (id != null && id !== '' && name != null && name !== '') {
    return { organization_id: Number(id), organization_name: name }
  }
  return null
})

const displayOrganization = computed(() => contactOrganizationFromConversation.value ?? contactOrganization.value)

const organizationLink = computed(() => {
  if (!displayOrganization.value) return ''
  const route = router.resolve({ name: 'organization-detail', params: { id: String(displayOrganization.value.organization_id) } })
  return route.href
})

async function fetchContactOrganization() {
  const contactId = conversation.value?.contact_id
  if (!contactId || !userStore.can('contacts:read')) {
    contactOrganization.value = null
    return
  }
  try {
    const res = await api.getContactOrganizations(contactId)
    const list = res?.data?.data ?? []
    contactOrganization.value = list.length > 0 ? list[0] : null
  } catch {
    contactOrganization.value = null
  }
}

watch(conversation, (c) => {
  contactOrganization.value = null
  if (!c?.contact_id || !userStore.can('contacts:read')) return
  // If conversation already has contact org (from API), no need to fetch
  const fromConv = c.contact_organization_id != null && c.contact_organization_id !== '' && c.contact_organization_name
  if (!fromConv) {
    fetchContactOrganization()
  }
}, { immediate: true })

const phoneNumber = computed(() => {
  const countryCodeValue = conversation.value?.contact?.phone_number_country_code || ''
  const number = conversation.value?.contact?.phone_number || t('conversation.sidebar.notAvailable')
  if (!countryCodeValue) return number

  // Lookup calling code
  const country = countries.find((c) => c.iso_2 === countryCodeValue)
  const callingCode = country ? country.calling_code : countryCodeValue
  return `${callingCode} ${number}`
})
</script>
