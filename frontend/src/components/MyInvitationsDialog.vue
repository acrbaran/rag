<template>
  <!-- "My invitations" inbox as a dialog (used by workspace onboarding).
       The sidebar bell shows the same list inside its popover. -->
  <t-dialog v-model:visible="visibleModel" :header="$t('tenantInvitation.myInbox.title')" :footer="false" width="540px"
    :close-on-overlay-click="true">
    <MyInvitationsList :active="visible" />
  </t-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import MyInvitationsList from '@/components/MyInvitationsList.vue'

// v-model:visible — the parent owns the open/close state. The list
// reloads on every open transition.
const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ (e: 'update:visible', v: boolean): void }>()
const visibleModel = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v),
})
</script>
