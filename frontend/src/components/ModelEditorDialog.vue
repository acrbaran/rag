<template>
  <SettingDrawer :visible="dialogVisible" :title="isEdit ? $t('model.editor.editTitle') : $t('model.editor.addTitle')"
    :description="getModalDescription()" :icon="modelTypeIcon" :confirm-loading="saving" :cancel-disabled="saving"
    :confirm-text="$t('model.editor.saveAndClose')"
    :close-on-overlay-click="!saving" :close-on-esc-keydown="!saving"
    @update:visible="(v: boolean) => dialogVisible = v" @confirm="handleConfirm" @cancel="handleCancel">

    <template v-if="formData.source === 'remote'" #footer-left>
      <t-button variant="outline" @click="checkRemoteAPI" :loading="checking"
        :disabled="saving || !formData.modelName || !formData.baseUrl || selectedRemoteModels.length > 1">
        <template #icon>
          <t-icon v-if="!checking && remoteChecked && remoteAvailable" name="check-circle-filled"
            class="status-icon available" />
          <t-icon v-else-if="!checking && remoteChecked && !remoteAvailable" name="close-circle-filled"
            class="status-icon unavailable" />
        </template>
        {{ checking ? $t('model.editor.testing') : $t('model.editor.testConnection') }}
      </t-button>
      <span v-if="remoteChecked" :class="['connection-status', remoteAvailable ? 'success' : 'error']">
        {{ remoteAvailable ? $t('model.editor.connectionSuccess') : $t('model.editor.connectionFailed') }}
      </span>
    </template>

    <template #footer-extra>
      <div v-if="saveError" class="connection-result error" role="alert">
        <strong>{{ $t('modelSettings.toasts.saveFailed') }}</strong>
        <div class="connection-result__details" tabindex="0">{{ saveError }}</div>
      </div>
      <div v-if="formData.source === 'remote'" class="connection-feedback" aria-live="polite">
        <p class="connection-hint">{{ $t(isEdit ? 'model.editor.testDraftEditHint' : 'model.editor.testDraftHint') }}</p>
        <p v-if="remoteStale" class="connection-hint">{{ $t('model.editor.testStale') }}</p>
        <div v-if="remoteChecked && !remoteAvailable" class="connection-result error">
          <div class="connection-result__header">
            <strong>{{ $t('model.editor.connectionFailed') }}</strong>
            <t-button size="small" variant="text" @click="copyWithToast(remoteMessage, 'common.copied')">
              {{ $t('common.copy') }}
            </t-button>
          </div>
          <div class="connection-result__details" tabindex="0">{{ remoteMessage }}</div>
        </div>
      </div>
    </template>

    <t-form :inert="saving || undefined" ref="formRef" :data="formData" :rules="rules" layout="vertical">

      <section v-if="!isEdit" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionType') }}</h4>
        <div class="model-type-options" role="radiogroup" :aria-label="$t('model.editor.typeLabel')">
          <button
            v-for="opt in modelTypeChoices"
            :key="opt.value"
            type="button"
            class="model-type-option"
            :class="{ 'is-active': activeModelType === opt.value }"
            role="radio"
            :aria-checked="activeModelType === opt.value"
            @click="selectModelType(opt.value)"
          >
            <t-icon :name="opt.icon" class="model-type-option__icon" />
            <span class="model-type-option__label">{{ opt.label }}</span>
          </button>
        </div>
      </section>

      <!--
        Bölüm 1 — Model kaynağı + model adı (kaynak aşağıdaki alanları doğrudan belirlediği için tek bölümde)
      -->
      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionSource') }}</h4>

        <div class="form-item">
          <!--
            Bölüm başlığı zaten 「Model kaynağı」 dediği için burada label tekrarlanmaz,
            bölümlü denetim doğrudan section'ın ilk içeriği olarak sunulur; böylece 「çift başlık」 hissi önlenir.
          -->
          <div class="source-options" role="radiogroup" :aria-label="$t('model.editor.sourceLabel')">
            <button
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.source === 'remote' }"
              role="radio"
              :aria-checked="formData.source === 'remote'"
              @click="formData.source = 'remote'"
            >
              <t-icon name="cloud" class="source-option__icon" />
              <span class="source-option__label">{{ $t('model.editor.sourceRemote') }}</span>
            </button>
          </div>
        </div>
      </section>

      <!-- Uzak API yapılandırması-->
      <template v-if="formData.source === 'remote'">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionProvider') }}</h4>

          <!-- Sağlayıcı seçici-->
          <div class="form-item">
            <label class="form-label">{{ $t('model.editor.providerLabel') }}</label>
            <t-select v-model="formData.provider" :placeholder="$t('model.editor.providerPlaceholder')"
              :loading="loadingProviders" filterable
              @change="handleProviderChange"
              :popup-props="{ overlayClassName: 'wk-popover provider-select-popup', overlayInnerStyle: matchTriggerWidth }">
              <!--
                Seçili değer: simge + yerelleştirilmiş ad (açıklama yalnızca açılır listede gösterilir; giriş kutusunun fazla uzaması önlenir).
                Dizinde bu provider artık yoksa (sağlayıcı kaldırılmış / eski veri) özgün id gösterimine geri dönülür,
                aksi takdirde giriş kutusu tamamen boş kalır ve "sağlayıcı seçilmemiş" gibi görünür, ancak kaydedilirken yine de eklenir.
              -->
              <template #valueDisplay>
                <span v-if="formData.provider" class="provider-value">
                  <img v-if="selectedProviderIcon" :src="selectedProviderIcon" class="provider-icon" alt="" />
                  <span class="provider-value__name">{{ selectedProviderDisplayLabel }}</span>
                </span>
              </template>
              <!--
                show-overflow-tooltip=false: TDesign varsayılan olarak hover sırasında seçeneğin üzerinde bir
                tam label içeren küçük baloncuk gösterir; ancak buradaki seçenek zaten iki satırlıdır (ana ad + açıklama), bu nedenle
                kısaltma oluşmaz; tooltip yalnızca zaten vurgulanmış gri arka planla çakışır. Doğrudan kapatın。
              -->
              <t-option v-for="opt in providerOptions" :key="opt.value" :value="opt.value"
                :label="providerDisplayLabel(opt)" :show-overflow-tooltip="false">
                <div class="provider-option">
                  <img v-if="providerIcon(opt)" :src="providerIcon(opt)" class="provider-icon" alt="" />
                  <span v-else class="provider-icon provider-icon--placeholder" aria-hidden="true">
                    {{ providerDisplayLabel(opt).slice(0, 1) }}
                  </span>
                  <div class="provider-option__text">
                    <span class="provider-name">{{ providerDisplayLabel(opt) }}</span>
                    <span class="provider-desc">{{ providerDisplayDescription(opt) }}</span>
                  </div>
                </div>
              </t-option>
            </t-select>
            <p v-if="vendorDocLink" class="form-desc provider-doc-link">
              <a :href="vendorDocLink" target="_blank" rel="noopener noreferrer">
                {{ $t('model.editor.providerDocs', { provider: selectedProviderDisplayLabel }) }}
                <t-icon name="jump" size="12px" />
              </a>
            </p>
          </div>

          <div class="form-item">
            <label class="form-label required">{{ $t('model.editor.baseUrlLabel') }}</label>
            <t-input v-model="formData.baseUrl" :placeholder="getBaseUrlPlaceholder()" />
          </div>

          <div class="form-item">
            <label class="form-label" :class="{ required: apiKeyRequired }">{{ apiKeyLabel }}</label>
            <!--
              Edit mode: credentials live behind the /credentials subresource
              of the model — managed by the shared CredentialResource card,
              which now renders an INPUT-LOOKING row (32px tall, same border
              + radius as t-input) so it sits flush with the Base URL field
              yukarıdaki ve aşağıdaki özel istek başlığı denetimleri — artık yok
              "card inside a card" feel.
              Create mode: the resource doesn't exist yet, so we render a
              plain password input with a leading lock icon; TDesign's password
              input provides the show/hide toggle.
            -->
            <CredentialResource v-if="isEdit && props.modelData?.id" :api="credentialApi" :fields="credentialFields"
              :meta="credentialMeta" @changed="invalidateConnectionTest()" />
            <t-input v-else v-model="formData.apiKey" type="password"
              :placeholder="apiKeyPlaceholder"
              class="api-key-input" autocomplete="new-password" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <p v-if="apiKeyHint" class="form-desc">{{ apiKeyHint }}</p>
          </div>

          <div v-if="!isEdit" class="form-item">
            <t-button type="button" variant="outline" :loading="discoveryLoading"
              :disabled="!formData.baseUrl || discoveryLoading" @click="loadRemoteModels">
              {{ $t('model.editor.discoverModels') }}
            </t-button>
            <p v-if="discoveryError" class="form-desc" role="alert">{{ discoveryError }}</p>
          </div>

          <!--
            Model adı: sağlayıcının yerleşik dizini aday olarak kullanılır, ayrıca serbest girişe izin verilir (filterable + creatable).
            Dizindeki bir model seçildiğinde bağlam penceresi / çıktı sınırı / görsel / boyut otomatik olarak getirilir (yalnızca boş alanlar doldurulur).
          -->
          <div class="form-item">
            <label class="form-label required">{{ $t('model.modelName') }}</label>
            <t-select v-if="!isEdit && discoveredModels.length"
              v-model="selectedRemoteModels" multiple filterable clearable
              :placeholder="$t('model.editor.selectRemoteModels')"
              :popup-props="{ overlayClassName: 'wk-popover catalog-model-select-popup', overlayInnerStyle: matchTriggerWidth }"
              @change="handleRemoteModelSelection">
              <t-option v-for="id in discoveredModels" :key="id" :value="id" :label="id" />
            </t-select>
            <!--
              Yerleşik dizin olmadığında düz giriş kutusu kullanılır: TDesign select, sıfır seçenek olduğunda tüm
              açılır katmanı (hideEmptyPopup) gizler; creatable "Oluştur" satırı da bununla birlikte gizlenir, böylece
              kullanıcı yazabilir ancak göndermenin hiçbir yolu yoktur; odak kaybolduktan sonra giriş doğrudan kaybolur. Özel
              (OpenAI uyumlu arayüz) tam olarak bu durumdur, bu yüzden model adı hiç girilemez.
            -->
            <t-input
              v-else-if="catalogModelOptions.length === 0"
              v-model="formData.modelName"
              :placeholder="getModelNamePlaceholder()"
              clearable
              autocomplete="off"
              spellcheck="false"
            />
            <t-select
              v-else
              v-model="formData.modelName"
              filterable
              creatable
              clearable
              :placeholder="getModelNamePlaceholder()"
              :popup-props="{ overlayClassName: 'wk-popover catalog-model-select-popup', overlayInnerStyle: matchTriggerWidth }"
              @create="handleCatalogModelCreate"
              @change="handleCatalogModelChange"
            >
              <t-option v-for="opt in catalogModelOptions" :key="opt.value" :value="opt.value" :label="opt.label"
                :show-overflow-tooltip="false">
                <div class="catalog-model-option">
                  <span class="catalog-model-option__name">{{ opt.label }}</span>
                  <span v-if="opt.value !== opt.label" class="catalog-model-option__id">{{ opt.value }}</span>
                  <span class="catalog-model-option__badges">
                    <span v-if="opt.contextWindow" class="catalog-badge">{{ opt.contextWindow }}</span>
                    <span v-if="opt.dimension" class="catalog-badge">{{ $t('model.editor.dimensionLabel') }} {{ opt.dimension }}</span>
                    <span v-if="opt.reasoning" class="catalog-badge catalog-badge--accent">{{ $t('model.editor.catalog.reasoning') }}</span>
                    <span v-if="opt.vision" class="catalog-badge catalog-badge--accent">{{ $t('model.editor.catalog.vision') }}</span>
                  </span>
                </div>
              </t-option>
            </t-select>
            <p v-if="catalogModelOptions.length > 0 && !discoveredModels.length" class="form-desc">{{ $t('model.editor.catalog.hint') }}</p>
          </div>

          <div class="form-item">
            <label class="form-label">{{ $t('model.editor.displayNameLabel') }}</label>
            <t-input v-model="formData.displayName" :disabled="selectedRemoteModels.length > 1"
              :placeholder="$t('model.editor.displayNamePlaceholder')" />
            <p class="form-desc">{{ $t(selectedRemoteModels.length > 1 ? 'model.editor.batchDisplayNameDesc' : 'model.editor.displayNameDesc') }}</p>
          </div>

          <!--
            Sağlayıcının belirttiği ek secret alanları (LKEAP / Volcengine Rerank SecretKey vb.):
            Oluşturma modunda app_secret yazılır; düzenleme modunda yukarıdaki CredentialResource kartı tarafından yönetilir.
          -->
          <div v-if="secretExtraField && !isEdit" class="form-item">
            <label class="form-label" :class="{ required: secretExtraField.required }">{{ extraFieldDisplayLabel(secretExtraField) }}</label>
            <t-input v-model="formData.appSecret" type="password"
              :placeholder="secretExtraField.placeholder || ''" autocomplete="new-password" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <p v-if="secretExtraField.placeholder" class="form-desc">{{ secretExtraField.placeholder }}</p>
          </div>

          <!-- Sağlayıcının bildirdiği diğer ek alanlar: `type` değerine göre dinamik olarak render edilir, değer `extra_config[key]` içine kaydedilir-->
          <div v-for="field in plainExtraFields" :key="field.key" class="form-item">
            <label class="form-label" :class="{ required: field.required }">{{ extraFieldDisplayLabel(field) }}</label>
            <div v-if="field.type === 'boolean'" class="vision-toggle">
              <t-switch :model-value="extraConfigBool(field.key)"
                @update:model-value="(v: boolean) => setExtraConfig(field.key, v ? 'true' : 'false')" />
              <span v-if="extraFieldDisplayPlaceholder(field)" class="form-desc form-desc--inline">{{ extraFieldDisplayPlaceholder(field) }}</span>
            </div>
            <t-select v-else-if="field.type === 'select'" :model-value="formData.extraConfig[field.key] || ''"
              :placeholder="extraFieldDisplayPlaceholder(field)" clearable
              @update:model-value="(v: string) => setExtraConfig(field.key, v)">
              <t-option v-for="opt in (field.options || [])" :key="opt.value" :value="opt.value"
                :label="extraFieldDisplayOptionLabel(opt)" />
            </t-select>
            <t-input v-else-if="field.type === 'number'" :model-value="formData.extraConfig[field.key] || ''"
              type="number" :placeholder="extraFieldDisplayPlaceholder(field)"
              @update:model-value="(v: string | number) => setExtraConfig(field.key, String(v ?? ''))" />
            <t-input v-else-if="field.type === 'password'" :model-value="formData.extraConfig[field.key] || ''"
              type="password" :placeholder="extraFieldDisplayPlaceholder(field)" autocomplete="new-password" spellcheck="false"
              @update:model-value="(v: string) => setExtraConfig(field.key, v)">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <t-input v-else :model-value="formData.extraConfig[field.key] || ''"
              :placeholder="extraFieldDisplayPlaceholder(field)"
              @update:model-value="(v: string) => setExtraConfig(field.key, v)" />
          </div>

          <!-- Özel HTTP Header (`OpenAI Python SDK` içindeki `extra_headers` benzeri)-->
          <div class="form-item">
            <div class="custom-headers-header">
              <label class="form-label" style="margin-bottom: 0;">{{ $t('model.editor.customHeadersLabel') }}</label>
              <t-button variant="text" size="small" theme="primary" @click="addCustomHeader">
                <template #icon><t-icon name="add" /></template>
                {{ $t('model.editor.customHeadersAdd') }}
              </t-button>
            </div>
            <p class="form-desc custom-headers-desc">{{ $t('model.editor.customHeadersDesc') }}</p>
            <div v-if="formData.customHeaders && formData.customHeaders.length > 0" class="custom-headers-list">
              <div v-for="(item, idx) in formData.customHeaders" :key="idx" class="custom-header-row">
                <t-input v-model="item.key" :placeholder="$t('model.editor.customHeadersKeyPlaceholder')"
                  class="custom-header-key" />
                <t-input v-model="item.value" :placeholder="$t('model.editor.customHeadersValuePlaceholder')"
                  class="custom-header-value" />
                <t-button variant="text" shape="square" size="small" class="custom-header-remove"
                  @click="removeCustomHeader(idx)" :aria-label="$t('common.delete')">
                  <t-icon name="close" />
                </t-button>
              </div>
            </div>
          </div>

          <!--
            Entegrasyon tanısı: arka uç dizininden gerçek zamanlı çözümlenen geçerli protokol / düşünme biçimi / destek seviyesi /
            dizinde olup olmadığı / bağlam penceresi, salt okunur. Sağlayıcı, model adı ve Base URL değiştikten sonra 400ms gecikmeli yenilenir.
            Yalnızca sohbet / VLM için anlamlıdır — Embedding, ReRank ve ASR'nin ne düşünme seviyesi ne de
            bağlam penceresi vardır; bunları göstermek yalnızca bu değerlerin onlar için de geçerli olduğu izlenimini verir.
          -->
          <div v-if="showResolvedPanel" class="form-item">
            <div class="resolved-panel" aria-live="polite">
              <div class="resolved-panel__header">
                <t-icon name="system-code" class="resolved-panel__icon" />
                <span class="resolved-panel__title">{{ $t('model.editor.resolved.title') }}</span>
                <t-icon v-if="resolving" name="loading" class="spinning resolved-panel__loading" />
              </div>
              <p v-if="!resolved && !resolving && !resolveFailed" class="form-desc">{{ $t('model.editor.resolved.empty') }}</p>
              <!-- Arka uç hata gövdesinin biçimi sabit değildir; okunabilir bir metin alınamazsa yalnızca başlığı göster, `[object Object]` gösterme-->
              <p v-else-if="resolveFailed" class="form-desc form-desc--warn">
                {{ $t('model.editor.resolved.failed') }}<template v-if="resolveError">: {{ resolveError }}</template>
              </p>
              <dl v-else-if="resolved" class="resolved-panel__grid">
                <dt>{{ $t('model.editor.resolved.protocol') }}</dt>
                <dd><code>{{ resolved.api }}</code></dd>
                <dt>{{ $t('model.editor.resolved.catalog') }}</dt>
                <dd>
                  <span class="catalog-badge" :class="{ 'catalog-badge--accent': resolved.cataloged }">
                    {{ resolved.cataloged ? $t('model.editor.resolved.catalogedYes') : $t('model.editor.resolved.catalogedNo') }}
                  </span>
                </dd>
                <template v-if="resolved.url">
                  <dt>{{ $t('model.editor.resolved.endpoint') }}</dt>
                  <dd><code class="resolved-endpoint">{{ resolved.url }}</code></dd>
                </template>
                <template v-if="resolved.remote_model && resolved.remote_model !== formData.modelName">
                  <dt>{{ $t('model.editor.advanced.remoteModelName.label') }}</dt>
                  <dd><code>{{ resolved.remote_model }}</code></dd>
                </template>
                <template v-if="isChatLike">
                  <dt>{{ $t('model.editor.resolved.thinkingFormat') }}</dt>
                  <dd><code>{{ resolved.capabilities?.thinking_format || '-' }}</code></dd>
                  <dt>{{ $t('model.editor.resolved.thinkingLevels') }}</dt>
                  <dd>
                    <template v-if="resolvedThinkingLevels.length > 0">
                      <span v-for="level in resolvedThinkingLevels" :key="level" class="catalog-badge">
                        {{ $t(levelLabelKey(level)) }}
                      </span>
                    </template>
                    <span v-else class="form-desc form-desc--inline">{{ $t('model.editor.resolved.noThinking') }}</span>
                  </dd>
                  <template v-if="resolved.capabilities?.context_window">
                    <dt>{{ $t('model.editor.contextWindowLabel') }}</dt>
                    <dd>{{ formatTokenCount(resolved.capabilities.context_window) }}</dd>
                  </template>
                  <template v-if="resolved.capabilities?.max_output_tokens">
                    <dt>{{ $t('model.editor.maxOutputTokensLabel') }}</dt>
                    <dd>{{ formatTokenCount(resolved.capabilities.max_output_tokens) }}</dd>
                  </template>
                </template>
              </dl>
            </div>
          </div>

          <!--
            Connection test action moved to the drawer footer (footer-left
            slot above) so primary actions live in one row at the bottom.
          -->
        </section>
      </template>

      <!-- Bölüm 3 — Gelişmiş seçenekler (yalnızca içerik varsa render edilir; boş bir bölümün alt ayırıcı çizgi göstermesini önler)-->
      <section v-if="['embedding', 'chat', 'vllm'].includes(activeModelType) || formData.source === 'remote'" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionAdvanced') }}</h4>

        <!-- Yalnızca Embedding için: boyut-->
        <div v-if="activeModelType === 'embedding'" class="form-item">
          <label class="form-label">{{ $t('model.editor.dimensionLabel') }}</label>
          <div class="dimension-control">
            <t-input v-model.number="formData.dimension" type="number" :min="128" :max="4096"
              :placeholder="$t('model.editor.dimensionPlaceholder')"
              :disabled="!formData.supportsDimensionOverride" />
          </div>
          <p v-if="dimensionChecked && dimensionMessage" class="dimension-hint" :class="{ success: dimensionSuccess }">
            {{ dimensionMessage }}
          </p>
        </div>

        <div v-if="activeModelType === 'embedding'" class="form-item">
          <label class="form-label">{{ $t('model.editor.dimensionOverrideLabel') }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.supportsDimensionOverride" />
            <span class="form-desc form-desc--inline">{{ $t('model.editor.dimensionOverrideDesc') }}</span>
          </div>
        </div>

        <!-- Chat / VLM: context window. Agent compaction sizes itself from this. -->
        <div v-if="activeModelType === 'chat' || activeModelType === 'vllm'" class="form-item">
          <label class="form-label">{{ $t('model.editor.contextWindowLabel') }}</label>
          <t-input v-model.number="formData.contextWindow" type="number" :min="1024" :max="10000000"
            :placeholder="$t('model.editor.contextWindowPlaceholder', { value: DEFAULT_MODEL_CONTEXT_WINDOW })" />
          <p class="form-desc">{{ $t('model.editor.contextWindowDesc') }}</p>
        </div>

        <!-- Chat: supports vision toggle (VLLM models are inherently multimodal) -->
        <div v-if="activeModelType === 'chat'" class="form-item">
          <label class="form-label">{{ $t('model.editor.supportsVisionLabel') }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.supportsVision" />
            <span class="form-desc form-desc--inline">{{ $t('model.editor.supportsVisionDesc') }}</span>
          </div>
        </div>

        <!-- Chat / VLM: max output tokens (catalog default when empty). -->
        <div v-if="activeModelType === 'chat' || activeModelType === 'vllm'" class="form-item">
          <label class="form-label">{{ $t('model.editor.maxOutputTokensLabel') }}</label>
          <t-input v-model.number="formData.maxOutputTokens" type="number" :min="1" :max="10000000"
            :placeholder="$t('model.editor.maxOutputTokensPlaceholder')" />
          <p class="form-desc">{{ $t('model.editor.maxOutputTokensDesc') }}</p>
        </div>

        <!--
          Background concurrency cap for this model. Only chat / embedding / vllm
          are gated by the governor (see internal/models/limiter), so we surface
          it just for those three. 0 = fall back to the global default.
        -->
        <div v-if="['embedding', 'chat', 'vllm'].includes(activeModelType)" class="form-item">
          <label class="form-label">{{ $t('model.editor.maxConcurrencyLabel') }}</label>
          <t-input v-model.number="formData.maxConcurrency" type="number" :min="0" :max="4096"
            :placeholder="$t('model.editor.maxConcurrencyPlaceholder')" />
          <p class="form-desc">{{ $t('model.editor.maxConcurrencyDesc') }}</p>
        </div>

        <!--
          Gelişmiş: protokol geçersiz kılma / uzak model adı / dizin compat geçersiz kılması ve yalnızca eski verilerde gösterilen
          thinking_control uyumluluk seçimi. Varsayılan olarak daraltılır; kullanıcıların büyük çoğunluğunun dokunmasına gerek yoktur.
        -->
        <template v-if="formData.source === 'remote'">
          <button type="button" class="advanced-toggle" :aria-expanded="advancedOpen" @click="advancedOpen = !advancedOpen">
            <t-icon name="chevron-right" class="toggle-arrow" :class="{ open: advancedOpen }" />
            <span>{{ $t('model.editor.advanced.toggle') }}</span>
          </button>

          <template v-if="advancedOpen">
            <!--
              extra_config.api yalnızca sohbet protokolünü seçer. embedding / rerank satırları bunu okumaz:
              embedding için protokol geçersiz kılması aşağıdaki compat JSON içinde ("api") yazılır; değer vektör protokolüdür.
            -->
            <div v-if="isChatLike" class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.api.label') }}</label>
              <t-select :model-value="formData.extraConfig.api || ''" clearable
                @update:model-value="(v: string) => setExtraConfig('api', v)">
                <t-option value="" :label="$t('model.editor.advanced.api.auto')" />
                <t-option v-for="api in PROTOCOL_OPTIONS" :key="api" :value="api" :label="api" />
              </t-select>
              <p class="form-desc">{{ $t('model.editor.advanced.api.desc') }}</p>
            </div>

            <div class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.remoteModelName.label') }}</label>
              <t-input :model-value="formData.extraConfig.remote_model_name || ''"
                :placeholder="$t('model.editor.advanced.remoteModelName.placeholder')"
                @update:model-value="(v: string) => setExtraConfig('remote_model_name', v)" />
              <p class="form-desc">{{ $t('model.editor.advanced.remoteModelName.desc') }}</p>
            </div>

            <!-- Yalnızca eski veriler: `extra_config.thinking_control` mevcutsa göster; temizlenebilir ve dizin varsayılanına dönülebilir-->
            <div v-if="showLegacyThinkingControl" class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.legacyThinking.label') }}</label>
              <t-select :model-value="formData.thinkingControl || ''"
                @update:model-value="(v: string) => formData.thinkingControl = v">
                <t-option value="" :label="$t('model.editor.advanced.legacyThinking.catalog')" />
                <t-option v-for="opt in LEGACY_THINKING_CONTROL_VALUES" :key="opt" :value="opt"
                  :label="opt === 'none' ? $t('model.editor.advanced.legacyThinking.none') : opt" />
              </t-select>
              <p class="form-desc form-desc--warn">{{ $t('model.editor.advanced.legacyThinking.desc') }}</p>
            </div>

            <div class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.compat.label') }}</label>
              <t-textarea v-model="formData.specCompat" :autosize="{ minRows: 3, maxRows: 10 }"
                :placeholder="$t('model.editor.advanced.compat.placeholder')" class="compat-textarea"
                :status="specCompatError ? 'error' : undefined" />
              <p v-if="specCompatError" class="form-desc form-desc--error">{{ $t('model.editor.advanced.compat.invalid') }}: {{ specCompatError }}</p>
              <p v-else class="form-desc">
                {{ $t('model.editor.advanced.compat.desc') }}
              </p>
            </div>
          </template>
        </template>
      </section>

    </t-form>
  </SettingDrawer>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import {
  checkRemoteModel, testEmbeddingModel, checkRerankModel, checkASRModel, resolveModelCatalog,
  type ModelProviderOption, type ModelProviderExtraField,
  type ModelProviderExtraFieldOption, type ModelCatalogEntry,
  type ResolvedModelCatalog,
} from '@/api/initialization'
import {
  discoverModels,
  putModelCredentials,
  deleteModelCredentialField,
  type ModelCredentialField,
  type ModelSpecOverride,
} from '@/api/model'
import { useI18n } from 'vue-i18n'
import { useModelProvidersStore } from '@/stores/modelProviders'
import {
  credentialLabelForModelType,
  extraFieldLabel,
  extraFieldOptionLabel,
  extraFieldPlaceholder,
  extraFieldsForModelType,
  pickLocalized,
  providerDescription,
  providerIcon,
  providerLabel,
} from '@/stores/modelProvidersState'
import { levelLabelKey, supportedLevels } from '@/utils/reasoningEffort'
import { DEFAULT_MODEL_CONTEXT_WINDOW, formatTokenCount } from '@/utils/contextWindow'
import { copyWithToast } from '@/utils/clipboard'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import CredentialResource, {
  type CredentialFieldDef,
  type CredentialResourceApi,
} from '@/components/credentials/CredentialResource.vue'

interface CustomHeaderItem {
  key: string
  value: string
}

interface ModelFormData {
  id: string
  name: string
  source: 'remote'
  provider?: string // Provider identifier: openai, aliyun, zhipu, generic, etc.
  modelName: string
  displayName?: string
  baseUrl?: string
  apiKey?: string
  dimension?: number
  supportsDimensionOverride?: boolean
  interfaceType?: 'openai'
  isDefault: boolean
  supportsVision?: boolean
  /** Sohbet/VLM bağlam penceresi (`token`). Boş/0, varsayılan `200000` değerinin kullanılması anlamına gelir.*/
  contextWindow?: number
  /** Arka plan görevleri için bu modelin eşzamanlılık üst sınırı; `0`/`undefined`, genel varsayılanın kullanılacağı anlamına gelir. Yalnızca `chat`/`embedding`/`vllm` için geçerlidir.*/
  maxConcurrency?: number
  /** Sohbet/VLM tek çıktı üst sınırı (`token`). Boş/0, dizin varsayılanının kullanılması anlamına gelir.*/
  maxOutputTokens?: number
  /**
   * Legacy extra_config.thinking_control (none | enable_thinking | thinking_type
   * | chat_template_kwargs). Only rows saved by older UIs carry it; the catalog
   * decides the encoding otherwise. Empty string = drop the key on save.
   */
  thinkingControl?: string
  /**
   * Provider-specific extra_config entries: vendor-declared extra fields
   * (api_version, region, ...) plus the advanced overrides `api` and
   * `remote_model_name`. thinking_control lives in its own field above.
   */
  extraConfig: Record<string, string>
  /** parameters.spec.compat as JSON text (validated before save). */
  specCompat?: string
  /** Other parameters.spec fields preserved verbatim from the loaded row. */
  spec?: ModelSpecOverride | null
  // Özel HTTP istek başlıkları (`OpenAI Python SDK` içindeki `extra_headers` benzeri)
  customHeaders?: CustomHeaderItem[]
  /** İkinci anahtar (sağlayıcının `LKEAP` / Volcengine Rerank `SecretKey` gibi ek `secret` alanı); oluşturulurken `app_secret` içine yazılır*/
  appSecret?: string
}

/** Protocols the backend accepts in extra_config.api (internal/models/api.API). */
const PROTOCOL_OPTIONS = [
  'openai-completions',
  'openai-responses',
  'anthropic-messages',
  'google-generative-ai',
] as const

/** Legacy thinking_control values still honoured by catalog.Resolve. */
const LEGACY_THINKING_CONTROL_VALUES = ['none', 'enable_thinking', 'thinking_type', 'chat_template_kwargs'] as const

/** Keys of extra_config that are edited by dedicated controls, not the generic renderer. */
const RESERVED_EXTRA_CONFIG_KEYS = new Set(['thinking_control'])

type EditorModelType = 'chat' | 'embedding' | 'rerank' | 'vllm' | 'asr'

interface Props {
  visible: boolean
  modelType: EditorModelType
  modelData?: ModelFormData | null
  saveModel: (data: ModelFormData & { modelType?: EditorModelType }) => Promise<void>
}

const { t, locale } = useI18n()
const providersStore = useModelProvidersStore()

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  modelData: null
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const draftModelType = ref<EditorModelType>(props.modelType)

const isEdit = computed(() => !!props.modelData)

const activeModelType = computed(() => (
  isEdit.value ? props.modelType : draftModelType.value
))

const modelTypeChoices = computed(() => ([
  { value: 'chat' as const, label: t('modelSettings.typeShort.chat'), icon: 'chat' },
  { value: 'embedding' as const, label: t('modelSettings.typeShort.embedding'), icon: 'chart-bubble' },
  { value: 'rerank' as const, label: t('modelSettings.typeShort.rerank'), icon: 'filter-sort' },
  { value: 'vllm' as const, label: t('modelSettings.typeShort.vllm'), icon: 'image' },
  { value: 'asr' as const, label: t('modelSettings.typeShort.asr'), icon: 'sound' },
]))

// Sağlayıcı listesi tamamen arka uç dizininden gelir (`store`, model türüne göre önbelleğe alır); ön uç artık hiçbir sağlayıcı tablosu tutmaz.
const loadingProviders = computed(() => providersStore.isLoading(activeModelType.value))

const loadProviders = async (force = false) => {
  try {
    await providersStore.ensureLoaded(activeModelType.value, force)
  } catch (error) {
    console.error('Failed to load providers from API', error)
  }
}

const providerOptions = computed<ModelProviderOption[]>(() => providersStore.providersFor(activeModelType.value))

const selectedProvider = computed<ModelProviderOption | undefined>(() => {
  const id = formData.value.provider
  if (!id) return undefined
  return providerOptions.value.find(p => p.value === id) || providersStore.providerById(id)
})

const currentLocale = computed(() => String(locale.value || ''))
const providerDisplayLabel = (p: ModelProviderOption) => providerLabel(p, currentLocale.value)
const providerDisplayDescription = (p: ModelProviderOption) => providerDescription(p, currentLocale.value)
const selectedProviderIcon = computed(() => providerIcon(selectedProvider.value))
/** Localized vendor name, or the raw stored id when the catalog no longer has it. */
const selectedProviderDisplayLabel = computed(() => (
  selectedProvider.value
    ? providerDisplayLabel(selectedProvider.value)
    : (formData.value.provider || '')
))
/**
 * Pin a select's dropdown to the width of its input.
 *
 * TDesign sizes a select popup as max(popup content, trigger)
 * (select-input/hooks/useOverlayInnerStyle), so one long option — a vendor
 * whose description lists half a dozen model ids — stretches the whole menu
 * past the field it belongs to, leaving a wide band of empty space and
 * pushing the tick mark far from the text. Passing a function here replaces
 * that matching outright (the hook keeps a function as-is), and the option
 * rows already ellipsize, so the long ones simply truncate.
 */
const matchTriggerWidth = (triggerElement: HTMLElement) => ({
  width: `${triggerElement.offsetWidth}px`,
})

const extraFieldDisplayLabel = (field: ModelProviderExtraField) => extraFieldLabel(field, currentLocale.value)
const extraFieldDisplayPlaceholder = (field: ModelProviderExtraField) =>
  extraFieldPlaceholder(field, currentLocale.value)
const extraFieldDisplayOptionLabel = (option: ModelProviderExtraFieldOption) =>
  extraFieldOptionLabel(option, currentLocale.value)

/**
 * Vendors whose API is not a bearer-token API name their first credential
 * themselves (LKEAP rerank takes a SecretId, Volcengine rerank an Access Key
 * ID). Falling back to the generic "API Key" wording is what led operators to
 * paste an `sk-` token into a signature field.
 */
const credentialLabel = computed(() =>
  credentialLabelForModelType(selectedProvider.value?.credentialLabels, activeModelType.value),
)
const apiKeyLabel = computed(() => {
  const label = credentialLabel.value
  return label ? pickLocalized(label.labels, currentLocale.value, label.label) : t('model.editor.apiKeyOptional')
})
const apiKeyRequired = computed(
  () => credentialLabel.value?.required === true || (!!selectedProvider.value?.requiresAuth && !isEdit.value),
)
const apiKeyHint = computed(() => {
  const label = credentialLabel.value
  if (!label?.hint) return ''
  return pickLocalized(label.hints, currentLocale.value, label.hint)
})

// Sağlayıcı ek alanları: geçerli model türüne göre filtrelenir; `secret` alanları `app_secret` kimlik bilgisini kullanır, diğerleri `extra_config` içine gider.
const visibleExtraFields = computed<ModelProviderExtraField[]>(() =>
  extraFieldsForModelType(selectedProvider.value?.extraFields, activeModelType.value)
    .filter(field => !RESERVED_EXTRA_CONFIG_KEYS.has(field.key)),
)
/**
 * A credential-bearing vendor field. `secret` is the backend's own marker;
 * `type: 'password'` is treated the same way so a vendor that forgets the flag
 * still cannot get its value echoed back from GET /models into extra_config.
 */
const isSecretExtraField = (field: ModelProviderExtraField) => field.secret === true || field.type === 'password'
// Only one credential slot (app_secret) exists per model, so the first such
// field wins; vendors today declare at most one (LKEAP / Volcengine rerank).
const secretExtraField = computed<ModelProviderExtraField | undefined>(() =>
  visibleExtraFields.value.find(isSecretExtraField),
)
const plainExtraFields = computed<ModelProviderExtraField[]>(() =>
  visibleExtraFields.value.filter(field => !isSecretExtraField(field)),
)

/** extra_config keys that belong to the connection, not to the vendor. */
const VENDOR_NEUTRAL_EXTRA_CONFIG_KEYS = ['api', 'remote_model_name'] as const

/** What survives a vendor (or model-type) switch: everything else is vendor-specific. */
const keepVendorNeutralExtraConfig = (): Record<string, string> => {
  const keep: Record<string, string> = {}
  for (const key of VENDOR_NEUTRAL_EXTRA_CONFIG_KEYS) {
    const value = formData.value.extraConfig?.[key]
    if (value) keep[key] = value
  }
  return keep
}

const setExtraConfig = (key: string, value: string | null | undefined) => {
  const next = { ...(formData.value.extraConfig || {}) }
  const normalized = value == null ? '' : String(value)
  if (normalized === '') delete next[key]
  else next[key] = normalized
  formData.value.extraConfig = next
}

const extraConfigBool = (key: string) => {
  const raw = (formData.value.extraConfig?.[key] || '').trim().toLowerCase()
  return raw === 'true' || raw === '1' || raw === 'yes'
}

/** Pre-fill vendor defaults for fields the user has not touched. */
const applyExtraFieldDefaults = () => {
  for (const field of plainExtraFields.value) {
    if (!field.default) continue
    if ((formData.value.extraConfig?.[field.key] ?? '') === '') {
      setExtraConfig(field.key, field.default)
    }
  }
}

// Yerleşik model dizini → model adı açılır adayları (serbestçe girilebilir)
interface CatalogModelOption {
  value: string
  label: string
  contextWindow: string
  dimension?: number
  reasoning: boolean
  vision: boolean
  entry?: ModelCatalogEntry
}

const catalogEntries = computed<ModelCatalogEntry[]>(() => {
  const entries = selectedProvider.value?.models || []
  // The list is already scoped: providers are fetched per model type, so the
  // backend returned exactly the entries that type can use. Filtering again
  // burada entry.type görsel için yanlıştı — bir VLM girdisi, şu olan bir sohbet modelidir
  // accepts images, so it arrives typed "chat" and every one of them was
  // dropped, leaving the picker empty for every vendor.
  return entries
})

const catalogModelOptions = computed<CatalogModelOption[]>(() => {
  const options: CatalogModelOption[] = catalogEntries.value.map(entry => ({
    value: entry.id,
    label: entry.name || entry.id,
    contextWindow: entry.context_window ? formatTokenCount(entry.context_window) : '',
    dimension: entry.dimension || undefined,
    reasoning: !!entry.reasoning || (entry.thinking_levels?.length ?? 0) > 0,
    vision: Array.isArray(entry.input) && entry.input.includes('image'),
    entry,
  }))
  for (const id of discoveredModels.value) {
    if (!options.some(o => o.value === id)) {
      options.push({ value: id, label: id, contextWindow: '', reasoning: false, vision: false })
    }
  }
  const current = (formData.value.modelName || '').trim()
  if (current && !options.some(o => o.value === current)) {
    options.unshift({ value: current, label: current, contextWindow: '', reasoning: false, vision: false })
  }
  return options
})

const discoveredModels = ref<string[]>([])
const selectedRemoteModels = ref<string[]>([])
const discoveryLoading = ref(false)
const discoveryError = ref('')
let discoveryRevision = 0

const loadRemoteModels = async () => {
  const revision = ++discoveryRevision
  discoveryLoading.value = true
  discoveryError.value = ''
  discoveredModels.value = []
  try {
    const ids = await discoverModels(formData.value.baseUrl!.trim(), formData.value.apiKey || '')
    if (revision !== discoveryRevision) return
    discoveredModels.value = ids
    selectedRemoteModels.value = ids.includes(formData.value.modelName) ? [formData.value.modelName] : []
    formData.value.modelName = selectedRemoteModels.value[0] || ''
    if (!ids.length) discoveryError.value = t('model.editor.noRemoteModels')
  } catch (error: any) {
    if (revision !== discoveryRevision) return
    discoveryError.value = error?.message || t('model.editor.discoverFailed')
  } finally {
    if (revision === discoveryRevision) discoveryLoading.value = false
  }
}

const handleRemoteModelSelection = (value: unknown) => {
  const names = Array.isArray(value) ? value.filter((name): name is string => typeof name === 'string') : []
  formData.value.modelName = names[0] || ''
}

const findCatalogEntry = (name: string) => catalogEntries.value.find(m => m.id === name)

/**
 * Where to read about what is configured right now.
 *
 * Prefer the page the selected model's facts were taken from — that is the
 * page listing its context window, thinking levels and price — and fall back
 * to the vendor's own site when the model is not in the catalog. Both come
 * from the catalog, so a new vendor gets the link without a UI change.
 */
const vendorDocLink = computed(() => {
  const provider = selectedProvider.value
  if (!provider) return ''
  const entry = findCatalogEntry((formData.value.modelName || '').trim())
  const url = entry?.source || provider.website || ''
  return /^https?:\/\//i.test(url) ? url : ''
})

/**
 * What applyCatalogEntry filled in for the model currently selected.
 *
 * Switching vendor has to take those values back — a context window from the
 * önceki sağlayıcının modeli tam olarak "formu doldurmak sıkıştırmanın tetiklenmemesine ve üst akışın doğrudan reddetmesine yol açar" durumudur
 * case this field warns about — but it must not touch a number the operator
 * typed. Remembering what was filled, and only clearing a field that still
 * holds it, separates the two.
 */
const catalogFilled = ref<Partial<ModelFormData>>({})

/** Fill blank capability fields from a catalog entry the user just picked. */
const applyCatalogEntry = (entry: ModelCatalogEntry) => {
  if (activeModelType.value === 'chat' || activeModelType.value === 'vllm') {
    if (!formData.value.contextWindow && entry.context_window) {
      formData.value.contextWindow = entry.context_window
      catalogFilled.value.contextWindow = entry.context_window
    }
    if (!formData.value.maxOutputTokens && entry.max_output_tokens) {
      formData.value.maxOutputTokens = entry.max_output_tokens
      catalogFilled.value.maxOutputTokens = entry.max_output_tokens
    }
  }
  if (activeModelType.value === 'chat' && !formData.value.supportsVision && Array.isArray(entry.input) && entry.input.includes('image')) {
    formData.value.supportsVision = true
    catalogFilled.value.supportsVision = true
  }
  if (activeModelType.value === 'embedding' && !formData.value.dimension && entry.dimension) {
    formData.value.dimension = entry.dimension
    catalogFilled.value.dimension = entry.dimension
  }
}

const handleCatalogModelCreate = (value: string | number | boolean | bigint) => {
  formData.value.modelName = String(value ?? '').trim()
}

const handleCatalogModelChange = (value: unknown) => {
  const name = typeof value === 'string' ? value.trim() : ''
  if (!name) return
  const entry = findCatalogEntry(name)
  if (entry) applyCatalogEntry(entry)
}

// Entegrasyon tanılama (salt okunur): sağlayıcı / model adı / Base URL / protokol geçersiz kılma değiştiğinde, 400ms gecikmeyle katalog çözümlemesini çağır
const resolved = ref<ResolvedModelCatalog | null>(null)
const resolving = ref(false)
const resolveFailed = ref(false)
/** Human-readable detail of the last failure; '' when the error carried none. */
const resolveError = ref('')
let resolveTimer: ReturnType<typeof setTimeout> | null = null
let resolveRevision = 0

/** Katalogdan çözümlenen protokol / düşünme seviyesi / bağlam penceresi yalnızca sohbet ve VLM modelleri için anlamlıdır.*/
const isChatLike = computed(() => activeModelType.value === 'chat' || activeModelType.value === 'vllm')

const showResolvedPanel = computed(() =>
  isChatLike.value
  && formData.value.source === 'remote'
  && !!formData.value.provider
 ,
)

const resolvedThinkingLevels = computed(() => supportedLevels(resolved.value?.capabilities))

const runResolve = async () => {
  const revision = ++resolveRevision
  const provider = (formData.value.provider || '').trim()
  // Embedding / ReRank / ASR never render the panel, so do not spend a
  // request (and do not leave a stale chat result behind) for them.
  if (!props.visible || !showResolvedPanel.value || !provider) {
    resolved.value = null
    resolving.value = false
    resolveFailed.value = false
    resolveError.value = ''
    return
  }
  resolving.value = true
  try {
    const result = await resolveModelCatalog({
      provider,
      spec: buildSpec(),
      model: formData.value.modelName || '',
      base_url: formData.value.baseUrl || '',
      model_type: activeModelType.value,
      api: formData.value.extraConfig?.api || '',
      thinking_control: formData.value.thinkingControl || '',
      remote_model_name: formData.value.extraConfig?.remote_model_name || '',
      // Vendor fields decide the request too — Azure's api_version picks
      // between the v1 data plane and the dated deployments path — so the
      // preview has to see them or it describes a different request than
      // the one this row will make. Secret fields never travel in a query
      // string; plainExtraFields already excludes them.
      ...Object.fromEntries(
        plainExtraFields.value
          .map(field => [field.key, formData.value.extraConfig?.[field.key] || ''])
          .filter(([, value]) => !!value),
      ),
    })
    if (revision !== resolveRevision) return
    resolved.value = result
    resolveFailed.value = false
    resolveError.value = ''
  } catch (error: any) {
    if (revision !== resolveRevision) return
    resolved.value = null
    resolveFailed.value = true
    const detail = typeof error === 'string'
      ? error
      : (error?.message || error?.error?.message || error?.error)
    resolveError.value = typeof detail === 'string' ? detail.trim() : ''
  } finally {
    if (revision === resolveRevision) resolving.value = false
  }
}

const scheduleResolve = () => {
  if (resolveTimer) clearTimeout(resolveTimer)
  resolveTimer = setTimeout(() => {
    resolveTimer = null
    void runResolve()
  }, 400)
}

// Gelişmiş daraltılabilir bölüm
const advancedOpen = ref(false)
/** Set when the loaded row carried thinking_control, so the select stays visible after clearing it. */
const legacyThinkingControlLoaded = ref(false)
const showLegacyThinkingControl = computed(() =>
  activeModelType.value === 'chat' && !!(formData.value.thinkingControl || legacyThinkingControlLoaded.value),
)

const specCompatError = computed(() => {
  const text = (formData.value.specCompat || '').trim()
  if (!text) return ''
  try {
    const parsed = JSON.parse(text)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return t('model.editor.advanced.compat.mustBeObject')
    }
    return ''
  } catch (error: any) {
    return error?.message || 'invalid JSON'
  }
})

// Match the parent save path: retain row metadata, replace the edited compat.
const buildSpec = (): ModelSpecOverride => {
  if (specCompatError.value) throw new Error(specCompatError.value)
  const spec: ModelSpecOverride = { ...(formData.value.spec || {}) }
  const text = (formData.value.specCompat || '').trim()
  if (text) spec.compat = JSON.parse(text)
  else delete spec.compat
  return spec
}

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => {
    if (!saving.value) emit('update:visible', val)
  }
})

/** Form modelData ile dolduruluyor; sağlayıcı/kaynak denetimlerinin programatik change yan etkileri yoksayılıyor*/
const hydratingForm = ref(false)

// Header icon for the SettingDrawer — uses the same TDesign icon name table
// as the model card list, so the drawer's leading badge visually matches the
// card the user just clicked on.
const modelTypeIcon = computed(() => {
  const map: Record<string, string> = {
    chat: 'chat',
    embedding: 'chart-bubble',
    rerank: 'filter-sort',
    vllm: 'image',
    asr: 'sound',
  }
  return map[activeModelType.value] || 'setting'
})

// Credential resource binding for the shared <CredentialResource> component.
// A vendor-declared secret extra field (LKEAP / Volcengine rerank SecretKey)
// is stored as the app_secret credential, so it shows up here in edit mode.
const credentialFields = computed<CredentialFieldDef<ModelCredentialField>[]>(() => {
  const fields: CredentialFieldDef<ModelCredentialField>[] = [
    { key: 'api_key', label: t('model.editor.apiKeyOptional') as string },
  ]
  if (secretExtraField.value) {
    fields.push({ key: 'app_secret', label: extraFieldDisplayLabel(secretExtraField.value) })
  }
  return fields
})

const credentialApi = computed<CredentialResourceApi<ModelCredentialField>>(() => {
  const id = props.modelData?.id ?? ''
  return {
    save: async (patch) => {
      const meta = await putModelCredentials(id, patch)
      return meta.fields
    },
    remove: async (field) => {
      await deleteModelCredentialField(id, field)
    },
  }
})

// Initial credential metadata. ModelSettings.convertToLegacyFormat
// preserves `credentials` from the main ListModels response so the card
// renders the correct "Configured" state on dialog open.
const credentialMeta = computed(() => (props.modelData as any)?.credentials ?? {
  api_key: { configured: false },
  app_secret: { configured: false },
})

// Placeholder hint for the create-mode API key input. Edit mode replaces
// this input entirely with a <CredentialResource> card.
const apiKeyPlaceholder = computed(() => {
  const label = credentialLabel.value
  if (label?.placeholder) return pickLocalized(label.placeholders, currentLocale.value, label.placeholder)
  return t('model.editor.apiKeyPlaceholder')
})

const formRef = ref()
const saving = ref(false)
const saveError = ref('')

// Settings itself listens on window for Escape. Capture it while saving so
// the parent cannot unmount this editor before the request finishes.
const handleSaveEscape = (event: KeyboardEvent) => {
  if (props.visible && saving.value && (event.key === 'Escape' || event.code === 'Escape')) {
    event.preventDefault()
    event.stopImmediatePropagation()
  }
}
watch(() => props.visible && saving.value, (locked) => {
  if (locked) window.addEventListener('keydown', handleSaveEscape, true)
  else window.removeEventListener('keydown', handleSaveEscape, true)
}, { flush: 'sync' })
// Toggles the create-mode API key input between masked and plain text. Lets
// the user proofread a freshly pasted secret without losing the password
// affordance for everyday use. Reset every time the drawer closes (see
// reset block in the visible watcher) so we never leak the previous value
// across editor sessions.
const checking = ref(false)
const remoteChecked = ref(false)
const remoteAvailable = ref(false)
const remoteMessage = ref('')
const remoteStale = ref(false)
let connectionRevision = 0
let applyingDetectedDimension = false

// Invalidate pending responses too, even when the user changes a field back.
const invalidateConnectionTest = (showStale = true) => {
  connectionRevision++
  remoteStale.value = showStale && (remoteStale.value || checking.value || remoteChecked.value)
  checking.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

const applyDetectedDimension = (dimension: number) => {
  applyingDetectedDimension = true
  try {
    formData.value.dimension = dimension
  } finally {
    applyingDetectedDimension = false
  }
}
const dimensionChecked = ref(false)
const dimensionSuccess = ref(false)
const dimensionMessage = ref('')

const formData = ref<ModelFormData>({
  id: '',
  name: '',
  source: 'remote',
  provider: 'generic',
  modelName: '',
  displayName: '',
  baseUrl: '',
  apiKey: '',
  dimension: undefined,
  supportsDimensionOverride: false,
  interfaceType: 'openai',
  isDefault: false,
  supportsVision: false,
  contextWindow: undefined,
  maxConcurrency: undefined,
  maxOutputTokens: undefined,
  thinkingControl: '',
  extraConfig: {},
  specCompat: '',
  spec: null,
  customHeaders: [],
  appSecret: '',
})

watch(() => [formData.value.baseUrl, formData.value.apiKey, formData.value.provider, props.visible], () => {
  discoveryRevision++
  discoveredModels.value = []
  selectedRemoteModels.value = []
  discoveryLoading.value = false
  discoveryError.value = ''
}, { flush: 'sync' })

const rules = computed(() => ({
  modelName: [
    { required: true, message: t('model.editor.validation.modelNameRequired') },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: t('model.editor.validation.modelNameEmpty') }
        }
        if (val.trim().length > 100) {
          return { result: false, message: t('model.editor.validation.modelNameMax') }
        }
        return { result: true }
      },
      trigger: 'blur'
    }
  ],
  baseUrl: [
    {
      required: true,
      message: t('model.editor.validation.baseUrlRequired'),
      trigger: 'blur'
    },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: t('model.editor.validation.baseUrlEmpty') }
        }
        // Basit URL biçimi doğrulaması
        try {
          new URL(val.trim())
          return { result: true }
        } catch {
          return { result: false, message: t('model.editor.validation.baseUrlInvalid') }
        }
      },
      trigger: 'blur'
    }
  ]
}))

// Entegrasyon tanılamasının tetikleyicileri: görünürlük / kaynak / sağlayıcı / model adı / Base URL / gelişmiş geçersiz kılmalar / model türü
watch(
  () => [
    props.visible, formData.value.source, formData.value.provider, formData.value.modelName,
    formData.value.baseUrl, formData.value.extraConfig?.api, formData.value.extraConfig?.remote_model_name,
    formData.value.thinkingControl, activeModelType.value,
 formData.value.specCompat, JSON.stringify(formData.value.spec),
  ],
  () => {
    if (!props.visible) return
    scheduleResolve()
  },
)

// Açılır pencerenin açıklama metnini al
const getModalDescription = () => {
  const key = `model.editor.description.${activeModelType.value}` as const
  return t(key) || t('model.editor.description.default')
}

// Model adı yer tutucusunu al
const getModelNamePlaceholder = () => {
  if (activeModelType.value === 'vllm') return t('model.editor.modelNamePlaceholder.remoteVllm')
  if (activeModelType.value === 'asr') {
    return t('model.editor.modelNamePlaceholder.remoteAsr')
  }
  return t('model.editor.modelNamePlaceholder.remote')
}

const getBaseUrlPlaceholder = () => {
  if (activeModelType.value === 'vllm') {
    return t('model.editor.baseUrlPlaceholderVllm')
  }
  if (activeModelType.value === 'asr') {
    return t('model.editor.baseUrlPlaceholderAsr')
  }
  return t('model.editor.baseUrlPlaceholder')
}

// Son açılıştaki modelData id: model değiştirme/yeni ekleme ile aynı yeni eklemenin art arda açılışını ayırt etmek için
const lastOpenedModelId = ref<string | null>(null)

const selectModelType = async (type: EditorModelType) => {
  if (isEdit.value || draftModelType.value === type) return
  draftModelType.value = type

  if (type !== 'embedding') {
    formData.value.dimension = undefined
    formData.value.supportsDimensionOverride = false
    dimensionChecked.value = false
    dimensionSuccess.value = false
    dimensionMessage.value = ''
  }
  if (type !== 'chat') {
    formData.value.supportsVision = false
  }
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''

  await loadProviders()
  const supported = providerOptions.value.some(p => p.value === formData.value.provider)
  if (!supported) {
    formData.value.provider = 'generic'
    formData.value.baseUrl = ''
    // Yalnızca sağlayıcıyla ilgili extra_config öğelerini kaldır; protokol / uzak model adı, kullanıcının bu entegrasyon için seçimidir,
    // sağlayıcıdan bağımsızdır ve handleProviderChange ile aynı kuralları izler.
    formData.value.extraConfig = keepVendorNeutralExtraConfig()
    formData.value.appSecret = ''
  } else {
    handleProviderChange(formData.value.provider || 'generic')
  }
}

// visible değişimini dinle ve formu başlat
watch(() => props.visible, (val) => {
  if (val) {
    // Model Provider listesini API'den yükle (mevcut satır düzenlenirken ek alan varsayılanlarını da tamamla).
    // Catalogs can be published by another administrator while this page is
    // open, so refresh this type without dropping other types' cached lists.
    loadProviders(true).then(() => {
      if (props.visible && !isEdit.value) applyExtraFieldDefaults()
    })
    advancedOpen.value = false

    // Her açılışta önceki doğrulama/denetim sonuçlarını temizle; başka bir model düzenlenirken
    // önceki "bağlantı başarılı" sonucunu doğrudan gösterme
    remoteChecked.value = false
    remoteAvailable.value = false
    remoteMessage.value = ''
    dimensionChecked.value = false
    dimensionSuccess.value = false
    dimensionMessage.value = ''

    const currentId = props.modelData?.id ?? null
    draftModelType.value = props.modelType

    hydratingForm.value = true
    try {
      if (props.modelData) {
        // Düzenleme: her zaman en güncel modelData ile üzerine yaz. apiKey alanı boş bırakılır —
        // edit mode the credential is owned by the <CredentialResource> card,
        // not by this form's apiKey field.
        const loadedExtra: Record<string, string> = {}
        for (const [key, value] of Object.entries(props.modelData.extraConfig || {})) {
          if (RESERVED_EXTRA_CONFIG_KEYS.has(key)) continue
          if (value != null && String(value) !== '') loadedExtra[key] = String(value)
        }
        const loadedSpec = props.modelData.spec || null
        formData.value = {
          ...props.modelData,
          source: 'remote',
          apiKey: '',
          appSecret: '',
          extraConfig: loadedExtra,
          thinkingControl: props.modelData.thinkingControl || '',
          spec: loadedSpec,
          specCompat: props.modelData.specCompat
            ?? (loadedSpec?.compat ? JSON.stringify(loadedSpec.compat, null, 2) : ''),
          customHeaders: Array.isArray(props.modelData.customHeaders)
            ? props.modelData.customHeaders.map(h => ({ key: h.key, value: h.value }))
            : [],
        }
        legacyThinkingControlLoaded.value = !!props.modelData.thinkingControl
      } else if (lastOpenedModelId.value !== null || !formData.value.id) {
        // Önceki işlem bir modeli düzenlemektiyse veya ilk kez yeni ekleniyorsa → boş duruma sıfırla
        resetForm()
      }
      // Aksi halde: art arda iki kez "yeni ekleme" açılışı (arada maske/ESC ile kapatılmış) → önceki girilenleri koru

      lastOpenedModelId.value = currentId

    } finally {
      nextTick(() => {
        hydratingForm.value = false
      })
    }
  }
})

watch(
  () => [
    activeModelType.value, props.modelData?.id, formData.value.source,
    formData.value.provider, formData.value.modelName, formData.value.baseUrl,
    formData.value.apiKey, formData.value.appSecret, formData.value.customHeaders,
    formData.value.dimension, formData.value.supportsDimensionOverride,
    formData.value.extraConfig, formData.value.thinkingControl,
 formData.value.specCompat, formData.value.spec,
  ],
  () => {
    if (!applyingDetectedDimension) invalidateConnectionTest(props.visible && !hydratingForm.value)
  },
  { deep: true, flush: 'sync' },
)

watch(() => props.visible, () => {
  invalidateConnectionTest(false)
  saveError.value = ''
}, { flush: 'sync' })

// Formu sıfırla
const resetForm = () => {
  legacyThinkingControlLoaded.value = false
  formData.value = {
    id: generateId(),
    name: '', // Alan korunur ancak kullanılmaz, kaydederken modelName kullanılır
    source: 'remote',
    provider: 'generic',
    modelName: '',
    displayName: '',
    baseUrl: '',
    apiKey: '',
    dimension: undefined, // Varsayılan olarak boş bırakılır; kullanıcının elle girmesine veya algılama düğmesiyle almasına izin verilir
    supportsDimensionOverride: false,
    interfaceType: undefined,
    isDefault: false,
    supportsVision: false,
    contextWindow: undefined,
    maxConcurrency: undefined,
    maxOutputTokens: undefined,
    thinkingControl: '',
    extraConfig: {},
    specCompat: '',
    spec: null,
    customHeaders: [],
    appSecret: '',
  }
  resolved.value = null
  resolveFailed.value = false
  resolveError.value = ''
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

// Sağlayıcı seçimi değişikliğini işle (varsayılan URL'yi otomatik doldur)
/**
 * Drop a model name the new vendor does not serve, along with whatever the
 * catalog filled in for it.
 *
 * The same id does exist at several vendors — deepseek-v4-pro is sold by
 * DeepSeek, Aliyun, Volcengine and the gateways — so switching between them
 * should keep the selection. Anything else is a name from the previous
 * vendor: left in place it is saved verbatim, resolves as an uncatalogued
 * model and fails at the first call.
 */
const resetModelSelectionForVendor = () => {
  const current = (formData.value.modelName || '').trim()
  if (!current || findCatalogEntry(current)) return

  formData.value.modelName = ''
  // Take back only the values applyCatalogEntry put there; a number the
  // operator typed is theirs and survives the switch.
  const filled = catalogFilled.value
  if (filled.contextWindow && formData.value.contextWindow === filled.contextWindow) {
    formData.value.contextWindow = undefined
  }
  if (filled.maxOutputTokens && formData.value.maxOutputTokens === filled.maxOutputTokens) {
    formData.value.maxOutputTokens = undefined
  }
  if (filled.dimension && formData.value.dimension === filled.dimension) {
    formData.value.dimension = undefined
  }
  if (filled.supportsVision && formData.value.supportsVision) {
    formData.value.supportsVision = false
  }
  catalogFilled.value = {}
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

const handleProviderChange = (value: string) => {
  const provider = providerOptions.value.find(opt => opt.value === value)
  if (provider?.defaultUrls) {
    // Geçerli model türüne karşılık gelen varsayılan URL'yi al
    const defaultUrl = provider.defaultUrls[activeModelType.value]
    if (defaultUrl) {
      formData.value.baseUrl = defaultUrl
    }
  }
  // Doğrulama durumunu sıfırla: önceki sağlayıcının bağlantısını açıklar ve yeni sağlayıcıyla ilgisizdir. Bunu defaultUrls
  // kontrolünün dışına koy; aksi halde varsayılan adresi olmayan bir sağlayıcıya geçildiğinde eski bir "bağlantı normal" sonucu kalır.
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  if (hydratingForm.value) return
  resetModelSelectionForVendor()
  // Sağlayıcı değiştirildiğinde: önceki sağlayıcıya ait alanları kaldırın (protokol / uzak model adı gibi gelişmiş geçersiz kılmaları koruyun), ardından yeni sağlayıcının varsayılan değerlerini uygulayın
  formData.value.extraConfig = keepVendorNeutralExtraConfig()
  formData.value.appSecret = ''
  applyExtraFieldDefaults()
}

// Kaynak değişikliklerini izle, doğrulama durumunu sıfırla (aşağıdaki `watch` ile birleştirildi)

// Benzersiz ID oluştur
const generateId = () => {
  return `model_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

// Özel HTTP Header düzenleme
const addCustomHeader = () => {
  if (!Array.isArray(formData.value.customHeaders)) {
    formData.value.customHeaders = []
  }
  formData.value.customHeaders.push({ key: '', value: '' })
}

const removeCustomHeader = (idx: number) => {
  if (!Array.isArray(formData.value.customHeaders)) return
  formData.value.customHeaders.splice(idx, 1)
}

/** extra_config exactly as it will be persisted: trimmed, empty values dropped. */
const buildExtraConfig = (): Record<string, string> => {
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(formData.value.extraConfig || {})) {
    const trimmed = (value ?? '').toString().trim()
    if (key && trimmed) out[key] = trimmed
  }
  const legacy = (formData.value.thinkingControl || '').trim()
  if (legacy && activeModelType.value === 'chat' && formData.value.source === 'remote') {
    out.thinking_control = legacy
  }
  return out
}

// Uzak API bağlantısını kontrol et (model türüne göre farklı arayüzler çağrılır)
const checkRemoteAPI = async () => {
  if (checking.value || saving.value) return
  if (!formData.value.modelName || (!formData.value.baseUrl)) {
    MessagePlugin.warning(t('model.editor.fillModelAndUrl'))
    return
  }

  const revision = ++connectionRevision
  checking.value = true
  remoteStale.value = false
  remoteChecked.value = false
  remoteMessage.value = ''

  try {
    let result: any

    // Formdaki Key-Value dizi biçimindeki özel Header'ları, arka ucun beklediği map biçimine dönüştür.
    // `ModelSettings.vue` kaydetme işlemiyle tutarlı olarak boş satırları otomatik kaldırır; bağlantı testi ile gerçek kaydetme sonrası
    // üretim çağrısının tamamen aynı Header kümesini kullanmasını sağlar.
    const customHeaders: Record<string, string> = {}
    if (Array.isArray(formData.value.customHeaders)) {
      for (const item of formData.value.customHeaders) {
        const key = (item?.key ?? '').trim()
        const value = (item?.value ?? '').trim()
        if (key && value) customHeaders[key] = value
      }
    }
    // Boş nesnelerin URL query / günlüklerde görünmesini önlemek için alanları yalnızca boş olmadıklarında ekle
    const headerPayload = Object.keys(customHeaders).length > 0
      ? { customHeaders }
      : {}

    // Model türüne göre farklı doğrulama arayüzleri çağır
    // Düzenleme modunda `apiKey`, `<CredentialResource>` tarafından bağımsız olarak yönetilir; `formData` içinde değildir.
    // `modelId` değerini arka uca ilet; böylece `apiKey` boş olduğunda depolanan şifresi çözülmüş değeri otomatik olarak yedek olarak kullanabilir,
    // "bağlantı testi apiKey olmadan doğrudan başarısız oldu" durumunu önler.
    const idPayload = isEdit.value && props.modelData?.id
      ? { modelId: props.modelData.id as string }
      : {}

    // `extra_config`, gerçek kaydetme işlemindekiyle tamamen aynıdır (sağlayıcı alanları + gelişmiş geçersiz kılmalar + eski `thinking_control`),
    // bağlantı testinin üretim çağrısıyla aynı dizin çözümleme yolunu kullanmasını sağlar.
    const extraConfig = buildExtraConfig()
    const extraPayload = {
      spec: buildSpec(),
      ...(Object.keys(extraConfig).length > 0 ? { extraConfig } : {}),
      ...(formData.value.appSecret?.trim() ? { appSecret: formData.value.appSecret.trim() } : {}),
    }

    switch (activeModelType.value) {
      case 'chat':
        // Sohbet modeli (`KnowledgeQA`)
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      case 'embedding':
        // Embedding modeli
        result = await testEmbeddingModel({
          source: 'remote',
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          dimension: formData.value.dimension,
          supportsDimensionOverride: formData.value.supportsDimensionOverride ?? false,
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        // Test başarılıysa ve boyut dönerse otomatik olarak doldur
        if (revision !== connectionRevision) return
        if (result.available && result.dimension) applyDetectedDimension(result.dimension)
        break

      case 'rerank':
        result = await checkRerankModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      case 'vllm':
        // VLLM modeli (çok modlu)
        // VLLM, temel bağlantı testi için `checkRemoteModel` kullanır
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      case 'asr':
        // ASR modeli (ses tanıma) — özel ASR test arayüzünü kullanır (`/v1/audio/transcriptions`)
        result = await checkASRModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      default:
        MessagePlugin.error(t('model.editor.unsupportedModelType'))
        return
    }

    if (revision !== connectionRevision) return
    remoteChecked.value = true
    remoteAvailable.value = result.available || false
    remoteMessage.value = result.available
      ? t('model.editor.connectionSuccess')
      : result.message || t('model.editor.connectionFailed')
  } catch (error: any) {
    if (revision !== connectionRevision) return
    remoteChecked.value = true
    remoteAvailable.value = false
    remoteMessage.value = error?.message || t('model.editor.connectionConfigError')
  } finally {
    if (revision === connectionRevision) checking.value = false
  }
}

// Kaydetmeyi onayla
const handleConfirm = async () => {
  if (saving.value) return
  saving.value = true
  if (checking.value) invalidateConnectionTest()
  saveError.value = ''
  try {
    // Zorunlu alanları manuel olarak doğrula
    if (!formData.value.modelName || !formData.value.modelName.trim()) {
      MessagePlugin.warning(t('model.editor.validation.modelNameRequired'))
      return
    }

    if ([formData.value.modelName, ...selectedRemoteModels.value].some(name => name.trim().length > 100)) {
      MessagePlugin.warning(t('model.editor.validation.modelNameMax'))
      return
    }

    // Uzak model için `baseUrl` doldurulmalıdır
    if (formData.value.source === 'remote') {
      if (!formData.value.baseUrl || !formData.value.baseUrl.trim()) {
        MessagePlugin.warning(t('model.editor.remoteBaseUrlRequired'))
        return
      }

      // Base URL biçimini doğrula
      try {
        new URL(formData.value.baseUrl.trim())
      } catch {
        MessagePlugin.warning(t('model.editor.validation.baseUrlInvalid'))
        return
      }
    }

    if (formData.value.source === 'remote') {
      // Üreticinin bildirdiği zorunlu ek alanlar
      for (const field of plainExtraFields.value) {
        if (field.required && !(formData.value.extraConfig?.[field.key] || '').trim()) {
          MessagePlugin.warning(t('model.editor.validation.extraFieldRequired', { name: extraFieldDisplayLabel(field) }))
          return
        }
      }
      if (!isEdit.value && secretExtraField.value?.required && !(formData.value.appSecret || '').trim()) {
        MessagePlugin.warning(t('model.editor.validation.extraFieldRequired', { name: extraFieldDisplayLabel(secretExtraField.value) }))
        return
      }
      if (specCompatError.value) {
        advancedOpen.value = true
        MessagePlugin.warning(`${t('model.editor.advanced.compat.invalid')}: ${specCompatError.value}`)
        return
      }
    }

    // Form doğrulamasını çalıştır
    const validation = await formRef.value?.validate()
    if (validation !== undefined && validation !== true) return

    // Credential removal in edit mode is handled inline by the
    // CredentialResource card (it confirms + DELETEs to /credentials), so
    // the main save flow no longer needs to confirm or handle clear flags.

    // Yeni ekleme ise ve id yoksa bir tane oluştur
    if (!formData.value.id) {
      formData.value.id = generateId()
    }

    const names = !isEdit.value && selectedRemoteModels.value.length
      ? [...selectedRemoteModels.value] : [formData.value.modelName]
    for (const name of names) {
      try {
        await props.saveModel({
          ...formData.value,
          modelName: name,
          displayName: names.length > 1 ? name : formData.value.displayName,
          extraConfig: {
            ...buildExtraConfig(),
            ...(names.length > 1 ? { remote_model_name: '' } : {}),
          },
          ...(isEdit.value ? {} : { modelType: activeModelType.value }),
        })
        if (selectedRemoteModels.value.length) {
          selectedRemoteModels.value = selectedRemoteModels.value.filter(id => id !== name)
          formData.value.modelName = selectedRemoteModels.value[0] || name
        }
      } catch (error: any) {
        throw new Error((names.length > 1 ? name + ': ' : '') + (error?.message || t('modelSettings.toasts.saveFailed')))
      }
    }
    emit('update:visible', false)
    invalidateConnectionTest(false)
    // Kaydetme başarılı olduktan sonra taslağı sıfırla; sonraki yeni model açılışında boş olsun
    resetForm()
    lastOpenedModelId.value = null
    // Buradaki başarı bildirimini kaldır; üst bileşen topluca yönetsin
  } catch (error: any) {
    saveError.value = error?.message || t('modelSettings.toasts.saveFailed')
  } finally {
    saving.value = false
  }
}

// Bileşen kaldırıldığında zamanlayıcıyı temizle
onUnmounted(() => {
  window.removeEventListener('keydown', handleSaveEscape, true)
  invalidateConnectionTest(false)
  if (resolveTimer) {
    clearTimeout(resolveTimer)
    resolveTimer = null
  }
  resolveRevision++
})

// Model adı değişikliklerini izle, boyut algılama durumunu temizle
watch(() => formData.value.modelName, () => {
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
})

// İptal (alttaki "İptal" düğmesine tıklanınca tetiklenir; maskeye tıklama/ESC tetiklemez, böylece taslak korunur)
const handleCancel = () => {
  if (saving.value) return
  resetForm()
  lastOpenedModelId.value = null
  dialogVisible.value = false
}
</script>

<style lang="less" scoped>
.provider-doc-link {
  margin-top: 6px;
}

.provider-doc-link a,
.compat-doc-link {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--td-text-color-link);
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}

// Yerel t-form-item kapsayıcısını boş bırak (bu bileşen özel .form-item + elle yazılmış label kullanır)
:deep(.t-form) {
  .t-form-item {
    display: none;
  }
}

// Form öğesi stilleri
.form-item {
  // No bottom margin — vertical rhythm is owned by the parent
  // .setting-drawer__section's `gap`. That keeps the spacing inside a section
  // tight and the gap between sections visually distinct.
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;

  // TDesign-style required marker: leading asterisk before the label text,
  // matching the rest of the app's <t-form-item required ...> appearance.
  &.required::before {
    content: '*';
    color: var(--td-error-color);
    margin-right: 4px;
    font-weight: 500;
    line-height: 1;
  }
}

.model-type-options,
.source-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

// "Model türü" ve "Model kaynağı" ikisi de tek seçimlidir; aynı tür düğmeyi kullan. Model kaynağı başlangıçta gri arka planlı bir şeritti
// segmented: #e7e7e7 şeridi açık renkli formda tamamen koyu gri bir blok oluşturuyor, beyaz kapsüller de öne çıkmıyordu,
// ayrıca hemen yanındaki model türü farklı bir görünüme sahipti. Şerit kaldırılınca iki grup doğal olarak aynı aileye dönüşür.
.model-type-option,
.source-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: inherit;
  padding: 6px 12px;
  min-height: 32px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.4;
  cursor: pointer;
  transition: border-color var(--app-motion-fast) ease, color var(--app-motion-fast) ease, background var(--app-motion-fast) ease;

  &__icon {
    font-size: var(--app-text-lg);
    flex-shrink: 0;
  }

  &__label {
    white-space: nowrap;
  }

  &:hover:not(.is-active) {
    border-color: var(--td-brand-color-3);
    color: var(--td-text-color-primary);
  }

  // Seçili durum alttaki "Model kaynağı" segmentiyle tutarlı: beyaz arka plan + tema rengi kenarlık + tema rengi metin.
  // Önceden %10 opaklıklı bir tema rengi katmanı da vardı; beş düğme içindeki bu bölüm ekrandaki en yoğun renk bloğuydu ve
  // açılır listeden henüz kaldırılan tam satır yeşil arka planla aynı sorunu taşıyordu.
  &.is-active {
    border-color: var(--td-brand-color);
    background: var(--td-bg-color-container);
    color: var(--td-brand-color);
    font-weight: 500;
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}



.source-option {
  // Görünüm yukarıdaki ortak kuraldan gelir; burada yalnızca devre dışı durumu eklenir.
  &.is-disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }
}

// Giriş alanı stilleri: yazı boyutunu yalnızca en dıştaki .t-input üzerinde ayarla; iç wrap/inner üzerinde kenarlığı tekrar eklemekten kaçın
// ve border-radius, görsel olarak "iç içe yuvarlatılmış köşe kapsayıcıları" yanılsaması oluşturur
:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: var(--app-text-md);
}

// Üretici seçici stilleri — t-select popup body altında oluşturulduğu için scoped olmayan bloğa taşındı
// .provider-option stilleri dosyanın sonunda yer alır

// Onay kutusu
:deep(.t-checkbox) {
  font-size: var(--app-text-md);

  .t-checkbox__label {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
  }
}

// API Key girişi: başta lock simgesi + sonda tıklanabilir "göster/gizle" küçük göz simgesi.
// TDesign varsayılan olarak prefix-icon simgesini gri gösterir, burada değiştirilmedi; suffix üzerindeki göz simgesi
// placeholder rengini kullanır, hover durumunda ana metin rengine geçer; böylece dikkat çekmez.
.api-key-input {
  :deep(.t-input__prefix) {
    color: var(--td-text-color-placeholder);
  }

  :deep(.t-input__suffix) {
    color: var(--td-text-color-placeholder);
  }

}

// API test alanı — zayıf kartlaştırma: "işlem + geri bildirim" bölümünü açık zemin + dashed kenarlıkla tek bir blok haline getir,
// kullanıcının bunu sıradan başka bir alan yerine bağımsız bir "eylem birimi" olarak görmesini sağla.
// (Geçmiş stil korundu: yalnızca bir dal test bloğunu hâlâ inline olarak oluşturduğunda kullanılır; mevcut RemoteAPI
// testi SettingDrawer footer-left yuvasına taşındı, ana akış artık bu bölümden geçmiyor.)
.api-test-section {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: var(--td-bg-color-container-hover);
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  .test-message {
    font-size: var(--app-text-md);
    line-height: 1.5;
    flex: 1;

    &.success {
      color: var(--td-brand-color-active);
    }

    &.error {
      color: var(--td-error-color);
    }
  }

  :deep(.t-button) {
    min-width: 88px;
    height: 32px;
    font-size: var(--app-text-md);
    border-radius: var(--app-radius-sm);
    flex-shrink: 0;
  }

  .status-icon {
    font-size: var(--app-text-xl);
    flex-shrink: 0;

    &.available {
      color: var(--td-brand-color);
    }

    &.unavailable {
      color: var(--td-error-color);
    }
  }
}

.connection-status {
  font-size: var(--app-text-sm);
  &.success { color: var(--td-brand-color-active); }
  &.error { color: var(--td-error-color); }
}

.connection-hint {
  margin: 0 0 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.5;
}

.connection-result {
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid var(--td-error-color-3);
  border-radius: var(--td-radius-default);
  background: var(--td-error-color-1);
  color: var(--td-error-color);
  font-size: var(--app-text-sm);
  text-align: left;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  &__details {
    max-height: min(160px, 20vh);
    overflow: auto;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    line-height: 1.5;
    user-select: text;
  }
}

// Status icon variant used inside the footer button.
.status-icon {
  font-size: var(--app-text-xl);
  flex-shrink: 0;

  &.available {
    color: var(--td-brand-color);
  }

  &.unavailable {
    color: var(--td-error-color);
  }
}

// Boyut kontrol stilleri
.dimension-control {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.t-input) {
    flex: 1;
  }
}

.dimension-check-btn {
  flex-shrink: 0;
}

.dimension-hint {
  margin: 8px 0 0 0;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-error-color);

  &.success {
    color: var(--td-brand-color);
  }
}

// Özel HTTP Header alanı
.custom-headers-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.custom-headers-desc {
  margin: 0 0 10px 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.custom-headers-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.custom-header-row {
  display: flex;
  align-items: center;
  gap: 8px;

  .custom-header-key {
    flex: 0 0 38%;
  }

  .custom-header-value {
    flex: 1;
  }

  // Ghost icon button — matches the model-card "more" affordance: invisible
  // until hover/focus, then a subtle background pops in. Avoids painting a
  // permanent red splotch next to every header row.
  .custom-header-remove {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    padding: 0;
    color: var(--td-text-color-placeholder);
    border-radius: var(--app-radius-sm);
    transition: all 0.18s ease;

    &:hover {
      background: var(--td-error-color-light);
      color: var(--td-error-color);
    }
  }
}

.form-desc {
  margin: 4px 0 0 0;
  font-size: var(--app-text-xs);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);

  // Inline with switches/checkboxes — drops the top margin so the label and
  // helper text sit on the same baseline.
  &--inline {
    margin: 0;
  }

  &--recommend {
    color: var(--td-brand-color);
  }

  &--warn {
    color: var(--td-warning-color);
  }

  &--error {
    color: var(--td-error-color);
  }
}

// Seçili sağlayıcı: simge + ad, t-select'in valueDisplay içinde yer alır
.provider-value {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;

  &__name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.provider-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  border-radius: 4px;
  object-fit: contain;

  &--placeholder {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: var(--app-text-xs);
    font-weight: 600;
    color: var(--td-text-color-secondary);
    background: var(--td-bg-color-secondarycontainer);
  }
}

// Dizin içi / çıkarım / görsel vb. soluk rozetler (kartlardaki chip ile uyumlu)
.catalog-badge {
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  height: 18px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  line-height: 18px;
  white-space: nowrap;

  & + & {
    margin-left: 4px;
  }

  &--accent {
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);
  }
}

// Bağlantı tanılama paneli: salt okunur, açık zeminli; düzenlenebilir alanlarla karışmasını önler
.resolved-panel {
  padding: 10px 12px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container-hover);

  &__header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  &__icon {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
  }

  &__title {
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-text-color-secondary);
  }

  &__loading {
    font-size: var(--app-text-md);
    color: var(--td-text-color-placeholder);
    animation: spin 1s linear infinite;
  }

  &__grid {
    display: grid;
    grid-template-columns: max-content minmax(0, 1fr);
    column-gap: 12px;
    row-gap: 4px;
    margin: 0;
    font-size: var(--app-text-sm);

    dt {
      color: var(--td-text-color-placeholder);
      white-space: nowrap;
      line-height: 18px;
    }

    dd {
      margin: 0;
      min-width: 0;
      color: var(--td-text-color-primary);
      line-height: 18px;
      overflow-wrap: anywhere;
    }

    code {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: var(--app-text-sm);
    }
  }
}

// Gelişmiş daraltma: bilgi tabanı bölüm ayarlarındaki advanced-toggle ile aynı görseli koru
.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  margin: 0;
  background: transparent;
  border: none;
  cursor: pointer;
  font-family: inherit;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  user-select: none;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
    border-radius: 4px;
  }

  .toggle-arrow {
    font-size: var(--app-text-base);
    transition: transform 0.15s ease;

    &.open {
      transform: rotate(90deg);
    }
  }
}

.compat-textarea {
  :deep(textarea) {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: var(--app-text-sm);
  }
}

.vision-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}

.catalog-model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;

  &__name {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__id {
    font-size: var(--app-text-xs);
    color: var(--td-text-color-placeholder);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__badges {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: auto;
    flex-shrink: 0;

    .catalog-badge {
      display: inline-flex;
      align-items: center;
      padding: 0 6px;
      height: 18px;
      border-radius: 4px;
      background: var(--td-bg-color-secondarycontainer);
      color: var(--td-text-color-secondary);
      font-size: var(--app-text-xs);
      line-height: 18px;
      white-space: nowrap;

      &--accent {
        background: var(--td-brand-color-light);
        color: var(--td-brand-color);
      }
    }
  }
}

.provider-select-popup {
  // Kapsayıcıya biraz nefes alanı bırak: seçeneklerin popup yuvarlatılmış köşelerine yapışmasını önle
  padding: 4px;

  // TDesign, select açılır katmanı için max-height: 300px değerini sabitliyor; buradaki seçenekler ise iki satırlı
  // (yaklaşık 56px), bu nedenle tam olarak yalnızca beş satır görünür — aşağıda yirmiden fazla sağlayıcı daha var, ancak macOS'un kayan
  // kaydırma çubuğu kaydırılmadıkça görünmez; bu da sanki "yalnızca bunlar varmış" gibi görünür. Yüksekliği bir ekranda sekiz dokuz satır görülecek şekilde artırınca kaydırma
  // çubuğu da kalıcı olur ve listenin devamı olduğu anlaşılır. &.wk-popover yalnızca TDesign'in
  // aynı derecede iki seviyeli seçicisini geçersiz kılmak içindir.
  &.wk-popover .t-popup__content {
    max-height: min(480px, 60vh);
    // Scrolling itself is restored for every skinned select in
    // assets/theme/tdesign-overrides.less; this only makes the bar visible,
    // because macOS overlay scrollbars stay hidden until something moves and
    // a capped list then looks complete.
    scrollbar-color: var(--td-component-border) transparent;
    scrollbar-width: thin;
  }

  // TDesign varsayılan olarak t-select-option üzerinde bir overflow tooltip ekler (sağ tarafta
  // label'ın tamamını gösterir). Seçenek düzenimiz iki satırlıdır: "ana ad + ikincil açıklama"; hiçbir zaman
  // kısaltma tetiklenmez, tooltip ise görsel gürültüye dönüşür → popup'ın yerleşik ipucunu doğrudan gizle.
  + .t-popup .t-tooltip,
  ~ .t-popup .t-tooltip {
    display: none !important;
  }

  .t-select-option {
    height: auto !important;
    padding: 8px 10px;
    border-radius: var(--app-radius-sm);
    margin: 2px 0;
    outline: none;
    transition: background-color var(--app-motion-fast) ease;

    &:focus,
    &:focus-visible {
      outline: none;
    }


  }

  // Vurgu durumu burada tanımlanmıyor: assets/theme/tdesign-overrides.less zaten TDesign'in
  // varsayılan, satırın tamamını kaplayan --td-brand-color-light değerini nötr bir zemin + küçük bir tema rengi
  // onay işaretiyle değiştirdi; bu, site genelinde tutarlıdır. Burada yeniden yazmak yalnızca bu açılır menünün "model türü / model kaynağı"
  // iki satırlı filtreyle uyuşmamasına neden olur. Seçili satırın ana adı, iki satırlı düzende bağlantı noktası olarak tema rengini yine biraz alır.
  .t-select-option.t-is-selected .provider-name {
    color: var(--td-brand-color);
  }

  .provider-option {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-width: 0;

    .provider-icon {
      width: 18px;
      height: 18px;
      flex-shrink: 0;
      border-radius: 4px;
      object-fit: contain;

      &--placeholder {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        font-size: var(--app-text-xs);
        font-weight: 600;
        color: var(--td-text-color-secondary);
        background: var(--td-bg-color-secondarycontainer);
      }
    }

    &__text {
      display: flex;
      flex-direction: column;
      gap: 2px;
      // Daralmaya izin ver; böylece açıklama açılır katman genişliği içinde kısaltılabilir (açılır katman genişliği matchTriggerWidth tarafından sabitlenir).
      min-width: 0;
      flex: 1;
    }

    .provider-name {
      font-size: var(--app-text-md);
      font-weight: 500;
      color: var(--td-text-color-primary);
      line-height: 20px;
    }

    .provider-desc {
      font-size: var(--app-text-sm);
      color: var(--td-text-color-placeholder);
      line-height: 18px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }
}
</style>
