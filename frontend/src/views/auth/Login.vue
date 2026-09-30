<template>
  <div class="login-layout">
    <!-- Light rays background -->
    <div class="light-rays" aria-hidden="true">
      <div class="light-rays__glow light-rays__glow--left" />
      <div class="light-rays__glow light-rays__glow--right" />
      <div v-for="ray in rays" :key="ray.id" class="light-ray" :style="ray.style" />
    </div>

    <!-- Login Card -->
    <div class="auth-card" v-if="!isRegisterMode">
      <div class="auth-header">
        <img src="@/assets/img/rethra-icon.png" alt="Rethra" class="auth-logo" />
        <h2 class="auth-title">{{ $t('auth.loginTitle') }}</h2>
        <p class="auth-subtitle">{{ $t('auth.loginSubtitle') }}</p>
      </div>

      <!-- invite_only modunda paylaşım bağlantısı giriş kartında kalır ve davet bağlamına da ihtiyaç duyar. -->
      <div v-if="inviteLookup" class="invite-banner">
        <t-icon name="link" class="invite-banner__icon" />
        <div class="invite-banner__text">
          <div class="invite-banner__title">
            {{ $t('inviteRegister.bannerTitle', { tenant: inviteLookup.tenant_name || '' }) }}
          </div>
          <div class="invite-banner__hint">
            {{ $t('inviteRegister.bannerHintLogin') }}
          </div>
        </div>
      </div>
      <div v-else-if="inviteLookupError" class="invite-banner invite-banner--error">
        {{ inviteLookupError }}
      </div>

      <t-form ref="formRef" class="auth-form" :data="formData" :rules="formRules" @submit="handleLogin"
        layout="vertical" label-align="top" :required-mark="false">
        <t-form-item :label="$t('auth.email')" name="email" @focusout="loginValidation.onFocusOut('email', $event)">
          <t-input v-model="formData.email" :placeholder="$t('auth.emailPlaceholder')" type="text"
            autocomplete="email" :disabled="loading" />
        </t-form-item>

        <t-form-item :label="$t('auth.password')" name="password" @focusout="loginValidation.onFocusOut('password', $event)">
          <t-input v-model="formData.password" class="auth-password" :placeholder="$t('auth.passwordPlaceholder')" :type="showPassword ? 'text' : 'password'"
            autocomplete="current-password" :disabled="loading" @enter="handleLogin">
            <template #suffix>
              <button type="button" class="auth-password-toggle" :disabled="loading"
                :aria-label="$t(showPassword ? 'auth.hidePassword' : 'auth.showPassword')"
                @click="showPassword = !showPassword">
                <svg v-if="showPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49" />
                  <path d="M14.084 14.158a3 3 0 0 1-4.242-4.242" />
                  <path d="M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143" />
                  <path d="m2 2 20 20" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              </button>
            </template>
          </t-input>
        </t-form-item>

        <div class="submit-row">
          <t-button type="submit" theme="primary" :loading="loading" class="submit-button">
            {{ loading ? $t('auth.loggingIn') : $t('auth.login') }}
          </t-button>
        </div>
      </t-form>

      <p v-if="registrationEnabled" class="auth-switch">
        {{ $t('auth.firstTime') }}
        <a href="#" class="auth-link" @click.prevent="!loading && toggleMode()">{{ $t('auth.createAccount') }}</a>
      </p>

      <div v-if="oidcEnabled" class="oidc-section">
        <p class="oidc-section__label">{{ $t('auth.orContinueWith') }}</p>
        <t-button theme="default" variant="outline" block :loading="oidcLoading" :disabled="loading"
          class="oidc-button" @click="handleOIDCLogin">
          {{ oidcLoading ? $t('auth.redirectingToOIDC') : oidcLoginText }}
        </t-button>
      </div>
    </div>

    <!-- Register Card. Renders when the user is in register mode
         AND either self-service registration is enabled OR they
         arrived with a valid share-link token (which bypasses the
         invite_only gate). -->
    <div class="auth-card auth-card--wide" v-if="isRegisterMode && (registrationEnabled || inviteLookup)">
      <div class="auth-header">
        <img src="@/assets/img/rethra-icon.png" alt="Rethra" class="auth-logo" />
        <h2 class="auth-title">{{ $t('auth.createAccount') }}</h2>
        <p class="auth-subtitle">{{ $t('auth.registerSubtitle') }}</p>
      </div>

      <!-- Share-link banner: shown only when ?token= resolved to a
           real invitation row, so the invitee instantly sees who
           invited them and into which workspace. -->
      <div v-if="inviteLookup" class="invite-banner">
        <t-icon name="link" class="invite-banner__icon" />
        <div class="invite-banner__text">
          <div class="invite-banner__title">
            {{ $t('inviteRegister.bannerTitle', { tenant: inviteLookup.tenant_name || '' }) }}
          </div>
          <div class="invite-banner__hint">
            {{ $t('inviteRegister.bannerHint') }}
          </div>
        </div>
      </div>
      <div v-else-if="inviteLookupError" class="invite-banner invite-banner--error">
        {{ inviteLookupError }}
      </div>

      <t-form ref="registerFormRef" class="auth-form auth-form--grid" :data="registerData" :rules="registerRules"
        @submit="handleRegister" layout="vertical" label-align="top" :required-mark="false">
        <t-form-item :label="$t('auth.firstName')" name="first_name" @focusout="registerValidation.onFocusOut('first_name', $event)">
          <t-input v-model="registerData.first_name" :placeholder="$t('auth.firstNamePlaceholder')" autocomplete="given-name" :disabled="loading" />
        </t-form-item>

        <t-form-item :label="$t('auth.lastName')" name="last_name" @focusout="registerValidation.onFocusOut('last_name', $event)">
          <t-input v-model="registerData.last_name" :placeholder="$t('auth.lastNamePlaceholder')" autocomplete="family-name" :disabled="loading" />
        </t-form-item>

        <t-form-item :label="$t('auth.username')" name="username" @focusout="registerValidation.onFocusOut('username', $event)">
          <t-input v-model="registerData.username" :placeholder="$t('auth.usernamePlaceholder')"
            :disabled="loading" />
        </t-form-item>

        <t-form-item :label="$t('auth.email')" name="email" @focusout="registerValidation.onFocusOut('email', $event)">
          <t-input v-model="registerData.email" :placeholder="$t('auth.emailPlaceholder')" type="text"
            autocomplete="email" :disabled="loading" />
        </t-form-item>

        <t-form-item class="auth-form__full" :label="$t('auth.phone')" name="phone" @focusout="registerValidation.onFocusOut('phone', $event)">
          <div class="phone-input">
            <t-select v-model="phoneCountry" :options="phoneCountries" filterable :disabled="loading"
              @change="registerValidation.revalidate('phone')"
              :aria-label="$t('auth.phoneCountry')" />
            <t-input v-model="registerData.phone" :placeholder="$t('auth.phonePlaceholder')" type="tel" autocomplete="tel" :maxlength="32"
              :disabled="loading" />
          </div>
        </t-form-item>

        <t-form-item :label="$t('auth.password')" name="password" @focusout="registerValidation.onFocusOut('password', $event)">
          <t-input v-model="registerData.password" class="auth-password" :placeholder="$t('auth.passwordPlaceholder')" :type="showPassword ? 'text' : 'password'"
            autocomplete="new-password" :disabled="loading">
            <template #suffix>
              <button type="button" class="auth-password-toggle" :disabled="loading"
                :aria-label="$t(showPassword ? 'auth.hidePassword' : 'auth.showPassword')"
                @click="showPassword = !showPassword">
                <svg v-if="showPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49" />
                  <path d="M14.084 14.158a3 3 0 0 1-4.242-4.242" />
                  <path d="M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143" />
                  <path d="m2 2 20 20" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              </button>
            </template>
          </t-input>
        </t-form-item>

        <t-form-item :label="$t('auth.confirmPassword')" name="confirmPassword" @focusout="registerValidation.onFocusOut('confirmPassword', $event)">
          <t-input v-model="registerData.confirmPassword" :placeholder="$t('auth.confirmPasswordPlaceholder')" class="auth-password"
            :type="showConfirmPassword ? 'text' : 'password'" autocomplete="new-password" :disabled="loading" @enter="handleRegister">
            <template #suffix>
              <button type="button" class="auth-password-toggle" :disabled="loading"
                :aria-label="$t(showConfirmPassword ? 'auth.hidePassword' : 'auth.showPassword')"
                @click="showConfirmPassword = !showConfirmPassword">
                <svg v-if="showConfirmPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49" />
                  <path d="M14.084 14.158a3 3 0 0 1-4.242-4.242" />
                  <path d="M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143" />
                  <path d="m2 2 20 20" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              </button>
            </template>
          </t-input>
        </t-form-item>

        <div class="submit-row auth-form__full">
          <t-button type="submit" theme="primary" :loading="loading" class="submit-button">
            {{ loading ? $t('auth.registering') : $t('auth.register') }}
          </t-button>
        </div>
      </t-form>

      <p class="auth-switch">
        {{ $t('auth.haveAccount') }}
        <a href="#" class="auth-link" @click.prevent="!loading && toggleMode()">{{ $t('auth.backToLogin') }}</a>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, nextTick, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin, type FormInstanceFunctions } from 'tdesign-vue-next'
import { deferRules, useDeferredValidation } from '@/composables/useDeferredValidation'
import { useRoleLabel } from '@/composables/useRoleLabel'
import { notifyLoginSuccess } from '@/utils/loginNotify'
import { newPasswordRules } from '@/utils/passwordPolicy'
import { normalizePhoneNumber } from '@/utils/phoneNumber'
import { getCountries, getCountryCallingCode, type CountryCode } from 'libphonenumber-js/max'
import {
  login,
  register,
  getOIDCAuthorizationURL,
  getOIDCConfig,
  getAuthConfig,
  userInfoFromApi,
  getInvitationByToken,
  registerByInvite,
  type InviteLookup,
} from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t, tm, locale } = useI18n()
const { formatRole, roleIcon } = useRoleLabel()

// Light rays background (randomised once per mount, animated via CSS)
const RAY_COUNT = 6
const RAY_CYCLE = 15
const rays = Array.from({ length: RAY_COUNT }, (_, index) => {
  const left = 8 + Math.random() * 84
  const rotate = -28 + Math.random() * 56
  const width = 160 + Math.random() * 160
  const swing = 0.8 + Math.random() * 1.8
  const delay = Math.random() * RAY_CYCLE
  const duration = RAY_CYCLE * (0.75 + Math.random() * 0.5)
  const intensity = 0.6 + Math.random() * 0.5
  return {
    id: index,
    style: {
      '--ray-left': left + '%',
      '--ray-width': width + 'px',
      '--ray-rotate': rotate + 'deg',
      '--ray-swing': swing + 'deg',
      '--ray-delay': '-' + delay + 's',
      '--ray-duration': duration * 1.1 + 's',
      '--ray-intensity': String(intensity),
    } as Record<string, string>,
  }
})

// Form references
const formRef = ref<FormInstanceFunctions>()
const registerFormRef = ref<FormInstanceFunctions>()

// State management
const loading = ref(false)
const oidcLoading = ref(false)
const isRegisterMode = ref(false)
const oidcEnabled = ref(false)
const oidcProviderName = ref('')
// registrationEnabled defaults to true so that on first paint the Register
// link is visible; the actual mode is fetched from /auth/config in onMounted.
// In invite_only mode the link/card are hidden.
const registrationEnabled = ref(true)
const complexPasswordEnabled = ref(false)

// invite-link state. When the URL carries ?token=xxx we resolve it to
// the originating tenant + role and switch the form into a "register
// via invitation" mode. The token bypasses the normal invite_only
// gate — possessing it IS the authorisation. Submitting the register
// form with this set hits /auth/register-by-invite (auto-login on
// success) instead of /auth/register.
const inviteToken = ref('')
const inviteLookup = ref<InviteLookup | null>(null)
const inviteLookupError = ref('')
const inviteLookupLoading = ref(false)

const oidcLoginText = computed(() => {
  if (oidcProviderName.value) {
    return t('auth.oidcLoginWithProvider', { provider: oidcProviderName.value })
  }
  return t('auth.oidcLogin')
})

// Login form data
const formData = reactive<{ [key: string]: any }>({
  email: '',
  password: '',
})

// Register form data
const registerData = reactive<{ [key: string]: any }>({
  username: '',
  first_name: '',
  last_name: '',
  phone: '',
  email: '',
  password: '',
  confirmPassword: ''
})
const phoneCountry = ref<CountryCode>('TR')
const phoneCountries = computed(() => {
  const names = new Intl.DisplayNames([locale.value], { type: 'region' })
  return getCountries().map(country => ({
    value: country,
    name: names.of(country) || country,
    label: `${[...country].map(letter => String.fromCodePoint(127397 + letter.charCodeAt(0))).join('')} +${getCountryCallingCode(country)} ${names.of(country) || country}`,
  })).sort((a, b) => a.name.localeCompare(b.name, locale.value))
})
const showPassword = ref(false)
const showConfirmPassword = ref(false)

// Login form validation rules
const formRules = computed(() => deferRules({
  email: [
    { required: true, message: t('auth.emailRequired'), type: 'error' },
    { email: true, message: t('auth.emailInvalid'), type: 'error' }
  ],
  password: [
    { required: true, message: t('auth.passwordRequired'), type: 'error' },
    { min: 8, message: t('auth.passwordMinLength'), type: 'error' },
    { max: 32, message: t('auth.passwordMaxLength'), type: 'error' }
  ],
}))

// Register form validation rules
const registerRules = computed(() => deferRules({
  username: [
    { required: true, message: t('auth.usernameRequired'), type: 'error' },
    { min: 2, message: t('auth.usernameMinLength'), type: 'error' },
    { max: 20, message: t('auth.usernameMaxLength'), type: 'error' },
    {
      pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5]+$/,
      message: t('auth.usernameInvalid'),
      type: 'error'
    }
  ],
  first_name: [{ validator: (value: string) => !!value?.trim(), message: t('auth.firstNameRequired'), type: 'error' }],
  last_name: [{ validator: (value: string) => !!value?.trim(), message: t('auth.lastNameRequired'), type: 'error' }],
  phone: [
    { required: true, message: t('auth.phoneRequired'), type: 'error' },
    { validator: (value: string) => !!normalizePhoneNumber(value || '', phoneCountry.value),
      message: t('auth.phoneInvalid'), type: 'error' },
  ],
  email: [
    { required: true, message: t('auth.emailRequired'), type: 'error' },
    { email: true, message: t('auth.emailInvalid'), type: 'error' }
  ],
  password: newPasswordRules(t, complexPasswordEnabled.value),
  confirmPassword: [
    { required: true, message: t('auth.confirmPasswordRequired'), type: 'error' },
    {
      validator: (val: string) => val === registerData.password,
      message: t('auth.passwordMismatch'),
      type: 'error'
    }
  ]
}))

const loginValidation = useDeferredValidation(formRef, formData)
const registerValidation = useDeferredValidation(registerFormRef, registerData, {
  dependents: { password: ['confirmPassword'] },
})

// Toggle login/register mode
const toggleMode = () => {
  isRegisterMode.value = !isRegisterMode.value
  showPassword.value = false
  showConfirmPassword.value = false

  Object.keys(registerData).forEach(key => {
    (registerData as any)[key] = ''
  })
  phoneCountry.value = 'TR'
  void loginValidation.reset()
  void registerValidation.reset()
}

const persistLoginResponse = async (response: any, skipRedirect = false) => {
  // Backend renamed `tenant` to `active_tenant` and added `memberships`
  // when tenant-level RBAC landed (issue #1303). The two are otherwise
  // identical — `active_tenant` is the tenant whose ID is encoded in the
  // JWT, defaulting to the user's home tenant on a fresh login.
  const activeTenant = response.active_tenant || response.tenant
  if (response.user && response.token) {
    // user.tenant_id must be the user's HOME tenant (the immutable row
    // on the users table); useHomeTenant() and the home-badge logic both
    // assume so. The ACTIVE tenant (which can differ from home when the
    // server honoured a remembered last-active-tenant preference) is
    // expressed separately via setSelectedTenant below.
    const homeTenantIdRaw = response.user.tenant_id ?? activeTenant?.id ?? ''
    authStore.setUser(userInfoFromApi(response.user, homeTenantIdRaw))
    authStore.setToken(response.token)
    if (response.refresh_token) {
      authStore.setRefreshToken(response.refresh_token)
    }
    if (activeTenant) {
      authStore.setTenant({
        id: String(activeTenant.id) || '',
        name: activeTenant.name || '',
        owner_id: response.user.id || '',
        created_at: activeTenant.created_at || new Date().toISOString(),
        updated_at: activeTenant.updated_at || new Date().toISOString()
      })
    } else {
      authStore.setTenant(null)
    }
    if (Array.isArray(response.memberships)) {
      authStore.setMemberships(response.memberships)
    }
    // If the backend dropped us into a non-home tenant (honoured a
    // remembered "last active tenant" preference), set the override so
    // subsequent requests carry X-Tenant-ID and the UI stays consistent.
    // Otherwise clear any stale override left in localStorage by a
    // previous session for a different account.
    const activeIdNum = Number(activeTenant?.id)
    const homeIdNum = Number(homeTenantIdRaw)
    if (Number.isFinite(activeIdNum) && Number.isFinite(homeIdNum) && activeIdNum !== homeIdNum) {
      authStore.setSelectedTenant(activeIdNum, activeTenant?.name || null)
    } else {
      authStore.setSelectedTenant(null, null)
    }
  }

  // Pull runtime capabilities (including whether ordinary users may create
  // workspaces) before entering the main UI so create actions never flash
  // briefly when the deployment is invitation-only.
  await authStore.refreshFromAuthMe()
  await nextTick()
  if (skipRedirect) return
  router.replace(authStore.hasValidTenant ? '/platform/knowledge-bases' : '/onboarding/workspace')
}

const getBackendOIDCRedirectURI = () => `${window.location.origin}/api/v1/auth/oidc/callback`

const loadOIDCConfig = async () => {
  try {
    const response = await getOIDCConfig()
    oidcEnabled.value = !!response.success && !!response.enabled
    oidcProviderName.value = response.provider_display_name || ''
  } catch {
    oidcEnabled.value = false
    oidcProviderName.value = ''
  }
}

// loadAuthConfig fetches /auth/config and caches whether self-service
// registration is allowed. Failures fall back to "enabled" so a transient
// network glitch doesn't lock new users out of an open deployment.
const loadAuthConfig = async () => {
  try {
    const response = await getAuthConfig()
    registrationEnabled.value = response.registration_mode !== 'invite_only'
    complexPasswordEnabled.value = response.complex_password_enabled
  } catch {
    registrationEnabled.value = true
    complexPasswordEnabled.value = false
  }
}

const handleOIDCLogin = async () => {
  try {
    oidcLoading.value = true
    const response = await getOIDCAuthorizationURL(getBackendOIDCRedirectURI())
    const authorizationURL = response.authorization_url

    if (!response.success || !authorizationURL) {
      MessagePlugin.error(response.message || t('auth.oidcLoginFailed'))
      return
    }

    // IdP'ye yönlendirme URL'deki token'ı kaybeder; sessionStorage'a geçici olarak kaydedilir ve geri çağrıdan sonra App.vue tarafından değiştirilir.
    if (inviteToken.value) {
      sessionStorage.setItem('rethra_pending_invite_token', inviteToken.value)
    }
    window.location.href = authorizationURL
  } catch (error: any) {
    console.error('OIDC login redirect failed:', error)
    MessagePlugin.error(error.message || t('auth.oidcLoginFailed'))
  } finally {
    oidcLoading.value = false
  }
}

// token ile alana katıl ve uygulamaya gir. Oturum bu noktada zaten geçerli olduğundan, token geçersiz olsa bile normal şekilde girilir (giriş sayfasında takılı kalmayı önlemek için).
const acceptAndEnter = async (token: string) => {
  loading.value = true
  try {
    const result = await authStore.acceptInvitationByTokenAndRefresh(token)
    if (result.ok) {
      MessagePlugin.success(t('inviteRegister.joined'))
    } else {
      MessagePlugin.warning(t('inviteRegister.invalidBody'))
    }
  } catch {
    MessagePlugin.warning(t('inviteRegister.invalidBody'))
  } finally {
    loading.value = false
    await nextTick()
    router.replace('/platform/knowledge-bases')
  }
}

// Handle login
const handleLogin = async () => {
  try {
    if (!(await loginValidation.validateAll())) return

    loading.value = true

    const response = await login({
      email: formData.email,
      password: formData.password,
    })

    if (response.success) {
      if (inviteToken.value) {
        // Davet bağlantısından giriş: kalıcı oturumdan sonra token'ı değiştir ve ilgili alana gir.
        await persistLoginResponse(response, true)
        await acceptAndEnter(inviteToken.value)
        return
      }
      await persistLoginResponse(response)
      notifyLoginSuccess(response, t, tm, formatRole, roleIcon)
    } else {
      MessagePlugin.error(response.message || t('auth.loginError'))
    }
  } catch (error: any) {
    console.error('Login failed:', error)
    MessagePlugin.error(error.message || t('auth.loginErrorRetry'))
  } finally {
    loading.value = false
  }
}

// Handle registration. Dispatches based on whether the user arrived
// with a share-link token: with token -> register-by-invite (auto-
// login on success); without -> the normal self-service register
// (drops back to the login form for the user to sign in).
const handleRegister = async () => {
  try {
    if (!(await registerValidation.validateAll())) return

    const phone = normalizePhoneNumber(registerData.phone, phoneCountry.value)
    if (!phone) return

    loading.value = true

    if (inviteToken.value) {
      const response = await registerByInvite({
        token: inviteToken.value,
        username: registerData.username,
        first_name: registerData.first_name.trim(),
        last_name: registerData.last_name.trim(),
        phone,
        email: registerData.email,
        password: registerData.password,
      })
      if (!response.success) {
        MessagePlugin.error(response.message || t('auth.registerFailed'))
        return
      }
      MessagePlugin.success(t('auth.registerSuccess'))
      // register-by-invite returns the same shape as login (token +
      // active_tenant + memberships), so reuse the login persistence
      // path — same store writes, same redirect target.
      await persistLoginResponse(response)
      return
    }

    const response = await register({
      username: registerData.username,
      first_name: registerData.first_name.trim(),
      last_name: registerData.last_name.trim(),
      phone,
      email: registerData.email,
      password: registerData.password
    })

    if (response.success) {
      MessagePlugin.success(t('auth.registerSuccess'))

      // Switch to login mode and fill in email
      isRegisterMode.value = false
      formData.email = registerData.email

      // Clear register form
      Object.keys(registerData).forEach(key => {
        (registerData as any)[key] = ''
      })
      void loginValidation.reset()
      void registerValidation.reset()
    } else {
      MessagePlugin.error(response.message || t('auth.registerFailed'))
    }
  } catch (error: any) {
    console.error('Registration failed:', error)
    MessagePlugin.error(error.message || t('auth.registerError'))
  } finally {
    loading.value = false
  }
}

// Check if already logged in.
onMounted(async () => {
  // Share-link landing: ?token=xxx switches the form into invite-
  // register mode before any other auto-flow (logged-in redirect /
  // OIDC) gets a chance to redirect. Resolution failure
  // surfaces inline; the user can still log in normally if they
  // already have an account. We check this BEFORE the isLoggedIn
  // redirect so an existing session doesn't bounce the user to
  // /platform (and possibly back to /login if the session is stale),
  // dropping the invite token along the way.
  const tokenFromQuery = String(route.query.token || '').trim()
  if (tokenFromQuery) {
    inviteToken.value = tokenFromQuery
    inviteLookupLoading.value = true
    // 1. Önce token'ı doğrula: geçersiz/süresi dolmuşsa hata vererek giriş sayfasında kal, kayıt moduna girme.
    try {
      const resp = await getInvitationByToken(tokenFromQuery)
      if (resp.success && resp.data) {
        inviteLookup.value = resp.data
      } else {
        inviteLookupError.value = resp.message || t('inviteRegister.invalidBody')
        loadOIDCConfig()
        loadAuthConfig()
        return
      }
    } catch {
      inviteLookupError.value = t('inviteRegister.invalidBody')
      loadOIDCConfig()
      loadAuthConfig()
      return
    } finally {
      inviteLookupLoading.value = false
    }

    // 2. Zaten giriş yapılmışsa token'ı doğrudan değiştirip alana gir (her iki mod için ortak).
    if (authStore.isLoggedIn && (await authStore.refreshFromAuthMe())) {
      await acceptAndEnter(tokenFromQuery)
      return
    }

    // 3. Giriş yapılmadıysa: kayıt moduna göre arayüzü belirle. invite_only giriş sayfasında kalır ve girişten sonra değiştirilir; self_serve kayıt akışını korur.
    const cfg = await getAuthConfig()
    const inviteOnly = cfg.registration_mode === 'invite_only'
    registrationEnabled.value = !inviteOnly
    isRegisterMode.value = !inviteOnly
    loadOIDCConfig()
    return
  }

  if (authStore.isLoggedIn) {
    router.replace('/platform/knowledge-bases')
    return
  }

  loadOIDCConfig()
  loadAuthConfig()
})
</script>

<style lang="less" scoped>
.login-layout {
  --auth-bg: #fefefd;
  --auth-card: #ffffff;
  --auth-fg: #262626;
  --auth-muted: #737373;
  --auth-border: #e2e8e5;
  --auth-primary: #17b88b;
  --auth-ray-color: rgba(34, 197, 94, 0.1);

  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  min-height: 100dvh;
  width: 100%;
  padding: 32px 16px;
  box-sizing: border-box;
  overflow: hidden;
  background: var(--auth-bg);
  color: var(--auth-fg);
  font-family: var(--app-font-family);
}

/* ---------- Light rays ---------- */
.light-rays {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
  isolation: isolate;
  opacity: 0.35;
}

.light-rays__glow {
  position: absolute;
  inset: 0;
  opacity: 0.6;

  &--left {
    background: radial-gradient(circle at 20% 15%, color-mix(in srgb, var(--auth-ray-color) 45%, transparent), transparent 70%);
  }

  &--right {
    background: radial-gradient(circle at 80% 10%, color-mix(in srgb, var(--auth-ray-color) 35%, transparent), transparent 75%);
  }
}

.light-ray {
  position: absolute;
  top: -12%;
  left: var(--ray-left);
  width: var(--ray-width);
  height: 70vh;
  border-radius: var(--app-radius-pill);
  transform-origin: top center;
  transform: translateX(-50%) rotate(var(--ray-rotate));
  background: linear-gradient(to bottom, color-mix(in srgb, var(--auth-ray-color) 70%, transparent), transparent);
  mix-blend-mode: screen;
  filter: blur(34px);
  opacity: 0;
  animation: rayPulse var(--ray-duration) ease-in-out var(--ray-delay) infinite;
  will-change: opacity, transform;
}

@keyframes rayPulse {
  0%,
  100% {
    opacity: 0;
    transform: translateX(-50%) rotate(calc(var(--ray-rotate) - var(--ray-swing)));
  }

  45% {
    opacity: var(--ray-intensity);
    transform: translateX(-50%) rotate(calc(var(--ray-rotate) + var(--ray-swing)));
  }

  90% {
    opacity: 0;
    transform: translateX(-50%) rotate(calc(var(--ray-rotate) - var(--ray-swing)));
  }
}

/* ---------- Card ---------- */
.auth-card {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 384px;
  box-sizing: border-box;
  padding: 32px 28px;
  border-radius: 40px;
  background: var(--auth-card);
  box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.16);
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.auth-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  text-align: center;
}

.auth-logo {
  width: 80px;
  height: 80px;
  margin-bottom: 8px;
  object-fit: contain;
}

.auth-title {
  margin: 0;
  font-size: var(--app-text-4xl);
  line-height: 32px;
  font-weight: 600;
  color: var(--auth-fg);
}

.auth-subtitle {
  margin: 0;
  font-size: var(--app-text-base);
  line-height: 20px;
  color: var(--auth-muted);
}

/* ---------- Form ---------- */
.auth-form {
  display: flex;
  flex-direction: column;

  :deep(.t-form__item) {
    margin-bottom: 20px;
  }

  :deep(.t-form__label) {
    padding: 0;
    margin-bottom: 8px;
    min-height: 0;
    line-height: 1;
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--auth-fg);

    label {
      line-height: 1;
    }
  }

  :deep(.t-form__controls) {
    min-height: 0;
  }

  .phone-input {
    display: flex;
    width: 100%;
    gap: 8px;

    :deep(.t-select) {
      width: 135px;
      flex: none;
    }

    :deep(.t-input__wrap) {
      min-width: 0;
    }
  }

  :deep(.t-input) {
    height: 36px;
    padding: 0 14px;
    border-radius: var(--app-radius-pill);
    border: 1px solid var(--auth-border);
    background: var(--auth-bg);
    box-shadow: none;
    font-size: var(--app-text-base);
    transition: border-color var(--app-motion-fast) ease, box-shadow var(--app-motion-fast) ease;

    &:hover {
      border-color: color-mix(in srgb, var(--auth-fg) 22%, var(--auth-border));
    }

    &.t-is-focused,
    &:focus-within {
      border-color: color-mix(in srgb, var(--auth-fg) 48%, var(--auth-border));
      box-shadow: 0 0 0 3px color-mix(in srgb, var(--auth-fg) 8%, transparent);
    }

    .t-input__inner {
      font-size: var(--app-text-base);
      color: var(--auth-fg);

      &::placeholder {
        color: var(--auth-muted);
      }
    }

    .t-input__suffix-icon {
      color: var(--auth-muted);
      font-size: var(--app-text-xl);
    }
  }

  :deep(.t-is-error .t-input) {
    border-color: var(--td-error-color);
  }

  :deep(.auth-confirm-password .t-input__suffix-icon) {
    display: none;
  }

  :deep(.auth-password .t-input__suffix-icon) {
    display: none;
  }

  :deep(.auth-password-toggle) {
    display: grid;
    place-items: center;
    width: 24px;
    height: 30px;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--auth-muted);
    cursor: pointer;

    svg {
      width: 16px;
      height: 16px;
    }

    &:focus-visible {
      outline: 2px solid var(--auth-primary);
      outline-offset: 2px;
      border-radius: 4px;
    }
  }

  :deep(.t-input__extra),
  :deep(.t-form__controls .t-input__tips) {
    position: static;
    margin-top: 4px;
    padding-left: 14px;
    font-size: 12px;
    line-height: 16px;
    font-weight: 400;
  }

  :deep(.t-is-error .t-input__extra) {
    color: color-mix(in srgb, var(--td-error-color) 80%, var(--auth-muted));
  }

  :deep(.t-form__item:has(.t-is-error)) {
    margin-bottom: 12px;
  }
}

.submit-row {
  display: flex;
  justify-content: center;
}

.submit-button {
  &.t-button {
    height: 36px;
    width: auto;
    padding: 0 16px;
    border-radius: var(--app-radius-pill);
    border: none;
    background: var(--auth-primary);
    color: #fff;
    font-size: var(--app-text-base);
    font-weight: 500;
    box-shadow: none;
    transition: opacity var(--app-motion-fast) ease;

    &:hover:not(.t-is-disabled),
    &:focus-visible {
      background: var(--auth-primary);
      opacity: 0.8;
    }

    &:active:not(.t-is-disabled) {
      background: var(--auth-primary);
    }
  }
}

.auth-switch {
  margin: 0;
  text-align: center;
  font-size: var(--app-text-base);
  line-height: 20px;
  color: var(--auth-muted);
}

.auth-link {
  margin-left: 2px;
  color: var(--auth-primary);
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}

.oidc-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 20px;
  border-top: 1px solid var(--auth-border);
}

.oidc-section__label {
  margin: 0;
  text-align: center;
  font-size: var(--app-text-sm);
  color: var(--auth-muted);
}

.oidc-button {
  &.t-button {
    height: 36px;
    border-radius: var(--app-radius-pill);
    border: 1px solid var(--auth-border);
    background: transparent;
    color: var(--auth-fg);
    font-size: var(--app-text-base);
    font-weight: 500;

    &:hover:not(.t-is-disabled) {
      background: color-mix(in srgb, var(--auth-fg) 5%, transparent);
      border-color: var(--auth-border);
      color: var(--auth-fg);
    }
  }
}

/* ---------- Invite banner ---------- */
.invite-banner {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: var(--app-radius-xl);
  background: color-mix(in srgb, var(--auth-primary) 7%, transparent);
  border: 1px solid color-mix(in srgb, var(--auth-primary) 20%, transparent);
  color: var(--auth-fg);
}

.invite-banner__icon {
  margin-top: 2px;
  font-size: var(--app-text-xl);
  flex-shrink: 0;
  color: var(--auth-primary);
}

.invite-banner__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.invite-banner__title {
  font-size: var(--app-text-base);
  font-weight: 500;
  line-height: 1.4;
}

.invite-banner__hint {
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--auth-muted);
}

.invite-banner--error {
  background: var(--td-error-color-1);
  border-color: var(--td-error-color-3);
  color: var(--td-error-color);
  font-size: var(--app-text-md);
  justify-content: center;
  text-align: center;
}

/* ---------- Responsive ---------- */
@media (min-width: 640px) {
  .login-layout {
    padding: 40px 24px;
  }

  .auth-card {
    padding: 40px 32px;
  }

  .auth-card--wide {
    max-width: 600px;
  }

  .auth-form--grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 16px;

    .auth-form__full {
      grid-column: 1 / -1;
    }

    .phone-input {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 16px;

      > :deep(*),
      :deep(.t-select__wrap),
      :deep(.t-select) {
        width: 100%;
        min-width: 0;
      }
    }
  }
}

@media (prefers-reduced-motion: reduce) {
  .light-ray {
    animation: none;
    opacity: calc(var(--ray-intensity) * 0.5);
  }
}
</style>

<style lang="less">
html[theme-mode="dark"] {
  .login-layout {
    --auth-bg: #181818;
    --auth-card: #212121;
    --auth-fg: #ececec;
    --auth-muted: #9b9b9b;
    --auth-border: #303030;
  }

  .login-layout .light-rays {
    opacity: 0.15;
  }

  .login-layout .auth-card {
    box-shadow: none;
  }

  .login-layout .auth-form .t-input {
    background: rgba(255, 255, 255, 0.06);
    border-color: transparent;

    &:hover:not(.t-is-disabled) {
      background: rgba(255, 255, 255, 0.10);
    }

    &.t-is-focused,
    &:focus-within {
      background: rgba(255, 255, 255, 0.12);
      border-color: transparent;
    }
  }

  .login-layout .auth-form .t-is-error .t-input {
    border-color: var(--td-error-color);
  }

}
</style>
