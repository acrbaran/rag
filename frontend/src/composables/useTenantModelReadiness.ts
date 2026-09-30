import { computed, onMounted, ref } from 'vue'
import { useChatResourcesStore } from '@/stores/chatResources'
import { evaluateTenantModelReadiness } from '@/utils/tenantModelReadiness'

/**
 * Mevcut alanın bilgi tabanı / akıllı ajan oluşturmak için gereken modelleri yapılandırıp yapılandırmadığı.
 *
 * Doğrudan chatResources içindeki model anlık görüntüsünden türetilir: ayarlar sayfasında model eklenip silindikten sonra `replaceModels` çağrılır,
 * buradaki computed buna göre güncellenir; yeniden çekmek için ayarlar penceresinin kapanması gibi dolaylı sinyalleri dinlemeye gerek yoktur.
 */
export function useTenantModelReadiness() {
  const chatResources = useChatResourcesStore()
  const loaded = ref(false)
  const loading = ref(false)

  const readiness = computed(() => evaluateTenantModelReadiness(chatResources.allModels))

  const refresh = async (force = false) => {
    loading.value = true
    try {
      await chatResources.ensureModels(force)
    } finally {
      loading.value = false
      loaded.value = true
    }
  }

  onMounted(() => {
    refresh()
  })

  const isReadyForDocumentKb = computed(() => readiness.value.isReadyForDocumentKb)

  const isReadyForAgent = computed(() => readiness.value.isReadyForAgent)

  const hasChat = computed(() => readiness.value.hasChat)

  return {
    readiness,
    loaded,
    loading,
    refresh,
    isReadyForDocumentKb,
    isReadyForAgent,
    hasChat,
  }
}
