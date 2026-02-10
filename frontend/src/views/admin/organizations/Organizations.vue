<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>
    <template #help>
      <p>Group contacts into organizations. Members can share tickets with their organization.</p>
      <p>Per-member "Share my tickets by default" controls whether new tickets are shared with the organization.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
