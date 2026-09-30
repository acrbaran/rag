<template>
  <div class="user-profile">
    <div class="section-header">
      <h2>{{ $t('userProfile.title') }}</h2>
      <p class="section-description">{{ $t('userProfile.description') }}</p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="shell" aria-busy="true">
      <div class="core hero hero--skeleton">
        <t-skeleton animation="gradient" :row-col="[{ type: 'circle', size: '76px' }]" />
        <div class="skeleton-lines">
          <t-skeleton animation="gradient" :row-col="[{ width: '45%', height: '22px' }, { width: '30%' }, { width: '55%' }]" />
        </div>
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">{{ $t('tenant.retry') }}</t-button>
        </template>
      </t-alert>
    </div>

    <template v-else>
      <!-- Identity -->
      <section class="shell" style="--i: 0" aria-labelledby="profile-identity-name">
        <div class="core hero">
          <div class="avatar" aria-hidden="true">
            <span class="avatar-initials">{{ initials }}</span>
          </div>
          <div class="hero-text">
            <h3 id="profile-identity-name" class="hero-name">{{ displayName }}</h3>
            <p v-if="userInfo?.email && userInfo.email !== displayName" class="hero-email">
              <t-icon name="mail" aria-hidden="true" />
              <span>{{ userInfo.email }}</span>
            </p>
            <ul class="hero-meta">
              <li v-if="userInfo?.username" class="meta-chip">
                <t-icon name="user" aria-hidden="true" />
                <span>@{{ userInfo.username }}</span>
              </li>
              <li v-if="memberSince" class="meta-chip">
                <t-icon name="calendar" aria-hidden="true" />
                <span>{{ $t('userProfile.memberSince', { date: memberSince }) }}</span>
              </li>
            </ul>
          </div>
        </div>
      </section>

      <!-- Personal information -->
      <section class="shell" style="--i: 1" aria-labelledby="profile-personal-title">
        <div class="core panel">
          <header class="panel-head">
            <span class="panel-icon" aria-hidden="true"><t-icon name="user-circle" /></span>
            <div class="panel-titles">
              <h3 id="profile-personal-title">{{ $t('userProfile.personal.title') }}</h3>
              <p>{{ $t('userProfile.personal.hint') }}</p>
            </div>
          </header>

          <t-form
            ref="profileFormRef"
            :data="profileForm"
            :rules="profileRules"
            label-align="top"
            class="profile-form"
            :disabled="saving"
            @submit.prevent
          >
            <t-form-item :label="$t('userProfile.firstName')" name="firstName">
              <t-input
                v-model="profileForm.firstName"
                autocomplete="given-name"
                :maxlength="100"
                @enter="submitProfile"
              />
            </t-form-item>
            <t-form-item :label="$t('userProfile.lastName')" name="lastName">
              <t-input
                v-model="profileForm.lastName"
                autocomplete="family-name"
                :maxlength="100"
                @enter="submitProfile"
              />
            </t-form-item>
            <t-form-item :label="$t('tenant.api.emailLabel')" :help="$t('userProfile.personal.emailHint')">
              <t-input :value="userInfo?.email || ''" readonly class="readonly-input">
                <template #suffix-icon><t-icon name="lock-on" /></template>
              </t-input>
            </t-form-item>
            <t-form-item :label="$t('userProfile.phone')" name="phone">
              <div class="phone-input">
                <t-select
                  v-model="profileForm.country"
                  :options="phoneCountries"
                  :value-display="phoneCountryDisplay"
                  filterable
                  :aria-label="$t('auth.phoneCountry')"
                  :popup-props="{ overlayClassName: 'user-profile-select-overlay' }"
                  @change="revalidatePhone"
                />
                <t-input
                  v-model="profileForm.phone"
                  type="tel"
                  autocomplete="tel-national"
                  :maxlength="32"
                  :placeholder="$t('auth.phonePlaceholder')"
                  @enter="submitProfile"
                />
              </div>
            </t-form-item>
          </t-form>

          <footer class="panel-footer">
            <span class="dirty-hint" :class="{ 'is-visible': isDirty }" aria-live="polite">
              <span class="dirty-dot" aria-hidden="true" />
              {{ isDirty ? $t('userProfile.personal.unsaved') : '' }}
            </span>
            <div class="panel-actions">
              <t-button variant="text" :disabled="!isDirty || saving" @click="resetProfileForm">
                {{ $t('common.cancel') }}
              </t-button>
              <t-button theme="primary" :loading="saving" :disabled="!isDirty" @click="submitProfile">
                {{ $t('userProfile.personal.save') }}
              </t-button>
            </div>
          </footer>
        </div>
      </section>

      <div class="bento">
        <!-- Account details -->
        <section class="shell bento-account" style="--i: 2" aria-labelledby="profile-account-title">
          <div class="core panel">
            <header class="panel-head">
              <span class="panel-icon" aria-hidden="true"><t-icon name="user-safety" /></span>
              <div class="panel-titles">
                <h3 id="profile-account-title">{{ $t('userProfile.account.title') }}</h3>
                <p>{{ $t('userProfile.account.hint') }}</p>
              </div>
            </header>
            <dl class="detail-list">
              <div class="detail-row">
                <dt>{{ $t('tenant.api.usernameLabel') }}</dt>
                <dd>{{ userInfo?.username || '-' }}</dd>
              </div>
              <div class="detail-row">
                <dt>{{ $t('tenant.api.userIdLabel') }}</dt>
                <dd class="detail-id">
                  <code class="mono">{{ userInfo?.id || '-' }}</code>
                  <t-tooltip v-if="userInfo?.id" :content="$t('userProfile.account.copyId')">
                    <button
                      type="button"
                      class="copy-btn"
                      :aria-label="$t('userProfile.account.copyId')"
                      @click="copyUserId"
                    >
                      <t-icon name="copy" />
                    </button>
                  </t-tooltip>
                </dd>
              </div>
              <div class="detail-row">
                <dt>{{ $t('tenant.api.createdAtLabel') }}</dt>
                <dd>{{ formatDate(userInfo?.created_at) }}</dd>
              </div>
            </dl>
          </div>
        </section>

        <!-- Security -->
        <section class="shell bento-security" style="--i: 3" aria-labelledby="profile-security-title">
          <div class="core panel security">
            <header class="panel-head">
              <span class="panel-icon panel-icon--brand" aria-hidden="true"><t-icon name="lock-on" /></span>
              <div class="panel-titles">
                <h3 id="profile-security-title">{{ $t('userProfile.security.title') }}</h3>
              </div>
            </header>
            <div class="security-body">
              <span class="security-label">{{ $t('userProfile.changePassword.label') }}</span>
              <p class="security-desc">
                {{ oidcOnlyLogin
                  ? $t('userProfile.changePassword.oidcOnlyDescription')
                  : $t('userProfile.changePassword.description') }}
              </p>
            </div>
            <t-popup
              v-if="!oidcOnlyLogin"
              v-model="passwordPopupVisible"
              trigger="click"
              placement="bottom-end"
              destroy-on-close
              overlay-class-name="wk-popover wk-popover--form user-profile-password-popup-overlay"
            >
              <t-button variant="outline" class="security-action">
                <template #icon><t-icon name="lock-on" /></template>
                {{ $t('userProfile.changePassword.label') }}
              </t-button>
              <template #content>
                <div class="password-popup-inner" @click.stop>
                  <div class="password-popup-title">{{ $t('userProfile.changePassword.label') }}</div>
                  <p class="password-popup-hint">{{ $t('userProfile.changePassword.description') }}</p>
                  <t-form
                    ref="passwordFormRef"
                    :data="passwordForm"
                    :rules="passwordRules"
                    label-align="top"
                    class="password-popup-form"
                    @submit.prevent
                  >
                    <t-form-item :label="$t('userProfile.changePassword.currentLabel')" name="oldPassword">
                      <t-input
                        v-model="passwordForm.oldPassword"
                        type="password"
                        autocomplete="current-password"
                        :disabled="passwordSubmitting"
                        :placeholder="$t('userProfile.changePassword.currentPlaceholder')"
                      />
                    </t-form-item>
                    <t-form-item :label="$t('userProfile.changePassword.newLabel')" name="newPassword">
                      <t-input
                        v-model="passwordForm.newPassword"
                        type="password"
                        autocomplete="new-password"
                        :disabled="passwordSubmitting"
                        :placeholder="$t('userProfile.changePassword.newPlaceholder')"
                      />
                    </t-form-item>
                    <t-form-item :label="$t('userProfile.changePassword.confirmLabel')" name="confirmPassword">
                      <t-input
                        v-model="passwordForm.confirmPassword"
                        type="password"
                        autocomplete="new-password"
                        :disabled="passwordSubmitting"
                        :placeholder="$t('userProfile.changePassword.confirmPlaceholder')"
                        @enter="submitPasswordChange"
                      />
                    </t-form-item>
                  </t-form>
                  <div class="password-popup-footer">
                    <t-button variant="outline" :disabled="passwordSubmitting" @click="closePasswordPopup">
                      {{ $t('common.cancel') }}
                    </t-button>
                    <t-button theme="primary" :loading="passwordSubmitting" @click="submitPasswordChange">
                      {{ $t('userProfile.changePassword.submit') }}
                    </t-button>
                  </div>
                </div>
              </template>
            </t-popup>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import {
  getCountries,
  getCountryCallingCode,
  parsePhoneNumberFromString,
  type CountryCode,
} from 'libphonenumber-js/max'
import {
  getCurrentUser,
  getAuthConfig,
  changePassword,
  updateMyProfile,
  logout as logoutApi,
  type UserInfo,
} from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { newPasswordRules } from '@/utils/passwordPolicy'
import { normalizePhoneNumber } from '@/utils/phoneNumber'
import { copyWithToast } from '@/utils/clipboard'

const DEFAULT_COUNTRY: CountryCode = 'TR'

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const userInfo = ref<UserInfo | null>(null)
const complexPasswordEnabled = ref(false)
const loading = ref(true)
const error = ref('')

/* ---------- Personal information ---------- */

interface ProfileFormState {
  firstName: string
  lastName: string
  country: CountryCode
  phone: string
}

const profileFormRef = ref<FormInstanceFunctions | null>(null)
const saving = ref(false)
const profileForm = reactive<ProfileFormState>({
  firstName: '',
  lastName: '',
  country: DEFAULT_COUNTRY,
  phone: '',
})
const savedProfile = ref<ProfileFormState>({ ...profileForm })

const flagOf = (country: string) =>
  [...country].map((letter) => String.fromCodePoint(127397 + letter.charCodeAt(0))).join('')

const phoneCountries = computed(() => {
  const names = new Intl.DisplayNames([locale.value], { type: 'region' })
  return getCountries()
    .map((country) => {
      const name = names.of(country) || country
      return {
        value: country,
        name,
        label: flagOf(country) + ' +' + getCountryCallingCode(country) + ' ' + name,
      }
    })
    .sort((a, b) => a.name.localeCompare(b.name, locale.value))
})

const phoneCountryDisplay = computed(
  () => flagOf(profileForm.country) + ' +' + getCountryCallingCode(profileForm.country),
)

const formStateFromUser = (user: UserInfo | null): ProfileFormState => {
  const parsed = user?.phone ? parsePhoneNumberFromString(user.phone) : undefined
  return {
    firstName: user?.first_name || '',
    lastName: user?.last_name || '',
    country: (parsed?.country as CountryCode | undefined) || DEFAULT_COUNTRY,
    phone: parsed ? parsed.formatNational() : user?.phone || '',
  }
}

const syncProfileForm = (user: UserInfo | null) => {
  const state = formStateFromUser(user)
  Object.assign(profileForm, state)
  savedProfile.value = { ...state }
  profileFormRef.value?.clearValidate?.()
}

const isDirty = computed(() => {
  const saved = savedProfile.value
  return (
    profileForm.firstName.trim() !== saved.firstName.trim() ||
    profileForm.lastName.trim() !== saved.lastName.trim() ||
    profileForm.country !== saved.country ||
    profileForm.phone.trim() !== saved.phone.trim()
  )
})

const displayName = computed(() => {
  const full = [profileForm.firstName.trim(), profileForm.lastName.trim()].filter(Boolean).join(' ')
  return full || userInfo.value?.username || userInfo.value?.email || '-'
})

const initials = computed(() => {
  const source = [profileForm.firstName.trim(), profileForm.lastName.trim()].filter(Boolean)
  const letters = source.length
    ? source.map((part) => part[0])
    : [(userInfo.value?.username || userInfo.value?.email || '?')[0]]
  return letters.join('').toLocaleUpperCase(locale.value)
})

const nameLengthOk = (v: string) => [...(v || '').trim()].length <= 100

const profileRules = computed<Record<string, FormRule[]>>(() => ({
  firstName: [
    { validator: (v: string) => !!v?.trim(), message: t('auth.firstNameRequired'), type: 'error' },
    { validator: nameLengthOk, message: t('userProfile.personal.nameTooLong'), type: 'error' },
  ],
  lastName: [
    { validator: (v: string) => !!v?.trim(), message: t('auth.lastNameRequired'), type: 'error' },
    { validator: nameLengthOk, message: t('userProfile.personal.nameTooLong'), type: 'error' },
  ],
  phone: [
    { validator: (v: string) => !!v?.trim(), message: t('auth.phoneRequired'), type: 'error' },
    {
      validator: (v: string) => !!normalizePhoneNumber(v || '', profileForm.country),
      message: t('auth.phoneInvalid'),
      type: 'error',
    },
  ],
}))

const revalidatePhone = () => {
  if (profileForm.phone.trim()) void profileFormRef.value?.validate?.({ fields: ['phone'] })
}

const resetProfileForm = () => syncProfileForm(userInfo.value)

const submitProfile = async () => {
  if (saving.value || !isDirty.value) return
  const result = await profileFormRef.value?.validate?.()
  if (result !== true) return
  const phone = normalizePhoneNumber(profileForm.phone, profileForm.country)
  if (!phone) return

  saving.value = true
  try {
    const resp = await updateMyProfile({
      first_name: profileForm.firstName.trim(),
      last_name: profileForm.lastName.trim(),
      phone,
    })
    const updated = resp.data?.user
    if (!resp.success || !updated || !userInfo.value) {
      MessagePlugin.error(resp.message || t('userProfile.personal.failed'))
      return
    }
    const changes = {
      first_name: updated.first_name || '',
      last_name: updated.last_name || '',
      phone: updated.phone || '',
      updated_at: updated.updated_at || new Date().toISOString(),
    }
    userInfo.value = { ...userInfo.value, ...changes }
    if (authStore.user) authStore.setUser({ ...authStore.user, ...changes })
    syncProfileForm(userInfo.value)
    MessagePlugin.success(t('userProfile.personal.saved'))
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('userProfile.personal.failed'))
  } finally {
    saving.value = false
  }
}

/* ---------- Password ---------- */

const passwordPopupVisible = ref(false)
const passwordFormRef = ref<FormInstanceFunctions | null>(null)
const passwordSubmitting = ref(false)
const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const loadPasswordPolicy = async () => {
  try {
    const resp = await getAuthConfig()
    complexPasswordEnabled.value = !!resp.complex_password_enabled
  } catch {
    complexPasswordEnabled.value = false
  }
}

const oidcOnlyLogin = computed(
  () => userInfo.value?.preferences?.oidc_only_login === true,
)

watch(passwordPopupVisible, (open) => {
  resetPasswordForm()
  if (open) void loadPasswordPolicy()
})

const passwordRules = computed<Record<string, FormRule[]>>(() => ({
  oldPassword: [
    { required: true, message: t('userProfile.changePassword.currentRequired'), type: 'error' },
  ],
  newPassword: newPasswordRules(t, complexPasswordEnabled.value, [
    {
      validator: (val: string) => val !== passwordForm.oldPassword,
      message: t('userProfile.changePassword.sameAsCurrent'),
      type: 'error',
    },
  ]),
  confirmPassword: [
    { required: true, message: t('auth.confirmPasswordRequired'), type: 'error' },
    {
      validator: (val: string) => val === passwordForm.newPassword,
      message: t('auth.passwordMismatch'),
      type: 'error',
      trigger: 'blur',
    },
  ],
}))

const resetPasswordForm = () => {
  passwordForm.oldPassword = ''
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  passwordFormRef.value?.clearValidate?.()
}

const closePasswordPopup = () => {
  if (passwordSubmitting.value) return
  passwordPopupVisible.value = false
  resetPasswordForm()
}

const submitPasswordChange = async () => {
  if (passwordSubmitting.value) return
  const result = await passwordFormRef.value?.validate?.()
  if (result !== true) return

  passwordSubmitting.value = true
  try {
    const resp = await changePassword({
      old_password: passwordForm.oldPassword,
      new_password: passwordForm.newPassword,
    })
    if (!resp.success) {
      MessagePlugin.error(resp.message || t('userProfile.changePassword.failed'))
      return
    }

    passwordPopupVisible.value = false
    MessagePlugin.success(t('userProfile.changePassword.success'))
    resetPasswordForm()

    // Backend revokes all sessions on success; mirror that locally and
    // force a fresh login with the new credential.
    try {
      await logoutApi()
    } catch {
      /* ignore: local cleanup still proceeds */
    }
    authStore.logout()
    router.push('/login')
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('userProfile.changePassword.failed'))
  } finally {
    passwordSubmitting.value = false
  }
}

/* ---------- Loading ---------- */

const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''
    const resp = await getCurrentUser()
    if ((resp as any).success && resp.data) {
      userInfo.value = resp.data.user
      syncProfileForm(resp.data.user)
    } else {
      error.value = resp.message || t('tenant.messages.fetchFailed')
    }
  } catch (err: any) {
    error.value = err?.message || t('tenant.messages.networkError')
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return t('tenant.unknown')
  try {
    return new Intl.DateTimeFormat(locale.value || 'tr-TR', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(dateStr))
  } catch {
    return t('tenant.formatError')
  }
}

const memberSince = computed(() => {
  const raw = userInfo.value?.created_at
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(locale.value || 'tr-TR', { year: 'numeric', month: 'long' }).format(date)
})

const copyUserId = () => copyWithToast(userInfo.value?.id, 'userProfile.account.copied')

onMounted(loadInfo)
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

@ease: cubic-bezier(0.32, 0.72, 0, 1);
@shell-pad: 5px;

.user-profile {
  width: 100%;
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--app-space-4);

  --up-shell: var(--app-surface-muted);
  --up-card: var(--td-bg-color-container);
  --up-ring: color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  --up-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.9);
  --up-lift: 0 1px 2px rgba(0, 0, 0, 0.03), 0 10px 28px -18px rgba(0, 0, 0, 0.14);
}

:root[theme-mode='dark'] .user-profile {
  --up-shell: #1a1a1a;
  --up-card: #242424;
  --up-ring: rgba(255, 255, 255, 0.07);
  --up-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.05);
  --up-lift: 0 1px 2px rgba(0, 0, 0, 0.3), 0 12px 32px -18px rgba(0, 0, 0, 0.7);
}

.section-header {
  .settings-section-header();
  margin-bottom: var(--app-space-2);
}

.error-inline {
  padding: var(--app-space-2) 0;
}

/* ---------- Double-bezel surfaces ---------- */

.shell {
  padding: @shell-pad;
  border-radius: var(--app-radius-2xl);
  background: var(--up-shell);
  box-shadow: 0 0 0 1px var(--up-ring);
  animation: profile-rise 560ms @ease both;
  animation-delay: calc(var(--i, 0) * 70ms);
}

.core {
  border-radius: calc(var(--app-radius-2xl) - @shell-pad);
  background: var(--up-card);
  box-shadow: 0 0 0 1px var(--up-ring), var(--up-highlight), var(--up-lift);
}

/* ---------- Identity hero ---------- */

.hero {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--app-space-5);
  padding: var(--app-space-6);
  overflow: hidden;
  background:
    radial-gradient(120% 160% at 0% 0%, color-mix(in srgb, var(--td-brand-color) 9%, transparent) 0%, transparent 55%),
    var(--up-card);
}

.hero--skeleton {
  .skeleton-lines {
    flex: 1;
    min-width: 0;
  }
}

.avatar {
  flex: none;
  position: relative;
  width: 76px;
  height: 76px;
  border-radius: 50%;
  padding: 3px;
  background: linear-gradient(
    140deg,
    color-mix(in srgb, var(--td-brand-color) 55%, transparent),
    color-mix(in srgb, var(--td-brand-color) 10%, transparent)
  );
  user-select: none;
}

.avatar-initials {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: linear-gradient(150deg, var(--td-brand-color-6) 0%, var(--td-brand-color-8) 100%);
  box-shadow: 0 0 0 3px var(--up-card), inset 0 1px 0 rgba(255, 255, 255, 0.25);
  color: #fff;
  font-family: var(--app-font-heading);
  font-size: var(--app-text-5xl);
  font-weight: 600;
  letter-spacing: 0.01em;
  line-height: 1;
}

.hero-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--app-space-1);
}

.hero-name {
  margin: 0;
  font-family: var(--app-font-heading);
  font-size: var(--app-text-4xl);
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.015em;
  color: var(--td-text-color-primary);
  overflow-wrap: anywhere;
}

.hero-email {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  overflow-wrap: anywhere;

  .t-icon {
    flex: none;
    color: var(--td-text-color-placeholder);
  }
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: var(--app-space-2) 0 0;
  padding: 0;
  list-style: none;
}

.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 24px;
  padding: 0 10px;
  border-radius: var(--app-radius-pill);
  background: var(--up-shell);
  box-shadow: inset 0 0 0 1px var(--up-ring);
  font-size: var(--app-text-sm);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  white-space: nowrap;

  .t-icon {
    font-size: var(--app-text-md);
    color: var(--td-text-color-placeholder);
  }
}

/* ---------- Panels ---------- */

.panel {
  display: flex;
  flex-direction: column;
  gap: var(--app-space-5);
  padding: var(--app-space-6);
  min-width: 0;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  gap: var(--app-space-3);
}

.panel-icon {
  flex: none;
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: var(--app-radius-md);
  background: var(--up-shell);
  box-shadow: inset 0 0 0 1px var(--up-ring);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xl);

  &--brand {
    background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 22%, transparent);
    color: var(--td-brand-color);
  }
}

.panel-titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 1px;

  h3 {
    margin: 0;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-lg);
    font-weight: 600;
    line-height: 1.35;
    color: var(--td-text-color-primary);
  }

  p {
    margin: 0;
    max-width: 60ch;
    font-size: var(--app-text-sm);
    line-height: 1.5;
    color: var(--td-text-color-secondary);
  }
}

/* ---------- Personal form ---------- */

.profile-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: var(--app-space-4);
  row-gap: var(--app-space-1);

  :deep(.t-form__item) {
    margin-bottom: 0;
    min-width: 0;
  }

  :deep(.t-form__label) {
    padding-bottom: 6px;
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-text-color-secondary);
    line-height: 1.4;
    min-height: 0;
  }

  :deep(.t-form__controls) {
    min-height: 62px;
  }

  :deep(.t-input) {
    height: 40px;
    border-radius: var(--app-radius-md);
    font-size: var(--app-text-base);
    transition: border-color var(--app-motion-base) @ease, box-shadow var(--app-motion-base) @ease;
  }

  :deep(.t-input__help),
  :deep(.t-form__help) {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
  }
}

.readonly-input :deep(.t-input) {
  background: var(--up-shell);
  border-color: transparent;
  cursor: default;

  input {
    color: var(--td-text-color-secondary);
    cursor: default;
  }

  .t-input__suffix-icon {
    color: var(--td-text-color-placeholder);
  }
}

.phone-input {
  display: flex;
  width: 100%;
  min-width: 0;

  :deep(.t-select__wrap) {
    width: 116px;
    flex: none;
  }

  :deep(.t-select .t-input) {
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
  }

  > :deep(.t-input__wrap) {
    flex: 1;
    min-width: 0;
    margin-left: -1px;

    .t-input {
      border-top-left-radius: 0;
      border-bottom-left-radius: 0;
    }
  }

  :deep(.t-input--focused),
  :deep(.t-input:hover) {
    position: relative;
    z-index: 1;
  }
}

.panel-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--app-space-4);
  margin: 0 calc(var(--app-space-6) * -1) calc(var(--app-space-6) * -1);
  padding: var(--app-space-3) var(--app-space-6);
  border-top: 1px solid var(--up-ring);
  border-radius: 0 0 calc(var(--app-radius-2xl) - @shell-pad) calc(var(--app-radius-2xl) - @shell-pad);
  background: color-mix(in srgb, var(--up-shell) 55%, transparent);
}

.dirty-hint {
  display: inline-flex;
  align-items: center;
  gap: var(--app-space-2);
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  opacity: 0;
  transform: translateY(2px);
  transition: opacity var(--app-motion-base) @ease, transform var(--app-motion-base) @ease;

  &.is-visible {
    opacity: 1;
    transform: none;
  }
}

.dirty-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--td-warning-color);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--td-warning-color) 18%, transparent);
}

.panel-actions {
  display: flex;
  gap: var(--app-space-2);
  margin-left: auto;

  :deep(.t-button) {
    min-width: 92px;
    height: 36px;
    padding: 0 var(--app-space-4);
    border-radius: var(--app-radius-pill);
    transition: transform var(--app-motion-fast) @ease, background-color var(--app-motion-base) @ease;

    &:active:not(.t-is-disabled) {
      transform: scale(0.98);
    }
  }
}

/* ---------- Bento ---------- */

.bento {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(0, 1fr);
  gap: var(--app-space-4);
  align-items: stretch;

  > .shell {
    display: flex;

    > .core {
      flex: 1;
    }
  }
}

.detail-list {
  margin: 0;
  display: flex;
  flex-direction: column;
}

.detail-row {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: var(--app-space-3) 0;

  & + & {
    border-top: 1px dashed var(--up-ring);
  }

  &:first-child {
    padding-top: 0;
  }

  &:last-child {
    padding-bottom: 0;
  }

  dt {
    font-size: var(--app-text-xs);
    font-weight: 500;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--td-text-color-placeholder);
  }

  dd {
    margin: 0;
    font-size: var(--app-text-base);
    color: var(--td-text-color-primary);
    overflow-wrap: anywhere;
  }
}

.detail-id {
  display: flex;
  align-items: center;
  gap: var(--app-space-2);
  min-width: 0;

  .mono {
    min-width: 0;
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
    overflow-wrap: anywhere;
  }
}

.copy-btn {
  flex: none;
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: background-color var(--app-motion-fast) @ease, color var(--app-motion-fast) @ease;

  &:hover {
    background: var(--up-shell);
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--app-ring-soft);
    outline-offset: 1px;
  }
}

.security {
  .security-body {
    display: flex;
    flex-direction: column;
    gap: var(--app-space-1);
    flex: 1;
  }
}

.security-label {
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.security-desc {
  margin: 0;
  font-size: var(--app-text-sm);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.security-action {
  align-self: flex-start;
  height: 36px;
  padding: 0 var(--app-space-4);
  border-radius: var(--app-radius-pill);
}

/* ---------- Password popup ---------- */

.password-popup-inner {
  max-width: 100%;
}

.password-popup-title {
  font-size: var(--app-text-lg);
  font-weight: 500;
  color: var(--td-text-color-primary);
  margin: 0 0 8px;
  line-height: 1.35;
}

.password-popup-hint {
  margin: 0 0 12px;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.password-popup-form {
  :deep(.t-form__item) {
    margin-bottom: 14px;

    &:last-child {
      margin-bottom: 4px;
    }
  }
}

.password-popup-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

@keyframes profile-rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* ---------- Narrow layouts ---------- */

@container (max-width: 720px) {
  .bento {
    grid-template-columns: minmax(0, 1fr);
  }
}

@container (max-width: 560px) {
  .hero {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--app-space-4);
    padding: var(--app-space-5);
  }

  .avatar {
    width: 64px;
    height: 64px;
  }

  .avatar-initials {
    font-size: var(--app-text-4xl);
  }

  .hero-name {
    font-size: var(--app-text-3xl);
  }

  .panel {
    padding: var(--app-space-5);
  }

  .panel-footer {
    flex-direction: column;
    align-items: stretch;
    margin: 0 calc(var(--app-space-5) * -1) calc(var(--app-space-5) * -1);
    padding: var(--app-space-3) var(--app-space-5);
  }

  .panel-actions :deep(.t-button) {
    flex: 1;
  }

  .profile-form {
    grid-template-columns: minmax(0, 1fr);
  }

  .security-action {
    align-self: stretch;
  }
}

@media (prefers-reduced-motion: reduce) {
  .shell {
    animation: none;
  }

  .dirty-hint,
  .copy-btn,
  .panel-actions :deep(.t-button) {
    transition: none;
  }
}
</style>

<style lang="less">
/* t-popup body içine eklenir, global stil gerekir; z-index ayarlar tam ekran maskesinden (2000) yüksek olmalı. */
.user-profile-password-popup-overlay,
.user-profile-select-overlay {
  z-index: 3050 !important;
}

.user-profile-select-overlay .t-select__dropdown {
  min-width: 260px;
}

:root[theme-mode='dark'] .user-profile-password-popup-overlay .t-popup__content {
  background: rgba(36, 36, 36, 0.92) !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.05),
    0 2px 4px rgba(0, 0, 0, 0.12),
    0 8px 32px rgba(0, 0, 0, 0.28) !important;
}
</style>
