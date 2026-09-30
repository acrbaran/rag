<template>
  <RoleMatrixPopover :title="$t('tenantMember.permissions.title')" :description="$t('tenantMember.permissions.desc')"
    :hint="$t('tenantMember.permissions.iconHint')" :roles="roles" :highlight="currentRole" :placement="placement" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TenantRole } from '@/api/tenant/members'
import RoleMatrixPopover, { type RoleMatrixEntry } from './RoleMatrixPopover.vue'

withDefaults(defineProps<{
  /** Vurgulanacak rol ("ben" rozeti); verilmezse hiçbir rol vurgulanmaz. */
  currentRole?: TenantRole | ''
  placement?: 'bottom-start' | 'bottom-end' | 'bottom' | 'top-start' | 'top-end' | 'left' | 'right'
}>(), {
  currentRole: '',
  placement: 'bottom-start',
})

const { t } = useI18n()

// Static role-permissions matrix. The keys reference i18n strings under
// `tenantMember.permissions.*` so each locale can rephrase per culture.
// Keep this aligned with the design-doc §4.3 matrix and the actual
// PR 2 enforcement; if a permission moves between roles, update both
// sides in the same PR.
const PERM_KEYS = ['manageMembers', 'manageTenantConfig', 'manageInfra', 'createOwnKB', 'readAll'] as const
const ROLE_MATRIX: { role: TenantRole; icon: string; has: (typeof PERM_KEYS)[number][] }[] = [
  { role: 'owner', icon: 'user-vip-filled', has: [...PERM_KEYS] },
  { role: 'admin', icon: 'user-safety', has: ['manageInfra', 'createOwnKB', 'readAll'] },
  { role: 'contributor', icon: 'edit', has: ['createOwnKB', 'readAll'] },
  { role: 'viewer', icon: 'browse', has: ['readAll'] },
]

const roles = computed<RoleMatrixEntry[]>(() => ROLE_MATRIX.map(r => ({
  key: r.role,
  label: t(`tenantMember.role.${r.role}`),
  icon: r.icon,
  perms: PERM_KEYS.map(key => ({ label: t(`tenantMember.permissions.${key}`), has: r.has.includes(key) })),
})))
</script>
