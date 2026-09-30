import type { ModelConfig } from '@/api/model'

export interface TenantModelReadiness {
  chatCount: number
  embeddingCount: number
  hasChat: boolean
  hasEmbedding: boolean
  /** Belge kütüphanesinde vektör/anahtar kelime araması varsayılan olarak etkin olduğunda gereken modellerin eksiksiz olup olmadığı*/
  isReadyForDocumentKb: boolean
  /** Bir agent oluşturmak için en azından bir sohbet modeli gerekir.*/
  isReadyForAgent: boolean
}

export function evaluateTenantModelReadiness(models: ModelConfig[]): TenantModelReadiness {
  const chatCount = models.filter((m) => m.type === 'KnowledgeQA').length
  const embeddingCount = models.filter((m) => m.type === 'Embedding').length
  const hasChat = chatCount > 0
  const hasEmbedding = embeddingCount > 0
  return {
    chatCount,
    embeddingCount,
    hasChat,
    hasEmbedding,
    isReadyForDocumentKb: hasChat && hasEmbedding,
    isReadyForAgent: hasChat,
  }
}
