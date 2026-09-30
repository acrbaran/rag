import { get, post, postUpload, put, del } from '../../utils/request';
import i18n from '@/i18n'
import { ModelInUseError, modelInUseErrorFromRequest } from './modelUsage'

export * from './modelUsage'

const t = (key: string) => i18n.global.t(key)

// Protocol-neutral thinking level. Mirrors internal/models/api.ReasoningEffort.
export type ReasoningEffortLevel =
  | 'off'
  | 'auto'
  | 'minimal'
  | 'low'
  | 'medium'
  | 'high'
  | 'xhigh'
  | 'max'

// Catalog view of a saved chat/VLM model. Mirrors internal/models/catalog.Capabilities.
// thinking_levels: empty array = the model cannot be asked to think.
export interface ModelCapabilities {
  provider: string;
  api: string;
  cataloged: boolean;
  reasoning: boolean;
  thinking_levels: ReasoningEffortLevel[];
  thinking_format: string;
  input?: string[];
  context_window?: number;
  max_output_tokens?: number;
  max_tokens_field?: string;
}

// Per-row override of a catalog entry. Mirrors internal/types.ModelSpecOverride.
// compat is the flat, protocol-specific object defined in
// internal/models/api/*_settings.go (free-form JSON).
export interface ModelSpecOverride {
  api?: string;
  reasoning?: boolean;
  input?: string[];
  context_window?: number;
  max_output_tokens?: number;
  thinking_levels?: Record<string, string | null>;
  compat?: Record<string, unknown>;
}

// Model türü tanımları
export interface ModelConfig {
  id?: string;
  tenant_id?: number;
  name: string;
  display_name?: string;
  type: 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM' | 'ASR';
  source: 'remote';
  description?: string;
  parameters: {
    base_url?: string;
    api_key?: string;
    provider?: string; // Provider identifier: openai, aliyun, zhipu, generic
    embedding_parameters?: {
      dimension?: number;
      truncate_prompt_tokens?: number;
      supports_dimension_override?: boolean;
    };
    interface_type?: 'openai'; // Yalnızca VLLM için
    parameter_size?: string;
    extra_config?: Record<string, string>; // Provider-specific configuration
    // Özel HTTP istek başlıkları (Python OpenAI SDK'deki extra_headers benzeri),
    // Uzak model API'si çağrılırken her isteğe eklenir. Authorization, Content-Type gibi ayrılmış başlıklar yok sayılır.
    custom_headers?: Record<string, string>;
    supports_vision?: boolean; // Whether the model accepts image/multimodal input
    // Sohbet/VLM bağlam penceresi (token). 0 veya boş bırakılması, arka ucun varsayılan 200000 değerinin kullanılacağı anlamına gelir.
    context_window?: number;
    max_output_tokens?: number;
    // Bu model için arka plan görevlerinin (veri alma/zenginleştirme) eşzamanlılık üst sınırı; model kimliğine göre tüm kopyalar arasında paylaşılır.
    // 0 veya boş bırakılması, genel varsayılanın (model.max_concurrency) kullanılacağı anlamına gelir; yalnızca chat/embedding/vllm için geçerlidir.
    max_concurrency?: number;
    app_id?: string;
    // Secret fields (api_key, app_secret) are never returned by the server in
    // this shape — they live behind the /credentials subresource. They are
    // kept on the type so create-mode payloads can still carry them in the
    // initial POST body.
    app_secret?: string;
    // Per-model catalog override (protocol, limits, protocol compat knobs).
    spec?: ModelSpecOverride;
  };
  // Catalog view (chat / VLM remote models only): protocol, thinking levels,
  // context window. Computed by the backend from provider + name + overrides.
  capabilities?: ModelCapabilities;
  is_default?: boolean;
  is_builtin?: boolean;
  status?: string;
  // Per-field configured? metadata from the main response. For builtin
  // models it is returned only to system administrators.
  credentials?: Record<ModelCredentialField, { configured: boolean }>;
  created_at?: string;
  updated_at?: string;
  deleted_at?: string | null;
}

// Modeli kopyala. Yalnızca görünen adı gönderin; name ve kimlik bilgileri sunucu tarafından kaynak modelden kopyalanır.
export function copyModelConfig(id: string, displayName: string): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    post(`/api/v1/models/${id}/copy`, { display_name: displayName })
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || ''));
        }
      })
      .catch((error: any) => {
        console.error('Failed to copy model:', error);
        reject(error);
      });
  });
}

// Model oluştur
export function createModel(data: ModelConfig): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    post('/api/v1/models', data)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || t('error.model.createFailed')));
        }
      })
      .catch((error: any) => {
        console.error('Failed to create model:', error);
        reject(error);
      });
  });
}

export async function discoverModels(baseUrl: string, apiKey: string): Promise<string[]> {
  const url = new URL(baseUrl.replace(/\/+$/, '') + '/models')
  if (!['https:', 'http:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) {
    throw new Error('Invalid Base URL')
  }
  let response: Response
  try {
    response = await fetch(url, {
      headers: { Accept: 'application/json', ...(apiKey ? { Authorization: `Bearer ${apiKey}` } : {}) },
    })
  } catch (error) {
    if (!(error instanceof TypeError)) throw error
    // Browsers blocked by provider CORS can still use the server-side route.
    const response = await post<{ success: boolean; data: string[] }>(
      '/api/v1/models/discover', { base_url: baseUrl, api_key: apiKey },
    )
    return response.data
  }
  if (!response.ok) throw new Error(`Model API returned HTTP ${response.status}`)
  const result = await response.json()
  if (!Array.isArray(result.data)) throw new Error('Invalid model list response')
  const ids: string[] = (result.data as Array<{ id?: unknown }>).flatMap((model) => typeof model?.id === 'string' ? [model.id.trim()] : [])
  return [...new Set<string>(ids.filter(Boolean))].sort()
}

// Model listesini al
export function listModels(type?: string): Promise<ModelConfig[]> {
  return new Promise((resolve, reject) => {
    const url = `/api/v1/models`;
    get(url)
      .then((response: any) => {
        if (response.success && response.data) {
          if (type) {
            response.data = response.data.filter((item: ModelConfig) => item.type === type);
          }
          resolve(response.data);
        } else {
          resolve([]);
        }
      })
      .catch((error: any) => {
        console.error('Failed to list models:', error);
        // Yutmak yerine fırlat: çağıran taraf (önbellek katmanı dahil) ancak bu şekilde «gerçek hata» ile «başarılı ama model yok» durumlarını ayırt edebilir,
        // Tek seferlik anlık bir hatanın boş sonucunun önbelleğe alınmasını önler. Tüm UI çağrı noktalarında try/catch yedeklemesi vardır.
        reject(error);
      });
  });
}

// Tek bir modeli al
export function getModel(id: string): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    get(`/api/v1/models/${id}`)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || t('error.model.getFailed')));
        }
      })
      .catch((error: any) => {
        console.error('Failed to get model:', error);
        reject(error);
      });
  });
}

// Modeli güncelle
export function updateModel(id: string, data: Partial<ModelConfig>): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    put(`/api/v1/models/${id}`, data)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || t('error.model.updateFailed')));
        }
      })
      .catch((error: any) => {
        console.error('Failed to update model:', error);
        reject(error);
      });
  });
}

// Modeli sil
export function deleteModel(id: string): Promise<void> {
  return new Promise((resolve, reject) => {
    del(`/api/v1/models/${id}`)
      .then((response: any) => {
        if (response.success) {
          resolve();
        } else {
          const conflict = modelInUseErrorFromRequest(response)
          if (conflict) {
            reject(conflict)
            return
          }
          reject(new Error(response.message || t('error.model.deleteFailed')));
        }
      })
      .catch((error: any) => {
        console.error('Failed to delete model:', error);
        if (error instanceof ModelInUseError) {
          reject(error)
          return
        }
        const conflict = modelInUseErrorFromRequest(error)
        if (conflict) {
          reject(conflict)
          return
        }
        reject(error);
      });
  });
}

export interface ModelDebugOptions {
  system_prompt?: string
  temperature?: number
  top_p?: number
  max_tokens?: number
  thinking?: boolean
  // Graded thinking level; takes precedence over the boolean when set.
  reasoning_effort?: ReasoningEffortLevel | string
}

export interface ModelDebugResult {
  ok: boolean
  elapsed_ms: number
  request: Record<string, unknown>
  raw_response: unknown
  observations: Record<string, unknown>
  error?: string
}

export async function debugModel(
  id: string,
  data: {
    input?: string
    documents?: string[]
    options?: ModelDebugOptions
    file?: File | null
  },
): Promise<ModelDebugResult> {
  const form = new FormData()
  form.append('input', data.input || '')
  form.append('documents', JSON.stringify(data.documents || []))
  form.append('options', JSON.stringify(data.options || {}))
  if (data.file) form.append('file', data.file)
  const response: any = await postUpload(
    `/api/v1/models/${id}/debug`,
    form,
    undefined,
    { timeout: 300000 },
  )
  if (response?.success && response?.data) return response.data
  throw new Error(response?.message || t('error.model.getFailed'))
}

// ----------------------------------------------------------------------------
// Model credential subresource. See mcp-service.ts for the matching MCP API
// shape and the design notes in internal/handler/dto/mcp.go.
// ----------------------------------------------------------------------------

export type ModelCredentialField = 'api_key' | 'app_secret'

export interface ModelCredentialsResponse {
  fields: Record<ModelCredentialField, { configured: boolean }>
}

export async function putModelCredentials(
  id: string,
  body: Partial<Record<ModelCredentialField, string>>,
): Promise<ModelCredentialsResponse> {
  const response: any = await put(`/api/v1/models/${id}/credentials`, body)
  return (response.data ?? response) as ModelCredentialsResponse
}

export async function deleteModelCredentialField(
  id: string,
  field: ModelCredentialField,
): Promise<void> {
  await del(`/api/v1/models/${id}/credentials/${field}`)
}
