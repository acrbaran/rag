import { get, post, put, del } from "../../utils/request";

// Agent yapılandırması
// Akıllı muhakeme altındaki ajan türü ön ayar kimliği
// 'rag-qa'       : Klasik belge/SSS parçalı RAG
// 'wiki-qa'      : Wiki bilgi grafiği gezinmeli soru-cevap
// 'hybrid-rag-wiki': Wiki + parçalı hibrit getirme
// 'custom'       : Tamamen özel (ön ayar uygulanmaz)
export type AgentType = 'rag-qa' | 'wiki-qa' | 'hybrid-rag-wiki' | 'data-analysis' | 'custom';

export interface QuestionSuggestionConfig {
  starters: {
    enabled: boolean;
    mode: 'curated' | 'knowledge' | 'hybrid';
    items: string[];
    count: number;
  };
  follow_ups: {
    enabled: boolean;
    mode: 'generated' | 'knowledge' | 'hybrid';
    count: number;
    model_id?: string;
    additional_instruction?: string;
    categories: Array<'clarify' | 'deepen' | 'action'>;
    max_context_turns: number;
    suppress_on_fallback: boolean;
    suppress_when_answer_asks_question: boolean;
    knowledge_fallback: boolean;
    allow_regenerate: boolean;
  };
}

export interface CustomAgentConfig {
  // ===== Temel Ayarlar =====
  agent_mode?: 'quick-answer' | 'smart-reasoning';  // Çalışma modu: quick-answer=RAG modu, smart-reasoning=ReAct Agent modu
  // Akıllı muhakeme modundaki tür ön ayarı; "sistem istemi + araçlar + KB uyumluluğu" birleşimini tek tıklamayla uygulamak için kullanılır
  // Yalnızca agent_mode === 'smart-reasoning' iken geçerlidir; quick-answer modu yok sayar
  agent_type?: AgentType;
  system_prompt?: string;           // Birleşik sistem istemi (davranışı dinamik olarak kontrol etmek için {{web_search_status}} yer tutucusu kullanılır)
  system_prompt_id?: string;        // Başvurulan prompt template ID (ön ayar bu alanı doldurur)
  context_template_id?: string;     // Inherit the referenced context template when text is empty
  context_template?: string;        // Bağlam şablonu (normal mod)

  // ===== Model Ayarları =====
  model_id?: string;
  rerank_model_id?: string;         // ReRank model ID
  temperature?: number;
  max_completion_tokens?: number;   // 0 = sistem varsayılanını takip et (hızlı yanıt 2048; akıllı muhakeme 4096, yazılabilir dosyalar için sandbox bağlandığında 24576). 0'dan büyük değer özel üst sınırdır
  thinking?: boolean;                      // Düşünme modunun etkinleştirilip etkinleştirilmeyeceği (genişletilmiş düşünmeyi destekleyen modeller)
  // Düşünme yoğunluğu: off | auto | minimal | low | medium | high | xhigh | max。
  // Boş olduğunda thinking boolean değerine geri döner (true == auto, false == off)。
  reasoning_effort?: string;
  citation_enabled?: boolean;        // Son yanıtta bilgi tabanı/web kaynağı alıntılarının gösterilip gösterilmeyeceği (varsayılan olarak etkin)

  // ===== Agent Modu Ayarları =====
  max_iterations?: number;          // Maksimum yineleme sayısı; -1 sınırsız anlamına gelir
  llm_call_timeout?: number;        // LLM çağrısı zaman aşımı süresi (saniye)
  allowed_tools?: string[];         // İzin verilen araçlar
  reflection_enabled?: boolean;     // Yansıtmanın etkinleştirilip etkinleştirilmeyeceği
  // MCP hizmet seçimi modu: all=tüm etkin MCP hizmetleri, selected=belirtilen hizmetler, none=MCP kullanma
  mcp_selection_mode?: 'all' | 'selected' | 'none';
  mcp_services?: string[];          // Seçilen MCP hizmet ID listesi
  // Konuşmada OAuth yetkilendirmesi tetiklendiğindeki bekleme zaman aşımı (saniye): süre dolduğunda yetkilendirme istemi otomatik olarak atlanır。
  // <=0 olduğunda sunucu varsayılan zaman aşımı kullanılır. Yalnızca OAuth kullanan MCP hizmetleri için geçerlidir。
  mcp_auth_wait_timeout?: number;

  // ===== Skills Ayarları (yalnızca Agent modu) =====
  // Skills seçim modu: all=tüm önceden yüklenmiş olanlar, selected=belirtilenler, none=kullanma
  skills_selection_mode?: 'all' | 'selected' | 'none';
  selected_skills?: string[];       // Seçilen Skill adları listesi

  // ===== Korumalı Alan Ayarları =====
  // Bu ajanın beceri betiklerinin hangi korumalı alan yapılandırmasında çalışacağını belirtir; boş olması korumalı alan yürütmesinin etkinleştirilmediği anlamına gelir。
  // Belirli bir sürüm yerine mantıksal yapılandırmaya işaret eder; kimlik bilgileri döndürüldüğünde her ajanı yeniden atamaya gerek yoktur。
  sandbox_config_id?: string;

  // ===== Bilgi Tabanı Ayarları =====
  // Bilgi tabanı seçim modu: all=tüm bilgi tabanları, selected=belirtilen bilgi tabanları, none=bilgi tabanı kullanma
  kb_selection_mode?: 'all' | 'selected' | 'none';
  knowledge_bases?: string[];
  // Bilgi tabanının yalnızca açıkça @ ile anıldığında getirilip getirilmeyeceği (varsayılan: false)
  // true: yalnızca kullanıcı bilgi tabanını/belgeyi @ ile açıkça andığında getirilir
  // false: kb_selection_mode değerine göre bilgi tabanı otomatik olarak getirilir
  retrieve_kb_only_when_mentioned?: boolean;

  // ===== Görsel yükleme/çok modlu ayarlar =====
  image_upload_enabled?: boolean;    // Görsel yüklemenin etkinleştirilip etkinleştirilmeyeceği (varsayılan: false)
  vlm_model_id?: string;            // VLM model ID (görsel analizi için)
  image_storage_provider?: string;   // Görsel depolama sağlayıcısı
  audio_upload_enabled?: boolean;    // Ses yükleme/ASR transkripsiyonunun etkinleştirilip etkinleştirilmeyeceği (varsayılan: false)
  asr_model_id?: string;            // ASR model ID (ses transkripsiyonu için)
  // Ek görsel anlama / taranmış belge OCR anahtarı (varsayılan: false, etkinleştirmek ayrıştırma süresini artırır)
  attachment_image_understanding?: boolean;
  // Taranmış belge OCR için maksimum sayfa sayısı (0 = genel varsayılan `RETHRA_CHAT_ATTACHMENT_OCR_MAX_PAGES` kullanılır)
  attachment_ocr_max_pages?: number;
  // Tek turlu soru-cevapta ek ayrıştırmasının tamamlanması için en uzun bekleme süresi (saniye, 0 = genel varsayılan `RETHRA_CHAT_ATTACHMENT_WAIT_TIMEOUT_SEC` kullanılır)
  attachment_parse_wait_timeout_sec?: number;

  // ===== Sohbet eki ayrıştırma motoru stratejisi =====
  // Dosya türüne göre ayrıştırma motorunu seçer; öncelik: istek `parser_engine` > ajan kuralı > kiracı kuralı > auto
  chat_parser_engine_rules?: { file_types: string[]; engine: string }[];

  // ===== Dosya türü sınırlamaları =====
  // Desteklenen dosya türleri (ör. ["csv", "xlsx", "xls"])
  // Boş olması tüm dosya türlerinin desteklendiğini belirtir
  supported_file_types?: string[];

  // ===== Web araması ayarları =====
  web_search_enabled?: boolean;
  web_search_provider_id?: string;
  web_search_max_results?: number;

  // ===== Çok turlu konuşma ayarları =====
  multi_turn_enabled?: boolean;     // Çok turlu konuşmanın etkinleştirilip etkinleştirilmeyeceği
  history_turns?: number;           // Korunacak geçmiş tur sayısı

  // ===== Uzun süreli bellek =====
  // Bu ajanın kullanıcının uzun süreli belleğini okuyup okuyamayacağı.
  // Varsayılan değer (eski veriler) true ile eşdeğerdir: bu yalnızca "kapatılabilen" bir anahtardır; çalışma alanı ayarı kapalıysa
  // burada açılması da etkili olmaz.
  memory_enabled?: boolean;

  // ===== Getirme stratejisi ayarları =====
  embedding_top_k?: number;         // Vektör geri getirme TopK
  keyword_threshold?: number;       // Anahtar kelime geri çağırma eşiği
  vector_threshold?: number;        // Vektör geri çağırma eşiği
  rerank_top_k?: number;            // Yeniden sıralama TopK
  rerank_threshold?: number;        // Yeniden sıralama eşiği

  // ===== Gelişmiş ayarlar (çoğunlukla normal mod için) =====
  enable_query_expansion?: boolean; // Sorgu genişletme etkinleştirilsin mi
  enable_rewrite?: boolean;         // Soru yeniden yazımı etkinleştirilsin mi
  rewrite_prompt_system?: string;   // Yeniden yazım sistem istemi
  rewrite_prompt_user?: string;     // Yeniden yazım kullanıcı istemi şablonu
  fallback_strategy?: 'fixed' | 'model'; // Yedek strateji
  fallback_response?: string;       // Sabit yedek yanıt
  fallback_prompt?: string;         // Yedek istemi (model oluştururken)
  // Niyet istemi: getirme dışı niyetlerde (selamlama, sohbet vb.) ana sistem isteminin üzerine yazılır
  intent_prompts?: Record<string, string>;

  // ===== Kullanımdan kaldırılmış alanlar (uyumluluk için korunur) =====
  welcome_message?: string;
  question_suggestions?: QuestionSuggestionConfig;
}

// Ajan
export interface CustomAgent {
  id: string;
  name: string;
  description?: string;
  avatar?: string;
  is_builtin: boolean;
  tenant_id?: number;
  created_by?: string;
  // `creator_name`, arka uçtaki list arayüzü tarafından toplu olarak doldurulur ve yalnızca liste kartlarındaki kaynak rozeti için kullanılır.
  creator_name?: string;
  config: CustomAgentConfig;
  created_at?: string;
  updated_at?: string;
}

// Ajan oluşturma isteği
export interface CreateAgentRequest {
  name: string;
  description?: string;
  avatar?: string;
  config?: CustomAgentConfig;
}

// Ajan güncelleme isteği
export interface UpdateAgentRequest {
  name: string;
  description?: string;
  avatar?: string;
  config?: CustomAgentConfig;
}

// Yerleşik ajan ID'si (kod başvurularını kolaylaştıran, yaygın kullanılan ayrılmış sabitler)
export const BUILTIN_QUICK_ANSWER_ID = 'builtin-quick-answer';
export const BUILTIN_SMART_REASONING_ID = 'builtin-smart-reasoning';

// `AgentMode` sabiti
export const AGENT_MODE_QUICK_ANSWER = 'quick-answer';
export const AGENT_MODE_SMART_REASONING = 'smart-reasoning';

// Deprecated: Use BUILTIN_QUICK_ANSWER_ID instead
export const BUILTIN_AGENT_NORMAL_ID = BUILTIN_QUICK_ANSWER_ID;
// Deprecated: Use BUILTIN_SMART_REASONING_ID instead
export const BUILTIN_AGENT_AGENT_ID = BUILTIN_SMART_REASONING_ID;

// Etkinleştirilmiş ajanlar dahil ajan listesini al
// disabled_own_agent_ids: Geçerli alanda sohbet açılır menüsünde devre dışı bırakılan "benim" ajan kimlikleri; yalnızca bu alanı etkiler
export function listAgents(params?: {
  /**
   * Optional creator filter; mirrors listKnowledgeBases. Built-in agents
   * (is_builtin=true) are always returned regardless of this filter so
   * the conversation dropdown never silently loses quick-answer /
   * smart-reasoning when a user picks "Created by me".
   */
  creator?: 'all' | 'mine' | 'others';
}) {
  const qs = params?.creator && params.creator !== 'all' ? `?creator=${params.creator}` : '';
  return get<{ data: CustomAgent[]; disabled_own_agent_ids?: string[] }>(`/api/v1/agents${qs}`);
}

// Ajan ayrıntılarını al
export function getAgentById(id: string) {
  return get<{ data: CustomAgent }>(`/api/v1/agents/${id}`);
}

// Ajan oluştur
export function createAgent(data: CreateAgentRequest) {
  return post<{ data: CustomAgent }>('/api/v1/agents', data);
}

// Ajanı güncelle
export function updateAgent(id: string, data: UpdateAgentRequest) {
  return put<{ data: CustomAgent }>(`/api/v1/agents/${id}`, data);
}

// Ajanı sil
export function deleteAgent(id: string) {
  return del<{ success: boolean }>(`/api/v1/agents/${id}`);
}

// Ajanı kopyala
export function copyAgent(id: string) {
  return post<{ data: CustomAgent }>(`/api/v1/agents/${id}/copy`);
}

// Yerleşik ajan olup olmadığını belirle (`agent.is_builtin` alanı veya ID önekiyle)
export function isBuiltinAgent(agentId: string): boolean {
  return agentId.startsWith('builtin-');
}

// Yer tutucu tanımları
export interface PlaceholderDefinition {
  name: string;
  label: string;
  description: string;
}

// Yer tutucu yanıtı
export interface PlaceholdersResponse {
  all: PlaceholderDefinition[];
  system_prompt: PlaceholderDefinition[];
  agent_system_prompt: PlaceholderDefinition[];
  context_template: PlaceholderDefinition[];
  rewrite_system_prompt: PlaceholderDefinition[];
  rewrite_prompt: PlaceholderDefinition[];
  fallback_prompt: PlaceholderDefinition[];
}

// Yer tutucu tanımlarını al
export function getPlaceholders() {
  return get<{ data: PlaceholdersResponse }>('/api/v1/agents/placeholders');
}

// ===== Ajan türü ön ayarları =====

// Arka uç `kb_filter` yapısı (bkz. `internal/types/agent_type_preset.go`)
export interface AgentTypeKBFilter {
  any_of?: string[];   // KB en az birine sahip olmalı
  all_of?: string[];   // KB hepsine sahip olmalı
  none_of?: string[];  // KB hiçbirine sahip olmamalı
}

// KB yetenek etiketleri (arka uç `types.KBCapabilities` JSON'u)
export interface KBCapabilities {
  vector: boolean;
  keyword: boolean;
  wiki: boolean;
  graph: boolean;
  faq: boolean;
}

// Ön ayarlı "otomatik doldurma" yapılandırma yükü: yalnızca ön ayarın geçersiz kıldığı alanları içerir; diğer alanlar değişmez
export interface AgentTypePresetConfig {
  system_prompt_id?: string;
  temperature?: number;
  max_iterations?: number;
  allowed_tools?: string[];
  retain_retrieval_history?: boolean;
  faq_priority_enabled?: boolean;
  web_search_enabled?: boolean;
  supported_file_types?: string[];
  kb_selection_mode?: 'all' | 'selected' | 'none';
}

export interface AgentTypePresetI18n {
  label: string;
  description: string;
}

export interface AgentTypePreset {
  id: AgentType;
  i18n: Record<string, AgentTypePresetI18n>;
  config?: AgentTypePresetConfig;     // Boş olması "özel" türü (ön ayar yok) anlamına gelir
  kb_filter?: AgentTypeKBFilter;      // Boş olması tüm KB'lerin seçilebilir olduğu anlamına gelir
}

// Tür ön ayarı listesini çek (düzenleyici için)
export function getAgentTypePresets() {
  return get<{ data: AgentTypePreset[] }>('/api/v1/agents/type-presets');
}

// ===== IM kanalları =====

export interface IMChannel {
  id: string;
  tenant_id?: number;
  agent_id: string;
  platform: 'slack' | 'telegram' | 'wechat' | 'qqbot';
  name: string;
  enabled: boolean;
  mode: 'webhook' | 'websocket' | 'longpoll';
  output_mode: 'stream' | 'full';
  locale?: '' | 'en-US' | 'tr-TR';
  session_mode?: 'user' | 'thread';
  knowledge_base_id?: string;
  credentials: Record<string, any>;
  created_at?: string;
  updated_at?: string;
}

export interface IMChannelSummary extends Omit<IMChannel, 'credentials'> {
  credentials_configured: boolean;
}

export function listIMChannels(agentId: string) {
  return get<{ data: IMChannelSummary[] }>(`/api/v1/agents/${agentId}/im-channels`);
}

// Tenant-wide overview row. Credentials are intentionally omitted.
export interface IMChannelOverview {
  id: string;
  tenant_id: number;
  agent_id: string;
  agent_name: string; // localized built-in name when the agent is built-in
  platform: IMChannel['platform'];
  name: string;
  enabled: boolean;
  mode: IMChannel['mode'];
  output_mode: IMChannel['output_mode'];
  locale?: IMChannel['locale'];
  session_mode?: IMChannel['session_mode'];
  bot_identity: string;
  created_at: string;
  updated_at: string;
}

export function listAllIMChannels() {
  return get<{ data: IMChannelOverview[] }>('/api/v1/im-channels');
}

export function createIMChannel(agentId: string, data: Partial<IMChannel>) {
  return post<{ data: IMChannel }>(`/api/v1/agents/${agentId}/im-channels`, data);
}

export function updateIMChannel(id: string, data: Partial<IMChannel>) {
  return put<{ data: IMChannel }>(`/api/v1/im-channels/${id}`, data);
}

export function deleteIMChannel(id: string) {
  return del<{ success: boolean }>(`/api/v1/im-channels/${id}`);
}

export function toggleIMChannel(id: string) {
  return post<{ data: IMChannel }>(`/api/v1/im-channels/${id}/toggle`);
}

// ===== Önerilen sorular =====

// Önerilen sorular
export interface SuggestedQuestion {
  question: string;
  source: 'faq' | 'document' | 'agent_config' | 'wiki';
  knowledge_base_id?: string;
  knowledge_id?: string;
}

// Ajanın önerilen sorularını al
// Ajanla ilişkili bilgi tabanı kapsamına göre önerilen soruları döndürür; ön uç sohbet panelinde hızlı soru sormak için kullanılır
export function getSuggestedQuestions(
  agentId: string,
  params?: {
    knowledge_base_ids?: string[];
    knowledge_ids?: string[];
    tag_scopes?: Array<{ knowledge_base_id: string; tag_ids: string[] }>;
    limit?: number;
  }
) {
  const query = new URLSearchParams();
  if (params?.knowledge_base_ids?.length) query.set('knowledge_base_ids', params.knowledge_base_ids.join(','));
  if (params?.knowledge_ids?.length) query.set('knowledge_ids', params.knowledge_ids.join(','));
  if (params?.tag_scopes?.length) query.set('tag_scopes', JSON.stringify(params.tag_scopes));
  if (params?.limit) query.set('limit', String(params.limit));
  const qs = query.toString();
  return get<{ data: { questions: SuggestedQuestion[] } }>(`/api/v1/agents/${agentId}/suggested-questions${qs ? '?' + qs : ''}`);
}
// ===== WeChat QR Code Login =====

export interface WeChatQRCodeResult {
  qrcode_url: string;
  qrcode: string;
}

export interface WeChatQRCodeStatus {
  status: 'wait' | 'scaned' | 'confirmed' | 'expired';
  credentials?: {
    bot_token: string;
    ilink_bot_id: string;
    ilink_user_id: string;
  };
  baseurl?: string;
}

export function getWeChatQRCode() {
  return post<{ data: WeChatQRCodeResult }>('/api/v1/wechat/qrcode');
}

export function pollWeChatQRCodeStatus(qrcode: string) {
  return post<{ data: WeChatQRCodeStatus }>('/api/v1/wechat/qrcode/status', { qrcode });
}
