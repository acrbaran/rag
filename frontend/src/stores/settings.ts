import { defineStore } from "pinia";
import { nextTick } from "vue";
import { BUILTIN_QUICK_ANSWER_ID, BUILTIN_SMART_REASONING_ID } from "@/api/agent";
import { getApiBaseUrl } from "@/utils/api-base";
import { isAgentStreamAgentId } from "@/utils/agent-mode";
import { loadAndReconcileSettings } from "@/stores/settingsStorage";
import { isReasoningLevel, type ReasoningLevel } from "@/utils/reasoningEffort";

// Ayar API'sini tanımlar
interface Settings {
  endpoint: string;
  apiKey: string;
  knowledgeBaseId: string;
  isAgentEnabled: boolean;
  agentConfig: AgentConfig;
  selectedKnowledgeBases: string[];  // Şu anda seçili bilgi bankası kimlikleri listesi
  selectedFiles: string[]; // Şu anda seçili dosya kimlikleri listesi
  selectedFileKbMap: Record<string, string>; // Dosya kimliği -> bilgi bankası kimliği; yenilemeden sonra paylaşılan bilgi bankası dosyalarını kb_id ile çekmek için
  selectedTags: Array<{ id: string; name: string; kbId: string; kbName?: string }>;
  selectedMCPServices: string[];
  selectedSkills: string[];
  selectedTools?: string[];
  modelConfig: ModelConfig;  // Model yapılandırması
  localBrowserEnabled: boolean; // Explicit source preference; composer activates it only while the extension is online
  webSearchEnabled: boolean;  // Ağ aramasının etkin olup olmadığı
  conversationModels: ConversationModels;
  selectedAgentId: string;  // Şu anda seçili akıllı ajan kimliği
  selectedAgentSourceTenantId: string | null;  // Paylaşılan akıllı ajan kullanıldığında kaynak alan kimliği (arka uçta model/KB/MCP çözümlemesi için)
}

// Agent yapılandırma API'si
interface AgentConfig {
  maxIterations: number;
  temperature: number;
  allowedTools: string[];
  system_prompt?: string;  // Unified system prompt (uses {{web_search_status}} placeholder)
}

interface ConversationModels {
  summaryModelId: string;
  rerankModelId: string;
  selectedChatModelId: string;  // Kullanıcının şu anda seçtiği sohbet modeli kimliği
}

// Tek model öğesi API'si
interface ModelItem {
  id: string;  // Benzersiz kimlik
  name: string;  // Görünen ad
  source: 'remote';  // Model kaynağı
  modelName: string;  // Model tanımlayıcısı
  baseUrl?: string;  // Uzak API URL'si
  apiKey?: string;  // Uzak API anahtarı
  dimension?: number;  // Yalnızca Embedding için: vektör boyutu
  interfaceType?: 'openai';  // Yalnızca VLLM için: arayüz türü
  isDefault?: boolean;  // Varsayılan model olup olmadığı
}

// Model yapılandırma API'si - birden çok modeli destekler
interface ModelConfig {
  chatModels: ModelItem[];
  embeddingModels: ModelItem[];
  rerankModels: ModelItem[];
  vllmModels: ModelItem[];  // VLLM görsel modeli
}

// Varsayılan ayarlar
const defaultSettings: Settings = {
  endpoint: getApiBaseUrl(),
  apiKey: "",
  knowledgeBaseId: "",
  isAgentEnabled: false,
  agentConfig: {
    maxIterations: 5,
    temperature: 0.7,
    allowedTools: [],  // Varsayılan olarak boş; API üzerinden arka uçtan yüklenmesi gerekir
    system_prompt: "",
  },
  selectedKnowledgeBases: [],  // Varsayılan olarak boş dizi
  selectedFiles: [], // Varsayılan olarak boş dizi
  selectedFileKbMap: {},  // Dosya ID'si -> bilgi tabanı ID'si
  selectedTags: [],
  selectedMCPServices: [],
  selectedSkills: [],
  modelConfig: {
    chatModels: [],
    embeddingModels: [],
    rerankModels: [],
    vllmModels: []
  },
  localBrowserEnabled: false,
  webSearchEnabled: false,  // Ağ araması varsayılan olarak kapalı
  conversationModels: {
    summaryModelId: "",
    rerankModelId: "",
    selectedChatModelId: "",  // Kullanıcının şu anda seçtiği sohbet modeli ID'si
  },
  selectedAgentId: BUILTIN_QUICK_ANSWER_ID,  // Hızlı soru-cevap modu varsayılan olarak seçili
  selectedAgentSourceTenantId: null as string | null,  // Paylaşılan ajan kaynak alanı ID'si
};

export const useSettingsStore = defineStore("settings", {
  state: () => ({
    // Ayarları yerel depolamadan yükler; yoksa varsayılan ayarları kullanır
    settings: loadAndReconcileSettings(defaultSettings),
    // Oturuma girerken "genel varsayılan"ın anlık görüntüsünü al; oturumdan çıkarken geri yükle. Kalıcı olmayan alanlar:
    // Sayfayı yenilemek, "oturuma girme" akışını yeniden çalıştırmaya eşdeğerdir; anlık görüntü doğal olarak yeniden alınır.
    _defaultsSnapshot: null as Settings | null,
    /** Agent değişikliğinde watch işleminin KB seçimini ezmesini önlemek için giriş alanı session.last_request_state üzerinden geri yükleniyor*/
    _isApplyingSessionState: false,
    // Session-only preference: never written into global settings/localStorage.
    reasoningEffortOverride: '' as ReasoningLevel | '',
  }),

  getters: {
    // Agent etkin mi
    isAgentEnabled: (state) => state.settings.isAgentEnabled || false,

    // Mevcut seçimin yerleşik hızlı soru-cevap olup olmadığı (`selectedAgentId` önceliklidir; `isAgentEnabled` ile sapmayı önler)
    isQuickAnswerMode: (state) =>
      (state.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID) === BUILTIN_QUICK_ANSWER_ID,

    // Agent akış hattının kullanılıp kullanılmayacağı (akıllı çıkarım / özel Agent); hızlı soru-cevap RAG akış hattını kullanır
    isAgentStreamMode: (state) =>
      isAgentStreamAgentId(
        state.settings.selectedAgentId,
        state.settings.isAgentEnabled || false,
      ),
    
    // Agent hazır mı (yapılandırma tamam)
    // Şunlar sağlanmalıdır: 1) izin verilen araçlar yapılandırılmış olmalı 2) konuşma modeli ayarlanmış olmalı 3) yeniden sıralama modeli ayarlanmış olmalı
    isAgentReady: (state) => {
      const config = state.settings.agentConfig || defaultSettings.agentConfig
      const models = state.settings.conversationModels || defaultSettings.conversationModels
      return Boolean(
        config.allowedTools && config.allowedTools.length > 0 &&
        models.summaryModelId && models.summaryModelId.trim() !== '' &&
        models.rerankModelId && models.rerankModelId.trim() !== ''
      )
    },
    
    // Normal mod (hızlı yanıt) hazır mı
    // Şunlar sağlanmalıdır: 1) konuşma modeli ayarlanmış olmalı 2) yeniden sıralama modeli ayarlanmış olmalı
    isNormalModeReady: (state) => {
      const models = state.settings.conversationModels || defaultSettings.conversationModels
      return Boolean(
        models.summaryModelId && models.summaryModelId.trim() !== '' &&
        models.rerankModelId && models.rerankModelId.trim() !== ''
      )
    },
    
    // Agent yapılandırmasını al
    agentConfig: (state) => state.settings.agentConfig || defaultSettings.agentConfig,

    conversationModels: (state) => state.settings.conversationModels || defaultSettings.conversationModels,
    
    // Model yapılandırmasını al
    modelConfig: (state) => state.settings.modelConfig || defaultSettings.modelConfig,
    
    // Bu sorgu turunun kaynağı
    isLocalBrowserEnabled: (state) => state.settings.localBrowserEnabled === true,
    isWebSearchEnabled: (state) => state.settings.webSearchEnabled || false,
    
    // Şu anda seçili akıllı ajan ID'si
    selectedAgentId: (state) => state.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID,
    // Paylaşılan akıllı ajanın kaynak alanı ID'si (isteğe bağlı)
    selectedAgentSourceTenantId: (state) => state.settings.selectedAgentSourceTenantId ?? null,
  },

  actions: {
    // Ayarları kaydet
    saveSettings(settings: Settings) {
      this.settings = { ...settings };
      // localStorage'a kaydet
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    // Ayarları al
    getSettings(): Settings {
      return this.settings;
    },

    // API uç noktasını al
    getEndpoint(): string {
      return this.settings.endpoint || defaultSettings.endpoint;
    },

    // API Key'i al
    getApiKey(): string {
      return this.settings.apiKey;
    },

    // Bilgi tabanı ID'sini al
    getKnowledgeBaseId(): string {
      return this.settings.knowledgeBaseId;
    },
    
    // Agent'i etkinleştir/devre dışı bırak
    toggleAgent(enabled: boolean) {
      this.settings.isAgentEnabled = enabled;
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Agent yapılandırmasını güncelle
    updateAgentConfig(config: Partial<AgentConfig>) {
      this.settings.agentConfig = { ...this.settings.agentConfig, ...config };
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    updateConversationModels(models: Partial<ConversationModels>) {
      const current = this.settings.conversationModels || defaultSettings.conversationModels;
      this.settings.conversationModels = { ...current, ...models };
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Model yapılandırmasını güncelle
    updateModelConfig(config: Partial<ModelConfig>) {
      this.settings.modelConfig = { ...this.settings.modelConfig, ...config };
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Model ekle
    addModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', model: ModelItem) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      // Varsayılan olarak ayarlanırsa, diğer modellerin varsayılan durumunu kaldır
      if (model.isDefault) {
        models.forEach(m => m.isDefault = false);
      }
      // İlk modelse otomatik olarak varsayılan yap
      if (models.length === 0) {
        model.isDefault = true;
      }
      models.push(model);
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Modeli güncelle
    updateModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string, updates: Partial<ModelItem>) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      const index = models.findIndex(m => m.id === modelId);
      if (index !== -1) {
        // Varsayılan olarak ayarlanacaksa, diğer modellerin varsayılan durumunu kaldır
        if (updates.isDefault) {
          models.forEach(m => m.isDefault = false);
        }
        models[index] = { ...models[index], ...updates };
        this.settings.modelConfig[key] = models as any;
        localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
      }
    },
    
    // Modeli sil
    deleteModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string) {
      const key = `${type}Models` as keyof ModelConfig;
      let models = [...this.settings.modelConfig[key]] as ModelItem[];
      const deletedModel = models.find(m => m.id === modelId);
      models = models.filter(m => m.id !== modelId);
      // Silinen model varsayılan modelse, ilk modeli varsayılan yap
      if (deletedModel?.isDefault && models.length > 0) {
        models[0].isDefault = true;
      }
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Varsayılan modeli ayarla
    setDefaultModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      models.forEach(m => m.isDefault = (m.id === modelId));
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Bilgi tabanlarını seç (tüm listeyi değiştir)
    selectKnowledgeBases(kbIds: string[]) {
      this.settings.selectedKnowledgeBases = kbIds;
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Tek bir bilgi tabanı ekle
    addKnowledgeBase(kbId: string) {
      if (!this.settings.selectedKnowledgeBases.includes(kbId)) {
        this.settings.selectedKnowledgeBases.push(kbId);
        localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
      }
    },
    
    // Tek bir bilgi tabanını kaldır
    removeKnowledgeBase(kbId: string) {
      this.settings.selectedKnowledgeBases = 
        this.settings.selectedKnowledgeBases.filter((id: string) => id !== kbId);
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Bilgi tabanı seçimini temizle
    clearKnowledgeBases() {
      this.settings.selectedKnowledgeBases = [];
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Seçili bilgi tabanları listesini al
    getSelectedKnowledgeBases(): string[] {
      return this.settings.selectedKnowledgeBases || [];
    },
    
    // Yerel tarayıcı ve çevrimiçi arama bağımsız olarak seçilebilir.
    toggleLocalBrowser(enabled: boolean) {
      this.settings.localBrowserEnabled = enabled;
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    toggleWebSearch(enabled: boolean) {
      this.settings.webSearchEnabled = enabled;
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    // File selection actions
    addFile(fileId: string) {
      if (!this.settings.selectedFiles) this.settings.selectedFiles = [];
      if (!this.settings.selectedFiles.includes(fileId)) {
        this.settings.selectedFiles.push(fileId);
        localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
      }
    },

    removeFile(fileId: string) {
      if (!this.settings.selectedFiles) return;
      this.settings.selectedFiles = this.settings.selectedFiles.filter((id: string) => id !== fileId);
      if (this.settings.selectedFileKbMap) delete this.settings.selectedFileKbMap[fileId];
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    clearFiles() {
      this.settings.selectedFiles = [];
      this.settings.selectedFileKbMap = {};
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    addTag(tag: { id: string; name: string; kbId: string; kbName?: string }) {
      if (!this.settings.selectedTags) this.settings.selectedTags = [];
      if (!this.settings.selectedTags.some(t => t.id === tag.id && t.kbId === tag.kbId)) {
        this.settings.selectedTags.push(tag);
        localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
      }
    },

    removeTag(tagId: string, kbId?: string) {
      if (!this.settings.selectedTags) return;
      this.settings.selectedTags = this.settings.selectedTags.filter(t => !(t.id === tagId && (!kbId || t.kbId === kbId)));
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    clearTags() {
      this.settings.selectedTags = [];
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    addMCPService(serviceId: string) {
      if (!this.settings.selectedMCPServices) this.settings.selectedMCPServices = [];
      if (!this.settings.selectedMCPServices.includes(serviceId)) {
        this.settings.selectedMCPServices.push(serviceId);
        localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
      }
    },

    removeMCPService(serviceId: string) {
      if (!this.settings.selectedMCPServices) return;
      this.settings.selectedMCPServices = this.settings.selectedMCPServices.filter(id => id !== serviceId);
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    addSkill(skillName: string) {
      if (!this.settings.selectedSkills) this.settings.selectedSkills = [];
      if (!this.settings.selectedSkills.includes(skillName)) {
        this.settings.selectedSkills.push(skillName);
        localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
      }
    },

    removeSkill(skillName: string) {
      if (!this.settings.selectedSkills) return;
      this.settings.selectedSkills = this.settings.selectedSkills.filter(name => name !== skillName);
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    setFileKbMap(updates: Record<string, string>) {
      if (!this.settings.selectedFileKbMap) this.settings.selectedFileKbMap = {};
      Object.assign(this.settings.selectedFileKbMap, updates);
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },

    removeFileKbId(fileId: string) {
      if (this.settings.selectedFileKbMap) delete this.settings.selectedFileKbMap[fileId];
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    getSelectedFiles(): string[] {
      return this.settings.selectedFiles || [];
    },

    /**
     * Scope for suggested-questions API (KB / file / tag @mentions).
     * Leave `limit` undefined to let the backend apply the agent's configured
     * starter count; pass a number only to request a specific count.
     */
    getSuggestedQuestionsParams(limit?: number) {
      const selectedKBs = this.getSelectedKnowledgeBases();
      const selectedFiles = this.getSelectedFiles();
      const tags = this.settings.selectedTags || [];
      const tagScopes = Object.entries(tags.reduce<Record<string, string[]>>((scopes, tag) => {
        if (!tag.id || !tag.kbId) return scopes;
        (scopes[tag.kbId] ||= []).push(tag.id);
        return scopes;
      }, {})).map(([knowledge_base_id, ids]) => ({
        knowledge_base_id,
        tag_ids: [...new Set(ids)],
      }));
      return {
        // A tag's parent KB is only an ownership hint, not an explicit whole-KB
        // selection. Keep it in tag_scopes so the backend cannot widen a tag to
        // every document in that KB.
        knowledge_base_ids: selectedKBs.length > 0 ? selectedKBs : undefined,
        knowledge_ids: selectedFiles.length > 0 ? selectedFiles : undefined,
        tag_scopes: tagScopes.length > 0 ? tagScopes : undefined,
        limit,
      };
    },
    
    // Aracıyı seç (`sourceTenantId` yalnızca paylaşılan aracı kullanılırken iletilir)
    selectAgent(agentId: string, sourceTenantId?: string | null) {
      this.reasoningEffortOverride = '';
      this.settings.selectedAgentId = agentId;
      this.settings.selectedAgentSourceTenantId = (sourceTenantId != null && sourceTenantId !== "") ? sourceTenantId : null;
      // Aracı yapılandırması yalnızca ağda arama yeteneğine sahip olup olmadığını belirler; kullanıcının bu turda kullanıp kullanmayacağına karar vermez.
      // Aracı her seçildiğinde varsayılan olarak kapalıdır; sonrasında yalnızca kullanıcı giriş kutusundan etkinleştirebilir.
      this.settings.webSearchEnabled = false;
      this.settings.localBrowserEnabled = false;
      // Aracı türüne göre Agent modunu otomatik olarak değiştir
      if (agentId === BUILTIN_QUICK_ANSWER_ID) {
        this.settings.isAgentEnabled = false;
      } else if (agentId === BUILTIN_SMART_REASONING_ID) {
        this.settings.isAgentEnabled = true;
      }
      // Özel aracılar, yapılandırmalarına göre belirlenmelidir
      
      // Aracı değiştirirken bilgi tabanı ve dosya seçim durumunu sıfırla
      // Farklı aracıların ilişkili bilgi tabanları farklı olduğundan, kullanıcının önceki seçimini temizlemek gerekir
      this.settings.selectedKnowledgeBases = [];
      this.settings.selectedFiles = [];
      this.settings.selectedFileKbMap = {};
      this.settings.selectedTags = [];
      this.settings.selectedMCPServices = [];
      this.settings.selectedSkills = [];
      localStorage.setItem("Rethra_settings", JSON.stringify(this.settings));
    },
    
    // Seçili aracı ID'sini al
    getSelectedAgentId(): string {
      return this.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID;
    },

    // —— Oturum düzeyinde giriş durumu geri yükleme —— //
    //
    // Giriş çubuğundaki agent / model / KB / çevrimiçi erişim / MCP vb. seçimler bu store tarafından tutulur ve oturumlar arasında paylaşılır.
    // Ancak kullanıcının beklentisi şudur: Eski bir oturum açıldığında, isteğin gönderildiği andaki durum kümesini görebilmelidir.
    // Uygulama stratejisi: Oturuma girerken "mevcut genel varsayılan" değerini kalıcı olmayan bir `_defaultsSnapshot` içinde geçici olarak sakla
    // Alanlara yazın, ardından store'u session.last_request_state ile geçersiz kılın; oturumdan çıkarken anlık görüntüden geri yükleyin.
    // Anlık görüntü localStorage'a yazılmaz, çünkü yalnızca "eski bir oturumdayken" bu rota süresi boyunca anlamlıdır;
    // Sayfayı yenilemek "oturuma yeniden girmek" anlamına gelir → anlık görüntüyü yeniden alın + geçersiz kılın; kullanıcının genel varsayılanları kaybolmaz.

    // Mevcut settings'i "oturumdan çıktıktan sonra geri yüklenecek varsayılan" olarak kaydedin.
    // Anlık görüntü zaten varsa üzerine yazmayın; oturumlar arası geçişte (B→B') geri yüklenmiş store'un varsayılan sanılmasını önler.
    snapshotAsDefaultsIfNeeded() {
      if (this._defaultsSnapshot) return;
      this._defaultsSnapshot = JSON.parse(JSON.stringify(this.settings));
    },

    // Varsayılanları geri yükleyin (anlık görüntü varsa); oturumdan çıkarken veya oturumlar arası geçişte kullanılır.
    restoreDefaultsIfSnapshotted() {
      this.reasoningEffortOverride = '';
      if (!this._defaultsSnapshot) return;
      this.settings = this._defaultsSnapshot;
      this._defaultsSnapshot = null;
      // localStorage'a yazmayın: varsayılan değerler anlık görüntüden önce zaten localStorage'a yazılmıştır, burada geri yüklenen
      // de localStorage'daki mevcut değerlerdir; yeniden yazmak yalnızca anlamsız IO ekler.
    },

    // Yeni oturumun ilk gönderimi createChat'in giriş durumunu kullanır; asenkron dönen boş/eski kayıtlar tarafından geçersiz kılınmamalıdır.
    // preserveDraft, session ayrıntıları istenmeden önce yakalanmalıdır; yanıt döndüğünde okunmamalıdır.
    hydrateSessionInputState(state: SessionLastRequestStatePayload | null | undefined, preserveDraft = false) {
      if (!state || preserveDraft) return;
      this.snapshotAsDefaultsIfNeeded();
      this.applyLastRequestState(state);
    },

    // Giriş çubuğuyla ilgili alanları session.last_request_state temelinde geçersiz kılın.
    // Yalnızca bu kaydın alanlarına dokunun; store içindeki diğer ilgisiz alanları (ör. model listesi) **temizlemeyin**.
    // Genel alanlar eksik olduğunda "mümkün olduğunca geri yükleme" yapın; bilgi tabanı kapsamı istisnadır, eksiklik bu oturumda bilgi tabanı seçilmediğini gösterir.
    applyLastRequestState(state: SessionLastRequestStatePayload | null | undefined) {
      if (!state) return;
      this._isApplyingSessionState = true;
      try {
        this.reasoningEffortOverride = isReasoningLevel(state.reasoning_effort) ? state.reasoning_effort : '';
        if (typeof state.agent_enabled === "boolean") {
          this.settings.isAgentEnabled = state.agent_enabled;
        }
        if (typeof state.agent_id === "string" && state.agent_id) {
          this.settings.selectedAgentId = state.agent_id;
          // Son kaydın kendi agent'ı mı yoksa paylaşılan agent mı olduğu şu anda belirlenemiyor; sunucu sourceTenantId değerini ayrım yapmadan geri iletiyor.
          // selectAgent()'ten farklı olarak burada KB/dosya seçimini **sıfırlamayın** — çünkü hemen ardından
          // state içindeki KB/dosya ile geçersiz kılacağız; önce temizleyip sonra yazmaya gerek yok.
        }
        if (state.model_id !== undefined) {
          const current = this.settings.conversationModels || defaultSettings.conversationModels;
          this.settings.conversationModels = { ...current, selectedChatModelId: state.model_id || "" };
        }
        // Arka uç boş listeleri omitempty ile atlar; bu nedenle eksik değer "bu oturumda KB kapsamı yok" anlamına gelir,
        // "önceki oturumun seçimini koru" anlamına gelmez. Aksi halde süresi geçmiş @KB sunucuya gönderilmeye devam eder.
        this.settings.selectedKnowledgeBases = Array.isArray(state.knowledge_base_ids)
          ? [...state.knowledge_base_ids]
          : [];
        if (Array.isArray(state.knowledge_ids)) {
          this.settings.selectedFiles = [...state.knowledge_ids];
          // selectedFileKbMap şu anda yeniden oluşturulamaz (state içinde KB sahipliği saklanmıyor); gerektiğinde istemcinin
          // lazy olarak çekmesine bırakın. Kullanıcının az önce eklediği dosya eşlemelerinin yanlışlıkla silinmesini önlemek için store'un mevcut değerini koruyun.
        }
        if (Array.isArray(state.mentioned_items)) {
          const fromMentions = state.mentioned_items
            .filter(item => item.type === "tag" && item.id && item.kb_id)
            .map(item => ({ id: item.id, name: item.name || item.id, kbId: item.kb_id!, kbName: item.kb_name }));
          const covered = new Set(fromMentions.map(t => t.id));
          const orphanTagIds = (state.tag_ids || []).filter(id => id && !covered.has(id));
          if (orphanTagIds.length > 0 && Array.isArray(state.knowledge_base_ids) && state.knowledge_base_ids.length === 1) {
            const kbId = state.knowledge_base_ids[0];
            orphanTagIds.forEach(id => {
              fromMentions.push({ id, name: id, kbId, kbName: undefined });
            });
          }
          this.settings.selectedTags = fromMentions;
        } else if (Array.isArray(state.tag_ids)) {
          const existing = this.settings.selectedTags || [];
          this.settings.selectedTags = existing.filter(tag => state.tag_ids?.includes(tag.id));
        }
        if (Array.isArray(state.mcp_service_ids)) {
          this.settings.selectedMCPServices = [...state.mcp_service_ids];
        } else if (Array.isArray(state.mentioned_items)) {
          this.settings.selectedMCPServices = state.mentioned_items
            .filter(item => item.type === "mcp" && item.id)
            .map(item => item.id);
        }
        if (Array.isArray(state.skill_names)) {
          this.settings.selectedSkills = [...state.skill_names];
        } else if (Array.isArray(state.mentioned_items)) {
          this.settings.selectedSkills = state.mentioned_items
            .filter(item => item.type === "skill" && item.id)
            .map(item => item.skill_name || item.id);
        }
        this.settings.localBrowserEnabled = state.local_browser_enabled === true;
        if (typeof state.web_search_enabled === "boolean") {
          this.settings.webSearchEnabled = state.web_search_enabled;
        }
      } finally {
        // Sıfırlama bir sonraki flush sonrasına ertelenmelidir: selectedAgentId watcher'ının varsayılanı
        // flush:'pre' olduğundan asenkron çalışır; burada eşzamanlı sıfırlanırsa watcher gerçekten çalıştığında bayrak çoktan
        // false olur, koruma etkisiz kalır ve geri yüklenen KB yine agent yapılandırması tarafından geçersiz kılınır. nextTick'e bırakmak
        // bu durum değişikliğinin tetiklediği watcher'ın bayrak hâlâ true iken çalışmasını sağlar.
        nextTick(() => {
          this._isApplyingSessionState = false;
        });
      }
      // Not: localStorage'a kasten yazılmıyor — eski oturumun durumu "kullanıcı varsayılanlarını" kirletmemelidir.
      // Oturumdan çıkarken restoreDefaultsIfSnapshotted, localStorage'daki tam
      // varsayılan değerleri yeniden this.settings ile senkronize eder.
    },
  },
});

// Arka uç sessions.last_request_state JSON biçimi (SessionLastRequestState ile uyumlu).
// Tüm alanlar isteğe bağlıdır; geçmiş oturumlarda veya yeni oturumların ilk istek öncesinde bu kayıt yoktur.
export interface SessionLastRequestStatePayload {
  reasoning_effort?: string;
  agent_id?: string;
  agent_enabled?: boolean;
  model_id?: string;
  knowledge_base_ids?: string[];
  knowledge_ids?: string[];
  tag_ids?: string[];
  mcp_service_ids?: string[];
  skill_names?: string[];
  mentioned_items?: Array<{
    id: string;
    name?: string;
    type: string;
    kb_id?: string;
    kb_name?: string;
    skill_name?: string;
  }>;
  local_browser_enabled?: boolean;
  web_search_enabled?: boolean;
}
