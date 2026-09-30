<template>
  <div class="parser-engine-settings">
    <div class="section-header">
      <h2>{{ $t('settings.parser.title') }}</h2>
      <p class="section-description">
        {{ $t('settings.parser.description') }}
      </p>
    </div>

    <div v-if="loading" class="loading-state">
      <t-loading size="small" />
      <span>{{ $t('settings.parser.loading') }}</span>
    </div>

    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadAll">{{ $t('settings.parser.retry') }}</t-button>
        </template>
      </t-alert>
    </div>

    <template v-else>
      <div v-if="engines.length === 0 && !hasBuiltinEngine" class="empty-state">
        <p class="empty-text">{{ $t('settings.parser.noEngineDetected') }}</p>
      </div>

      <!-- Diğer settings listeleriyle aynı biçim: solda monogram rozeti + başlık + durum rozeti + iki satırlık açıklama.
           Kartın tamamı tıklanabilir; yapılandırma çekmecesini açar. Mevcut çekmeceye karşılık gelen kart, marka renkli bir kenarlık alır. -->
      <div v-else class="engine-cards">
        <!-- Arka uç builtin motor öğesini döndürmediğinde bile DocReader durum kartını göster -->
        <button
          v-if="!hasBuiltinEngine"
          type="button"
          class="engine-card engine-card--builtin"
          :class="{ 'engine-card--active': drawerVisible && currentEngine?.Name === 'builtin' }"
          @click="openDrawer({ Name: 'builtin' } as any)"
        >
          <div class="engine-card__badge">{{ engineInitial('builtin') }}</div>
          <div class="engine-card__body">
            <div class="engine-card__header">
              <h3 class="engine-card__title">{{ getEngineDisplayName('builtin') }}</h3>
              <span
                class="engine-card__status"
                :class="connected ? 'engine-card__status--on' : 'engine-card__status--err'"
              >
                <span class="engine-card__status-dot" />
                {{ connected ? $t('settings.parser.connected') : $t('settings.parser.disconnected') }}
              </span>
            </div>
            <p class="engine-card__desc">{{ $t('settings.parser.builtinDesc') }}</p>
          </div>
        </button>

        <button
          v-for="engine in sortedEngines"
          :key="engine.Name"
          type="button"
          class="engine-card"
          :class="[
            `engine-card--${engine.Name}`,
            { 'engine-card--active': drawerVisible && currentEngine?.Name === engine.Name }
          ]"
          @click="openDrawer(engine)"
        >
          <div class="engine-card__badge">{{ engineInitial(engine.Name) }}</div>
          <div class="engine-card__body">
            <div class="engine-card__header">
              <h3 class="engine-card__title">{{ getEngineDisplayName(engine.Name) }}</h3>
              <span v-if="engine.Available" class="engine-card__status engine-card__status--on">
                <span class="engine-card__status-dot" />
                {{ $t('settings.parser.available') }}
              </span>
              <t-tooltip
                v-else-if="engine.UnavailableReason"
                :content="getUnavailableReason(engine.UnavailableReason)"
                placement="top"
              >
                <span class="engine-card__status engine-card__status--err engine-card__status--help">
                  <span class="engine-card__status-dot" />
                  {{ $t('settings.parser.unavailable') }}
                </span>
              </t-tooltip>
              <span v-else class="engine-card__status engine-card__status--err">
                <span class="engine-card__status-dot" />
                {{ $t('settings.parser.unavailable') }}
              </span>
            </div>
            <p class="engine-card__desc">{{ getEngineDisplayDesc(engine.Name, engine.Description) }}</p>
          </div>
        </button>
      </div>

    </template>

    <!-- Yapılandırma çekmecesi — SettingDrawer ile sar, ModelEditorDialog ile aynı görsel/etkileşimi koru -->
    <SettingDrawer
      v-model:visible="drawerVisible"
      :title="drawerTitle"
      :class="currentEngine ? `parser-engine-drawer parser-engine-drawer--${currentEngine.Name}` : 'parser-engine-drawer'"
      :hide-footer="!authStore.hasRole('admin') && !needsTestButton"
      :confirm-loading="saving"
      @confirm="onSave"
      @cancel="drawerVisible = false"
    >
      <!--
        Header icon — liste kartıyla aynı monogram rozeti: ilk harf + motor başına renklendirme,
        .parser-engine-drawer--{name} .setting-drawer__header-icon aracılığıyla
        scoped olmayan blokta arka planı ve metin rengini geçersiz kıl. Parser motorunun gerçek bir logosu yok, bu
        rada yalnızca harfler render edilir; depolama motoru tarafında logo resmi/mask kullanılır, desen tutarlı.
      -->
      <template v-if="currentEngine" #headerIcon>
        <span class="header-icon__text">{{ engineInitial(currentEngine.Name) }}</span>
      </template>
      <!--
        Subtitle slot: motor açıklaması + satır içi doküman bağlantısı. "Referanslar"ı bir
        bağımsız section'dan başlık alt başlığına geri aldık — tek bir dış bağlantı koca bir section kaplamaya değmez.
      -->
      <template v-if="currentEngine" #subtitle>
        <span>{{ getEngineDisplayDesc(currentEngine.Name, currentEngine.Description) }}</span>
        <a
          v-if="engineDocLink(currentEngine.Name)"
          :href="engineDocLink(currentEngine.Name)"
          target="_blank"
          rel="noopener noreferrer"
          class="doc-link doc-link--inline"
        >
          {{ engineDocLabel(currentEngine.Name) }}
          <t-icon name="link" class="link-icon" />
        </a>
      </template>
      <!--
        Footer-left slot: bağlantıyı test et düğmesi + durum metni — ana işlem çubuğu alt kenar boyunca hizalanır,
        ModelEditorDialog uzak model çekmecesiyle tutarlıdır. Yalnızca motorun doğrulanabilir
        yapılandırması/durumu olduğunda bağlanır.
      -->
      <template v-if="needsTestButton" #footer-left>
        <t-button variant="outline" :loading="checking" @click="onCheck">
          <template #icon>
            <t-icon v-if="!checking && saveSuccess && checkMessage" name="check-circle-filled"
              class="status-icon available" />
            <t-icon v-else-if="!checking && checkMessage && !saveSuccess" name="close-circle-filled"
              class="status-icon unavailable" />
          </template>
          {{ checking ? $t('settings.parser.checking', $t('settings.parser.testConnection')) : $t('settings.parser.testConnection') }}
        </t-button>
        <span v-if="checkMessage" :class="['footer-test-message', saveSuccess ? 'success' : 'error']" :title="checkMessage">
          {{ checkMessage }}
        </span>
      </template>

      <div v-if="currentEngine">
        <!--
          Section 1 — desteklenen dosya türleri. İçeriğin başına, motorun "neler yapabildiğine" dair
          ilk bakışta anlaşılır bir genel bakış olarak yerleştirildi; durum/yapılandırmadan ayrıdır.
        -->
        <section
          v-if="currentEngine.FileTypes && currentEngine.FileTypes.length"
          class="setting-drawer__section"
        >
          <h4 class="setting-drawer__section-title">{{ $t('settings.parser.supportedFileTypes') }}</h4>
          <div class="file-types">
            <span v-for="ft in currentEngine.FileTypes" :key="ft" class="file-type-chip">
              {{ ft }}
            </span>
          </div>
        </section>

        <!--
          Section 2 — durum bilgisi (DocReader bağlantısı)
          Yalnızca içerik varsa render edilir; boş section ve boş alt ayırıcı çizgiler önlenir.
        -->
        <section
          v-if="currentEngine.Name === 'builtin'"
          class="setting-drawer__section"
        >
          <h4 class="setting-drawer__section-title">{{ $t('settings.parser.statusSection') }}</h4>

          <!-- builtin: DocReader bağlantı bilgileri -->
          <div v-if="currentEngine.Name === 'builtin'" class="docreader-block">
            <div class="status-line">
              <t-tag v-if="connected" theme="success" variant="light" size="small">
                {{ $t('settings.parser.connected') }}
              </t-tag>
              <t-tag v-else theme="danger" variant="light" size="small">
                {{ $t('settings.parser.disconnected') }}
              </t-tag>
              <t-tag theme="default" variant="light" size="small">
                {{ docreaderTransport === 'http' ? 'HTTP' : 'gRPC' }}
              </t-tag>
              <span v-if="docreaderAddrEnv" class="env-hint">
                {{ $t('settings.parser.currentAddr') }}: {{ docreaderAddrEnv }}
              </span>
            </div>
            <p class="form-desc">{{ $t('settings.parser.envVarHint') }}</p>
          </div>

        </section>

        <!-- Bölüm 3 — mineru özel kurulum yapılandırması -->
        <section v-if="currentEngine.Name === 'mineru'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ $t('settings.parser.configSection') }}</h4>

          <div class="form-item">
            <label class="form-label">{{ t('settings.parser.selfHostedEndpoint') }}</label>
            <t-input
              v-model="config.mineru_endpoint"
              :placeholder="$t('settings.parser.mineruEndpointPlaceholder')"
              clearable
            />
            <p class="form-desc">{{ $t('settings.parser.mineruEndpointHint') }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">API Key</label>
            <t-input
              v-model="config.mineru_server_api_key"
              type="password"
              :placeholder="$t('settings.parser.mineruServerApiKeyPlaceholder')"
              clearable
            >
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <p class="form-desc">{{ $t('settings.parser.mineruServerApiKeyHint') }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ $t('settings.parser.mineruTierLabel') }}</label>
            <t-select v-model="config.mineru_tier" :placeholder="$t('settings.parser.mineruTierDefault')" clearable>
              <t-option value="flash" :label="$t('settings.parser.mineruTierFlash')" />
              <t-option value="basic" :label="$t('settings.parser.mineruTierBasic')" />
              <t-option value="standard" :label="$t('settings.parser.mineruTierStandard')" />
              <t-option value="advanced" :label="$t('settings.parser.mineruTierAdvanced')" />
            </t-select>
            <p class="form-desc">{{ $t('settings.parser.mineruTierHint') }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ $t('settings.parser.parseMethodLabel') }}</label>
            <t-select v-model="config.mineru_parse_method">
              <t-option value="auto" :label="$t('settings.parser.parseMethodAuto')" />
              <t-option value="ocr" :label="$t('settings.parser.parseMethodOCR')" />
              <t-option value="txt" :label="$t('settings.parser.parseMethodText')" />
            </t-select>
            <p class="form-desc">{{ $t('settings.parser.parseMethodHint') }}</p>
          </div>
        </section>

        <section v-if="currentEngine.Name === 'mineru'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ $t('settings.parser.mineruLegacySection') }}</h4>
          <p class="form-desc">{{ $t('settings.parser.mineruLegacySectionHint') }}</p>

          <div class="form-item">
            <label class="form-label">Backend</label>
            <t-select v-model="config.mineru_model" :placeholder="$t('settings.parser.defaultPipeline')" clearable>
              <t-option value="pipeline" label="pipeline" />
              <t-option value="vlm-auto-engine" label="vlm-auto-engine" />
              <t-option value="vlm-http-client" label="vlm-http-client" />
              <t-option value="hybrid-auto-engine" label="hybrid-auto-engine" />
              <t-option value="hybrid-http-client" label="hybrid-http-client" />
            </t-select>
          </div>
          <div class="form-item">
            <label class="form-label">vLLM {{ $t('settings.parser.serverUrl') }}</label>
            <t-input
              v-model="config.mineru_vlm_server_url"
              :placeholder="$t('settings.parser.vlmServerUrlPlaceholder')"
              clearable
            />
            <p class="form-desc">{{ $t('settings.parser.vlmServerUrlHint') }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ $t('settings.parser.featuresLabel') }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.mineru_enable_formula">{{ $t('settings.parser.formulaRecognition') }}</t-checkbox>
              <t-checkbox v-model="config.mineru_enable_table">{{ $t('settings.parser.tableRecognition') }}</t-checkbox>
            </div>
          </div>
          <div class="form-item">
            <label class="form-label">{{ t('settings.parser.language') }}</label>
            <t-input
              v-model="config.mineru_language"
              :placeholder="$t('settings.parser.languagePlaceholder')"
              clearable
            />
          </div>
        </section>

        <!-- Bölüm 3 — mineru_cloud bulut API yapılandırması -->
        <section v-if="currentEngine.Name === 'mineru_cloud'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ $t('settings.parser.configSection') }}</h4>

          <div class="form-item">
            <label class="form-label required">API Key</label>
            <t-input
              v-model="config.mineru_api_key"
              type="password"
              :placeholder="$t('settings.parser.mineruCloudApiKeyPlaceholder')"
              clearable
            >
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </div>
          <div class="form-item">
            <label class="form-label">Model Version</label>
            <t-select v-model="config.mineru_cloud_model" :placeholder="$t('settings.parser.defaultPipeline')" clearable>
              <t-option value="pipeline" label="pipeline" />
              <t-option value="vlm" :label="$t('settings.parser.vlmLabel')" />
              <t-option value="MinerU-HTML" :label="$t('settings.parser.mineruHtmlLabel')" />
            </t-select>
          </div>
          <div class="form-item">
            <label class="form-label">{{ $t('settings.parser.featuresLabel') }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.mineru_cloud_enable_formula">{{ $t('settings.parser.formulaRecognition') }}</t-checkbox>
              <t-checkbox v-model="config.mineru_cloud_enable_table">{{ $t('settings.parser.tableRecognition') }}</t-checkbox>
              <t-checkbox v-model="config.mineru_cloud_enable_ocr">OCR</t-checkbox>
            </div>
          </div>
          <div class="form-item">
            <label class="form-label">{{ t('settings.parser.language') }}</label>
            <t-input
              v-model="config.mineru_cloud_language"
              :placeholder="$t('settings.parser.languagePlaceholder')"
              clearable
            />
          </div>
        </section>

        <!-- Bölüm 3 — paddleocr_vl özel kurulum yapılandırması -->
        <section v-if="currentEngine.Name === 'paddleocr_vl'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ $t('settings.parser.configSection') }}</h4>

          <div class="form-item">
            <label class="form-label required">{{ t('settings.parser.selfHostedEndpoint') }}</label>
            <t-input
              v-model="config.paddleocr_vl_endpoint"
              :placeholder="$t('settings.parser.paddleocrVlEndpointPlaceholder')"
              clearable
            />
            <p class="form-desc">{{ $t('settings.parser.paddleocrVlEndpointHint') }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ $t('settings.parser.featuresLabel') }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.paddleocr_vl_use_seal_recognition">{{ $t('settings.parser.sealRecognition') }}</t-checkbox>
              <t-checkbox v-model="config.paddleocr_vl_use_chart_recognition">{{ $t('settings.parser.chartRecognition') }}</t-checkbox>
            </div>
          </div>
        </section>

        <!-- Bölüm 3 — paddleocr_vl_cloud bulut API yapılandırması -->
        <section v-if="currentEngine.Name === 'paddleocr_vl_cloud'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ $t('settings.parser.configSection') }}</h4>

          <div class="form-item">
            <label class="form-label required">Token</label>
            <t-input
              v-model="config.paddleocr_vl_cloud_token"
              type="password"
              :placeholder="$t('settings.parser.paddleocrVlCloudTokenPlaceholder')"
              clearable
            >
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </div>
          <div class="form-item">
            <label class="form-label">Model</label>
            <t-input
              v-model="config.paddleocr_vl_cloud_model"
              placeholder="PaddleOCR-VL-1.6"
              clearable
            />
          </div>
          <div class="form-item">
            <label class="form-label">{{ $t('settings.parser.featuresLabel') }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.paddleocr_vl_cloud_use_seal_recognition">{{ $t('settings.parser.sealRecognition') }}</t-checkbox>
              <t-checkbox v-model="config.paddleocr_vl_cloud_use_chart_recognition">{{ $t('settings.parser.chartRecognition') }}</t-checkbox>
            </div>
          </div>
        </section>
      </div>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { MessagePlugin } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import {
  getParserEngines,
  getParserEngineConfig,
  updateParserEngineConfig,
  checkParserEngines,
  type ParserEngineInfo,
  type ParserEngineConfig,
} from '@/api/system'

const { t } = useI18n()
const authStore = useAuthStore()

const CONFIGURABLE_ENGINES = new Set(['mineru', 'mineru_cloud', 'paddleocr_vl', 'paddleocr_vl_cloud'])

const UNAVAILABLE_REASON_KEYS: Record<string, string> = {
  'DocReader service not connected': 'docreaderDisconnected',
  'MinerU service not configured': 'mineruNotConfigured',
  'MinerU API Key not configured': 'mineruApiKeyMissing',
  'PaddleOCR-VL service not configured': 'paddleocrNotConfigured',
  'PaddleOCR-VL Cloud Token not configured': 'paddleocrCloudTokenMissing',
  'MinerU endpoint is not configured': 'mineruNotConfigured',
  'MinerU Cloud API key is not configured': 'mineruApiKeyMissing',
  'PaddleOCR-VL endpoint is not configured': 'paddleocrNotConfigured',
  'PaddleOCR-VL Cloud token is not configured': 'paddleocrCloudTokenMissing',
  'MinerU requires an API key': 'mineruRequiresApiKey',
  'MinerU API key is invalid': 'mineruApiKeyInvalid',
  'MinerU Cloud API key is invalid': 'mineruApiKeyInvalid',
}

/** Her ayrıştırma motorunun proje/resmi dokümantasyon adresleri */
const ENGINE_DOC_LINKS: Record<string, string> = {
  markitdown: 'https://github.com/microsoft/markitdown',
  mineru: 'https://github.com/opendatalab/MinerU',
  mineru_cloud: 'https://mineru.net/apiManage/docs',
  paddleocr_vl: 'https://github.com/PaddlePaddle/PaddleOCR',
  paddleocr_vl_cloud: 'https://aistudio.baidu.com/paddleocr',
}

/** Ayrıştırma motoru yapılandırması varsayılan değerleri (DocReader/Python tarafıyla uyumlu) */
const DEFAULT_PARSER_CONFIG: ParserEngineConfig = {
  docreader_addr: '',
  docreader_transport: 'grpc',
  mineru_endpoint: '',
  mineru_api_key: '',
  mineru_server_api_key: '',
  mineru_tier: '',
  mineru_model: 'pipeline',
  mineru_vlm_server_url: '',
  mineru_enable_formula: true,
  mineru_enable_table: true,
  mineru_parse_method: 'auto',
  mineru_enable_ocr: true,
  mineru_language: 'ch',
  mineru_cloud_model: 'pipeline',
  mineru_cloud_enable_formula: true,
  mineru_cloud_enable_table: true,
  mineru_cloud_enable_ocr: true,
  mineru_cloud_language: 'ch',
  paddleocr_vl_endpoint: '',
  paddleocr_vl_use_seal_recognition: true,
  paddleocr_vl_use_chart_recognition: false,
  paddleocr_vl_cloud_token: '',
  paddleocr_vl_cloud_model: 'PaddleOCR-VL-1.6',
  paddleocr_vl_cloud_use_seal_recognition: true,
  paddleocr_vl_cloud_use_chart_recognition: false,
}

const engines = ref<ParserEngineInfo[]>([])
const docreaderAddrEnv = ref('')
const docreaderTransport = ref<'grpc' | 'http'>('grpc')
const connected = ref(false)
const loading = ref(true)
const error = ref('')

const config = ref<ParserEngineConfig>({ ...DEFAULT_PARSER_CONFIG })
const saving = ref(false)
const saveMessage = ref('')
const saveSuccess = ref(false)
const checking = ref(false)
const checkMessage = ref('')

const hasBuiltinEngine = computed(() => engines.value.some(e => e.Name === 'builtin'))

const drawerVisible = ref(false)
const currentEngine = ref<ParserEngineInfo | null>(null)
const drawerTitle = computed(() => {
  return currentEngine.value ? getEngineDisplayName(currentEngine.value.Name) : ''
})

// SettingDrawer başlık simgesi #headerIcon yuvasını kullanır (ilk harf monogramı + motor bazlı
// renk şeması, liste kartıyla tamamen tutarlı); artık t-icon name geri dönüşüne gerek yok.

// Whether the footer test-connection button should appear. Engines without
// configurable fields and that aren't the builtin DocReader (whose connection
// status is the whole point of the drawer) skip the test affordance — for
// e.g. simple/markitdown there's nothing to validate beyond presence.
const needsTestButton = computed(() => {
  if (!currentEngine.value) return false
  return hasConfigFields(currentEngine.value.Name) || currentEngine.value.Name === 'builtin'
})

/** Sabit gösterim sırası; listelenmeyen motorlar sonda ada göre sıralanır */
const ENGINE_ORDER: Record<string, number> = {
  builtin: 0,
  simple: 2,
  anydoc: 3,
  markitdown: 4,
  mineru: 5,
  mineru_cloud: 6,
  paddleocr_vl: 7,
  paddleocr_vl_cloud: 8,
}

const sortedEngines = computed(() => {
  return [...engines.value].sort((a, b) => {
    const oa = ENGINE_ORDER[a.Name] ?? 100
    const ob = ENGINE_ORDER[b.Name] ?? 100
    if (oa !== ob) return oa - ob
    return a.Name.localeCompare(b.Name)
  })
})

function hasConfigFields(engineName: string): boolean {
  return CONFIGURABLE_ENGINES.has(engineName)
}

function engineDocLink(name: string): string | undefined {
  return ENGINE_DOC_LINKS[name]
}

function engineDocLabel(_name: string): string {
  return t('settings.parser.docs')
}

// Kart rozeti ilk harfi. Öncelikle yerelleştirilmiş adın ilk karakterini kullan (yerleşik/basit gibi Çince senaryoları kapsar),
// geri dönüş olarak engine name kullan; İngilizce/Çince için kararlı ve okunabilir bir monogram gösterilmesini sağla.
function engineInitial(engineName: string): string {
  const display = getEngineDisplayName(engineName)
  return (display.trim().charAt(0) || engineName.charAt(0) || '?').toUpperCase()
}

function getEngineDisplayName(engineName: string): string {
  const key = `kbSettings.parser.engines.${engineName}.name`
  const translated = t(key)
  return translated !== key ? translated : engineName
}

function getEngineDisplayDesc(engineName: string, fallback: string): string {
  const key = `kbSettings.parser.engines.${engineName}.desc`
  const translated = t(key)
  return translated !== key ? translated : fallback
}

function getUnavailableReason(reason: string): string {
  const key = UNAVAILABLE_REASON_KEYS[reason]
  return key ? t('settings.parser.unavailableReasons.' + key) : reason
}

function openDrawer(engine: ParserEngineInfo) {
  currentEngine.value = engine
  drawerVisible.value = true
  saveMessage.value = ''
  checkMessage.value = ''
}

async function loadEngines() {
  try {
    const res = await getParserEngines()
    engines.value = res?.data ?? []
    docreaderAddrEnv.value = res?.docreader_addr ?? ''
    const transport = (res?.docreader_transport ?? 'grpc').toLowerCase()
    docreaderTransport.value = transport === 'http' ? 'http' : 'grpc'
    connected.value = res?.connected ?? (engines.value.length > 0)
  } catch (e: any) {
    error.value = e?.message || t('settings.parser.loadFailed')
    engines.value = []
    connected.value = false
  }
}

async function loadConfig() {
  try {
    const res = await getParserEngineConfig()
    const data = res?.data
    config.value = {
      docreader_addr: data?.docreader_addr ?? DEFAULT_PARSER_CONFIG.docreader_addr ?? '',
      docreader_transport: data?.docreader_transport ?? DEFAULT_PARSER_CONFIG.docreader_transport ?? 'grpc',
      mineru_endpoint: data?.mineru_endpoint ?? DEFAULT_PARSER_CONFIG.mineru_endpoint ?? '',
      mineru_api_key: data?.mineru_api_key ?? DEFAULT_PARSER_CONFIG.mineru_api_key ?? '',
      mineru_server_api_key: data?.mineru_server_api_key ?? DEFAULT_PARSER_CONFIG.mineru_server_api_key ?? '',
      mineru_tier: data?.mineru_tier ?? DEFAULT_PARSER_CONFIG.mineru_tier ?? '',
      mineru_model: data?.mineru_model ?? DEFAULT_PARSER_CONFIG.mineru_model ?? '',
      mineru_vlm_server_url: data?.mineru_vlm_server_url ?? DEFAULT_PARSER_CONFIG.mineru_vlm_server_url ?? '',
      mineru_enable_formula: data?.mineru_enable_formula ?? DEFAULT_PARSER_CONFIG.mineru_enable_formula ?? true,
      mineru_enable_table: data?.mineru_enable_table ?? DEFAULT_PARSER_CONFIG.mineru_enable_table ?? true,
      mineru_parse_method: data?.mineru_parse_method ?? (data?.mineru_enable_ocr === false ? 'txt' : 'auto'),
      mineru_enable_ocr: data?.mineru_enable_ocr ?? DEFAULT_PARSER_CONFIG.mineru_enable_ocr ?? true,
      mineru_language: data?.mineru_language ?? DEFAULT_PARSER_CONFIG.mineru_language ?? 'ch',
      mineru_cloud_model: data?.mineru_cloud_model ?? DEFAULT_PARSER_CONFIG.mineru_cloud_model ?? '',
      mineru_cloud_enable_formula: data?.mineru_cloud_enable_formula ?? DEFAULT_PARSER_CONFIG.mineru_cloud_enable_formula ?? true,
      mineru_cloud_enable_table: data?.mineru_cloud_enable_table ?? DEFAULT_PARSER_CONFIG.mineru_cloud_enable_table ?? true,
      mineru_cloud_enable_ocr: data?.mineru_cloud_enable_ocr ?? DEFAULT_PARSER_CONFIG.mineru_cloud_enable_ocr ?? true,
      mineru_cloud_language: data?.mineru_cloud_language ?? DEFAULT_PARSER_CONFIG.mineru_cloud_language ?? 'ch',
      paddleocr_vl_endpoint: data?.paddleocr_vl_endpoint ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_endpoint ?? '',
      paddleocr_vl_use_seal_recognition: data?.paddleocr_vl_use_seal_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_use_seal_recognition ?? true,
      paddleocr_vl_use_chart_recognition: data?.paddleocr_vl_use_chart_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_use_chart_recognition ?? false,
      paddleocr_vl_cloud_token: data?.paddleocr_vl_cloud_token ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_token ?? '',
      paddleocr_vl_cloud_model: data?.paddleocr_vl_cloud_model ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_model ?? 'PaddleOCR-VL-1.6',
      paddleocr_vl_cloud_use_seal_recognition: data?.paddleocr_vl_cloud_use_seal_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_use_seal_recognition ?? true,
      paddleocr_vl_cloud_use_chart_recognition: data?.paddleocr_vl_cloud_use_chart_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_use_chart_recognition ?? false,
    }
  } catch {
    config.value = { ...DEFAULT_PARSER_CONFIG }
  }
}

async function loadAll() {
  loading.value = true
  error.value = ''
  await Promise.all([loadEngines(), loadConfig()])
  loading.value = false
}

function buildConfigPayload(): ParserEngineConfig {
  return {
    docreader_addr: config.value.docreader_addr?.trim() ?? '',
    docreader_transport: (config.value.docreader_transport ?? 'grpc').trim() || 'grpc',
    mineru_endpoint: config.value.mineru_endpoint?.trim() ?? '',
    mineru_api_key: config.value.mineru_api_key?.trim() ?? '',
    mineru_server_api_key: config.value.mineru_server_api_key?.trim() ?? '',
    mineru_tier: config.value.mineru_tier?.trim() ?? '',
    mineru_model: config.value.mineru_model?.trim() ?? '',
    mineru_vlm_server_url: config.value.mineru_vlm_server_url?.trim() ?? '',
    mineru_enable_formula: config.value.mineru_enable_formula,
    mineru_enable_table: config.value.mineru_enable_table,
    mineru_parse_method: config.value.mineru_parse_method ?? 'auto',
    // Keep the legacy toggle during rolling upgrades. New servers prefer parse_method.
    mineru_enable_ocr: config.value.mineru_parse_method !== 'txt',
    mineru_language: config.value.mineru_language?.trim() ?? '',
    mineru_cloud_model: config.value.mineru_cloud_model?.trim() ?? '',
    mineru_cloud_enable_formula: config.value.mineru_cloud_enable_formula,
    mineru_cloud_enable_table: config.value.mineru_cloud_enable_table,
    mineru_cloud_enable_ocr: config.value.mineru_cloud_enable_ocr,
    mineru_cloud_language: config.value.mineru_cloud_language?.trim() ?? '',
    paddleocr_vl_endpoint: config.value.paddleocr_vl_endpoint?.trim() ?? '',
    paddleocr_vl_use_seal_recognition: config.value.paddleocr_vl_use_seal_recognition,
    paddleocr_vl_use_chart_recognition: config.value.paddleocr_vl_use_chart_recognition,
    paddleocr_vl_cloud_token: config.value.paddleocr_vl_cloud_token?.trim() ?? '',
    paddleocr_vl_cloud_model: config.value.paddleocr_vl_cloud_model?.trim() ?? '',
    paddleocr_vl_cloud_use_seal_recognition: config.value.paddleocr_vl_cloud_use_seal_recognition,
    paddleocr_vl_cloud_use_chart_recognition: config.value.paddleocr_vl_cloud_use_chart_recognition,
  }
}

async function onCheck() {
  if (!connected) {
    checkMessage.value = t('settings.parser.ensureDocreaderConnected')
    return
  }
  checking.value = true
  checkMessage.value = ''
  saveMessage.value = ''
  try {
    const res = await checkParserEngines(buildConfigPayload())
    engines.value = res?.data ?? []
    if (res?.connected !== undefined) {
      connected.value = res.connected
    }

    if (currentEngine.value) {
      if (currentEngine.value.Name === 'builtin') {
        if (connected.value) {
          checkMessage.value = t('settings.parser.checkSuccess')
          saveSuccess.value = true
        } else {
          checkMessage.value = t('settings.parser.checkFailed')
          saveSuccess.value = false
        }
      } else {
        const updatedEngine = engines.value.find(e => e.Name === currentEngine.value!.Name)
        if (updatedEngine) {
          if (updatedEngine.Available) {
            checkMessage.value = t('settings.parser.checkSuccess')
            saveSuccess.value = true
          } else {
            checkMessage.value = updatedEngine.UnavailableReason
              ? getUnavailableReason(updatedEngine.UnavailableReason)
              : t('settings.parser.checkFailed')
            saveSuccess.value = false
          }
        } else {
          checkMessage.value = t('settings.parser.checkFailed')
          saveSuccess.value = false
        }
      }
    } else {
      checkMessage.value = t('settings.parser.checkDoneStatusUpdated')
      saveSuccess.value = true
    }

    setTimeout(() => { checkMessage.value = '' }, 3000)
  } catch (e: any) {
    checkMessage.value = e?.message || t('settings.parser.checkFailed')
    saveSuccess.value = false
  } finally {
    checking.value = false
  }
}

async function onSave() {
  saving.value = true
  saveMessage.value = ''
  try {
    await updateParserEngineConfig(buildConfigPayload())
    saveSuccess.value = true
    saveMessage.value = t('settings.parser.saveSuccess')
    drawerVisible.value = false
    loadEngines()
  } catch (e: any) {
    saveSuccess.value = false
    saveMessage.value = e?.message || t('settings.parser.saveFailed')
  } finally {
    saving.value = false
  }
}

onMounted(loadAll)
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/provider-card.less';

@import (reference) '@/components/css/settings-section.less';

.parser-engine-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-base);
}

.error-inline {
  padding: 16px 0;
}

.empty-state {
  padding: 48px 0;
  text-align: center;

  .empty-text {
    font-size: var(--app-text-base);
    color: var(--td-text-color-placeholder);
    margin: 0;
  }
}

// ---- Motor kartı düzeni ----
.engine-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
  margin-top: 24px;
}

// ModelSettings / WebSearchSettings / McpSettings ile aynı biçimde sağlayıcı kartı.
// Burada kartın tamamı bir button'dır — tıklanınca yapılandırma çekmecesi açılır; active durumu marka rengi çerçevesi kullanır.
.engine-card {
  .provider-card();
  .provider-card-interactive();
  text-align: left;
  font: inherit;
  color: inherit;
  cursor: pointer;

  &--active {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-1);
  }
}

.engine-card__badge {
  .provider-card-badge();
  .provider-card-badge-color(#0052d9);
}

// Ayrıştırma motoru rozet renkleri — yerleşik/resmi grup yeşil, harici araçlar özelliklerine göre farklı renkler alır.
.engine-card--builtin .engine-card__badge {
  .provider-card-badge-color(#17b88b);
}
.engine-card--simple .engine-card__badge {
  .provider-card-badge-color(#464646);
}
.engine-card--markitdown .engine-card__badge {
  .provider-card-badge-color(#0089ff);
}
.engine-card--mineru .engine-card__badge,
.engine-card--mineru_cloud .engine-card__badge,
.engine-card--paddleocr_vl .engine-card__badge,
.engine-card--paddleocr_vl_cloud .engine-card__badge {
  .provider-card-badge-color(#6235bb);
}

.engine-card__body {
  .provider-card-body();
}

.engine-card__header {
  .provider-card-header();
}

.engine-card__title {
  .provider-card-title();
}

// McpSettings ile tutarlı dot+metin durum rozeti. on=yeşil, err=kırmızı, help için cursor:help ipucu kullanılır.
.engine-card__status {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 1px 8px 1px 6px;
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 16px;
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-secondarycontainer);

  &--on {
    color: var(--td-success-color-7);

    .engine-card__status-dot { background: var(--td-success-color); }
  }

  &--err {
    color: var(--td-error-color-7);

    .engine-card__status-dot { background: var(--td-error-color); }
  }

  &--help {
    cursor: help;
  }
}

.engine-card__status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.engine-card__desc {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  margin: 0;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

// ---- Çekmece içeriği — ModelEditorDialog ile aynı kurallar ----
// .form-item / .form-label / .form-desc / .api-test
// frontend/src/components/ModelEditorDialog.vue içindeki adlandırma ile yazı boyutu/boşluklara bakın
.form-item {
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;

  // ModelEditorDialog ile tutarlı: zorunlu yıldız işareti başta
  &.required::before {
    content: '*';
    color: var(--td-error-color);
    margin-right: 4px;
    font-weight: 500;
    line-height: 1;
  }
}

.form-desc {
  margin: 4px 0 0 0;
  font-size: var(--app-text-xs);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

// Girdi alanları için yazı boyutunu birleştirin
:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: var(--app-text-md);
}

:deep(.t-checkbox) {
  font-size: var(--app-text-md);

  .t-checkbox__label {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
  }
}

// ---- DocReader bağlantı bilgileri (builtin motoru) ----
.docreader-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  .form-desc {
    margin-top: 0;
  }
}

.status-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.env-hint {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

// ---- Dosya türü chip'i ----
.file-types {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.file-type-chip {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  font-size: var(--app-text-xs);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-component);
  border-radius: var(--app-radius-xs);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  letter-spacing: 0.02em;
}

// ---- Form geçiş grubu (formül/tablo/OCR) ----
.form-toggles {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 8px 0 0;
}

// ---- footer-left bağlantı testi mesajı (ModelEditorDialog ile aynı) ----
.footer-test-message {
  font-size: var(--app-text-sm);
  line-height: 1.4;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &.success {
    color: var(--td-brand-color-active);
  }

  &.error {
    color: var(--td-error-color);
  }
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

// ---- Belge dış bağlantısı ----
.doc-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-brand-color);
  text-decoration: none;
  transition: color var(--app-motion-fast) ease;

  &:hover {
    color: var(--td-brand-color-active);
  }

  .link-icon {
    font-size: var(--app-text-base);
  }

  // Alt başlıktaki inline belge bağlantısı: açıklama metniyle aynı satırda, küçük metinle eşdeğer boyutta
  &--inline {
    margin-left: 6px;
    font-size: var(--app-text-sm);
    font-weight: 500;
    vertical-align: baseline;

    .link-icon {
      font-size: var(--app-text-sm);
    }
  }
}

// ---- Header simgesinin ilk harf monogramı (motor başına renkler scoped olmayan blokta) ----
.header-icon__text {
  font-size: var(--app-text-lg);
  font-weight: 500;
  letter-spacing: 0.02em;
}
</style>

<!--
  Non-scoped block: per-engine header-icon coloring. Keep these rules global
  so they always apply
  regardless of whether the drawer panel inherits the parent's scoped
  data attributes. Each rule mirrors the matching .engine-card--{name}
  .engine-card__badge from the scoped block above.
-->
<style lang="less">
.parser-engine-drawer--builtin .setting-drawer__header-icon {
  background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  color: var(--td-brand-color);
}
.parser-engine-drawer--simple .setting-drawer__header-icon {
  background: rgba(70, 70, 70, 0.1);
  color: #464646;
}
.parser-engine-drawer--markitdown .setting-drawer__header-icon {
  background: rgba(0, 137, 255, 0.12);
  color: #0089FF;
}
.parser-engine-drawer--mineru .setting-drawer__header-icon,
.parser-engine-drawer--mineru_cloud .setting-drawer__header-icon,
.parser-engine-drawer--paddleocr_vl .setting-drawer__header-icon,
.parser-engine-drawer--paddleocr_vl_cloud .setting-drawer__header-icon {
  background: rgba(98, 53, 187, 0.12);
  color: #6235BB;
}
</style>
