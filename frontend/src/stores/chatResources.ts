import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { listKnowledgeBases, getKnowledgeBaseById } from '@/api/knowledge-base'
import { listAgents, type CustomAgent } from '@/api/agent'
import { listModels, type ModelConfig } from '@/api/model'
import { listWebSearchProviders, type WebSearchProviderEntity } from '@/api/web-search-provider'
import { isNamedSandboxBackend, listSandboxConfigs, type SandboxConfigRecord } from '@/api/system'
import { useOrganizationStore } from '@/stores/organization'
import { getCurrentLanguage } from '@/utils/request'
import { isLocalizedSnapshotUsable, shouldForceLocalizedRefetch } from './localizedResourceCache'
import { createCachedResource, createKeyedSnapshotCache } from './resourceCache'

type ResourceKey = 'knowledgeBases' | 'agents' | 'models' | 'webSearchProviders' | 'sandboxConfigs'

export type ListCreatorFilter = 'all' | 'mine' | 'others'

function isKbModelReady(kb: any): boolean {
  if (!kb.summary_model_id || kb.summary_model_id === '') return false
  const strategy = kb.indexing_strategy
  const needsEmbedding = !strategy || strategy.vector_enabled || strategy.keyword_enabled
  if (needsEmbedding && (!kb.embedding_model_id || kb.embedding_model_id === '')) return false
  return true
}

/**
 * Alan düzeyindeki kaynakların (bilgi tabanı / akıllı ajan / model / çevrimiçi arama / sandbox arka ucu) paylaşılan anlık görüntüsü.
 *
 * TTL yoktur: her `ensureX()` istek gönderir (aynı anda yalnızca biri devam eder; eşzamanlı bağlanma dalgası onu paylaşır),
 * yazma işleminden sonra çağıran, `ensureX(true)`, `replaceModels()` veya `invalidate()` ile açıkça yeniler.
 * Sayfa, çizim için buradaki ref'i doğrudan okur; yeni veriler gelene kadar eski anlık görüntü görünmeye devam eder.
 */
export const useChatResourcesStore = defineStore('chatResources', () => {
  const rawKnowledgeBases = ref<any[]>([])
  const agents = ref<CustomAgent[]>([])
  const disabledOwnAgentIds = ref<string[]>([])
  const allModels = ref<ModelConfig[]>([])
  const webSearchProviders = ref<WebSearchProviderEntity[]>([])
  const sandboxConfigs = ref<SandboxConfigRecord[]>([])

  const validKnowledgeBases = computed(() => rawKnowledgeBases.value.filter(isKbModelReady))
  const chatModels = computed(() => allModels.value.filter((m) => m.type === 'KnowledgeQA'))

  const knowledgeBasesResource = createCachedResource(
    async () => {
      const res: any = await listKnowledgeBases()
      return res?.data && Array.isArray(res.data) ? (res.data as any[]) : []
    },
    (data) => {
      rawKnowledgeBases.value = data
    },
  )

  // Yerleşik akıllı ajanların adları/açıklamaları, arka uç tarafından Accept-Language'a göre yerelleştirilerek döndürülür; anlık görüntü yalnızca çekildiği andaki
  // UI dili geçerli. agentsLoadedLocale, «istek başlatıldığı andaki» dil olmalıdır; await sonrasında yeniden okunamaz.
  let agentsLoadedLocale = ''
  let agentsInflightLocale = ''
  const agentsResource = createCachedResource(
    async () => {
      const locale = getCurrentLanguage()
      agentsInflightLocale = locale
      const res = (await listAgents()) as { data?: CustomAgent[]; disabled_own_agent_ids?: string[] }
      return { data: res.data || [], disabled: res.disabled_own_agent_ids || [], locale }
    },
    ({ data, disabled, locale }) => {
      agents.value = data
      disabledOwnAgentIds.value = disabled
      agentsLoadedLocale = locale
    },
  )

  watch(
    () => getCurrentLanguage(),
    (locale) => {
      if (agentsLoadedLocale && agentsLoadedLocale !== locale) {
        agentsResource.invalidate()
        agentsLoadedLocale = ''
      }
    },
  )

  const modelsResource = createCachedResource(
    async () => {
      const models = await listModels()
      return Array.isArray(models) ? models : []
    },
    (models) => {
      allModels.value = models
    },
  )

  const webSearchProvidersResource = createCachedResource(
    async () => {
      const response = await listWebSearchProviders()
      const providers = (response as any)?.data
      return Array.isArray(providers) ? (providers as WebSearchProviderEntity[]) : []
    },
    (providers) => {
      webSearchProviders.value = providers
    },
  )

  // Hata yalnızca yutulur, fırlatılmaz: bu isteğe bağlı bir kaynaktır — alınamazsa geriye yalnızca «sandbox'ı etkinleştirme» seçeneği kalır,
  // Ajan yine de düzenleyip kaydedebilir. Çağıran taraf bunu genellikle bir sürü zorunlu kaynakla aynı
  // Promise.all içinde tutar; burada hata fırlatılırsa tüm düzenleyicinin bağımlılık yüklemesi de etkilenir
  // (beceri kullanılabilirliği alınamazsa ⇒ beceri yapılandırma grubu doğrudan kaybolur).
  const sandboxConfigsResource = createCachedResource(
    async () => {
      try {
        const res = await listSandboxConfigs()
        const rows = Array.isArray(res?.data) ? res.data : []
        return rows.filter((cfg) => isNamedSandboxBackend(cfg.sandbox_type))
      } catch {
        return [] as SandboxConfigRecord[]
      }
    },
    (rows) => {
      sandboxConfigs.value = rows
    },
  )

  const resources: Record<ResourceKey, ReturnType<typeof createCachedResource>> = {
    knowledgeBases: knowledgeBasesResource,
    agents: agentsResource,
    models: modelsResource,
    webSearchProviders: webSearchProvidersResource,
    sandboxConfigs: sandboxConfigsResource,
  }

  // Ajanın görebildiği bilgi tabanları ve tekil bilgi tabanı ayrıntıları: bir kez alındıysa açıkça geçersiz kılınana kadar kullanılmaya devam edilir.
  // @ bahsetme listesi count değerini tek tek tamamlar; key temelli bu tür okumalar her seferinde API'ye istek atmamalıdır.
  const agentKbCache = createKeyedSnapshotCache(async (cacheKey: string) => {
    const [agentId, sourceTenantId] = cacheKey.split(':')
    const res: any = await listKnowledgeBases({
      agent_id: agentId,
      agent_source_tenant_id: sourceTenantId === 'current' ? undefined : sourceTenantId,
    })
    return res?.data && Array.isArray(res.data) ? (res.data as any[]) : []
  })
  const kbDetailCache = createKeyedSnapshotCache(async (kbId: string) => {
    try {
      const res: any = await getKnowledgeBaseById(kbId)
      return res?.data ?? null
    } catch {
      return null
    }
  })

  /** Zaten bir anlık görüntü olup olmadığı («taze olup olmadığı»ndan farklıdır: bu store artık tazelik penceresine sahip değildir).*/
  function isLoaded(key: ResourceKey): boolean {
    if (key === 'agents') {
      return isLocalizedSnapshotUsable(agentsResource.isLoaded(), agentsLoadedLocale, getCurrentLanguage())
    }
    return resources[key].isLoaded()
  }

  /**
   * Bilgi tabanı listesi (creator filtresini destekler). creator=all, sohbet sayfasında yeniden kullanılmak üzere paylaşılan anlık görüntüyü kullanır.
   */
  async function fetchKnowledgeBasesForList(
    params?: { creator?: ListCreatorFilter },
    force = false,
  ): Promise<any[]> {
    const creator = params?.creator ?? 'all'
    // creator filtreli liste yalnızca liste sayfasına özeldir, anlık görüntüye girmez ve isteği doğrudan iletir.
    // Yazma işleminden sonraki zorunlu yenileme, tam anlık görüntüyü de yenilemelidir; aksi hâlde sohbet giriş çubuğu silinmiş bilgi tabanını görür.
    if (creator !== 'all') {
      if (force) void ensureKnowledgeBases(true).catch(() => {})
      const res: any = await listKnowledgeBases({ creator })
      return res?.data && Array.isArray(res.data) ? res.data : []
    }
    await ensureKnowledgeBases(force)
    return rawKnowledgeBases.value
  }

  // Paylaşılan kaynak ek yenilemedir: API'si çökerse kendi listesini de etkilememelidir; aksi hâlde çağıranın catch'i
  // zaten alınmış kendi bilgi tabanlarını / ajanlarını birlikte temizler (manuel belge düzenleyicisi, API entegrasyonu sayfası).
  // Paylaşılan liste başarısız olduğunda önceki anlık görüntü korunur; ayrı bir zorunlu yenileme gerekip gerekmediğine sayfa kendisi karar verir.
  const swallow = (p: Promise<unknown>) => p.catch(() => undefined)

  async function ensureKnowledgeBases(force = false): Promise<void> {
    const orgStore = useOrganizationStore()
    await Promise.all([
      knowledgeBasesResource.ensure(force),
      swallow(orgStore.fetchSharedKnowledgeBases({ force })),
    ])
  }

  /**
   * Ajan listesi (creator filtresini destekler). creator=all, paylaşılan anlık görüntüyü kullanır.
   */
  async function fetchAgentsForList(
    params?: { creator?: ListCreatorFilter },
    force = false,
  ): Promise<{ data: CustomAgent[]; disabled_own_agent_ids: string[] }> {
    const creator = params?.creator ?? 'all'
    const orgStore = useOrganizationStore()

    // creator filtreli liste anlık görüntüye girmez, ancak yine de paylaşılan ajanlar yenilenmelidir (tam liste yoluyla tutarlı kalır).
    if (creator !== 'all') {
      if (force) void ensureAgents(true).catch(() => {})
      const [agentsRes] = await Promise.all([
        listAgents({ creator }),
        orgStore.fetchSharedAgents({ force }),
      ])
      const res = agentsRes as { data?: CustomAgent[]; disabled_own_agent_ids?: string[] }
      return { data: res.data || [], disabled_own_agent_ids: res.disabled_own_agent_ids || [] }
    }

    await ensureAgents(force)
    return { data: agents.value, disabled_own_agent_ids: disabledOwnAgentIds.value }
  }

  async function ensureAgents(force = false): Promise<void> {
    const orgStore = useOrganizationStore()
    const locale = getCurrentLanguage()
    // Devam eden istek başka bir dille başlatılmıştır: yeniden kullanılamaz; arkasından sıraya alınıp tekrar gönderilmelidir.
    const mustForce =
      force ||
      shouldForceLocalizedRefetch(agentsResource.hasInFlightRequest(), agentsInflightLocale, locale)
    await Promise.all([
      agentsResource.ensure(mustForce),
      swallow(orgStore.fetchSharedAgents({ force })),
    ])
  }

  async function ensureModels(force = false): Promise<void> {
    return modelsResource.ensure(force)
  }

  /** Anlık görüntüyü yeni çekilen listeyle değiştirin (ayarlar sayfasında model ekleme/silme/değiştirme sonrası çağrılır) ve devam eden eski istekleri geçersiz kılın.*/
  function replaceModels(models: ModelConfig[]) {
    allModels.value = Array.isArray(models) ? models : []
    modelsResource.markLoaded()
  }

  /** @deprecated ensureModels kullanın; sohbet giriş çubuğunun çağrısı için takma ad korunur*/
  async function ensureChatModels(force = false): Promise<void> {
    return ensureModels(force)
  }

  async function ensureWebSearchProviders(force = false): Promise<void> {
    return webSearchProvidersResource.ensure(force)
  }

  /**
   * Sandbox arka uç yapılandırması; ajan düzenleyicisindeki arka uç seçicisi tarafından kullanılır.
   * prefetchChatInput içine girmez: yalnızca ajan düzenlenirken gerekir, sohbet giriş çubuğunda gerekmez.
   */
  async function ensureSandboxConfigs(force = false): Promise<void> {
    return sandboxConfigsResource.ensure(force)
  }

  /** Sohbet giriş çubuğu ve liste sayfasında sık kullanılan alan düzeyindeki kaynakları paralel olarak önceden getirir*/
  async function prefetchChatInput(force = false): Promise<void> {
    const orgStore = useOrganizationStore()
    await Promise.all([
      ensureKnowledgeBases(force),
      ensureAgents(force),
      ensureModels(force),
      ensureWebSearchProviders(force),
      orgStore.fetchOrganizations({ force }),
    ])
  }

  async function ensureAgentKnowledgeBases(agentId: string, sourceTenantId?: string, force = false): Promise<any[]> {
    return agentKbCache.ensure(`${agentId}:${sourceTenantId || 'current'}`, force)
  }

  /** Tekil bilgi tabanı ayrıntıları (kenar çubuğu + ayrıntı sayfası ortak kullanır, eşzamanlı istekleri tekilleştirir)*/
  async function fetchKnowledgeBaseById(kbId: string, force = false): Promise<any | null> {
    if (!kbId) return null
    return kbDetailCache.ensure(kbId, force)
  }

  function invalidateKnowledgeBaseDetail(kbId?: string) {
    kbDetailCache.invalidate(kbId)
  }

  function invalidate(...keys: ResourceKey[]) {
    if (keys.length === 0) {
      ;(Object.keys(resources) as ResourceKey[]).forEach((k) => resources[k].invalidate())
      rawKnowledgeBases.value = []
      agents.value = []
      disabledOwnAgentIds.value = []
      allModels.value = []
      webSearchProviders.value = []
      sandboxConfigs.value = []
      agentsLoadedLocale = ''
      agentsInflightLocale = ''
      agentKbCache.invalidate()
      invalidateKnowledgeBaseDetail()
      return
    }
    keys.forEach((k) => resources[k].invalidate())
    if (keys.includes('knowledgeBases')) {
      agentKbCache.invalidate()
      invalidateKnowledgeBaseDetail()
    }
    if (keys.includes('agents')) {
      agentsLoadedLocale = ''
    }
  }

  return {
    rawKnowledgeBases,
    validKnowledgeBases,
    agents,
    disabledOwnAgentIds,
    allModels,
    chatModels,
    webSearchProviders,
    sandboxConfigs,
    isLoaded,
    fetchKnowledgeBasesForList,
    fetchAgentsForList,
    ensureKnowledgeBases,
    ensureAgents,
    ensureModels,
    replaceModels,
    ensureChatModels,
    ensureWebSearchProviders,
    ensureSandboxConfigs,
    ensureAgentKnowledgeBases,
    prefetchChatInput,
    fetchKnowledgeBaseById,
    invalidateKnowledgeBaseDetail,
    invalidate,
  }
})
