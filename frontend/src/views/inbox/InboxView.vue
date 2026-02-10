<template>
  <ConversationPlaceholder v-if="['inbox', 'team-inbox', 'view-inbox'].includes(route.name)" />
  <router-view />
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useConversationStore } from '@/stores/conversation'
import { CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const type = computed(() => route.params.type)
const teamID = computed(() => route.params.teamID)
const viewID = computed(() => route.params.viewID)

const conversationStore = useConversationStore()

// Open status is default ID 1; use its current label so renaming works.
const openStatusLabel = () =>
  conversationStore.statusOptions.find((s) => Number(s.value) === 1)?.label ?? 'Open'

// Init conversations list based on route params
onMounted(() => {
  // Fetch list based on type
  if (type.value) {
    if (!conversationStore.getListStatus) {
      conversationStore.setListStatus(openStatusLabel(), false)
    }
    conversationStore.fetchConversationsList(true, type.value)
  }
  if (teamID.value) {
    if (!conversationStore.getListStatus) {
      conversationStore.setListStatus(openStatusLabel(), false)
    }
    conversationStore.fetchConversationsList(
      true,
      CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
      teamID.value
    )
  }
  if (viewID.value) {
    conversationStore.setListStatus('', false)
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], viewID.value)
  }
})

// Refetch when route params change
watch(
  [type, teamID, viewID],
  ([newType, newTeamID, newViewID], [oldType, oldTeamID, oldViewID]) => {
    if (newType !== oldType && newType) {
      if (!conversationStore.getListStatus) {
        conversationStore.setListStatus(openStatusLabel(), false)
      }
      conversationStore.fetchConversationsList(true, newType)
    }
    if (newTeamID !== oldTeamID && newTeamID) {
      if (!conversationStore.getListStatus) {
        conversationStore.setListStatus(openStatusLabel(), false)
      }
      conversationStore.fetchConversationsList(
        true,
        CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
        newTeamID
      )
    }
    if (newViewID !== oldViewID && newViewID) {
      // Empty out list status as views are already filtered.
      conversationStore.setListStatus('', false)
      conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], newViewID)
    }
  }
)
</script>
