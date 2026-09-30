<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, onUnmounted, computed, watch, nextTick, h, type PropType } from "vue";
import { storeToRefs } from 'pinia';
import { useRoute, useRouter } from 'vue-router';
import { onBeforeRouteUpdate } from 'vue-router';
import { MessagePlugin } from "tdesign-vue-next";
import type { SendMessageOptions } from '@/utils/questionOrigin';
import { useSettingsStore } from '@/stores/settings';
import { useBrowserConnectionStore } from '@/stores/browserConnection';
import { useUIStore } from '@/stores/ui';
import BrowserIcon from '@/components/icons/BrowserIcon.vue';
import ResourceIcon from '@/components/icons/ResourceIcon.vue';
import { useMenuStore } from '@/stores/menu';
import { listKnowledgeBases, searchKnowledge, batchQueryKnowledge, listKnowledgeTags } from '@/api/knowledge-base';
import { listMCPServices, type MCPService } from '@/api/mcp-service';
import { stopSession } from '@/api/chat';
import type { SteerQueueItem } from '@/api/chat/steer';
import { chatSubmitShortcut } from '@/utils/chatSubmitShortcut';
import { useOrganizationStore } from '@/stores/organization';
import KnowledgeBaseSelector from './KnowledgeBaseSelector.vue';
import MentionSelector from './MentionSelector.vue';
import AgentSelector from './AgentSelector.vue';
import { getCaretCoordinates } from '@/utils/caret';
import { getRootZoom, rectToCssPx, cssViewportSize } from '@/utils/zoom';
import { type ModelConfig } from '@/api/model';
import {
  formatContextWindow,
  isDefaultContextWindow,
  effectiveContextWindow,
} from '@/utils/contextWindow';
import { type CustomAgent, BUILTIN_QUICK_ANSWER_ID, BUILTIN_SMART_REASONING_ID } from '@/api/agent';
import { agentDisplayName } from '@/utils/agent-mode';
import { useChatResourcesStore } from '@/stores/chatResources';
import { useEditorResourcesStore } from '@/stores/editorResources';
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities';
import { hostSkillsOnly, mentionSkillTargetId } from '@/utils/skillTarget';
import { useI18n } from 'vue-i18n';
import AttachmentUpload, { type AttachmentFile } from './AttachmentUpload.vue';
import {
  kbSatisfiesAgentRequirements,
  deriveKbFilterForAgent,
  toolsConsumeFiles,
  type ScopeCapabilities,
} from '@/utils/tool-capabilities';
import {
  isAgentWebSearchEnabled,
  isAgentWebSearchReady,
  isTenantWebSearchReady,
} from '@/utils/agentWebSearch';
import {
  getAgentNotReadyReasonKeys,
  resolveAgentNotReadySection,
  resolveAgentNotReadyHighlight,
  canLocallyConfigureAgent,
  type AgentNotReadyReasonKey,
} from '@/utils/agent-readiness';
import { formatLocalizedList } from '@/utils/format-list';
import { SKILL_ICON, type MentionItem, type MentionItemType, type MentionRequestItem } from '@/types/mention';
import { toolboxLocation } from '@/config/toolbox';
import { supportedLevels, levelLabelKey, levelFromLegacy, clampLevel, type ReasoningLevel } from '@/utils/reasoningEffort';

const route = useRoute();
const router = useRouter();
const settingsStore = useSettingsStore();
const browserConnection = useBrowserConnectionStore();
const uiStore = useUIStore();
const orgStore = useOrganizationStore();
const menuStore = useMenuStore();
const chatResources = useChatResourcesStore();
const editorResources = useEditorResourcesStore();
const deploymentCapabilities = useDeploymentCapabilitiesStore();
const {
  agents,
  disabledOwnAgentIds,
  allModels,
  chatModels: availableModels,
  webSearchProviders,
} = storeToRefs(chatResources);
const { t, locale } = useI18n();

let query = ref("");
const showKbSelector = ref(false);

// Image upload state
const uploadedImages = ref<Array<{ file: File; preview: string }>>([]);
const imageInputRef = ref<HTMLInputElement>();
const imageUploading = ref(false);

// Attachment upload state
const attachmentUploadRef = ref<InstanceType<typeof AttachmentUpload>>();
const uploadedAttachments = ref<AttachmentFile[]>([]);
const CHAT_FILE_DROP_EVENT = 'rethra:chat-file-drop';

const isImageFile = (file: File) => {
  if (file.type.startsWith('image/')) {
    return true;
  }
  const fileName = file.name.toLowerCase();
  return ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp'].some(ext => fileName.endsWith(ext));
};

const handleDroppedFiles = (files: File[]) => {
  if (!files.length) return;

  const imageFiles = files.filter(isImageFile);
  const attachmentFiles = files.filter(file => !isImageFile(file));

  if (imageFiles.length > 0) {
    if (isImageUploadEnabledByAgent.value) {
      addImageFiles(imageFiles);
    } else {
      MessagePlugin.warning(t('input.imageUploadDisabledByAgent'));
    }
  }

  if (attachmentFiles.length > 0) {
    attachmentUploadRef.value?.addFiles(attachmentFiles);
  }
};

const handleChatFileDrop = (event: Event) => {
  const customEvent = event as CustomEvent<{ files?: File[] }>;
  const files = customEvent.detail?.files;
  if (!files || files.length === 0) return;
  handleDroppedFiles(files);
};

const handleImageSelect = (event: Event) => {
  const input = event.target as HTMLInputElement;
  if (!input.files) return;
  addImageFiles(Array.from(input.files));
  input.value = '';
};

const addImageFiles = (files: File[]) => {
  if (!isImageUploadEnabledByAgent.value) return;
  const allowed = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
  const maxSize = 10 * 1024 * 1024;
  for (const file of files) {
    if (uploadedImages.value.length >= 5) {
      MessagePlugin.warning(t('chat.imageTooMany'));
      break;
    }
    if (!allowed.includes(file.type)) {
      MessagePlugin.warning(t('chat.imageTypeSizeError'));
      continue;
    }
    if (file.size > maxSize) {
      MessagePlugin.warning(t('chat.imageTypeSizeError'));
      continue;
    }
    uploadedImages.value.push({ file, preview: URL.createObjectURL(file) });
  }
};

const removeImage = (index: number) => {
  const removed = uploadedImages.value.splice(index, 1);
  if (removed.length > 0) URL.revokeObjectURL(removed[0].preview);
};

const triggerImageUpload = () => {
  imageInputRef.value?.click();
};
const atButtonRef = ref<HTMLElement>();
const showAgentModeSelector = ref(false);
const agentModeButtonRef = ref<HTMLElement>();
const agentModeDropdownStyle = ref<Record<string, string>>({});

const selectedAgentId = computed({
  get: () => settingsStore.selectedAgentId || BUILTIN_QUICK_ANSWER_ID,
  set: (val: string) => settingsStore.selectAgent(val)
});
const selectedAgent = computed(() => {
  // When a shared-agent source tenant is set, resolve from sharedAgents FIRST.
  // Builtin agents (e.g. builtin-smart-reasoning) use the same constant ID across
  // tenants, so falling back to agents.value first would incorrectly return the
  // current tenant's own builtin instead of the shared one.
  const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
  if (sourceTenantId && orgStore.sharedAgents?.length) {
    const shared = orgStore.sharedAgents.find(
      s => s.agent.id === selectedAgentId.value && String(s.source_tenant_id) === sourceTenantId
    );
    if (shared?.agent) return shared.agent as CustomAgent;
  }
  const mine = agents.value.find(a => a.id === selectedAgentId.value);
  if (mine) return mine;
  return {
    id: BUILTIN_QUICK_ANSWER_ID,
    name: t('input.normalMode'),
    is_builtin: true,
    config: { agent_mode: 'quick-answer' as const }
  } as CustomAgent;
});
const selectedSharedAgent = computed(() => {
  const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
  if (!sourceTenantId) return undefined;
  return orgStore.sharedAgents?.find(
    s => s.agent.id === selectedAgentId.value && String(s.source_tenant_id) === sourceTenantId
  );
});

// Özel akıllı aracın (yerleşik olmayan) olup olmadığını belirle
const isCustomAgent = computed(() => {
  const agent = selectedAgent.value;
  return agent && !agent.is_builtin;
});

// Akıllı araç yapılandırması olup olmadığını belirle (yerleşik akıllı araçlar dahil)
const hasAgentConfig = computed(() => {
  const agent = selectedAgent.value;
  // Paylaşılan akıllı aracın config'i kaynak alandan gelir; bu alanın aynı ID'li builtin'i tarafından geçersiz kılınmasını önlemek için doğrudan agent.config kullanılır
  const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
  if (agent?.is_builtin && !sourceTenantId) {
    const builtinAgent = agents.value.find(a => a.id === agent.id);
    return !!builtinAgent?.config;
  }
  return !!agent?.config;
});

// Geçerli akıllı aracın gerçek yapılandırmasını al (yerleşik akıllı araçlar agents listesinden alınır)
const currentAgentConfig = computed(() => {
  const agent = selectedAgent.value;
  // For shared agents, agent.config already carries the source tenant's settings.
  // Re-looking it up by ID in the local agents.value would clobber it with the
  // current tenant's own builtin config (same constant ID for builtins).
  const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
  if (agent?.is_builtin && !sourceTenantId) {
    const builtinAgent = agents.value.find(a => a.id === agent.id);
    return builtinAgent?.config || {};
  }
  return agent?.config || {};
});

// Akıllı aracın önceden yapılandırılmış bilgi tabanı ID'leri
const agentKnowledgeBases = computed(() => {
  if (!hasAgentConfig.value) return [];
  return currentAgentConfig.value?.knowledge_bases || [];
});

// Akıllı aracın bilgi tabanı seçim modu
const agentKBSelectionMode = computed(() => {
  if (!hasAgentConfig.value) return null; // null, ajan tarafından kontrol edilmediğini belirtir
  return currentAgentConfig.value?.kb_selection_mode || 'all';
});

// Paylaşılan ajan altındaki bilgi tabanı listesi (listKnowledgeBases(agent_id)'den gelir); seçili bilgi tabanlarını göstermek ve org rozeti için kullanılır
const sharedAgentKbList = ref<Array<{ id: string; name: string; type?: string; knowledge_count?: number; chunk_count?: number }>>([]);

// Ajan değiştiğinde model ve @ ile kullanılabilir bilgi tabanı listesi yeni ajan yapılandırmasını izler; web araması kullanıcı tarafından etkinleştirilir
// Bilgi tabanı: seçili olanları yeni ajan yapılandırmasının listesiyle değiştirerek seçili ve @ ile kullanılabilir listelerin tutarlı olmasını sağlar (paylaşılan ajan dahil)
watch([selectedAgentId, agentKnowledgeBases, agentKBSelectionMode], ([newAgentId, newAgentKbs, newKbMode], [oldAgentId]) => {
  if (settingsStore._isApplyingSessionState) return;
  if (newAgentId !== oldAgentId && oldAgentId !== undefined) {
    if (newKbMode === 'none') {
      settingsStore.selectKnowledgeBases([]);
    } else {
      settingsStore.selectKnowledgeBases(newAgentKbs && newAgentKbs.length > 0 ? [...newAgentKbs] : []);
    }
    // @ paneli açıksa, yeni ajanın bilgi tabanı kapsamını hemen yansıtmak için @ ile kullanılabilir listeyi yenile
    if (showMention.value) {
      loadMentionItems(mentionQuery.value, true);
    }
    // Clear images when switching to an agent that doesn't support image upload
    if (!isImageUploadEnabledByAgent.value && uploadedImages.value.length > 0) {
      uploadedImages.value.forEach(img => URL.revokeObjectURL(img.preview));
      uploadedImages.value = [];
    }
  }
}, { immediate: true });

// Paylaşılan ajan kullanılırken bu ajanın bilgi tabanı listesini önceden getir; böylece seçili etiketler @ açılmadan da paylaşılan alan rozetini gösterebilir
watch([selectedAgentId, () => settingsStore.selectedAgentSourceTenantId], async ([agentId, sourceTenantId]) => {
  if (sourceTenantId && agentId) {
    try {
      const list = await chatResources.ensureAgentKnowledgeBases(agentId, sourceTenantId);
      sharedAgentKbList.value = list.map((kb: any) => ({
        id: kb.id,
        name: kb.name,
        type: kb.type || 'document',
        knowledge_count: kb.knowledge_count,
        chunk_count: kb.chunk_count
      }));
    } catch {
      sharedAgentKbList.value = [];
    }
  } else {
    sharedAgentKbList.value = [];
  }
}, { immediate: true });

// Ajanın web aramasını etkinleştirip etkinleştirmediği (yalnızca açıkça etkinleştirildiyse desteklenmiş sayılır)
const isWebSearchEnabledByAgent = computed(() => {
  if (!hasAgentConfig.value) return null;
  return isAgentWebSearchEnabled(currentAgentConfig.value);
});

// Web aramasının ajan tarafından devre dışı bırakılıp bırakılmadığı
const isWebSearchDisabledByAgent = computed(() => {
  return hasAgentConfig.value && isWebSearchEnabledByAgent.value !== true;
});

// Bilgi tabanı seçiminin ajan tarafından kilitlenip kilitlenmediği
// 1. Ajan kb_selection_mode = 'none' yapılandırmasına sahipse → bilgi tabanını tamamen devre dışı bırak
// Diğer tüm durumlarda kullanıcı, izin verilen kapsamda @ aracılığıyla bilgi tabanı seçebilir
const isKnowledgeBaseLockedByAgent = computed(() => {
  if (!hasAgentConfig.value) return false;
  // Yalnızca bilgi tabanı devre dışı bırakıldığında kilitle
  return agentKBSelectionMode.value === 'none';
});

// Bilgi tabanının ajan tarafından tamamen devre dışı bırakılıp bırakılmadığı (kb_selection_mode = 'none')
const isKnowledgeBaseDisabledByAgent = computed(() => {
  if (!hasAgentConfig.value) return false;
  return agentKBSelectionMode.value === 'none';
});
const isMentionDisabled = computed(() => {
  if (settingsStore.isAgentStreamMode && isKnowledgeBaseDisabledByAgent.value) {
    return agentMCPSelectionMode.value === 'none' && agentSkillsSelectionMode.value === 'none';
  }
  return isKnowledgeBaseLockedByAgent.value && !settingsStore.isAgentStreamMode;
});

// Ajan yapılandırmasındaki model ID'si
const agentModelId = computed(() => {
  if (!hasAgentConfig.value) return null;
  return currentAgentConfig.value?.model_id || null;
});

// Ajanın desteklediği dosya türleri (boş dizi tüm türlerin desteklendiği anlamına gelir)
const agentSupportedFileTypes = computed(() => {
  if (!hasAgentConfig.value) return [];
  return currentAgentConfig.value?.supported_file_types || [];
});

// Ajan yapılandırmasındaki araç listesi; @ menüsündeki KB uyumluluk filtrelemesini yönlendirir
const agentAllowedTools = computed<string[]>(() => {
  if (!hasAgentConfig.value) return [];
  return currentAgentConfig.value?.allowed_tools || [];
});

type SelectionMode = 'all' | 'selected' | 'none';
const normalizeSelectionMode = (mode?: string): SelectionMode => {
  return mode === 'all' || mode === 'selected' || mode === 'none' ? mode : 'none';
};

const agentMCPSelectionMode = computed<SelectionMode>(() => {
  if (!settingsStore.isAgentStreamMode || !hasAgentConfig.value) return 'none';
  return normalizeSelectionMode(currentAgentConfig.value?.mcp_selection_mode);
});

const agentMCPServiceIds = computed<string[]>(() => {
  if (agentMCPSelectionMode.value !== 'selected') return [];
  return currentAgentConfig.value?.mcp_services || [];
});

const isMCPAllowedByAgent = (service: MCPService) => {
  if (!settingsStore.isAgentStreamMode || !service.enabled) return false;
  const mode = agentMCPSelectionMode.value;
  if (mode === 'none') return false;
  if (mode === 'selected') return agentMCPServiceIds.value.includes(service.id);
  return true;
};

const agentSkillsSelectionMode = computed<SelectionMode>(() => {
  if (!settingsStore.isAgentStreamMode || !hasAgentConfig.value) return 'none';
  return normalizeSelectionMode(currentAgentConfig.value?.skills_selection_mode);
});

const agentSelectedSkills = computed<string[]>(() => {
  if (agentSkillsSelectionMode.value !== 'selected') return [];
  return currentAgentConfig.value?.selected_skills || [];
});

const isSkillAllowedByAgent = (skillName: string) => {
  if (!settingsStore.isAgentStreamMode || !editorResources.skillsAvailable) return false;
  const mode = agentSkillsSelectionMode.value;
  if (mode === 'none') return false;
  if (mode === 'selected') return agentSelectedSkills.value.includes(skillName);
  return true;
};

// Ajan değiştirildiğinde izin verilmeyen MCP / Skill @mention'ları temizle
watch([selectedAgentId, agentMCPSelectionMode, agentSkillsSelectionMode], ([newAgentId], [oldAgentId]) => {
  if (settingsStore._isApplyingSessionState) return;
  if (newAgentId === oldAgentId || oldAgentId === undefined) return;

  const mcpMode = agentMCPSelectionMode.value;
  if (mcpMode === 'none') {
    settingsStore.settings.selectedMCPServices = [];
  } else if (mcpMode === 'selected') {
    const allowed = new Set(agentMCPServiceIds.value);
    settingsStore.settings.selectedMCPServices = (settingsStore.settings.selectedMCPServices || [])
      .filter(id => allowed.has(id));
  }

  const skillsMode = agentSkillsSelectionMode.value;
  if (skillsMode === 'none') {
    settingsStore.settings.selectedSkills = [];
  } else if (skillsMode === 'selected') {
    const allowed = new Set(agentSelectedSkills.value);
    settingsStore.settings.selectedSkills = (settingsStore.settings.selectedSkills || [])
      .filter(name => allowed.has(name));
  }
});

// Yetenek bitlerini KB nesnesinden çıkar; öncelikle backend'in açık capabilities alanını kullan; yoksa indexing_strategy'ye geri dön,
// Son olarak kb.type === 'faq' ile yedekle. shared / owned / agent-scope olmak üzere üç yoldaki KB yanıt yapısı tutarlıdır.
const kbToScopeCaps = (kb: any): Partial<ScopeCapabilities> => {
  if (kb?.capabilities) {
    return {
      vector: !!kb.capabilities.vector,
      keyword: !!kb.capabilities.keyword,
      wiki: !!kb.capabilities.wiki,
      graph: !!kb.capabilities.graph,
      faq: !!kb.capabilities.faq,
    };
  }
  const s = kb?.indexing_strategy;
  return {
    vector: s ? !!s.vector_enabled : false,
    keyword: s ? !!s.keyword_enabled : false,
    wiki: s ? !!s.wiki_enabled : false,
    graph: s ? !!s.graph_enabled : false,
    faq: kb?.type === 'faq',
  };
};

// Mevcut ajanın agent_mode'u (quick-answer / smart-reasoning); bununla
// "Yalnızca RAG modu wiki-only bilgi tabanını @ ile kullanamaz" gibi örtük kısıtları KB filtrelemesine dahil et.
const agentMode = computed(() => {
  if (!hasAgentConfig.value) return '';
  return currentAgentConfig.value?.agent_mode || '';
});

// "all" modu + ajanın araçlarında KB bağımlılığı olduğunda uyumluluk filtrelemesi; 'selected'/'none' burada ikinci kez filtrelenmez
// (selected düzenleyici tarafından yönetilir, none zaten boş listedir).
const isKbCompatibleWithAgent = (kb: any): boolean => {
  if (!hasAgentConfig.value) return true;
  if (agentKBSelectionMode.value !== 'all') return true;
  return kbSatisfiesAgentRequirements(kbToScopeCaps(kb), agentMode.value, agentAllowedTools.value);
};

// Yalnızca kullanıcı arama terimi girmediğinde ve liste ajan araç uyumluluğu nedeniyle boşaltıldığında özel boş durum metnini göster
const mentionEmptyHint = computed(() => {
  if (mentionQuery.value) return '';
  if (!hasAgentConfig.value) return '';
  if (agentKBSelectionMode.value !== 'all') return '';
  // Liste boş && uyumluluk filtresi gerçekten geçerliyse (aksi halde "tümü" elenerek boşaltılmaz)
  if (mentionItems.value.length !== 0) return '';
  const filter = deriveKbFilterForAgent(agentMode.value, agentAllowedTools.value);
  if (!filter) return '';
  return t('mentionDetail.noCompatibleKbForAgent');
});

// Aracının görsel yüklemeyi (çok modlu) etkinleştirip etkinleştirmediği
const isImageUploadEnabledByAgent = computed(() => {
  if (!hasAgentConfig.value) return false;
  return currentAgentConfig.value?.image_upload_enabled === true;
});

// Input araç çubuğu: Yalnızca aracı etkin olduğunda ve arama motoru kullanılabilir olduğunda gösterilir
const showWebSearchButton = computed(() => {
  if (hasAgentConfig.value && settingsStore.selectedAgentSourceTenantId && !isWebSearchReadinessKnown.value) {
    return false;
  }
  if (!hasAgentConfig.value) {
    return isTenantWebSearchReady(webSearchProviders.value);
  }
  return isAgentWebSearchReady(
    currentAgentConfig.value,
    webSearchProviders.value,
    selectedSharedAgent.value?.web_search_ready,
  );
});
const showImageUploadButton = computed(() => isImageUploadEnabledByAgent.value);

// Model seçiminin aracı tarafından kilitlenip kilitlenmediği - Kilitleme mantığı kaldırıldı, kullanıcıların modeller arasında serbestçe geçiş yapmasına izin verilir
const isModelLockedByAgent = computed(() => {
  return false;
});

// Mention related state
const showMention = ref(false);
const mentionQuery = ref("");
const mentionItems = ref<MentionItem[]>([]);
/** Dosya ID -> bilgi tabanı ID (toplu sorgularda kb_id iletmek için, paylaşılan bilgi tabanlarındaki belgeleri destekler) */
const fileIdToKbId = ref<Record<string, string>>({});
const mcpServices = ref<MCPService[]>([]);
const mentionActiveIndex = ref(0);
const mentionStyle = ref<Record<string, string>>({});
const textareaRef = ref<any>(null); // Ref to t-textarea component
const mentionSelectorRef = ref<any>(null);
const mentionStartPos = ref(0);
const isComposing = ref(false);
const isMentionTriggeredByButton = ref(false);
const mentionHasMore = ref(false);
const mentionGroupCounts = ref<Partial<Record<MentionItemType, number>>>({});
// Geçerli @ oturumunun görebildiği KB ID kümesi (araç uyumluluğu filtrelemesi dahil), dosyalar sayfalı yüklenirken yeniden kullanılır,
// append isteğinin uyumsuz KB dosyalarını atlamasını önler.`null` "sınırsız" anlamına gelir (aracı olmayan senaryolar)
const mentionAllowedKbIds = ref<Set<string> | null>(null);
const mentionLoading = ref(false);
const mentionOffset = ref(0);
const MENTION_PAGE_SIZE = 20;

// Ajan paylaşıldığında «paylaşılan alanı» belirten görünen ad (kuruluş adı veya paylaşan); @ listesi ve seçili etiketlerde rozet göstermek için
const sharedAgentOrgName = computed(() => {
  const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
  const agentId = selectedAgentId.value;
  if (!sourceTenantId || !agentId || !orgStore.sharedAgents?.length) return '';
  const shared = orgStore.sharedAgents.find(
    (s: any) => s.agent?.id === agentId && String(s.source_tenant_id) === sourceTenantId
  );
  return shared?.org_name || shared?.shared_by_username || '';
});

const props = defineProps({
  compact: {
    type: Boolean,
    default: false
  },
  autoFocus: {
    type: Boolean,
    default: false
  },
  isReplying: {
    type: Boolean,
    required: false
  },
  composerLocked: {
    type: Boolean,
    default: false
  },
  sessionId: {
    type: String,
    required: false
  },
  assistantMessageId: {
    type: String,
    required: false
  },
  embeddedMode: {
    type: Boolean,
    default: false
  },
  queuedSteers: {
    type: Array as PropType<SteerQueueItem[]>,
    default: () => []
  },
  // Only agent turns have a loop that can take a mid-run message. In a
  // quick-answer session the composer keeps its old behaviour: Stop is the
  // only action while a reply is streaming.
  canSteer: {
    type: Boolean,
    default: false
  }
});

const isAgentEnabled = computed(() => settingsStore.isAgentEnabled);
const isWebSearchEnabled = computed(() => settingsStore.isWebSearchEnabled);
const selectedKbIds = computed(() => settingsStore.settings.selectedKnowledgeBases || []);
const selectedFileIds = computed(() => settingsStore.settings.selectedFiles || []);
const selectedTags = computed(() => settingsStore.settings.selectedTags || []);
const selectedMCPServiceIds = computed(() => settingsStore.settings.selectedMCPServices || []);
const selectedSkillNames = computed(() => settingsStore.settings.selectedSkills || []);

// Hazır bilgi tabanları (alan düzeyi önbellekten)
const knowledgeBases = computed(() => chatResources.validKnowledgeBases);
const fileList = ref<Array<{ id: string; name: string }>>([]);

// Seçili bilgi tabanları: kendininkiler + kuruluş tarafından paylaşılanlar + paylaşılan aracı altındakiler (seçili listeyi ve org rozetini göstermek için)
const selectedKbs = computed(() => {
  const own = knowledgeBases.value.filter(kb => selectedKbIds.value.includes(kb.id));
  const sharedList = orgStore.sharedKnowledgeBases || [];
  const sharedMapped = sharedList
    .filter((s: any) => s.knowledge_base != null && selectedKbIds.value.includes(s.knowledge_base.id))
    .map((s: any) => ({
      id: s.knowledge_base.id,
      name: s.knowledge_base.name,
      type: s.knowledge_base.type || 'document',
      knowledge_count: s.knowledge_base.knowledge_count,
      chunk_count: s.knowledge_base.chunk_count,
      org_name: s.org_name || ''
    }));
  const ownIds = new Set(own.map(kb => kb.id));
  const sharedOnly = sharedMapped.filter((kb: any) => !ownIds.has(kb.id));
  const sharedOnlyIds = new Set(sharedOnly.map((kb: any) => kb.id));
  // Paylaşılan aracı altındaki bilgi tabanları: sharedAgentKbList içinden seçili listedekileri alır ve paylaşılan alan tanımlayıcısını ekler
  const agentOrg = sharedAgentOrgName.value;
  const sharedFromAgent = (sharedAgentKbList.value || []).filter(kb => selectedKbIds.value.includes(kb.id) && !ownIds.has(kb.id) && !sharedOnlyIds.has(kb.id)).map(kb => ({
    id: kb.id,
    name: kb.name,
    type: kb.type || 'document',
    knowledge_count: kb.knowledge_count,
    chunk_count: kb.chunk_count,
    org_name: agentOrg || ''
  }));
  return [...own, ...sharedOnly, ...sharedFromAgent];
});

const selectedFiles = computed(() => {
  // If we have file details in fileList, use them.
  // Otherwise we might show ID or Loading...
  return selectedFileIds.value.map((id: string) => {
    const found = fileList.value.find(f => f.id === id);
    return found || { id, name: 'Loading...' };
  });
});

const skillMentionItems = computed<MentionItem[]>(() => {
  return selectedSkillNames.value
    .filter((name: string) => isSkillAllowedByAgent(name))
    .map((name: string) => {
    const skill = editorResources.skills.find(s => s.name === name);
    return {
      id: name,
      name: skill?.name || name,
      type: 'skill' as const,
      skillName: name,
      description: skill?.description || '',
    };
  });
});

const toMCPMentionItem = (svc: MCPService): MentionItem => ({
  id: svc.id,
  name: svc.name,
  type: 'mcp',
  description: svc.usage_instructions || svc.description || '',
  toolCount: svc.catalog?.tool_count,
  catalogStale: Boolean(svc.catalog?.stale),
  catalogSynced: Boolean(svc.catalog),
});

const selectedMCPItems = computed<MentionItem[]>(() => {
  return selectedMCPServiceIds.value
    .map((id: string) => mcpServices.value.find(service => service.id === id))
    .filter((svc): svc is MCPService => !!svc && isMCPAllowedByAgent(svc))
    .map(toMCPMentionItem);
});

// Tüm seçili öğeleri birleştir (giriş kutusunda göstermek için)
// Artık aracı yapılandırmasının bilgi tabanları da store içinde, tümü selectedKbs üzerinden alınır
const allSelectedItems = computed(() => {
  // Aracının önceden yapılandırılmış bilgi tabanı ID'lerini al (işaretleme ve sıralama için)
  const agentKbIds = agentKnowledgeBases.value;

  // Tüm seçili bilgi tabanlarını, aracı yapılandırması olup olmadıklarına göre işaretle
  const allKbs = selectedKbs.value.map(kb => ({
    ...kb,
    type: 'kb' as const,
    kbType: kb.type,
    isAgentConfigured: agentKbIds.includes(kb.id)
  }));

  // Kullanıcının seçtiği dosyalar (rozet için fileIdToKbId + paylaşılan liste/paylaşılan aracı aracılığıyla org_name tamamlanır)
  const sharedKbOrgMap: Record<string, string> = {};
  (orgStore.sharedKnowledgeBases || []).forEach((s: any) => {
    if (s.knowledge_base?.id != null && s.org_name) {
      sharedKbOrgMap[String(s.knowledge_base.id)] = s.org_name;
    }
  });
  if (sharedAgentOrgName.value) {
    (sharedAgentKbList.value || []).forEach((kb) => {
      sharedKbOrgMap[String(kb.id)] = sharedAgentOrgName.value;
    });
  }
  const files = selectedFiles.value.map((f: { id: string; name: string }) => {
    const kbId = fileIdToKbId.value[f.id];
    const org_name = kbId ? sharedKbOrgMap[String(kbId)] || '' : '';
    return {
      ...f,
      type: 'file' as const,
      isAgentConfigured: false,
      org_name
    };
  });

  // Aracı yapılandırılmış olanları öne yerleştir
  const agentConfiguredKbs = allKbs.filter(kb => kb.isAgentConfigured);
  const userSelectedKbs = allKbs.filter(kb => !kb.isAgentConfigured);
  const tags = selectedTags.value.map((tag: any) => ({
    id: tag.id,
    name: tag.name,
    type: 'tag' as const,
    kbId: tag.kbId,
    kbName: tag.kbName,
    description: tag.kbName || '',
    isAgentConfigured: false,
  }));

  return [...agentConfiguredKbs, ...userSelectedKbs, ...files, ...tags, ...selectedMCPItems.value, ...skillMentionItems.value];
});

// Seçili öğeleri kaldır (aracı yapılandırılmış öğeler de kaldırılabilir)
const removeSelectedItem = (item: MentionItem) => {
  if (item.type === 'kb') {
    settingsStore.removeKnowledgeBase(item.id);
  } else if (item.type === 'file') {
    settingsStore.removeFile(item.id);
  } else if (item.type === 'tag') {
    settingsStore.removeTag(item.id, item.kbId);
  } else if (item.type === 'mcp') {
    settingsStore.removeMCPService(item.id);
  } else if (item.type === 'skill') {
    settingsStore.removeSkill(item.skillName || item.id);
  }
};

const getMentionIcon = (item: MentionItem) => {
  switch (item.type) {
    case 'file': return 'file';
    case 'tag': return 'tag';
    case 'mcp': return 'tools';
    case 'skill': return SKILL_ICON;
    default: return 'folder';
  }
};

const getMentionChipClass = (item: MentionItem) => {
  if (item.type === 'kb') return item.kbType === 'faq' ? 'mention-chip--faq' : 'mention-chip--kb';
  return `mention-chip--${item.type}`;
};

// store'dan okumak için computed kullan ve setter aracılığıyla store'a geri eşitle
const selectedModelId = computed({
  get: () => settingsStore.conversationModels.selectedChatModelId || '',
  set: (val: string) => settingsStore.updateConversationModels({ selectedChatModelId: val })
});
const modelsLoading = ref(false);
const showModelSelector = ref(false);
const modelButtonRef = ref<HTMLElement>();
const modelDropdownStyle = ref<Record<string, string>>({});

// Gösterilen bilgi tabanı etiketleri (en fazla 2 adet gösterilir)
const displayedKbs = computed(() => selectedKbs.value.slice(0, 2));
const remainingCount = computed(() => Math.max(0, selectedKbs.value.length - 2));

// Bilgi tabanı listesini yükle (kendininkiler + paylaşılanlar, @ bahsetmeleri vb. için)
const loadKnowledgeBases = async (force = false) => {
  try {
    await chatResources.ensureKnowledgeBases(force);
    const validKbs = knowledgeBases.value;

    const validKbIds = new Set(validKbs.map((kb: any) => kb.id));
    const sharedKbIds = new Set(
      (orgStore.sharedKnowledgeBases || []).map((s: any) => s.knowledge_base?.id).filter(Boolean)
    );
    let sharedAgentKbIdSet = new Set<string>();
    const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
    const agentId = settingsStore.selectedAgentId;
    if (sourceTenantId && agentId) {
      try {
        const list = await chatResources.ensureAgentKnowledgeBases(agentId, sourceTenantId, force);
        list.forEach((kb: any) => kb?.id && sharedAgentKbIdSet.add(kb.id));
      } catch {
        sharedAgentKbIdSet = new Set();
      }
    }
    const currentSelectedIds = settingsStore.settings.selectedKnowledgeBases || [];
    const validSelectedIds = currentSelectedIds.filter(
      (id: string) => validKbIds.has(id) || sharedKbIds.has(id) || sharedAgentKbIdSet.has(id)
    );

    if (validSelectedIds.length !== currentSelectedIds.length) {
      settingsStore.selectKnowledgeBases(validSelectedIds);
    }
  } catch (error) {
    console.error('Failed to load knowledge bases:', error);
  }
};

const loadFiles = async () => {
  const ids = selectedFileIds.value;
  if (ids.length === 0) return;

  const missingIds = ids.filter((id: string) => !fileList.value.find(f => f.id === id));
  if (missingIds.length === 0) return;

  try {
    // kb_id ile grupla: paylaşılan bilgi tabanlarındaki belgelerin doğru sorgulanabilmesi için kb_id taşıması gerekir
    const byKbId = new Map<string, string[]>();
    const noKbId: string[] = [];
    missingIds.forEach((id: string) => {
      const kbId = fileIdToKbId.value[id];
      if (kbId) {
        if (!byKbId.has(kbId)) byKbId.set(kbId, []);
        byKbId.get(kbId)!.push(id);
      } else {
        noKbId.push(id);
      }
    });

    const allNewFiles: Array<{ id: string; name: string }> = [];
    const agentIdForBatch = settingsStore.selectedAgentSourceTenantId ? settingsStore.selectedAgentId : undefined;
    const runBatch = async (batchIds: string[], kbId?: string, agentId?: string) => {
      const query = new URLSearchParams();
      batchIds.forEach((id: string) => query.append('ids', id));
      const sourceTenantId = agentId ? settingsStore.selectedAgentSourceTenantId ?? undefined : undefined;
      const res: any = await batchQueryKnowledge(query.toString(), kbId, agentId, sourceTenantId);
      if (res.data && Array.isArray(res.data)) {
        res.data.forEach((f: any) => allNewFiles.push({ id: f.id, name: f.title || f.file_name }));
      }
    };

    for (const [kbId, batchIds] of byKbId) {
      await runBatch(batchIds, kbId);
    }
    if (noKbId.length > 0) {
      await runBatch(noKbId, undefined, agentIdForBatch);
    }
    if (allNewFiles.length > 0) {
      fileList.value = [...fileList.value, ...allNewFiles];
    }
  } catch (e) {
    console.error("Failed to load files", e);
  }
};

// A shared agent @mentions its OWNER's MCP services; this workspace's service
// ids match none of its preset, and the backend drops such a mention outright.
// So the list follows the selected agent and is refetched when it changes.
const currentAgentScope = computed(() => {
  const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
  if (!sourceTenantId || !selectedAgentId.value) return undefined;
  return { agentId: selectedAgentId.value, sourceTenantId };
});

const agentScopeKey = (scope?: { agentId: string; sourceTenantId: string | number }) =>
  scope ? `${scope.sourceTenantId}:${scope.agentId}` : '';

// Guards against a slow response for a previously selected agent overwriting
// the list with another workspace's services after the user switched away.
let mcpServicesRequestKey = '';

const loadMCPServices = async () => {
  const scope = currentAgentScope.value;
  const requestKey = agentScopeKey(scope);
  mcpServicesRequestKey = requestKey;
  try {
    const list = await listMCPServices(scope);
    if (mcpServicesRequestKey !== requestKey) return;
    mcpServices.value = list;
  } catch (error) {
    console.error('Failed to load MCP services:', error);
    if (mcpServicesRequestKey !== requestKey) return;
    mcpServices.value = [];
  }
};

watch(currentAgentScope, () => {
  void loadMCPServices();
});

watch(selectedFileIds, () => {
  loadFiles();
}, { immediate: true });

const isWebSearchConfigured = computed(() => {
  if (hasAgentConfig.value) {
    return isAgentWebSearchReady(
      currentAgentConfig.value,
      webSearchProviders.value,
      selectedSharedAgent.value?.web_search_ready,
    );
  }
  return isTenantWebSearchReady(webSearchProviders.value);
});
const isWebSearchReadinessKnown = computed(
  () => !settingsStore.selectedAgentSourceTenantId || selectedSharedAgent.value !== undefined
);

const browserSourceUnavailableHint = computed(() => {
  if (!browserConnection.enabled) return 'localBrowser.unavailable';
  if (browserConnection.device) return 'localBrowser.reconnectHint';
  return 'localBrowser.settingsHint';
});

const openBrowserConnectionSettings = () => {
  void router.push(toolboxLocation('browserconnection'));
};

const toggleBrowserSource = () => {
  showMention.value = false;
  showModelSelector.value = false;
  showAgentModeSelector.value = false;
  if (browserConnection.knownOffline) {
    openBrowserConnectionSettings();
    return;
  }
  settingsStore.toggleLocalBrowser(!settingsStore.isLocalBrowserEnabled);
};

const loadWebSearchConfig = async (force = false) => {
  try {
    await chatResources.ensureWebSearchProviders(force);

    if (isWebSearchReadinessKnown.value && !isWebSearchConfigured.value && settingsStore.isWebSearchEnabled) {
      settingsStore.toggleWebSearch(false);
    }
  } catch (error) {
    console.error('Failed to load web search config:', error);
    chatResources.invalidate('webSearchProviders');
    if (!settingsStore.selectedAgentSourceTenantId && settingsStore.isWebSearchEnabled) {
      settingsStore.toggleWebSearch(false);
    }
  }
};

// Aracı listesini yükle (benimkiler + paylaşılanlar, seçili durum ve hazır olma kontrolleri için)
const loadAgents = async (force = false) => {
  try {
    await chatResources.ensureAgents(force);
    ensureSelectedAgentNotDisabled();
  } catch (error) {
    console.error('Failed to load agents:', error);
  }
};

// Varsayılan olarak seçilen builtin (builtin-quick-answer) da geçerli alan yöneticisi tarafından devre dışı bırakılmış olabilir.
// Liste yüklendikten sonra bir kez düzeltme yapılır: seçili ajan bu alanda devre dışı bırakılmış bir agent ise (yalnızca «benim/builtin»,
// Paylaşılan aracı kaynak alan tarafından belirlenir, yerel devre dışı bırakma listesi geçerli değildir), Akıllı çıkarım → Hızlı yanıt →
// İlk kullanılabilir olana sıralı geri dönüş geçişi. Hepsi devre dışı olduğunda mevcut seçimi değiştirmeden koru (uç senaryo, UI yine de
// enabledAgents filtrelemesinden sonra boş gösterilir; kullanıcı herhangi birini ajan sayfasından geri etkinleştirir).
const ensureSelectedAgentNotDisabled = () => {
  if (settingsStore.selectedAgentSourceTenantId) return
  const currentId = settingsStore.selectedAgentId || BUILTIN_QUICK_ANSWER_ID
  if (!disabledOwnAgentIds.value.includes(currentId)) return

  const isEnabled = (id: string) =>
    agents.value.some(a => a.id === id) && !disabledOwnAgentIds.value.includes(id)

  let fallback: CustomAgent | undefined
  if (isEnabled(BUILTIN_SMART_REASONING_ID)) {
    fallback = agents.value.find(a => a.id === BUILTIN_SMART_REASONING_ID)
  } else if (isEnabled(BUILTIN_QUICK_ANSWER_ID)) {
    fallback = agents.value.find(a => a.id === BUILTIN_QUICK_ANSWER_ID)
  } else {
    fallback = agents.value.find(a => !disabledOwnAgentIds.value.includes(a.id))
  }
  if (!fallback) return

  settingsStore.selectAgent(fallback.id)
  // selectAgent içinde yalnızca iki builtin sabiti için isAgentEnabled otomatik olarak değiştirilir; özel agent geri dönüşünde
  // mod rozetinin ve sohbet davranışının tutarlı olmasını sağlamak için agent_mode değerine göre bir kez açıkça eşitlemek gerekir.
  if (fallback.id !== BUILTIN_QUICK_ANSWER_ID && fallback.id !== BUILTIN_SMART_REASONING_ID) {
    settingsStore.toggleAgent(fallback.config?.agent_mode === 'smart-reasoning')
  }
}

// Sohbet açılır menüsünde gösterilen 「Benim」 ajanlarım (mevcut alanda devre dışı bırakılanlar hariç)
const enabledAgents = computed(() =>
  agents.value.filter(a => !disabledOwnAgentIds.value.includes(a.id))
);

// LAST_CHAT_MODEL_KEY scopes the per-user "last selected chat model"
// to localStorage. The previous implementation wrote this back to the
// tenant-level KV /tenants/kv/conversation-config — which (a) required
// Admin+ to mutate, so a Viewer/Contributor switching models in the
// chat input got a 403, and (b) silently overwrote the tenant default
// for everyone else. localStorage is per-user-per-browser, which is
// what "remember my last pick" actually wants.
const LAST_CHAT_MODEL_KEY = 'rethra_last_chat_model_id'

const readLastChatModelID = (): string => {
  try {
    return localStorage.getItem(LAST_CHAT_MODEL_KEY) || ''
  } catch {
    return ''
  }
}

const writeLastChatModelID = (id: string) => {
  try {
    if (id) {
      localStorage.setItem(LAST_CHAT_MODEL_KEY, id)
    } else {
      localStorage.removeItem(LAST_CHAT_MODEL_KEY)
    }
  } catch {
    // localStorage may be disabled in incognito mode; ignore.
  }
}

// Initial chat-model selection priority: per-user last pick
// (localStorage) > current store value (e.g. carried over from
// settings page) > first available model. The tenant-level
// conversation-config used to feed summary_model_id/rerank_model_id
// into the dropdown, but those fields were removed: per-user last pick
// belongs in localStorage, agent-level model belongs on the agent.
const initChatModelSelection = () => {
  const lastPick = readLastChatModelID();
  const currentSelectedModel = settingsStore.conversationModels.selectedChatModelId;
  const initialSelection = lastPick || currentSelectedModel || '';
  settingsStore.updateConversationModels({
    summaryModelId: initialSelection,
    selectedChatModelId: initialSelection,
    rerankModelId: '',
  });
  if (!selectedModelId.value) {
    selectedModelId.value = initialSelection;
  }
  ensureModelSelection();
};

const loadChatModels = async (force = false) => {
  if (modelsLoading.value) return;
  modelsLoading.value = true;
  try {
    await chatResources.ensureChatModels(force);
    ensureModelSelection();
  } catch (error) {
    console.error('Failed to load chat models:', error);
    chatResources.invalidate('models');
  } finally {
    modelsLoading.value = false;
  }
};

const ensureModelSelection = () => {
  if (selectedModelId.value) {
    return;
  }
  const lastPick = readLastChatModelID();
  if (lastPick) {
    selectedModelId.value = lastPick;
    return;
  }
  if (availableModels.value.length > 0) {
    selectedModelId.value = availableModels.value[0].id || '';
  }
};

// Ajan kimliği veya verileri hazır olduğunda, sohbet modelini ajanın yapılandırılmış model_id değerine eşitle.
// Düzeltme senaryosu: sayfadan ayrılıp geri dönüldüğünde initChatModelSelection, localStorage içindeki lastPick değerini kullanarak
// paylaşılan ajan bağlı kaynak alanın model_id değerinin üzerine yazar, UI 「Yapılandırılmadı」 gösterir — bu durumda ajan modeli geri yüklenmelidir.
// Ancak kullanıcı bu sayfada modeli elle değiştirdiyse (lastPick ajan varsayılanından farklıysa ve mevcut seçim lastPick ise),
// kullanıcı seçimini koru; creatChat → chat geçişinden sonra model B'nin ajan varsayılanı A'ya geri dönmesini önle.
watch(
  [selectedAgentId, () => settingsStore.selectedAgentSourceTenantId, agentModelId],
  ([, sourceTenantId, newModelId]) => {
    if (!newModelId || newModelId.trim() === '') return;

    const lastPick = readLastChatModelID();
    const isSharedAgent = !!sourceTenantId;
    const agentModelInList = availableModels.value.some(m => m.id === newModelId);

    if (
      lastPick &&
      selectedModelId.value === lastPick &&
      lastPick !== newModelId &&
      (!isSharedAgent || agentModelInList)
    ) {
      return;
    }

    if (newModelId !== selectedModelId.value) {
      selectedModelId.value = newModelId;
    }
  },
  { immediate: true }
);

const handleGoToConversationModels = () => {
  showModelSelector.value = false;
  router.push('/platform/settings');
  setTimeout(() => {
    const event = new CustomEvent('settings-nav', {
      detail: { section: 'models', subsection: 'chat' },
    });
    window.dispatchEvent(event);
  }, 100);
};

const handleModelChange = (value: string | number | Array<string | number> | undefined) => {
  const normalized = Array.isArray(value) ? value[0] : value;
  const val = normalized !== undefined && normalized !== null ? String(normalized) : '';

  if (!val) {
    selectedModelId.value = '';
    return;
  }
  if (val === '__add_model__') {
    selectedModelId.value = readLastChatModelID();
    handleGoToConversationModels();
    return;
  }

  // The chat-level model picker now persists per-user-per-browser via
  // localStorage instead of writing to the tenant-shared KV. This is what
  // "remember my last pick" should always have meant — the previous PUT
  // /tenants/kv/conversation-config required Admin+, so a Viewer or
  // Contributor switching models from the chat input got a 403.
  writeLastChatModelID(val);
  selectedModelId.value = val;
  showModelSelector.value = false;

  settingsStore.updateConversationModels({
    summaryModelId: val,
    selectedChatModelId: val,
    rerankModelId: '',
  });
};

const selectedModel = computed(() => {
  return availableModels.value.find(model => model.id === selectedModelId.value);
});

const reasoningLevels = computed(() => supportedLevels(selectedModel.value?.capabilities));
const agentReasoningLevel = computed(() => clampLevel(
  levelFromLegacy(currentAgentConfig.value?.thinking, currentAgentConfig.value?.reasoning_effort),
  reasoningLevels.value,
));
// Show the effective level while leaving the request unset until the user
// chooses a different level. Selecting the agent's level restores inheritance.
const displayedReasoningLevel = computed(() => settingsStore.reasoningEffortOverride || agentReasoningLevel.value);
const showReasoningSelector = ref(false);
const selectReasoningLevel = (level: ReasoningLevel) => {
  settingsStore.reasoningEffortOverride = level === agentReasoningLevel.value ? '' : level;
  showReasoningSelector.value = false;
};
watch([selectedModel, reasoningLevels, () => settingsStore.reasoningEffortOverride, () => settingsStore._isApplyingSessionState], () => {
  // Wait for model resources during session restoration. Once known, an
  // unsupported override returns to inheritance instead of inventing a level.
  if (!settingsStore._isApplyingSessionState && selectedModel.value && settingsStore.reasoningEffortOverride
    && !reasoningLevels.value.includes(settingsStore.reasoningEffortOverride)) {
    settingsStore.reasoningEffortOverride = '';
  }
}, { immediate: true, flush: 'sync' });

// Model görünen adı: bu alan listesinde varsa adı kullanılır; paylaşılan ajan ise ve model_id değeri bu alan listesinde yoksa “Paylaşılan ajan tarafından yapılandırılan model” gösterilir
const selectedModelDisplayName = computed(() => {
  if (selectedModel.value) return modelDisplayName(selectedModel.value);
  if (!selectedModelId.value) return t('input.notConfigured');
  const isSharedAgent = !!settingsStore.selectedAgentSourceTenantId;
  const modelFromAgent = agentModelId.value && agentModelId.value === selectedModelId.value;
  if (isSharedAgent && modelFromAgent) return t('input.sharedAgentModelLabel');
  return t('input.notConfigured');
});

const modelDisplayName = (model: ModelConfig) => {
  const displayName = model.display_name?.trim();
  return displayName || model.name;
};

const contextWindowTitle = (tokens?: number) => {
  if (isDefaultContextWindow(tokens)) {
    return t('model.editor.contextWindowDefaultHint', { value: formatContextWindow(tokens) });
  }
  return t('model.editor.contextWindowTokens', { count: effectiveContextWindow(tokens) });
};

const selectedModelContextLabel = computed(() => {
  if (!selectedModel.value) return '';
  return formatContextWindow(selectedModel.value.parameters?.context_window);
});

const selectedModelContextIsDefault = computed(() => {
  return isDefaultContextWindow(selectedModel.value?.parameters?.context_window);
});

const selectedModelContextTitle = computed(() => {
  if (!selectedModel.value) return '';
  return contextWindowTitle(selectedModel.value.parameters?.context_window);
});

const updateModelDropdownPosition = () => {
  const anchor = modelButtonRef.value;
  if (!anchor) {
    modelDropdownStyle.value = {
      position: 'fixed',
      top: '50%',
      left: '50%',
      transform: 'translate(-50%, -50%)',
    };
    return;
  }

  // Normalize coordinates to CSS pixels so they are interpreted the same way
  // the browser will render them under the root `zoom` (see utils/zoom.ts).
  const zoom = getRootZoom();
  const rect = rectToCssPx(anchor.getBoundingClientRect(), zoom);
  console.log('[Model Dropdown] Button rect:', {
    top: rect.top,
    bottom: rect.bottom,
    left: rect.left,
    right: rect.right,
    width: rect.width,
    height: rect.height
  });

  const dropdownWidth = 280;
  const offsetY = 8;
  const { width: vw, height: vh } = cssViewportSize(zoom);

  // Tetikleyici öğenin sol kenarına sola hizala
  // Piksel hizalama sorunlarını önlemek için Math.round yerine Math.floor kullan
  let left = Math.floor(rect.left);

  // Sınır işlemesi: görünüm alanının sol ve sağına taşma (16px margin bırak)
  const minLeft = 16;
  const maxLeft = Math.max(16, vw - dropdownWidth - 16);
  left = Math.max(minLeft, Math.min(maxLeft, left));

  // Dikey konumlandırma: düğmeye bitişik, boşluğu önlemek için makul bir yükseklik kullan
  const preferredDropdownHeight = 280; // Tercih edilen yükseklik (kompakt ve yeterli)
  const maxDropdownHeight = 360; // Maksimum yükseklik
  const minDropdownHeight = 200; // Minimum yükseklik
  const topMargin = 20; // Üst boşluk
  const spaceBelow = vh - rect.bottom; // Altta kalan alan
  const spaceAbove = rect.top; // Üstte kalan alan

  console.log('[Model Dropdown] Space check:', {
    spaceBelow,
    spaceAbove,
    windowHeight: vh
  });

  let actualHeight: number;
  let shouldOpenBelow: boolean;

  // Öncelikle alttaki alanı dikkate al
  if (spaceBelow >= minDropdownHeight + offsetY) {
    // Altta yeterli alan varsa aşağı doğru aç
    actualHeight = Math.min(preferredDropdownHeight, spaceBelow - offsetY - 16);
    shouldOpenBelow = true;
    console.log('[Model Dropdown] Position: below button', { actualHeight });
  } else {
    // Yukarı doğru aç; öncelikle preferredHeight kullan, yalnızca gerekirse maxHeight değerine genişlet
    const availableHeight = spaceAbove - offsetY - topMargin;
    if (availableHeight >= preferredDropdownHeight) {
      // Tercih edilen yüksekliği göstermek için yeterli alan var
      actualHeight = preferredDropdownHeight;
    } else {
      // Alan yetersiz, kullanılabilir alanı kullan (ancak minimum yükseklikten küçük olmasın)
      actualHeight = Math.max(minDropdownHeight, availableHeight);
    }
    shouldOpenBelow = false;
    console.log('[Model Dropdown] Position: above button', { actualHeight });
  }

  // Açılır yönüne göre farklı konumlandırma yöntemleri kullan
  if (shouldOpenBelow) {
    // Aşağı açılma: top konumlandırması, sola hizalı
    const top = Math.floor(rect.bottom + offsetY);
    console.log('[Model Dropdown] Opening below, top:', top);
    modelDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      top: `${top}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important'
    };
  } else {
    // Yukarı açılma: bottom konumlandırması, sola hizalı
    const bottom = vh - rect.top + offsetY;
    console.log('[Model Dropdown] Opening above, bottom:', bottom);
    modelDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      bottom: `${bottom}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important'
    };
  }

  console.log('[Model Dropdown] Applied style:', modelDropdownStyle.value);
};

// Mention Logic
let lastMentionQuery = '';
const loadMentionItems = async (q: string, resetIndex = true, append = false) => {
  console.log('[Mention] loadMentionItems called with query:', q, 'append:', append);

  if (!append) {
    mentionOffset.value = 0;
  }

  // Bilgi tabanlarını ajanın kb_selection_mode değerine göre filtrele; paylaşılan ajan seçildiğinde bu alan altındaki bilgi tabanlarını, aksi halde kendi alanı + kendisiyle paylaşılanları kullan
  let kbItems: any[] = [];
  let tagItems: MentionItem[] = [];
  let mcpItems: MentionItem[] = [];
  let skillItems: MentionItem[] = [];
  if (!append) {
    let availableKbs: any[];
    const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
    const agentId = selectedAgentId.value;
    if (sourceTenantId && agentId) {
      // Paylaşılan ajan: agent_id ile bu ajanın yapılandırdığı bilgi tabanı kapsamını getir (arka uç alanı paylaşım ilişkisinden çözümler)
      try {
        const list = await chatResources.ensureAgentKnowledgeBases(agentId, sourceTenantId);
        const orgLabel = sharedAgentOrgName.value || '';
        // capabilities / indexing_strategy değerlerini koru, sonraki filtrelemede kullanılacaklar
        availableKbs = list.map((kb: any) => ({
          id: kb.id,
          name: kb.name,
          type: kb.type || 'document',
          knowledge_count: kb.knowledge_count,
          chunk_count: kb.chunk_count,
          org_name: orgLabel,
          capabilities: kb.capabilities,
          indexing_strategy: kb.indexing_strategy,
        }));
        sharedAgentKbList.value = list.map((kb: any) => ({
          id: kb.id,
          name: kb.name,
          type: kb.type || 'document',
          knowledge_count: kb.knowledge_count,
          chunk_count: kb.chunk_count
        }));
      } catch (e) {
        console.error('[Mention] listKnowledgeBases(agent_id) error:', e);
        availableKbs = [];
        sharedAgentKbList.value = [];
      }
    } else {
      sharedAgentKbList.value = [];
      availableKbs = [...knowledgeBases.value];
      const sharedList = orgStore.sharedKnowledgeBases || [];
      const sharedKbsForMention = sharedList
        .filter((s: any) => s.knowledge_base != null)
        .map((s: any) => ({
          id: s.knowledge_base.id,
          name: s.knowledge_base.name,
          type: s.knowledge_base.type || 'document',
          knowledge_count: s.knowledge_base.knowledge_count,
          chunk_count: s.knowledge_base.chunk_count,
          org_name: s.org_name || '',
          capabilities: s.knowledge_base.capabilities,
          indexing_strategy: s.knowledge_base.indexing_strategy,
        }));
      const ownIds = new Set(availableKbs.map((kb: any) => kb.id));
      sharedKbsForMention.forEach((kb: any) => {
        if (!ownIds.has(kb.id)) {
          availableKbs.push(kb);
          ownIds.add(kb.id);
        }
      });
    }

    if (hasAgentConfig.value) {
      const kbMode = agentKBSelectionMode.value;
      // Paylaşılan ajan yolu: `availableKbs` zaten `listKnowledgeBases({agent_id})` kaynağından gelir,
      // Arka uç kb_selection_mode + allowed_tools ile yetkili filtreleme yaptı; ön uç bunu tekrar yapmaz.
      // Kendi ajanı yolu: own KBs + kullanıcıyla paylaşılan KB'lerin birleştirilmesini kullanır, arka uç ajan bağlamını alamaz,
      // bu nedenle 'selected' yapılandırma kümesine daraltılmalı, 'all' ise araçlardan türetilen yeteneklere göre filtrelenmelidir.
      const isSharedAgent = !!(sourceTenantId && agentId);
      if (kbMode === 'none') {
        availableKbs = [];
      } else if (!isSharedAgent) {
        if (kbMode === 'selected') {
          // 'selected', kullanıcının düzenleyicideki seçimlerine tamamen güvenir; düzenleyici zaten kb_filter ile devre dışı bırakıyor
          // uyumsuz öğeleri burada tekrar filtreleme; kullanıcının açık seçimini yetki aşımıyla silmekten kaçın.
          const configuredKbIds = agentKnowledgeBases.value;
          availableKbs = availableKbs.filter((kb: any) => configuredKbIds.includes(kb.id));
        } else if (kbMode === 'all') {
          // 'all' anlamı "tüm uyumlu KB'ler"dir — araçlardan türetilen yetenek kümesine göre filtrele,
          // wiki-qa için "tümü" seçildiğinde @ ile wiki araçlarının çalıştıramayacağı bir sürü KB'nin gelmesini önle.
          availableKbs = availableKbs.filter((kb: any) => isKbCompatibleWithAgent(kb));
        }
      }
    }

    // Ajan dışı senaryolarda dosya filtrelemesi kısıtlanmaz; ajan senaryolarında dosyaları mevcut availableKbs'nin ID kümesine göre filtrele
    mentionAllowedKbIds.value = hasAgentConfig.value
      ? new Set(availableKbs.map((kb: any) => String(kb.id)))
      : null;

    const kbs = availableKbs.filter((kb: any) =>
      !q || (kb.name && kb.name.toLowerCase().includes(q.toLowerCase()))
    );
    kbItems = await Promise.all(kbs.map(async (kb: any) => {
      const kbType = kb.type || 'document';
      let count = kbType === 'faq' ? Number(kb.chunk_count || 0) : Number(kb.knowledge_count || 0);
      if (!count) {
        const detail = await chatResources.fetchKnowledgeBaseById(kb.id);
        if (detail) {
          count = detail.type === 'faq'
            ? Number(detail.chunk_count || 0)
            : Number(detail.knowledge_count || 0);
        }
      }
      return {
        id: kb.id,
        name: kb.name,
        type: 'kb' as const,
        kbType: kbType === 'faq' ? 'faq' as const : 'document' as const,
        count,
        orgName: kb.org_name || sharedAgentOrgName.value || undefined
      };
    }));
    mentionGroupCounts.value.kb = kbItems.length;

    const tagKeyword = q.trim();
    const tagSources = availableKbs;
    try {
      const tagResults = await Promise.all(tagSources.map(async (kb: any) => {
        const res: any = await listKnowledgeTags(kb.id, { page: 1, page_size: 20, keyword: tagKeyword || undefined });
        const payload = res?.data ?? res;
        const list = Array.isArray(payload?.data) ? payload.data : (Array.isArray(payload) ? payload : []);
        return list.map((tag: any) => ({
          id: tag.id,
          name: tag.name,
          type: 'tag' as const,
          kbId: kb.id,
          kbName: kb.name,
        }));
      }));
      tagItems = tagResults.flat();
      mentionGroupCounts.value.tag = tagItems.length;
    } catch (e) {
      console.error('[Mention] listKnowledgeTags error:', e);
      tagItems = [];
    }

    const mcpMode = agentMCPSelectionMode.value;
    if (mcpMode !== 'none') {
      mcpItems = mcpServices.value
        .filter(service => isMCPAllowedByAgent(service))
        .filter(service => !q || service.name?.toLowerCase().includes(q.toLowerCase()) || (service.usage_instructions || service.description || '').toLowerCase().includes(q.toLowerCase()))
        .map(toMCPMentionItem);
    }

    const skillsMode = agentSkillsSelectionMode.value;
    if (skillsMode !== 'none') {
      // The scope makes a shared agent's skills resolve in its owner's
      // workspace, where they are actually installed. Lite agents store no
      // sandbox config id; their skills live on the host target.
      await deploymentCapabilities.ensureLoaded();
      await editorResources.ensureSkills(
        mentionSkillTargetId(
          hostSkillsOnly(
            deploymentCapabilities.isSupported('settings.sandbox.remote'),
            deploymentCapabilities.isSupported('settings.sandbox.host'),
          ),
          currentAgentConfig.value?.sandbox_config_id,
        ),
        currentAgentScope.value,
      );
      skillItems = editorResources.skills
        .filter(skill => isSkillAllowedByAgent(skill.name))
        .map(skill => ({
          id: skill.name,
          name: skill.name,
          type: 'skill' as const,
          skillName: skill.name,
          description: skill.description || '',
        }))
        .filter(skill => {
          if (!q) return true;
          const keyword = q.toLowerCase();
          return skill.name.toLowerCase().includes(keyword)
            || (skill.description || '').toLowerCase().includes(keyword);
        });
    }
  }

  // Fetch Files from API
  // Dosyaları yalnızca şu iki koşul sağlandığında yükle:
  //   1. Ajan gerçekten bilgi tabanını kullanıyorsa (kb_selection_mode !== 'none');
  //   2. Ajanın etkinleştirdiği araçlardan en az biri @ ile belirtilen dosya ID'sini tüketebiliyor
  //      (örneğin wiki-qa tamamen wiki_* araçlarından oluşuyorsa, kullanıcının @ ile belirttiği dosya hiçbir araca giremez; bu yüzden göstermeye gerek yoktur).
  let fileItems: any[] = [];
  const kbModeAllowsFiles = !hasAgentConfig.value || agentKBSelectionMode.value !== 'none';
  const toolsAllowFiles = !hasAgentConfig.value || toolsConsumeFiles(agentAllowedTools.value);
  const shouldLoadFiles = kbModeAllowsFiles && toolsAllowFiles;

  // Anahtar sözcük boş olduğunda son dosyaları açıkça iste; anahtar sözcük olduğunda eşleşen dosyaları döndür.
  // `recent=true` yalnızca göz atma durumunda kullanılır; diğer arama çağrılarında anahtar sözcük gönderilmediğinde sessizce son listeye düşmesini önler.
  const fileSearchKeyword = q.trim();
  if (shouldLoadFiles) {
    mentionLoading.value = true;
    try {
      const fileTypesParam = agentSupportedFileTypes.value.length > 0 ? agentSupportedFileTypes.value : undefined;
      const sourceTenantId = settingsStore.selectedAgentSourceTenantId;
      const agentId = selectedAgentId.value;
      const searchOptions = {
        ...(sourceTenantId && agentId ? { agent_id: agentId, agent_source_tenant_id: sourceTenantId } : {}),
        recent: !fileSearchKeyword,
      };
      const res: any = await searchKnowledge(
        fileSearchKeyword,
        mentionOffset.value,
        MENTION_PAGE_SIZE,
        fileTypesParam,
        searchOptions
      );
      console.log('[Mention] searchKnowledge response:', res);
      if (res.data && Array.isArray(res.data)) {
        let files = res.data;
        const rawTotal = typeof res.total === 'number' ? res.total : undefined;
        const apiPageSize = res.data.length;
        // Mevcut @ oturumunun uyumlu KB kümesine göre filtrele:
        //   - Ajan dışı senaryo: `mentionAllowedKbIds` null'dır, atla;
        //   - Ajan senaryosu (paylaşılan ajan dahil): 'selected', ID'leri kullanıcının seçtiği KB'lere daraltır,
        //     'all', "uyumlu" KB'ye yakınsar; 'none' buraya hiç gelmez (shouldLoadFiles=false).
        //   Böylece sayfalı append işlemi de aynı koleksiyonu kullanabilir; artık yalnızca 'selected' + paylaşımsız dalı kapsamaz.
        if (mentionAllowedKbIds.value) {
          const allowed = mentionAllowedKbIds.value;
          files = files.filter((f: any) => {
            const kbId = f.knowledge_base_id ?? f.kb_id;
            return kbId != null && allowed.has(String(kbId));
          });
        }
        const sharedKbOrgMap: Record<string, string> = {};
        (orgStore.sharedKnowledgeBases || []).forEach((s: any) => {
          if (s.knowledge_base?.id != null && s.org_name) {
            sharedKbOrgMap[String(s.knowledge_base.id)] = s.org_name;
          }
        });
        const agentOrgLabel = sourceTenantId && agentId ? sharedAgentOrgName.value : '';
        fileItems = files.map((f: any) => {
          const kbId = f.knowledge_base_id ?? f.kb_id;
          const kbIdStr = kbId != null ? String(kbId) : '';
          const fileOrgName = agentOrgLabel || (kbIdStr ? sharedKbOrgMap[kbIdStr] : undefined);
          return {
            id: f.id,
            name: f.title || f.file_name,
            type: 'file' as const,
            kbName: f.knowledge_base_name || '',
            kbId: kbId || undefined,
            orgName: fileOrgName || undefined
          };
        });
        if (!append) {
          const clientFiltered = !!mentionAllowedKbIds.value && fileItems.length < apiPageSize;
          if (!clientFiltered && rawTotal != null) {
            mentionGroupCounts.value.file = rawTotal;
          } else {
            delete mentionGroupCounts.value.file;
          }
        }
      }
      mentionHasMore.value = res.has_more || false;
      mentionOffset.value += fileItems.length;
    } catch (e) {
      console.error('[Mention] searchKnowledge error:', e);
      mentionHasMore.value = false;
    } finally {
      mentionLoading.value = false;
    }
  } else {
    mentionHasMore.value = false;
  }

  if (append) {
    // Append file items to existing list
    mentionItems.value = [...mentionItems.value, ...fileItems];
  } else {
    mentionItems.value = [...kbItems, ...tagItems, ...mcpItems, ...skillItems, ...fileItems];
  }
  console.log('[Mention] Total items:', mentionItems.value.length, { kbItems: kbItems.length, fileItems: fileItems.length, tagItems: tagItems.length, mcpItems: mcpItems.length, skillItems: skillItems.length });

  // Only reset index if query changed or explicitly requested
  if (resetIndex || q !== lastMentionQuery) {
    mentionActiveIndex.value = 0;
  }
  // Ensure index is within bounds
  if (mentionActiveIndex.value >= mentionItems.value.length) {
    mentionActiveIndex.value = Math.max(0, mentionItems.value.length - 1);
  }
  lastMentionQuery = q;
};

const loadMoreMentionItems = () => {
  if (mentionHasMore.value && !mentionLoading.value) {
    loadMentionItems(lastMentionQuery, false, true);
  }
};

const getTextareaEl = () => {
  if (!textareaRef.value) return null;
  // If it's a native element
  if (textareaRef.value instanceof HTMLTextAreaElement) return textareaRef.value;
  // If it's a component wrapper
  const el = textareaRef.value.$el || textareaRef.value;
  if (!el) return null;
  if (el.tagName === 'TEXTAREA') return el as HTMLTextAreaElement;
  return el.querySelector('textarea');
};

const focusInput = async () => {
  await nextTick();
  const textarea = getTextareaEl();
  if (textarea?.isConnected) textarea.focus({ preventScroll: true });
};

const onInput = (val: string | InputEvent) => {
  // Girdi yöntemi birleştirme sırasında ise arama mantığını işleme, compositionend'i bekle.
  if (isComposing.value) return;

  // TDesign t-textarea passes the value directly, not an event
  const inputVal = typeof val === 'string' ? val : query.value;

  const textarea = getTextareaEl();
  if (!textarea) {
    console.warn('[Mention] Could not get textarea element');
    return;
  }

  const cursor = textarea.selectionStart;
  const textBeforeCursor = inputVal.slice(0, cursor);

  console.log('[Mention] onInput called', { inputVal, cursor, textBeforeCursor, showMention: showMention.value });

  if (showMention.value) {
    // Düğme tarafından tetiklenmediyse @ sembolünü kontrol et.
    if (!isMentionTriggeredByButton.value) {
      if (!inputVal || inputVal.length <= mentionStartPos.value || inputVal.charAt(mentionStartPos.value) !== '@') {
        showMention.value = false;
        return;
      }
    }

    // Düğme tarafından tetiklendiyse mentionStartPos imleç konumunu gösterir (yani sanal @ konumunun önünü), bu nedenle aslında sola doğru silmemelidir.
    // Ancak kullanıcı önceki içeriği silip uzunluğu kısalttıysa bunun da işlenmesi gerekir.
    if (cursor < mentionStartPos.value) {
      showMention.value = false;
      return;
    }

    // Get query
    // Düğme tarafından tetiklendiyse mentionStartPos başlangıç konumudur; @ işaretini atlamak için +1 gerekmez.
    const start = isMentionTriggeredByButton.value ? mentionStartPos.value : mentionStartPos.value + 1;
    const q = inputVal.slice(start, cursor);

    if (q.includes(' ')) {
      showMention.value = false;
      return;
    }
    // Only reload if query changed
    if (q !== mentionQuery.value) {
      mentionQuery.value = q;
      loadMentionItems(q, true); // Reset index when query changes
    }
  } else {
    if (textBeforeCursor.endsWith('@')) {
      // Aracı bilgi tabanını devre dışı bıraktıysa @ menüsünü tetikleme.
      if (isMentionDisabled.value) {
        return;
      }

      console.log('[Mention] @ detected, opening menu');
      isMentionTriggeredByButton.value = false;
      mentionStartPos.value = cursor - 1;
      showMention.value = true;
      mentionQuery.value = "";

      const coords = getCaretCoordinates(textarea, cursor);
      // Normalize coordinates to CSS pixels (root <html> may carry `zoom`).
      const zoom = getRootZoom();
      const rect = rectToCssPx(textarea.getBoundingClientRect(), zoom);
      const { width: vw, height: vh } = cssViewportSize(zoom);
      const scrollTop = textarea.scrollTop;
      const menuHeight = 320; // Tahmini maksimum yükseklik

      let left = rect.left + coords.left;
      // Prevent menu from going off-screen horizontally
      if (left + 300 > vw) {
        left = vw - 300 - 10;
      }

      // İmlecin görüntü alanına göre gerçek top konumu (CSS pikselleri)
      const cursorAbsoluteTop = rect.top + coords.top - scrollTop;
      const lineHeight = coords.height; // İmleç yüksekliği

      // Check vertical space below cursor
      const spaceBelow = vh - (cursorAbsoluteTop + lineHeight);

      if (spaceBelow < menuHeight && cursorAbsoluteTop > menuHeight) {
        // Show above cursor (using bottom positioning)
        const bottom = vh - cursorAbsoluteTop;
        mentionStyle.value = {
          left: `${left}px`,
          bottom: `${bottom}px`,
          top: 'auto'
        };
      } else {
        // Show below cursor (using top positioning)
        const top = cursorAbsoluteTop + lineHeight;
        mentionStyle.value = {
          left: `${left}px`,
          top: `${top}px`,
          bottom: 'auto'
        };
      }

      loadMentionItems("");
    }
  }
};

const onCompositionStart = () => {
  isComposing.value = true;
};

const onCompositionEnd = (e: CompositionEvent) => {
  isComposing.value = false;
  // onInput mantığını manuel olarak tetikle
  // Not: compositionend sırasında v-model henüz güncellenmemiş olabilir veya güncellenmiş olsa bile en güncel değeri kullanmamız gerekir.
  // TDesign textarea için nextTick gerekebilir
  nextTick(() => {
    onInput(query.value);
  });
};

const triggerMention = () => {
  // Şu anda bahsedilebilir hiçbir kaynak yoksa seçicinin açılmasına izin verme.
  if (isMentionDisabled.value) {
    const msgKey = isKnowledgeBaseDisabledByAgent.value ? 'input.kbDisabledByAgent' : 'input.kbLockedByAgent';
    MessagePlugin.warning(t(msgKey));
    return;
  }

  const textarea = getTextareaEl();
  if (!textarea) return;

  // Diğer seçicileri kapat
  showAgentModeSelector.value = false;
  showModelSelector.value = false;

  textarea.focus();

  // Menüyü doğrudan göster, @ ekleme.
  showMention.value = true;
  isMentionTriggeredByButton.value = true;
  mentionQuery.value = "";
  mentionStartPos.value = textarea.selectionStart;

  updateMentionButtonPosition();

  loadMentionItems("");
};

// Anchor the button-triggered mention menu directly under the knowledge base icon.
const updateMentionButtonPosition = () => {
  const anchor = atButtonRef.value;
  if (!anchor) return;

  // Normalize coordinates to CSS pixels (root <html> may carry `zoom`).
  const zoom = getRootZoom();
  const rect = rectToCssPx(anchor.getBoundingClientRect(), zoom);
  const { width: vw, height: vh } = cssViewportSize(zoom);
  const gap = 6;
  const margin = 8;
  const menuWidth = 220;
  const preferredHeight = 420;
  const minHeight = 200;

  const left = Math.max(margin, Math.min(rect.left, vw - menuWidth - margin));
  const spaceBelow = vh - rect.bottom - gap - margin;
  const spaceAbove = rect.top - gap - margin;

  if (spaceBelow >= minHeight || spaceBelow >= spaceAbove) {
    mentionStyle.value = {
      left: `${left}px`,
      top: `${rect.bottom + gap}px`,
      bottom: 'auto',
      maxHeight: `${Math.min(preferredHeight, spaceBelow)}px`
    };
  } else {
    mentionStyle.value = {
      left: `${left}px`,
      bottom: `${vh - rect.top + gap}px`,
      top: 'auto',
      maxHeight: `${Math.min(preferredHeight, spaceAbove)}px`
    };
  }
};

const onMentionSelect = (item: any) => {
  if (item.type === 'kb') {
    settingsStore.addKnowledgeBase(item.id);
  } else if (item.type === 'file') {
    settingsStore.addFile(item.id);
    if (item.kbId) {
      fileIdToKbId.value[item.id] = item.kbId;
      settingsStore.setFileKbMap({ [item.id]: item.kbId });
    }
    // Add to local cache immediately
    if (!fileList.value.find(f => f.id === item.id)) {
      fileList.value.push({ id: item.id, name: item.name });
    }
  } else if (item.type === 'tag') {
    if (item.kbId) {
      settingsStore.addTag({ id: item.id, name: item.name, kbId: item.kbId, kbName: item.kbName });
    }
  } else if (item.type === 'mcp') {
    settingsStore.addMCPService(item.id);
  } else if (item.type === 'skill') {
    settingsStore.addSkill(item.skillName || item.id);
  }

  const textarea = getTextareaEl();
  if (textarea) {
    // @ yazarak tetiklendiyse @ işaretini ve sonrasındaki sorgu metnini silmek gerekir.
    if (!isMentionTriggeredByButton.value) {
      const cursor = textarea.selectionStart;
      const textBeforeAt = query.value.slice(0, mentionStartPos.value);
      const textAfterCursor = query.value.slice(cursor);
      query.value = textBeforeAt + textAfterCursor;

      nextTick(() => {
        textarea.selectionStart = textarea.selectionEnd = mentionStartPos.value;
        textarea.focus();
      });
    } else {
      // Düğme aracılığıyla tetiklendiyse ve kullanıcı bir sorgu sözcüğü girdiyse sorgu sözcüğünü silmek gerekir.
      const cursor = textarea.selectionStart;
      if (cursor > mentionStartPos.value) {
        const textBeforeStart = query.value.slice(0, mentionStartPos.value);
        const textAfterCursor = query.value.slice(cursor);
        query.value = textBeforeStart + textAfterCursor;

        nextTick(() => {
          textarea.selectionStart = textarea.selectionEnd = mentionStartPos.value;
          textarea.focus();
        });
      } else {
        // Doğrudan odaklan
        textarea.focus();
      }
    }
  }

  showMention.value = false;
};

const removeFile = (id: string) => {
  settingsStore.removeFile(id);
  delete fileIdToKbId.value[id];
};

const toggleModelSelector = () => {
  showReasoningSelector.value = false;
  // Aracı modeli kilitlediyse seçicinin açılmasına izin verme.
  if (isModelLockedByAgent.value) {
    MessagePlugin.warning(t('input.modelLockedByAgent'));
    return;
  }

  // Karşılıklı dışlama: diğerlerini kapat
  showMention.value = false;
  showAgentModeSelector.value = false;

  showModelSelector.value = !showModelSelector.value;
  if (showModelSelector.value) {
    if (!availableModels.value.length) {
      loadChatModels();
    }
    // Doğruluğu sağlamak için konumu birden çok kez güncelle
    nextTick(() => {
      updateModelDropdownPosition();
      requestAnimationFrame(() => {
        updateModelDropdownPosition();
        setTimeout(() => {
          updateModelDropdownPosition();
        }, 50);
      });
    });
  }
};

const closeModelSelector = () => {
  showModelSelector.value = false;
};

const handleReasoningVisibleChange = (visible: boolean) => {
  if (!visible) return;
  closeModelSelector();
  showMention.value = false;
  showAgentModeSelector.value = false;
};

// Agent modu seçicisini kapat (dışarı tıklama)
const closeAgentModeSelector = () => {
  showAgentModeSelector.value = false;
};

const closeMentionSelector = (e: MouseEvent) => {
  const target = e.target as HTMLElement;
  // Tıklanan yer giriş alanı bölgesiyse Mention listesini kapatma (imleç mantığı tarafından kontrol edilir).
  if (target.closest('.rich-input-container')) {
    return;
  }
  showMention.value = false;
};

// Pencere olay işleyicisi
let resizeHandler: (() => void) | null = null;
let scrollHandler: (() => void) | null = null;

onMounted(() => {
  if (props.autoFocus) void focusInput();
  // Embed kanalı, agent/KB'yi ana bilgisayardan alır; JWT gerektiren platform kaynaklarını çekmeyin
  if (props.embeddedMode) return;

  browserConnection.watchStatus();

  // Paralel olarak çek; platform önceden çekilmişse ve önbellek süresi dolmamışsa doğrudan yeniden kullan
  initChatModelSelection();
  void Promise.all([
    loadKnowledgeBases(),
    loadWebSearchConfig(),
    loadChatModels(),
    loadAgents(),
    loadMCPServices(),
  ]);
  window.addEventListener(CHAT_FILE_DROP_EVENT, handleChatFileDrop as EventListener);

  // Kalıcı depolamadan fileId -> kbId eşlemesini geri yükle; yenilemeden sonra paylaşılan bilgi bankası dosyaları kb_id ile çekilebilir (yalnızca hâlâ seçili olan dosyalar korunur)
  const persisted = settingsStore.settings.selectedFileKbMap;
  const ids = settingsStore.settings.selectedFiles || [];
  if (persisted && typeof persisted === 'object' && ids.length > 0) {
    const next: Record<string, string> = {};
    ids.forEach((id: string) => {
      if (persisted[id]) next[id] = persisted[id];
    });
    fileIdToKbId.value = next;
  }

  // Bilgi bankasının içinden girilirse, bu bilgi bankasını otomatik olarak seç
  const kbId = (route.params as any)?.kbId as string;
  if (kbId && !selectedKbIds.value.includes(kbId)) {
    settingsStore.addKnowledgeBase(kbId);
  }

  const prefill = menuStore.consumePrefillQuery();
  if (prefill) {
    query.value = prefill;
    nextTick(() => {
      const textarea = getTextareaEl();
      if (textarea) textarea.focus();
    });
  }

  // Açılır menüyü dışarı tıklanınca kapatmayı dinle
  document.addEventListener('click', closeAgentModeSelector);
  document.addEventListener('click', closeModelSelector);
  document.addEventListener('click', closeMentionSelector);

  // Pencere boyutu değişikliklerini ve kaydırmayı dinle, konumu yeniden hesapla
  resizeHandler = () => {
    if (showModelSelector.value) {
      updateModelDropdownPosition();
    }
    if (showAgentModeSelector.value) {
      updateAgentModeDropdownPosition();
    }
    if (showMention.value && isMentionTriggeredByButton.value) {
      updateMentionButtonPosition();
    }
  };
  scrollHandler = () => {
    if (showModelSelector.value) {
      updateModelDropdownPosition();
    }
    if (showAgentModeSelector.value) {
      updateAgentModeDropdownPosition();
    }
    if (showMention.value && isMentionTriggeredByButton.value) {
      updateMentionButtonPosition();
    }
  };

  window.addEventListener('resize', resizeHandler, { passive: true });
  window.addEventListener('scroll', scrollHandler, { passive: true, capture: true });
});

onBeforeUnmount(() => {
  // Let TDesign handle blur while its textarea is still attached to the DOM.
  const textarea = getTextareaEl();
  if (textarea?.isConnected && document.activeElement === textarea) textarea.blur();
});

onUnmounted(() => {
  if (!props.embeddedMode) browserConnection.unwatchStatus();
  window.removeEventListener(CHAT_FILE_DROP_EVENT, handleChatFileDrop as EventListener);
  document.removeEventListener('click', closeAgentModeSelector);
  document.removeEventListener('click', closeModelSelector);
  document.removeEventListener('click', closeMentionSelector);
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler);
  }
  if (scrollHandler) {
    window.removeEventListener('scroll', scrollHandler, { capture: true });
  }
});

// Rota değişikliklerini dinle
watch(() => route.params.kbId, (newKbId) => {
  if (newKbId && typeof newKbId === 'string' && !selectedKbIds.value.includes(newKbId)) {
    settingsStore.addKnowledgeBase(newKbId);
  }
});

// Model / web araması listesi, ayarlar sayfası yazma işleminden sonra doğrudan chatResources içine geri yazılır
// （replaceModels / ensureWebSearchProviders(true)）, burada aynı anlık görüntü okunur,
// Artık “ayarlar penceresinin kapanması” ve “ayarlar rotasından ayrılma” olmak üzere iki sinyale dayanarak ayrı ayrı zorla yenileme yapılmaz.
watch(() => uiStore.showSettingsModal, (visible, prevVisible) => {
  if (prevVisible && !visible && !props.embeddedMode) {
    void browserConnection.refresh();
  }
});

watch([selectedKbIds, selectedFileIds], ([kbIds, fileIds]) => {
  if (!kbIds.length && !fileIds.length) {
    closeModelSelector();
  }
}, { deep: true });

const emit = defineEmits<{
  (e: 'send-msg', query: string, modelId: string, mentionedItems: MentionRequestItem[], imageFiles: File[], attachmentFiles: AttachmentFile[], options: SendMessageOptions): void;
  (e: 'stop-generation'): void;
  (e: 'stop-confirmed'): void;
  (e: 'stop-failed'): void;
  // Running input defaults to after; the explicit shortcut/action steers.
  // Empty input while replying shows Stop; typed text also shows Send.
  (e: 'steer-msg', query: string, mentionedItems: MentionRequestItem[], delivery: 'inject' | 'after'): void;
  (e: 'promote-steer', steerId: string): void;
  (e: 'remove-steer', steerId: string): void;
  (e: 'retry-steer', steerId: string): void;
}>();

// options ride along with this one send only: a send that returns early (or
// steers into the running turn) drops them instead of leaving them behind.
const createSession = async (
  val: string,
  delivery: 'inject' | 'after' = 'after',
  options: SendMessageOptions = {},
) => {
  if (props.composerLocked) {
    return;
  }
  if (!val.trim()) {
    MessagePlugin.info(t('input.messages.enterContent'));
    return;
  }
  if (props.isReplying) {
    if (!props.canSteer) {
      // Quick-answer turns have no round boundary to take a message at, and
      // no follow-up handoff on teardown — queueing here would park the
      // message until it expired. Stop first.
      MessagePlugin.info(t('input.messages.replying'));
      return;
    }
    // Queue the selected delivery mode until its next safe boundary.
    // Attachments are intentionally not allowed on the steer path — the
    // running turn already resolved its own scope.
    if (uploadedAttachments.value.some(item => item.status === 'uploading')) {
      MessagePlugin.warning(t('input.messages.steerAttachmentPending'));
      return;
    }
    if (uploadedAttachments.value.length || uploadedImages.value.length) {
      MessagePlugin.warning(t('input.messages.steerHasAttachments'));
      return;
    }
    const steerMentions: MentionRequestItem[] = allSelectedItems.value.map(item => ({
      id: item.id,
      name: item.name,
      type: item.type,
      kb_type: item.type === 'kb' ? (item.kbType || 'document') : undefined,
      kb_id: item.kbId,
      kb_name: item.kbName,
      service_id: item.serviceId,
      skill_name: item.skillName,
    }));
    emit('steer-msg', val.trim(), steerMentions, delivery);
    clearvalue();
    void focusInput();
    return;
  }
  // Only block while the file is still uploading (no document ID yet). Once
  // uploaded, sending is allowed even if parsing is still in progress: the
  // backend shows a "parsing attachment" step on the timeline and waits.
  const pendingAttachment = uploadedAttachments.value.find(item =>
    item.status === 'uploading'
  );
  if (pendingAttachment) {
    MessagePlugin.warning(t('chat.attachmentStillProcessing', { name: pendingAttachment.name }));
    return;
  }
  const failedAttachment = uploadedAttachments.value.find(item => item.status === 'failed');
  if (failedAttachment) {
    MessagePlugin.error(failedAttachment.error || t('chat.attachmentParseFailed'));
    return;
  }

  // Embed kanalı, agent/KB'yi arka uçtan bağlar; platform tarafındaki agent listesi ve hazır olma doğrulamasından geçmeyin
  if (props.embeddedMode) {
    emit('send-msg', val, selectedModelId.value || '', [], [], [], options);
    clearvalue();
    void focusInput();
    return;
  }

  // Images and non-embedded attachments both travel to the backend as
  // `attachment_ids`, which enforces a combined cap (MaxTemporaryAttachmentsPerMessage).
  // The per-picker limits (5 images / 5 attachments) are independent, so guard the
  // merged total here to avoid a late 400 after the files are already uploaded.
  const MAX_TOTAL_ATTACHMENTS = 5;
  const combinedAttachmentCount =
    uploadedImages.value.length +
    uploadedAttachments.value.filter(item => item.status !== 'failed').length;
  if (combinedAttachmentCount > MAX_TOTAL_ATTACHMENTS) {
    MessagePlugin.warning(t('chat.attachmentTotalTooMany', { max: MAX_TOTAL_ATTACHMENTS }));
    return;
  }

  if (!chatResources.isLoaded('models')) {
    await loadChatModels()
  }

  // Göndermeden önce, mevcut seçili akıllı agent'ın (varsayılan hızlı soru-cevap dahil) yapılandırmasının tamamlanıp tamamlanmadığını doğrula
  const agentToCheck = selectedAgent.value;
  let actualAgent = agentToCheck;
  if (agentToCheck.is_builtin && !settingsStore.selectedAgentSourceTenantId) {
    let builtin = agents.value.find(a => a.id === selectedAgentId.value);
    if (!builtin) {
      await loadAgents();
      builtin = agents.value.find(a => a.id === selectedAgentId.value);
    }
    actualAgent = builtin || agentToCheck;
  }
  const isAgentMode = actualAgent.config?.agent_mode === 'smart-reasoning';
  const { keys: notReadyKeys, labels: notReadyReasons } = collectAgentNotReadyReasons(
    actualAgent,
    isAgentMode,
    settingsStore.selectedAgentSourceTenantId ?? undefined,
  );
  if (notReadyReasons.length > 0) {
    showAgentNotReadyMessage(
      actualAgent,
      notReadyReasons,
      notReadyKeys,
      settingsStore.selectedAgentSourceTenantId ?? undefined,
    );
    return;
  }
  // @ ile anılan bilgi bankası ve dosya bilgilerini al
  const mentionedItems: MentionRequestItem[] = allSelectedItems.value.map(item => ({
    id: item.id,
    name: item.name,
    type: item.type,
    kb_type: item.type === 'kb' ? (item.kbType || 'document') : undefined,
    kb_id: item.kbId,
    kb_name: item.kbName,
    service_id: item.serviceId,
    skill_name: item.skillName,
  }));
  const imageFiles = uploadedImages.value.map(img => img.file);
  const attachmentFiles = uploadedAttachments.value;

  emit('send-msg', val, selectedModelId.value, mentionedItems, imageFiles, attachmentFiles, options);

  // Clean up image previews
  uploadedImages.value.forEach(img => URL.revokeObjectURL(img.preview));
  uploadedImages.value = [];

  // Clean up attachments
  attachmentUploadRef.value?.clear();
  uploadedAttachments.value = [];

  clearvalue();
  void focusInput();
}

const updateAgentModeDropdownPosition = () => {
  const anchor = agentModeButtonRef.value;

  if (!anchor) {
    agentModeDropdownStyle.value = {
      position: 'fixed',
      top: '50%',
      left: '50%',
      transform: 'translate(-50%, -50%)'
    };
    return;
  }

  // Normalize coordinates to CSS pixels (root <html> may carry `zoom`).
  const zoom = getRootZoom();
  const rect = rectToCssPx(anchor.getBoundingClientRect(), zoom);
  const dropdownWidth = 200;
  const offsetY = 8;
  const { width: vw, height: vh } = cssViewportSize(zoom);

  // Yatay konum: sola hizala
  let left = Math.floor(rect.left);
  const minLeft = 16;
  const maxLeft = Math.max(16, vw - dropdownWidth - 16);
  left = Math.max(minLeft, Math.min(maxLeft, left));

  // Dikey konum: düğmeye bitişik; boşluğu önlemek için uygun bir yükseklik kullan
  const preferredDropdownHeight = 140; // Agent modu seçicisinde daha az içerik vardır, daha küçük tercih edilen yükseklik kullan
  const maxDropdownHeight = 150;
  const minDropdownHeight = 100;
  const topMargin = 20;
  const spaceBelow = vh - rect.bottom;
  const spaceAbove = rect.top;

  console.log('[Agent Dropdown] Space check:', {
    spaceBelow,
    spaceAbove,
    windowHeight: vh
  });

  let actualHeight: number;

  // Öncelikle aşağıdaki alanı dikkate al
  if (spaceBelow >= minDropdownHeight + offsetY) {
    // Aşağıda yeterli alan var, aşağı doğru aç
    actualHeight = Math.min(preferredDropdownHeight, spaceBelow - offsetY - 16);
    const top = Math.floor(rect.bottom + offsetY);

    agentModeDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      top: `${top}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important',
    };
    console.log('[Agent Dropdown] Position: below button', { actualHeight });
  } else {
    // Yukarı doğru aç; düğmeye bitişik olmasını sağlamak için bottom konumlandırmasını kullan
    const availableHeight = spaceAbove - offsetY - topMargin;
    if (availableHeight >= preferredDropdownHeight) {
      actualHeight = preferredDropdownHeight;
    } else {
      actualHeight = Math.max(minDropdownHeight, availableHeight);
    }

    const bottom = vh - rect.top + offsetY;

    agentModeDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      bottom: `${bottom}px`, // Düğmeye bitişik olmasını sağlamak için bottom konumlandırmasını kullan
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important',
    };
    console.log('[Agent Dropdown] Position: above button', { actualHeight, bottom });
  }
};

const toggleAgentModeSelector = () => {
  // Karşılıklı dışlayıcı
  showMention.value = false;
  showModelSelector.value = false;

  showAgentModeSelector.value = !showAgentModeSelector.value;
  if (showAgentModeSelector.value) {
    if (!chatResources.isLoaded('agents')) {
      void loadAgents(true);
    }
    // Doğruluğu sağlamak için konumu birden çok kez güncelle
    nextTick(() => {
      updateAgentModeDropdownPosition();
      requestAnimationFrame(() => {
        updateAgentModeDropdownPosition();
        setTimeout(() => {
          updateAgentModeDropdownPosition();
        }, 50);
      });
    });
  }
}

const selectAgentMode = async (mode: 'quick-answer' | 'smart-reasoning') => {
  if (!chatResources.isLoaded('models')) {
    await loadChatModels()
  }

  const builtinAgentId = mode === 'smart-reasoning' ? BUILTIN_SMART_REASONING_ID : BUILTIN_QUICK_ANSWER_ID;
  const builtinAgent = agents.value.find(a => a.id === builtinAgentId);

  if (builtinAgent) {
    const { keys: notReadyKeys, labels: notReadyReasons } = collectAgentNotReadyReasons(
      builtinAgent,
      mode === 'smart-reasoning',
    );
    if (notReadyReasons.length > 0) {
      showAgentModeSelector.value = false;
      showAgentNotReadyMessage(builtinAgent, notReadyReasons, notReadyKeys);
      return;
    }
  }

  const shouldEnableAgent = mode === 'smart-reasoning';
  if (shouldEnableAgent !== isAgentEnabled.value) {
    settingsStore.toggleAgent(shouldEnableAgent);
    // Aynı anda seçili agent'ı güncelle
    settingsStore.selectAgent(shouldEnableAgent ? BUILTIN_SMART_REASONING_ID : BUILTIN_QUICK_ANSWER_ID);
    MessagePlugin.success(shouldEnableAgent ? t('input.messages.agentSwitchedOn') : t('input.messages.agentSwitchedOff'));
  }
  showAgentModeSelector.value = false;
}

// Agent seç (yeni sürüm); sourceTenantId, paylaşılan agent olduğunda iletilir
const handleAgentNotReady = (
  agent: CustomAgent,
  labels: string[],
  keys: AgentNotReadyReasonKey[],
  sourceTenantId?: string,
) => {
  showAgentNotReadyMessage(agent, labels, keys, sourceTenantId);
};

const handleSelectAgent = async (agent: CustomAgent, sourceTenantId?: string) => {
  if (!chatResources.isLoaded('models')) {
    await loadChatModels()
  }

  // Akıllı aracın agent_mode değerine göre Agent modu olup olmadığını belirle
  const isAgentType = agent.config?.agent_mode === 'smart-reasoning';

  // Akıllı aracın hazır olup olmadığını birleşik olarak kontrol et (yerleşik ve özel akıllı araçlar aynı mantığı kullanır)
  const actualAgent = agent.is_builtin && !sourceTenantId
    ? (agents.value.find(a => a.id === agent.id) || agent)
    : agent;

  const { keys: notReadyKeys, labels: notReadyReasons } = collectAgentNotReadyReasons(
    actualAgent,
    isAgentType,
    sourceTenantId,
  );

  if (notReadyReasons.length > 0) {
    showAgentModeSelector.value = false;
    showAgentNotReadyMessage(actualAgent, notReadyReasons, notReadyKeys, sourceTenantId);
    return;
  }

  settingsStore.selectAgent(agent.id, sourceTenantId);
  settingsStore.toggleAgent(!!isAgentType);

  // Modeli eşzamanla (seçilen sohbet modeli, paylaşılan akıllı araçlar dahil olmak üzere akıllı araç değiştirildiğinde değişir).
  // Web araması selectAgent tarafından kapalı olarak sıfırlandı; akıllı araç yapılandırması yalnızca bu anahtarın kullanılabilir olup olmadığını kontrol eder.
  const agentModel = agent.config?.model_id;
  if (agentModel && agentModel.trim() !== '') {
    selectedModelId.value = agentModel;
  } else {
    const lastPick = readLastChatModelID();
    if (lastPick) {
      selectedModelId.value = lastPick;
    }
  }

  showAgentModeSelector.value = false;

  // Only the two "mode-entry" built-ins are re-branded as "Normal / Agent Mode"
  // in the dropdown — the switched-on/off toasts only make sense for them.
  // Other built-ins (wiki researcher, data analyst, etc.) share `is_builtin`
  // but should fall back to the generic agentSelected toast like custom agents,
  // otherwise selecting e.g. the Wiki Questioner incorrectly says
  // "Switched to Intelligent Reasoning".
  const isModeBuiltin =
    agent.id === BUILTIN_QUICK_ANSWER_ID || agent.id === BUILTIN_SMART_REASONING_ID;
  const message = isModeBuiltin
    ? (isAgentType ? t('input.messages.agentSwitchedOn') : t('input.messages.agentSwitchedOff'))
    : t('input.messages.agentSelected', { name: agentDisplayName(agent, t) });
  MessagePlugin.success(message);
}

const clearvalue = () => {
  // Guard: only clear when the textarea DOM element is still mounted,
  // otherwise TDesign's autosize will call getComputedStyle on a non-Element.
  if (!getTextareaEl()) return;
  query.value = "";
}

// Drop any pending images/attachments and stop their status polling. Used when
// switching sessions: leftover documentIds belong to the previous session, so
// keeping them would make polling 404 (falsely marking them failed) or send IDs
// the new session does not own ("attachment ... not found in this session").
const clearPendingUploads = () => {
  uploadedImages.value.forEach(img => URL.revokeObjectURL(img.preview));
  uploadedImages.value = [];
  attachmentUploadRef.value?.clear();
  uploadedAttachments.value = [];
}

const steerShortcutLabel = /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘ Enter' : 'Alt+Enter';
const firstQueuedSteer = computed(() => props.queuedSteers.find(item =>
  item.delivery === 'after' && !item.pending && !item.promoting && !item.failed));
const injectCurrentInput = () => {
  if (props.composerLocked) return;
  if (!props.isReplying || !props.canSteer) return;
  if (query.value.trim()) void createSession(query.value, 'inject');
  else if (firstQueuedSteer.value) emit('promote-steer', firstQueuedSteer.value.steer_id);
};

const onKeydown = (val: string, event: { e: KeyboardEvent }) => {
  if (isComposing.value || event.e.isComposing || event.e.keyCode === 229) return;
  if (showMention.value) {
    if (event.e.keyCode === 38) { // Up
      event.e.preventDefault();
      mentionSelectorRef.value?.moveActive(-1);
      return;
    }
    if (event.e.keyCode === 40) { // Down
      event.e.preventDefault();
      mentionSelectorRef.value?.moveActive(1);
      return;
    }
    if (event.e.keyCode === 13) { // Enter
      event.e.preventDefault();
      mentionSelectorRef.value?.confirmActive();
      return;
    }
    if (event.e.keyCode === 27) { // Esc
      if (mentionSelectorRef.value?.leaveGroup()) {
        return;
      }
      showMention.value = false;
      return;
    }
  }

  // Geri tuşu: Giriş kutusu boşken ve seçili öğeler varken son seçili öğeyi sil
  if (event.e.keyCode === 8) { // Backspace
    const textarea = getTextareaEl();
    if (textarea && textarea.selectionStart === 0 && textarea.selectionEnd === 0 && query.value === '') {
      const items = allSelectedItems.value;
      if (items.length > 0) {
        event.e.preventDefault();
        const lastItem = items[items.length - 1];
        removeSelectedItem(lastItem);
        return;
      }
    }
  }

  const delivery = chatSubmitShortcut(event.e, props.isReplying && props.canSteer);
  if (delivery) {
    event.e.preventDefault();
    if (props.composerLocked) return;
    if (delivery === 'inject' && props.isReplying && props.canSteer) injectCurrentInput();
    else void createSession(val, delivery);
  }
}

const onPaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items;
  if (!items) return;
  const imageFiles: File[] = [];
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      const file = item.getAsFile();
      if (file) imageFiles.push(file);
    }
  }
  if (imageFiles.length > 0 && isImageUploadEnabledByAgent.value) {
    e.preventDefault();
    addImageFiles(imageFiles);
  }
};

const onDrop = (e: DragEvent) => {
  e.preventDefault();
  const files = e.dataTransfer?.files;
  if (!files || files.length === 0) return;
  handleDroppedFiles(Array.from(files));
};

const onDragOver = (e: DragEvent) => {
  e.preventDefault();
};

const handleGoToWebSearchSettings = () => {
  uiStore.openSettings('websearch');
  if (route.path !== '/platform/settings') {
    router.push('/platform/settings');
  }
};

const handleGoToWebSearchConfig = () => {
  if (hasAgentConfig.value && selectedAgent.value) {
    handleGoToAgentSettings('websearch');
    return;
  }
  handleGoToWebSearchSettings();
};

const handleGoToAgentSettings = (section?: string) => {
  const agent = selectedAgent.value;
  if (!agent) {
    router.push('/platform/agents');
    return;
  }
  const query: Record<string, string> = { edit: agent.id };
  if (section) {
    query.section = section;
  }
  router.push({ path: '/platform/agents', query });
};

const formatAgentNotReadyReasons = (
  reasonKeys: AgentNotReadyReasonKey[],
  isBuiltin: boolean,
): string[] => {
  return reasonKeys.map((key) => {
    if (key === 'summary_model') {
      return isBuiltin
        ? t('input.agentMissingSummaryModel')
        : t('input.customAgentMissingSummaryModel');
    }
    if (key === 'rerank_model') {
      return isBuiltin
        ? t('input.agentMissingRerankModel')
        : t('input.customAgentMissingRerankModel');
    }
    return t('input.agentMissingAllowedTools');
  });
};

const collectAgentNotReadyReasons = (
  agent: CustomAgent,
  isAgentMode: boolean,
  sourceTenantId?: string,
): { keys: AgentNotReadyReasonKey[]; labels: string[] } => {
  const isSharedAgent = !!sourceTenantId;
  const keys = getAgentNotReadyReasonKeys(agent.config, allModels.value, {
    isAgentMode,
    isSharedAgent,
  });
  return {
    keys,
    labels: formatAgentNotReadyReasons(keys, agent.is_builtin),
  };
};

const goToAgentEditor = (
  agent: CustomAgent,
  section = 'model',
  highlight?: AgentNotReadyReasonKey,
  sourceTenantId?: string,
) => {
  router.push({
    path: '/platform/agents',
    query: {
      edit: agent.id,
      section,
      ...(highlight ? { highlight } : {}),
      ...(sourceTenantId ? { sourceTenantId } : {}),
    },
  });
};

// Akıllı araç hazır olmadığında mesajı göster (yerleşik ve özel akıllı araçları birleşik olarak işle)
const showAgentNotReadyMessage = (
  agent: CustomAgent,
  reasons: string[],
  reasonKeys?: AgentNotReadyReasonKey[],
  sourceTenantId?: string,
) => {
  const reasonsText = formatLocalizedList(reasons, locale.value)
  const isRemoteShared = !canLocallyConfigureAgent(sourceTenantId)

  const messageContent = h('div', { style: 'display: flex; flex-direction: column; gap: 8px; max-width: 320px;' }, [
    h(
      'span',
      { style: 'color: var(--td-text-color-primary); line-height: 1.5;' },
      isRemoteShared
        ? t('input.sharedAgentNotReadyDetail', { agentName: agentDisplayName(agent, t), reasons: reasonsText })
        : t('input.agentNotReadyDetail', { agentName: agentDisplayName(agent, t), reasons: reasonsText }),
    ),
    ...(isRemoteShared ? [] : [
      h('a', {
        href: '#',
        onClick: (e: Event) => {
          e.preventDefault();
          const section = resolveAgentNotReadySection(reasonKeys || ['summary_model'])
          const highlight = resolveAgentNotReadyHighlight(reasonKeys || ['summary_model'])
          goToAgentEditor(agent, section, highlight, sourceTenantId);
        },
        style: 'color: var(--td-brand-color); text-decoration: none; font-weight: 500; cursor: pointer; align-self: flex-start;',
        onMouseenter: (e: Event) => {
          (e.target as HTMLElement).style.textDecoration = 'underline';
        },
        onMouseleave: (e: Event) => {
          (e.target as HTMLElement).style.textDecoration = 'none';
        }
      }, t('input.goToAgentEditor')),
    ]),
  ]);

  MessagePlugin.warning({
    content: () => messageContent,
    duration: 5000
  });
}

const toggleWebSearch = () => {
  // Karşılıklı dışlama: Açılır katman olmasa da işlem sırasında diğer açılır katmanları kapatmak daha iyi bir deneyim sağlar
  showMention.value = false;
  showModelSelector.value = false;
  showAgentModeSelector.value = false;

  // Akıllı araç web aramasını devre dışı bıraktıysa açılmasına izin verme
  if (isWebSearchDisabledByAgent.value) {
    MessagePlugin.warning(t('input.webSearchDisabledByAgent'));
    return;
  }

  if (!isWebSearchConfigured.value) {
    const messageContent = h('div', { style: 'display: flex; flex-direction: column; gap: 6px; max-width: 280px;' }, [
      h('span', { style: 'color: var(--td-text-color-primary); line-height: 1.5;' }, t('input.messages.webSearchNotConfigured')),
      h('a', {
        href: '#',
        onClick: (e: Event) => {
          e.preventDefault();
          handleGoToWebSearchConfig();
        },
        style: 'color: var(--td-brand-color); text-decoration: none; font-weight: 500; cursor: pointer; align-self: flex-start;',
        onMouseenter: (e: Event) => {
          (e.target as HTMLElement).style.textDecoration = 'underline';
        },
        onMouseleave: (e: Event) => {
          (e.target as HTMLElement).style.textDecoration = 'none';
        }
      }, t('input.goToAgentSettings'))
    ]);
    MessagePlugin.warning({
      content: () => messageContent,
      duration: 5000
    });
    return;
  }

  const currentValue = settingsStore.isWebSearchEnabled;
  const newValue = !currentValue;
  settingsStore.toggleWebSearch(newValue);
  MessagePlugin.success(newValue ? t('input.messages.webSearchEnabled') : t('input.messages.webSearchDisabled'));
};

const toggleKbSelector = () => {
  showKbSelector.value = !showKbSelector.value;
}

const removeKb = (kbId: string) => {
  settingsStore.removeKnowledgeBase(kbId);
}

const handleStop = async () => {
  if (!props.sessionId) {
    MessagePlugin.warning(t('input.messages.sessionMissing'));
    return;
  }

  if (!props.assistantMessageId) {
    console.error('[Stop] Assistant message ID is empty');
    MessagePlugin.warning(t('input.messages.messageMissing'));
    return;
  }

  console.log('[Stop] Stopping generation for message:', props.assistantMessageId);

  emit('stop-generation');

  try {
    await stopSession(props.sessionId, props.assistantMessageId);
    emit('stop-confirmed');
    MessagePlugin.success(t('input.messages.stopSuccess'));
  } catch (error) {
    console.error('Failed to stop session:', error);
    emit('stop-failed');
    MessagePlugin.error(t('input.messages.stopFailed'));
  }
}

onBeforeRouteUpdate((to, from, next) => {
  clearvalue()
  clearPendingUploads()
  next()
})

defineExpose({
  focusInput,
  triggerSend(text: string, options: SendMessageOptions = {}) {
    if (!text.trim()) return;
    query.value = text;
    nextTick(() => createSession(text, 'after', options));
  },
  /**
   * Puts text in the composer WITHOUT sending it. Session fork uses this so
   * the user lands on the branch with the original question ready to edit —
   * the whole point of branching at a user message.
   */
  prefill(text: string) {
    query.value = text;
  }
});

</script>
<template>
  <div class="answers-input" :class="{ 'is-embedded': embeddedMode, 'is-compact': compact }" @drop="onDrop" @dragover="onDragOver">
    <!-- Hidden file input for image upload -->
    <input ref="imageInputRef" type="file" accept="image/jpeg,image/png,image/gif,image/webp" multiple
      style="display:none" @change="handleImageSelect" />
    <!-- Kuyruk, giriş kutusunun hemen üstündedir ve giriş alanının odak vurgusuna dahil olmaz -->
    <div v-if="queuedSteers.length" class="steer-queue" role="list" :aria-label="$t('input.steerQueueWaiting')">
      <div v-for="item in queuedSteers" :key="item.steer_id" class="steer-queue-item" role="listitem">
        <t-tooltip :content="$t('input.steerAfter')">
          <t-icon name="time" class="steer-queue-icon" :aria-label="$t('input.steerAfter')" />
        </t-tooltip>
        <span class="steer-queue-text" :title="item.content">{{ item.content }}</span>
        <div class="steer-queue-actions">
          <t-tooltip v-if="item.failed" :content="$t('input.steerRetry')">
            <button type="button" class="steer-queue-action" :aria-label="$t('input.steerRetry')" @click="emit('retry-steer', item.steer_id)"><t-icon name="refresh" /></button>
          </t-tooltip>
          <t-icon v-else-if="item.pending" name="loading" class="steer-sending" :aria-label="$t('common.loading')" />
          <t-tooltip v-else :content="`${$t('input.steerQueueSendNow')}${item.steer_id === firstQueuedSteer?.steer_id ? ` · ${steerShortcutLabel}` : ''}`">
            <button type="button" class="steer-queue-action"
              :aria-label="$t('input.steerQueueSendNow')" :disabled="item.promoting || item.pending"
              @click="emit('promote-steer', item.steer_id)"><t-icon name="arrow-up" /></button>
          </t-tooltip>
          <t-tooltip :content="$t('common.remove')">
            <button type="button" class="steer-queue-action steer-queue-remove"
              :aria-label="$t('common.remove')" :disabled="item.promoting || item.pending"
              @click="emit('remove-steer', item.steer_id)"><t-icon name="close" /></button>
          </t-tooltip>
        </div>
      </div>
    </div>
    <div class="rich-input-container" data-guide="chat-input">
      <!-- Görsel önizleme alanı -->
      <div v-if="uploadedImages.length > 0" class="image-preview-bar">
        <div v-for="(img, idx) in uploadedImages" :key="idx" class="image-preview-item">
          <img :src="img.preview" class="image-preview-thumb" />
          <span class="image-preview-remove" @click="removeImage(idx)">×</span>
        </div>
      </div>

      <!-- Ek listesi alanı (AttachmentUpload bileşeni tarafından işlenir) -->
      <AttachmentUpload ref="attachmentUploadRef" :max-files="5"
        :session-id="sessionId" :agent-id="selectedAgentId"
        :agent-source-tenant-id="settingsStore.selectedAgentSourceTenantId ?? undefined"
        @update:files="uploadedAttachments = $event" />

      <!-- Seçili bilgi tabanı ve dosya etiketleri (giriş kutusunun üst kısmında gösterilir) -->
      <div v-if="allSelectedItems.length > 0" class="selected-tags-inline">
        <span v-for="item in allSelectedItems" :key="`${item.type}:${item.id}`" class="mention-chip" :class="[
          getMentionChipClass(item),
          { 'mention-chip--agent': item.isAgentConfigured }
        ]">
          <span class="mention-chip__icon-wrap" :class="{ 'has-org': item.org_name }">
            <span class="mention-chip__icon">
              <t-icon v-if="item.type === 'kb'" :name="item.kbType === 'faq' ? 'chat-bubble-help' : 'folder'" />
              <t-icon v-else :name="getMentionIcon(item)" />
            </span>
            <span v-if="item.org_name" class="mention-chip__org-badge">
              <img :src="getImgSrc(item.type === 'file' ? 'organization-grey.svg' : 'organization-green.svg')"
                class="mention-chip__org-img" alt="" aria-hidden="true" />
            </span>
          </span>
          <span class="mention-chip__name" :title="item.name">{{ item.name }}</span>
          <span class="mention-chip__remove" @click.stop="removeSelectedItem(item)"
            :aria-label="$t('common.remove')">×</span>
        </span>
      </div>

      <!-- Asıl giriş kutusu -->
      <t-textarea ref="textareaRef" v-model="query" :placeholder="t('input.placeholder')" name="description" :autosize="true"
        @keydown="onKeydown" @input="onInput" @compositionstart="onCompositionStart" @compositionend="onCompositionEnd"
        @paste="onPaste" />

      <!-- Kontrol çubuğu belge akışına göre düzenlenir; satır kaydırıldığında kapsayıcıyı otomatik olarak genişletir -->
      <div class="control-bar" :class="{ 'is-embedded': embeddedMode }">
        <!-- Sol kontrol düğmeleri -->
        <div class="control-left" v-if="!embeddedMode">
          <!-- Agent modu değiştirme düğmesi -->
          <div ref="agentModeButtonRef" class="control-btn agent-mode-btn" :class="{
            'is-normal': !isCustomAgent && !isAgentEnabled,
            'is-agent': !isCustomAgent && isAgentEnabled,
            'is-custom': isCustomAgent
          }" @click.stop="toggleAgentModeSelector">
            <span class="agent-mode-text">
              {{ agentDisplayName(selectedAgent, t) || (isAgentEnabled ? $t('input.agentMode') : $t('input.normalMode')) }}
            </span>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor" class="dropdown-arrow"
              :class="{ 'rotate': showAgentModeSelector }">
              <path d="M2.5 4.5L6 8L9.5 4.5H2.5Z" />
            </svg>
          </div>

          <!-- Agent seçici açılır menüsü -->
          <AgentSelector :visible="showAgentModeSelector" :anchorEl="agentModeButtonRef"
            :currentAgentId="selectedAgentId" :agents="enabledAgents" :all-models="allModels"
            @close="closeAgentModeSelector" @select="handleSelectAgent" @not-ready="handleAgentNotReady" />

          <t-tooltip v-if="settingsStore.isAgentStreamMode" placement="top" theme="light"
            :popupProps="{ overlayClassName: 'input-field-tooltip' }">
            <template #content>
              <div v-if="!browserConnection.knownOffline" class="browser-source-tooltip">
                <strong>{{ $t('localBrowser.local') }}</strong>
                <span>{{ $t('localBrowser.sourceHint') }}</span>
              </div>
              <div v-else class="tooltip-with-link">
                <span>{{ $t(browserSourceUnavailableHint) }}</span>
                <a href="#" @click.prevent="openBrowserConnectionSettings">{{ $t('localBrowser.openSettings') }}</a>
              </div>
            </template>
            <button type="button" class="control-btn browser-source-btn"
              :class="{
                active: settingsStore.isLocalBrowserEnabled && browserConnection.online,
                disabled: browserConnection.knownOffline,
              }"
              :aria-pressed="settingsStore.isLocalBrowserEnabled && browserConnection.online"
              :aria-disabled="browserConnection.knownOffline"
              :aria-label="$t('localBrowser.local')"
              @click.stop="toggleBrowserSource">
              <BrowserIcon class="control-icon" />
            </button>
          </t-tooltip>

          <!-- WebSearch anahtar düğmesi (akıllı araç etkin değilse gösterilmez) -->
          <t-tooltip v-if="showWebSearchButton" placement="top" theme="light"
            :popupProps="{ overlayClassName: 'input-field-tooltip' }">
            <template #content>
              <span v-if="isWebSearchConfigured">{{ isWebSearchEnabled ? $t('input.webSearch.toggleOff') :
                $t('input.webSearch.toggleOn') }}</span>
              <div v-else class="tooltip-with-link">
                <span>{{ $t('input.webSearch.notConfigured') }}</span>
                <a href="#" @click.prevent="handleGoToWebSearchConfig">{{ $t('input.goToAgentSettings') }}</a>
              </div>
            </template>
            <div class="control-btn websearch-btn" :class="{
              'active': isWebSearchEnabled && isWebSearchConfigured,
              'disabled': !isWebSearchConfigured
            }" @click.stop="toggleWebSearch">
              <t-icon name="earth" class="control-icon websearch-icon" aria-hidden="true" />
            </div>
          </t-tooltip>

          <!-- Görsel yükleme düğmesi (akıllı araç etkin değilse gösterilmez) -->
          <t-tooltip v-if="showImageUploadButton" placement="top" theme="light"
            :popupProps="{ overlayClassName: 'input-field-tooltip' }">
            <template #content>
              <span>{{ $t('chat.imageUploadTooltip') }}</span>
            </template>
            <div class="control-btn image-upload-btn" :class="{
              'active': uploadedImages.length > 0
            }" @click.stop="triggerImageUpload()">
              <svg width="18" height="18" viewBox="0 0 1024 1024" fill="currentColor" class="control-icon">
                <path
                  d="M896 128H128c-35.3 0-64 28.7-64 64v640c0 35.3 28.7 64 64 64h768c35.3 0 64-28.7 64-64V192c0-35.3-28.7-64-64-64zM128 832V192h768l0.1 640H128z" />
                <path d="M352 448a96 96 0 1 0 0-192 96 96 0 0 0 0 192z" />
                <path d="M128 768l224-288 160 160 192-256L896 640v128H128z" />
              </svg>
              <span v-if="uploadedImages.length > 0" class="image-count">{{ uploadedImages.length }}</span>
            </div>
          </t-tooltip>

          <!-- Ek yükleme düğmesi -->
          <t-tooltip placement="top" theme="light" :popupProps="{ overlayClassName: 'input-field-tooltip' }">
            <template #content>
              <span>{{ uploadedAttachments.length > 0 ? $t('chat.attachmentWithCount', {
                count: uploadedAttachments.length
              }) : $t('chat.attachmentUploadTooltip') }}</span>
            </template>
            <div class="control-btn attachment-upload-btn" :class="{ 'active': uploadedAttachments.length > 0 }"
              @click.stop="attachmentUploadRef?.triggerFileSelect()">
              <!-- Ataş simgesi -->
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"
                stroke-linecap="round" stroke-linejoin="round" class="control-icon">
                <path
                  d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48" />
              </svg>
              <span v-if="uploadedAttachments.length > 0" class="attachment-count">{{ uploadedAttachments.length
              }}</span>
            </div>
          </t-tooltip>

          <!-- @ bilgi tabanı/dosya seçme düğmesi -->
          <t-tooltip placement="top" theme="light" :popupProps="{ overlayClassName: 'input-field-tooltip' }">
            <template #content>
              <div v-if="isMentionDisabled && isKnowledgeBaseDisabledByAgent" class="tooltip-with-link">
                <span>{{ $t('input.kbDisabledByAgent') }}</span>
                <a href="#" @click.prevent="handleGoToAgentSettings('knowledge')">{{ $t('input.goToAgentSettings')
                }}</a>
              </div>
              <span v-else>{{ allSelectedItems.length > 0 ? $t('input.knowledgeBaseWithCount', {
                count:
                  allSelectedItems.length
              }) : $t('input.knowledgeBase') }}</span>
            </template>
            <div ref="atButtonRef" class="control-btn kb-btn" data-guide="chat-kb-mention" :class="{
              'active': allSelectedItems.length > 0,
              'disabled': isMentionDisabled
            }" @click.stop @mousedown.prevent="triggerMention">
              <ResourceIcon type="file" :size="16" />
              <span v-if="allSelectedItems.length > 0" class="kb-count">{{ allSelectedItems.length }}</span>
            </div>
          </t-tooltip>

          <!-- Model gösterimi -->
          <t-tooltip :content="isModelLockedByAgent ? $t('input.modelLockedByAgent') : ''"
            :disabled="!isModelLockedByAgent">
            <div class="model-display" :class="{ 'agent-controlled': isModelLockedByAgent }">
              <div ref="modelButtonRef" class="model-selector-trigger" @click.stop="toggleModelSelector">
                <span class="model-selector-name">
                  {{ selectedModelDisplayName }}
                </span>
                <span
                  v-if="selectedModelContextLabel"
                  class="model-selector-ctx"
                  :class="{ 'is-default': selectedModelContextIsDefault }"
                  :title="selectedModelContextTitle"
                >{{ selectedModelContextLabel }}</span>
                <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor" class="model-dropdown-arrow"
                  :class="{ 'rotate': showModelSelector }">
                  <path d="M2.5 4.5L6 8L9.5 4.5H2.5Z" />
                </svg>
              </div>
            </div>
          </t-tooltip>
          <t-popup v-if="reasoningLevels.length > 0" v-model:visible="showReasoningSelector"
            trigger="click" placement="top-right" :disabled="composerLocked"
            :overlay-inner-style="{ padding: '4px', borderRadius: 'var(--app-radius-lg)' }"
            @visible-change="handleReasoningVisibleChange">
            <button type="button" class="model-selector-trigger reasoning-effort-trigger"
              :disabled="composerLocked" :class="{ disabled: composerLocked }"
              :aria-label="`${$t('modelSettings.debug.reasoningEffort')}: ${$t(levelLabelKey(displayedReasoningLevel))}`"
              :title="$t('modelSettings.debug.reasoningEffort')" aria-haspopup="menu" :aria-expanded="showReasoningSelector"
              @keydown.esc="showReasoningSelector = false">
              <span class="model-selector-name">{{ $t(levelLabelKey(displayedReasoningLevel)) }}</span>
              <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor" class="model-dropdown-arrow"
                :class="{ rotate: showReasoningSelector }">
                <path d="M2.5 4.5L6 8L9.5 4.5H2.5Z" />
              </svg>
            </button>
            <template #content>
              <div class="reasoning-effort-menu" role="menu" :aria-label="$t('modelSettings.debug.reasoningEffort')"
                @keydown.esc="showReasoningSelector = false">
                <div class="reasoning-effort-title" role="presentation">{{ $t('modelSettings.debug.reasoningEffort') }}</div>
                <button v-for="level in reasoningLevels" :key="level" type="button" role="menuitemradio"
                  class="reasoning-effort-option" :class="{ selected: level === displayedReasoningLevel }"
                  :aria-checked="level === displayedReasoningLevel" @click="selectReasoningLevel(level)">
                  <span>{{ $t(levelLabelKey(level)) }}</span>
                  <t-icon v-if="level === displayedReasoningLevel" name="check" size="14px" />
                </button>
              </div>
            </template>
          </t-popup>
        </div>

        <Teleport to="body">
          <div v-if="showModelSelector" class="model-selector-overlay" @click="closeModelSelector">
            <div class="model-selector-dropdown" :style="modelDropdownStyle" @click.stop>
              <div class="model-selector-header">
                <span>{{ $t('conversationSettings.models.chatGroupLabel') }}</span>
                <button class="model-selector-add" type="button" @click="handleModelChange('__add_model__')">
                  <span class="add-icon">+</span>
                  <span class="add-text">{{ $t('input.addModel') }}</span>
                </button>
              </div>
              <div class="model-selector-content">
                <div v-for="model in availableModels" :key="model.id" class="model-option"
                  :class="{ selected: model.id === selectedModelId }" @click="handleModelChange(model.id || '')">
                  <div class="model-option-left">
                    <div class="model-option-icon">
                      <t-icon name="chat" size="14px" />
                    </div>
                    <div class="model-option-name-wrap">
                      <span class="model-option-name">{{ modelDisplayName(model) }}</span>
                      <span v-if="model.display_name" class="model-option-raw-name">{{ model.name }}</span>
                    </div>
                  </div>
                  <span
                    class="model-option-ctx"
                    :class="{ 'is-default': isDefaultContextWindow(model.parameters?.context_window) }"
                    :title="contextWindowTitle(model.parameters?.context_window)"
                  >{{ formatContextWindow(model.parameters?.context_window) }}</span>
                </div>
                <div v-if="availableModels.length === 0" class="model-option empty">
                  {{ $t('input.noModel') }}
                </div>
              </div>
            </div>
          </div>
        </Teleport>

        <!-- Sağ kontrol: Yanıt sürerken giriş boşsa durdur; yeni içerik girildiğinde aynı konum gönder düğmesine dönüşür -->
        <div class="control-right">
          <t-tooltip v-if="isReplying && (!canSteer || !query.trim())" :content="$t('input.stopGeneration')" placement="top">
            <button type="button" @click="handleStop" class="control-btn stop-btn" :aria-label="$t('input.stopGeneration')">
              <t-icon name="stop" />
            </button>
          </t-tooltip>
          <t-tooltip v-else :content="`${isReplying && canSteer ? $t('input.steerAfter') : $t('input.send')} · Enter`">
            <button type="button" @click="createSession(query)" class="control-btn send-btn" data-guide="chat-send"
              :disabled="!query.trim() || composerLocked" :class="{ 'disabled': !query.trim() || composerLocked }"
              :aria-label="isReplying && canSteer ? $t('input.steerAfter') : $t('input.send')">
              <t-icon name="arrow-up" />
            </button>
          </t-tooltip>
        </div>
      </div>
    </div>

    <!-- Mention Selector -->
    <Teleport to="body">
      <MentionSelector ref="mentionSelectorRef" :visible="showMention" :style="mentionStyle" :items="mentionItems" :hasMore="mentionHasMore"
        :loading="mentionLoading" :emptyHint="mentionEmptyHint" :query="mentionQuery" :group-counts="mentionGroupCounts" v-model:activeIndex="mentionActiveIndex"
        :hint="isMentionTriggeredByButton ? $t('input.kbMentionHint') : ''"
        @select="onMentionSelect" @loadMore="loadMoreMentionItems" />
    </Teleport>

    <!-- Bilgi tabanı seçim açılır menüsü (Teleport kullanılarak body öğesine taşınır; üst kapsayıcının konumlandırma etkisi önlenir) -->
    <Teleport to="body">
      <KnowledgeBaseSelector v-model:visible="showKbSelector" :anchorEl="atButtonRef" @close="showKbSelector = false" />
    </Teleport>
  </div>
</template>
<script lang="ts">
const getImgSrc = (url: string) => {
  return new URL(`/src/assets/img/${url}`, import.meta.url).href;
}
</script>
<style scoped lang="less">
@import './css/chat-resource-chips.less';

.answers-input {
  position: absolute;
  z-index: 99;
  bottom: 60px;
  left: 50%;
  transform: translateX(-50%);
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;

  &.is-embedded {
    position: relative;
    bottom: auto;
    left: auto;
    transform: none;
    z-index: auto;

    .rich-input-container,
    .steer-queue {
      max-width: 100%;
    }
  }
}

.steer-queue {
  width: calc(100% - 24px);
  max-width: 936px;
  box-sizing: border-box;
  max-height: 140px;
  overflow-y: auto;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-bottom: 0;
  border-radius: 10px 10px 0 0;
}

.steer-queue-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
}

.steer-queue-item + .steer-queue-item {
  border-top: 1px solid var(--td-component-stroke);
}

.steer-queue-text {
  flex: 1;
  min-width: 0;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.steer-queue-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 4px;
}

.steer-queue-icon {
  flex-shrink: 0;
  font-size: var(--app-text-base);
  color: var(--td-text-color-secondary);
}

.steer-queue-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xl);
  cursor: pointer;
  &:hover:not(:disabled) { background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-primary); }
  &:disabled { opacity: 0.4; cursor: default; }
}

.steer-sending { animation: wk-spin 1s linear infinite; }
@media (prefers-reduced-motion: reduce) { .steer-sending { animation: none; } }

/* Zengin metin giriş kutusu kapsayıcısı */
.rich-input-container {
  position: relative;
  width: 100%;
  max-width: 960px;
  background: var(--td-bg-color-container);
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--td-component-stroke);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04), 0 8px 16px -4px rgba(0, 0, 0, 0.06);

  &:focus-within {
    border-color: var(--td-brand-color);
  }
}

/* Seçili bilgi tabanı/dosya etiketleri (mention list seçili öğeleri) */
.selected-tags-inline {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 5px;
  padding: 6px 12px 6px;
  border-bottom: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
  border-radius: var(--app-menu-item-radius) var(--app-menu-item-radius) 0 0;
  /* .rich-input-container iç kenarının üst köşe yarıçapıyla aynı (12px - 1px kenarlık) */
}

.mention-chip {
  .chat-resource-chip-surface();

  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 26px;
  padding: 3px 7px 3px 6px;
  border-radius: var(--td-radius-medium);
  box-sizing: border-box;
  font-size: var(--app-text-sm);
  font-weight: 500;
  cursor: default;
  transition: background var(--app-motion-fast), border-color var(--app-motion-fast);
  line-height: 18px;

  &:hover {
    .chat-resource-chip-hover();
  }
}

.mention-chip__icon-wrap {
  position: relative;
  display: inline-flex;
  width: 16px;
  height: 16px;
  flex: 0 1 auto;
  min-width: 0;
  align-items: center;
  justify-content: center;
}

.mention-chip__icon {
  font-size: var(--app-text-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  color: inherit;
}

.mention-chip__org-badge {
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--td-bg-color-secondarycontainer);
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.06);
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.mention-chip__org-img {
  width: 5px;
  height: 5px;
  object-fit: contain;
}

.mention-chip__name {
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: currentColor;
}

.mention-chip__remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  margin-left: 1px;
  border-radius: 50%;
  font-size: var(--app-text-base);
  line-height: 1;
  font-weight: 400;
  cursor: pointer;
  opacity: 0.5;
  transition: opacity var(--app-motion-fast), background var(--app-motion-fast), color var(--app-motion-fast);
  color: currentColor;
  flex-shrink: 0;
}

.mention-chip:hover .mention-chip__remove {
  opacity: 0.85;
}

.mention-chip__remove:hover {
  opacity: 1;
  background: var(--td-bg-color-component);
  color: var(--td-text-color-primary);
}

/* Etiket yüzeyi nötr kalır; kaynak türü yalnızca simge rengiyle ifade edilir. */
.mention-chip--kb {
  color: var(--td-text-color-primary);
}

.mention-chip--kb .mention-chip__icon-wrap {
  color: var(--td-brand-color);
}

.mention-chip--faq {
  color: var(--td-text-color-primary);
}

.mention-chip--faq .mention-chip__icon-wrap {
  color: var(--rethra-faq-color, #0052d9);
}

.mention-chip--file {
  color: var(--td-text-color-primary);
}

.mention-chip--file .mention-chip__icon-wrap {
  color: var(--td-text-color-secondary);
}

.mention-chip--tag,
.mention-chip--mcp,
.mention-chip--tool {
  color: var(--td-text-color-primary);
}

.mention-chip--tag .mention-chip__icon-wrap {
  color: #9f7aea;
}

.mention-chip--mcp .mention-chip__icon-wrap {
  color: #0f766e;
}

.mention-chip--tool .mention-chip__icon-wrap {
  color: #b7791f;
}

/* Aracı ön yapılandırması: kesik çizgili kenarlıkla ayırt edilir */
.mention-chip--agent {
  border-style: dashed;
  border-color: var(--td-component-border);
}

:deep(.t-textarea__inner) {
  width: 100%;
  max-height: 152px !important;
  min-height: var(--composer-input-min-height, 72px) !important;
  resize: none;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-xl);
  font-weight: 400;
  line-height: 24px;
  font-family: var(--app-font-family);
  padding: 12px 16px;
  border-radius: 0 0 12px 12px;
  border: none;
  box-sizing: border-box;
  background: transparent;
  box-shadow: none;

  &:focus {
    border: none;
    box-shadow: none;
  }

  &::placeholder {
    color: var(--td-text-color-placeholder);
    font-family: var(--app-font-family);
    font-size: var(--app-text-xl);
    font-weight: 400;
    line-height: 24px;
  }
}

/* Hiçbir etiket seçilmediğinde textarea stili */
.rich-input-container:not(:has(.selected-tags-inline)) :deep(.t-textarea__inner) {
  border-radius: var(--app-radius-xl);
  padding-top: 16px;
}

/* Kontrol çubuğu */
.control-bar {
  position: relative;
  margin: 0 16px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
  z-index: 10;
  pointer-events: auto;
  padding-top: 8px;

  &.is-embedded {
    justify-content: flex-end;
  }
}

.answers-input.is-compact {
  --composer-input-min-height: 56px;

  .rich-input-container :deep(.t-textarea__inner) {
    padding: 12px 14px;
  }

  .control-bar {
    margin: 0 12px 8px;
    padding-top: 4px;
  }

  .control-icon {
    width: 16px;
    height: 16px;
  }
}

.control-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  flex-wrap: wrap;
  min-width: 0;
}

.control-btn {
  border: 0;
  font: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 10px;
  border-radius: var(--app-radius-sm);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: background var(--app-motion-instant), color var(--app-motion-instant);
  user-select: none;
  flex-shrink: 0;

  &:hover {
    background: var(--td-bg-color-secondarycontainer-hover);
  }

  &.disabled {
    opacity: 0.5;
    cursor: not-allowed;

    &:hover:not(.send-btn):not(.stop-btn) {
      background: var(--td-bg-color-secondarycontainer);
    }
  }
}

.agent-mode-btn {
  height: 28px;
  padding: 0 10px;
  min-width: auto;
  font-weight: 500;
  position: relative;
  border: .5px solid var(--td-component-border);
}

.agent-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.agent-btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 5px;
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.agent-mode-text {
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  font-weight: 500;
  white-space: nowrap;
  margin: 0 4px;
}

.control-icon {
  width: 18px;
  height: 18px;
}

.kb-btn {
  height: 28px;
  width: 28px;
  padding: 0;
  min-width: auto;
  position: relative;

  &:hover:not(.disabled):not(.active) {
    color: var(--td-text-color-primary);
  }

  &.active {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);

    &:hover {
      color: var(--td-brand-color);
      background: var(--td-bg-color-secondarycontainer);
    }
  }

  &.agent-controlled {
    cursor: not-allowed;
    opacity: 0.85;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
    }

    &.active:hover {
      background: var(--td-bg-color-secondarycontainer);
    }
  }
}

.kb-count {
  position: absolute;
  top: -2px;
  right: -2px;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  min-width: 14px;
  height: 14px;
  padding: 0 3px;
  border-radius: 7px;
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
  font-size: var(--app-text-2xs);
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  pointer-events: none;
}

.kb-btn-text {
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  font-weight: 500;
  white-space: nowrap;
}

.kb-btn.active .kb-btn-text {
  color: var(--td-brand-color);
}

/* Image upload */
.image-upload-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  min-width: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  color: var(--td-text-color-secondary);

  &:hover {
    background: var(--td-bg-color-secondarycontainer-hover);
    color: var(--td-text-color-primary);
  }

  &.active {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }

  .image-count {
    position: absolute;
    top: -2px;
    right: -2px;
    background: var(--td-brand-color);
    color: #fff;
    font-size: var(--app-text-2xs);
    width: 14px;
    height: 14px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
  }
}

/* Attachment upload */
.attachment-upload-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  min-width: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  color: var(--td-text-color-secondary);

  &:hover {
    background: var(--td-bg-color-secondarycontainer-hover);
    color: var(--td-text-color-primary);
  }

  &.active {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }

  .attachment-count {
    position: absolute;
    top: -2px;
    right: -2px;
    background: var(--td-brand-color);
    color: #fff;
    font-size: var(--app-text-2xs);
    width: 14px;
    height: 14px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
  }
}

.image-preview-bar {
  display: flex;
  gap: 8px;
  padding: 8px 12px 4px;
  flex-wrap: wrap;
}

.image-preview-item {
  position: relative;
  width: 60px;
  height: 60px;
  border-radius: var(--app-radius-md);
  overflow: hidden;
  border: 1px solid var(--td-border-level-1-color);

  .image-preview-thumb {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .image-preview-remove {
    position: absolute;
    top: 2px;
    right: 2px;
    width: 16px;
    height: 16px;
    background: rgba(0, 0, 0, 0.5);
    color: #fff;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--app-text-sm);
    cursor: pointer;
    line-height: 1;

    &:hover {
      background: rgba(0, 0, 0, 0.7);
    }
  }
}

.browser-source-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;

  &:hover:not(.disabled):not(.active) {
    color: var(--td-text-color-primary);
  }

  &.active {
    color: var(--td-brand-color);
    background: var(--td-bg-color-secondarycontainer);

    &:hover {
      color: var(--td-brand-color);
      background: var(--td-bg-color-secondarycontainer);
    }
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}

.browser-source-tooltip {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 240px;
  line-height: 1.5;

  strong {
    font-weight: 500;
  }
}

.websearch-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  min-width: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;

  &.active {
    background: var(--td-bg-color-secondarycontainer);

    .websearch-icon {
      color: var(--td-brand-color);
    }

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
    }
  }

  &:not(.active) {
    .websearch-icon {
      color: var(--td-text-color-secondary);
    }

    &:hover {
      background: var(--td-bg-color-secondarycontainer-hover);

      .websearch-icon {
        color: var(--td-text-color-primary);
      }
    }
  }

  &.agent-controlled {
    cursor: not-allowed;
    opacity: 0.85;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
    }

    &.active:hover {
      background: var(--td-bg-color-secondarycontainer);
    }
  }
}

:global(.input-field-tooltip) {
  .t-popup__content {
    box-shadow: var(--td-shadow-2);
    border: .5px solid var(--td-component-border);
  }
}

:global(.tooltip-with-link) {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 220px;
  font-size: var(--app-text-sm);
}

:global(.tooltip-with-link a) {
  color: var(--td-brand-color);
  font-weight: 500;
  text-decoration: none;
}

:global(.tooltip-with-link a:hover) {
  text-decoration: underline;
}

.websearch-icon {
  width: 18px;
  height: 18px;
}

.dropdown-arrow {
  width: 10px;
  height: 10px;
  margin-left: 2px;
  transition: transform var(--app-motion-instant);

  &.rotate {
    transform: rotate(180deg);
  }
}

.control-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stop-btn, .send-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  box-sizing: border-box;
  font-size: var(--app-text-xl);
  line-height: 1;

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}

.stop-btn, .send-btn {
  background-color: var(--td-brand-color);
  color: #fff;

  &:hover:not(.disabled) {
    background-color: var(--td-brand-color-active);
  }

  &.disabled {
    background-color: var(--td-success-color-light);
  }

  img {
    width: 16px;
    height: 16px;
  }
}

/* Model görüntüleme stili */
.model-selector-trigger.reasoning-effort-trigger {
  flex-shrink: 0;
  min-width: 0;
  box-sizing: content-box;
  background: transparent;
  font: inherit;
}

.reasoning-effort-menu {
  min-width: 120px;
  display: flex;
  flex-direction: column;
  gap: 0;
}

.reasoning-effort-title {
  padding: 6px 8px;
  margin-bottom: 2px;
  border-bottom: .5px solid var(--td-component-stroke);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  font-weight: 500;
  line-height: 20px;
}

.reasoning-effort-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 36px;
  padding: 8px 12px;
  border: 0;
  border-radius: var(--app-menu-item-radius);
  background: transparent;
  color: var(--td-text-color-primary);
  font: inherit;
  font-size: var(--app-text-base);
  line-height: 20px;
  text-align: left;
  cursor: pointer;

  &:hover, &:focus-visible {
    background: var(--td-bg-color-secondarycontainer-hover);
  }

  &.selected {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }
}

.model-display {
  display: flex;
  align-items: center;
  margin-left: auto;
  flex-shrink: 0;

  &.agent-controlled {
    .model-selector-trigger {
      cursor: not-allowed;
      opacity: 0.5;
    }
  }
}

.model-selector-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  min-width: 100px;
  height: 22px;
  border-radius: var(--app-radius-sm);
  border: .5px solid var(--td-component-border);
  transition: background var(--app-motion-instant), border-color var(--app-motion-instant);
  cursor: pointer;

  &:hover {
    background: var(--td-bg-color-secondarycontainer-hover);
  }

  &.disabled {
    opacity: 0.5;
    cursor: not-allowed;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
    }
  }
}

.model-selector-name {
  flex: 1;
  font-size: var(--app-text-sm);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-selector-ctx {
  flex-shrink: 0;
  font-size: var(--app-text-xs);
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-placeholder);
  font-weight: 400;

  &.is-default {
    opacity: 0.85;
  }
}

.model-dropdown-arrow {
  width: 10px;
  height: 10px;
  color: var(--td-text-color-placeholder);
  flex-shrink: 0;
  transition: transform var(--app-motion-instant);

  &.rotate {
    transform: rotate(180deg);
  }
}

.model-selector-trigger.disabled .model-dropdown-arrow {
  color: var(--td-text-color-placeholder);
}

.model-selector-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: transparent;
  touch-action: none;
}

.model-selector-dropdown {
  position: fixed !important;
  z-index: 10000;
  background: var(--td-bg-color-container);
  border: 0;
  border-radius: var(--app-radius-lg);
  box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.16);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  margin: 0 !important;
  padding: 4px !important;
  transform: none !important;
  transform-origin: top left;
  animation: modelSelectorFadeIn 0.15s ease-out;
}

@keyframes modelSelectorFadeIn {
  from {
    opacity: 0;
    transform: scale(0.98);
  }

  to {
    opacity: 1;
    transform: scale(1);
  }
}

.model-selector-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  border-bottom: .5px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
  font-size: var(--app-text-sm);
  font-weight: 500;
  color: var(--td-text-color-secondary);
}

.model-selector-content {
  flex: 1;
  min-height: 0;
  max-height: 260px;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  padding: 6px 8px;
}

.model-selector-add {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--app-radius-sm);
  border: .5px solid transparent;
  background: transparent;
  color: var(--td-brand-color);
  font-size: var(--app-text-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--app-motion-instant);

  .add-icon {
    font-size: var(--app-text-base);
    line-height: 1;
    font-weight: 400;
  }

  &:hover {
    color: var(--td-brand-color-hover);
    background: var(--td-bg-color-secondarycontainer);
  }
}

.model-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 36px;
  box-sizing: border-box;
  padding: 8px 12px;
  cursor: pointer;
  transition: background var(--app-motion-instant);
  border-radius: var(--app-menu-item-radius);
  margin-bottom: 0;

  &:hover,
  &.selected {
    background: var(--td-bg-color-secondarycontainer);
  }

  &.empty {
    color: var(--td-text-color-placeholder);
    cursor: default;
    text-align: center;
    padding: 20px 8px;

    &:hover {
      background: transparent;
    }
  }
}

.model-option-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.model-option-icon {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.model-option-name-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  flex: 1;
}

.model-option-name {
  font-size: var(--app-text-md);
  color: var(--td-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.model-option-raw-name {
  font-size: var(--app-text-xs);
  color: var(--td-text-color-placeholder);
  flex-shrink: 0;
}

.model-option-ctx {
  flex-shrink: 0;
  font-size: var(--app-text-xs);
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  padding: 0 6px;
  border-radius: var(--app-radius-xs);
  line-height: 18px;

  &.is-default {
    color: var(--td-text-color-placeholder);
  }
}

/* Agent modu seçim açılır menüsü */
.agent-mode-selector-overlay {
  position: fixed;
  inset: 0;
  z-index: 9998;
  background: transparent;
  touch-action: none;
}

.agent-mode-selector-dropdown {
  position: fixed !important;
  z-index: 9999;
  background: var(--td-bg-color-container);
  border-radius: var(--app-radius-lg);
  box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.16);
  border: 0;
  overflow: hidden;
  padding: 4px !important;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  margin: 0 !important;
  transform: none !important;
}

.agent-mode-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 36px;
  box-sizing: border-box;
  padding: 8px 12px;
  cursor: pointer;
  transition: background var(--app-motion-instant);
  border-radius: var(--app-menu-item-radius);
  position: relative;
  margin: 0;

  &:hover:not(.disabled) {
    background: var(--td-bg-color-container-hover);
  }

  &.disabled {
    opacity: 0.6;
    cursor: not-allowed;

    &:hover {
      background: transparent;
    }
  }

  &.selected {
    background: var(--td-brand-color-light);

    .agent-mode-option-name {
      color: var(--td-success-color);
      font-weight: 500;
    }
  }
}

.agent-mode-option-main {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex: 1;
  min-width: 0;
}

.agent-mode-option-name {
  font-size: var(--app-text-sm);
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;
  transition: color var(--app-motion-instant);
}

.agent-mode-option-desc {
  font-size: var(--app-text-xs);
  color: var(--td-text-color-secondary);
  line-height: 1.3;
}

.check-icon {
  width: 14px;
  height: 14px;
  color: var(--td-success-color);
  flex-shrink: 0;
  margin-left: 6px;
}

.agent-mode-warning {
  display: flex;
  align-items: center;
  margin-left: 6px;

  .warning-icon {
    color: var(--td-warning-color);
    font-size: var(--app-text-base);
  }
}

.agent-mode-footer {
  padding: 6px 10px;
  border-top: 1px solid var(--td-component-border);
  margin-top: 2px;
  background: var(--td-bg-color-secondarycontainer);
}

.agent-mode-link {
  color: var(--td-success-color);
  text-decoration: none;
  font-size: var(--app-text-xs);
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  transition: all var(--app-motion-instant);

  &:hover {
    color: var(--td-brand-color-active);
    text-decoration: underline;
  }
}
</style>
