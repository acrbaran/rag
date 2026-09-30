import { get, post, put } from '../../utils/request';
import i18n from '@/i18n'
import type { ModelCapabilities, ModelSpecOverride, ReasoningEffortLevel } from '../model'

const t = (key: string) => i18n.global.t(key)

// GET /initialization/config/:kbId exposes credential presence, not values.
export interface ModelCredentialStatus {
    apiKey?: boolean;
}

export interface COSCredentialStatus {
    secretId?: boolean;
    secretKey?: boolean;
}

// Yapılandırma veri türünü başlat
export interface InitializationConfig {
    llm: {
        source: string;
        modelName: string;
        baseUrl?: string;
        /** @deprecated Use credentials.apiKey from GET responses */
        apiKey?: string;
        credentials?: ModelCredentialStatus;
    };
    embedding: {
        source: string;
        modelName: string;
        baseUrl?: string;
        /** @deprecated Use credentials.apiKey from GET responses */
        apiKey?: string;
        dimension?: number; // embedding boyutu alanı ekle
        credentials?: ModelCredentialStatus;
    };
    rerank: {
        modelName: string;
        baseUrl: string;
        /** @deprecated Use credentials.apiKey from GET responses */
        apiKey?: string;
        enabled: boolean;
        credentials?: ModelCredentialStatus;
    };
    multimodal: {
        enabled: boolean;
        storageType: 'cos' | 'minio';
        vlm?: {
            modelName: string;
            baseUrl: string;
            /** @deprecated Use credentials.apiKey from GET responses */
            apiKey?: string;
            interfaceType?: string; // "openai"
            credentials?: ModelCredentialStatus;
        };
        cos?: {
            region: string;
            bucketName: string;
            appId: string;
            pathPrefix?: string;
            /** @deprecated Use credentials from GET responses */
            secretId?: string;
            /** @deprecated Use credentials from GET responses */
            secretKey?: string;
            credentials?: COSCredentialStatus;
        };
        minio?: {
            bucketName: string;
            pathPrefix?: string;
        };
    };
    documentSplitting: {
        chunkSize: number;
        chunkOverlap: number;
        separators: string[];
        // Adaptive chunking strategy. Empty / "legacy" = classic recursive splitter.
        // "auto" lets the backend profiler pick a tier; "heading" / "heuristic"
        // pin the tier explicitly. See backend chunker package for details.
        strategy?: string;
        // Cap chunk size in approx tokens. 0 = char-based budget only.
        tokenLimit?: number;
        // Language hints for heuristic patterns ("de", "en", "zh"). Empty = auto-detect.
        languages?: string[];
    };
    // Frontend-only hint for storage selection UI
    storageType?: 'cos' | 'minio';
    nodeExtract: {
        enabled: boolean,
        text: string,
        tags: string[],
        nodes: Node[],
        relations: Relation[]
    }
}

// İndirme görevi durum türü


// Basitleştirilmiş bilgi tabanı yapılandırma güncelleme arayüzü (yalnızca model ID gönderilir)
export interface KBModelConfigRequest {
    llmModelId: string
    embeddingModelId: string
    vlm_config?: {
        enabled: boolean
        model_id?: string
        description_language?: string
        custom_instructions?: string
    }
    asr_config?: {
        enabled: boolean
        model_id?: string
        language?: string
    }
    documentSplitting: {
        chunkSize: number
        chunkOverlap: number
        separators: string[]
        parserEngineRules?: {
            file_types: string[]
            engine: string
            xlsx_first_row_as_header?: boolean
        }[]
        enableParentChild?: boolean
        parentChunkSize?: number
        childChunkSize?: number
        // Adaptive chunking strategy ("auto" | "heading" | "heuristic" | "legacy").
        // The backend uses pointer-based DTOs for these three fields:
        // - undefined / not set in payload → no change on server
        // - "" / 0 / [] explicitly sent     → clears the value
        // Send the field whenever the user has opened the editor — even
        // empty values — so the user can always reset back to defaults.
        strategy?: string
        // Approximate token budget per chunk; 0 = char-based.
        tokenLimit?: number
        // Language hints for heuristic patterns. Empty array = auto-detect.
        languages?: string[]
        tableMetadataInstructions?: string
    }
    multimodal: {
        enabled: boolean
    }
    /** Depolama motoru seçimi: "local" | "minio" | "cos" | "obs" vb.; belge yüklemeyi ve belge içindeki görsellerin depolanmasını etkiler*/
    storageBackendId?: string
    storageProvider?: string
    nodeExtract: {
        enabled: boolean
        text: string
        tags: string[]
        nodes: Node[]
        relations: Relation[]
        customInstructions?: string
    }
    questionGeneration?: {
        enabled: boolean
        questionCount: number
        customInstructions?: string
    }
}

export function updateKBConfig(kbId: string, config: KBModelConfigRequest): Promise<any> {
    return new Promise((resolve, reject) => {
        console.log('Starting KB config update (simplified)...', kbId, config);
        put(`/api/v1/initialization/config/${kbId}`, config)
            .then((response: any) => {
                console.log('KB config update completed', response);
                resolve(response);
            })
            .catch((error: any) => {
                console.error('Failed to update KB config:', error);
                reject(error.error || error);
            });
    });
}

// Bilgi tabanı ID'sine göre yapılandırma güncellemesini gerçekleştirir (eski sürüm, uyumluluk için korunur)
export function initializeSystemByKB(kbId: string, config: InitializationConfig): Promise<any> {
    return new Promise((resolve, reject) => {
        console.log('Starting KB config update...', kbId, config);
        post(`/api/v1/initialization/initialize/${kbId}`, config)
            .then((response: any) => {
                console.log('KB config update completed', response);
                resolve(response);
            })
            .catch((error: any) => {
                console.error('Failed to update KB config:', error);
                reject(error.error || error);
            });
    });
}

export function getCurrentConfigByKB(kbId: string): Promise<InitializationConfig & { hasFiles: boolean }> {
    return new Promise((resolve, reject) => {
        get(`/api/v1/initialization/config/${kbId}`)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to get KB config:', error);
                reject(error);
            });
    });
}

// Tüm "bağlantıyı test et" arayüzlerinin ortak kullandığı genel isteğe bağlı parametreler.
// customHeaders / extraConfig / interfaceType, arka uçtaki ModelTestRequest içindeki aynı adlı alanlara karşılık gelir,
// Gerçek model oluşturma sürecine doğrudan iletilir; böylece bağlantı testi ve üretim çağrıları tamamen aynı yolu izler.
interface BaseModelTestPayload {
    spec?: ModelSpecOverride;
    customHeaders?: Record<string, string>;
    extraConfig?: Record<string, string>;
    interfaceType?: string;
    /** İkinci anahtar (ör. LKEAP Rerank için Tencent Cloud SecretKey)*/
    appSecret?: string;
}

// Uzak API modelini kontrol et
export function checkRemoteModel(modelConfig: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    // Mevcut bir modeli düzenlerken modelId gönderilir; arka uç apiKey değerini depolamadan otomatik olarak getirir
    // (Ön uç artık düz metin anahtarı yeniden göstermez; bu nedenle bağlantı testi bu geri doldurma yolunu kullanmalıdır)
    modelId?: string;
} & BaseModelTestPayload): Promise<{
    available: boolean;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/remote/check', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to check remote model:', error);
                reject(error);
            });
    });
}

// Uzak Embedding modelinin kullanılabilir olup olmadığını test et
export function testEmbeddingModel(modelConfig: {
    source: 'remote';
    modelName: string;
    baseUrl?: string;
    apiKey?: string;
    dimension?: number;
    supportsDimensionOverride?: boolean;
    provider?: string;
    modelId?: string;
} & BaseModelTestPayload): Promise<{ available: boolean; message?: string; dimension?: number }> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/embedding/test', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to test Embedding model:', error);
                reject(error);
            });
    });
}


export function checkRerankModel(modelConfig: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    modelId?: string;
} & BaseModelTestPayload): Promise<{
    available: boolean;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/rerank/check', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to check Rerank model:', error);
                reject(error);
            });
    });
}

// ASR model bağlantısını kontrol et (/v1/audio/transcriptions uç noktası üzerinden test edilir)
export function checkASRModel(modelConfig: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    modelId?: string;
} & BaseModelTestPayload): Promise<{
    available: boolean;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/asr/check', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to check ASR model:', error);
                reject(error);
            });
    });
}

export function testMultimodalFunction(testData: {
    image: File;
    vlm_model: string;
    vlm_base_url: string;
    vlm_api_key?: string;
    vlm_interface_type?: string;
    storage_type?: 'cos' | 'minio';
    // COS optional fields (required only when storage_type === 'cos')
    cos_secret_id?: string;
    cos_secret_key?: string;
    cos_region?: string;
    cos_bucket_name?: string;
    cos_app_id?: string;
    cos_path_prefix?: string;
    // MinIO optional fields
    minio_bucket_name?: string;
    minio_path_prefix?: string;
    chunk_size: number;
    chunk_overlap: number;
    separators: string[];
}): Promise<{
    success: boolean;
    caption?: string;
    ocr?: string;
    processing_time?: number;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('image', testData.image);
        formData.append('vlm_model', testData.vlm_model);
        formData.append('vlm_base_url', testData.vlm_base_url);
        if (testData.vlm_api_key) {
            formData.append('vlm_api_key', testData.vlm_api_key);
        }
        if (testData.vlm_interface_type) {
            formData.append('vlm_interface_type', testData.vlm_interface_type);
        }
        if (testData.storage_type) {
            formData.append('storage_type', testData.storage_type);
        }
        // Append COS fields only when storage_type is COS
        if (testData.storage_type === 'cos') {
            if (testData.cos_secret_id) formData.append('cos_secret_id', testData.cos_secret_id);
            if (testData.cos_secret_key) formData.append('cos_secret_key', testData.cos_secret_key);
            if (testData.cos_region) formData.append('cos_region', testData.cos_region);
            if (testData.cos_bucket_name) formData.append('cos_bucket_name', testData.cos_bucket_name);
            if (testData.cos_app_id) formData.append('cos_app_id', testData.cos_app_id);
            if (testData.cos_path_prefix) formData.append('cos_path_prefix', testData.cos_path_prefix);
        }
        // MinIO fields
        if (testData.minio_bucket_name) formData.append('minio_bucket_name', testData.minio_bucket_name);
        if (testData.minio_path_prefix) formData.append('minio_path_prefix', testData.minio_path_prefix);
        formData.append('chunk_size', testData.chunk_size.toString());
        formData.append('chunk_overlap', testData.chunk_overlap.toString());
        formData.append('separators', JSON.stringify(testData.separators));

        // Kimlik doğrulama Token'ını al
        const token = localStorage.getItem('rethra_token');
        const headers: Record<string, string> = {};
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        // Alanlar arası erişim istek başlığı: doğrudan eklenir; short-circuit "selectedTenantId
        // === defaultTenantId durumunda eklenmez" ifadesinin bazı uç durumlarda header'ın sessizce kaybolmasına yol açması önlenir.
        // utils/request.ts ve api/chat/streame.ts davranışıyla tutarlıdır.
        const selectedTenantId = localStorage.getItem('rethra_selected_tenant_id');
        if (selectedTenantId) {
            headers['X-Tenant-ID'] = selectedTenantId;
        }

        // FormData gönderilmesi gerektiği için yerel fetch kullanılır
        fetch('/api/v1/initialization/multimodal/test', {
            method: 'POST',
            headers,
            body: formData
        })
            .then(response => response.json())
            .then((data: any) => {
                if (data.success) {
                    resolve(data.data || {});
                } else {
                    resolve({ success: false, message: data.message || t('error.initialization.testFailed') });
                }
            })
            .catch((error: any) => {
                console.error('Failed multimodal test:', error);
                reject(error);
            });
    });
}

// Metin içeriği ilişki çıkarma arayüzü
export interface TextRelationExtractionRequest {
    text: string;
    tags: string[];
    model_id: string;
}

export interface Node {
    name: string;
    attributes: string[];
}

export interface Relation {
    node1: string;
    node2: string;
    type: string;
}

export interface TextRelationExtractionResponse {
    nodes: Node[];
    relations: Relation[];
}

// Metin içeriği ilişki çıkarma
export function extractTextRelations(request: TextRelationExtractionRequest): Promise<TextRelationExtractionResponse> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/extract/text-relation', request, { timeout: 60000 })
            .then((response: any) => {
                resolve(response.data || { nodes: [], relations: [] });
            })
            .catch((error: any) => {
                console.error('Failed to extract text relations:', error);
                reject(error);
            });
    });
}

export interface FabriTextRequest {
    tags: string[];
    model_id: string;
}

export interface FabriTextResponse {
    text: string;
}

// Metin içeriği oluşturma
export function fabriText(request: FabriTextRequest): Promise<FabriTextResponse> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/extract/fabri-text', request)
            .then((response: any) => {
                resolve(response.data || { text: '' });
            })
            .catch((error: any) => {
                console.error('Failed to generate text:', error);
                reject(error);
            });
    });
}

export interface FabriTagRequest {
}

export interface FabriTagResponse {
    tags: string[];
}

// Etiket oluşturma
export function fabriTag(request: FabriTagRequest): Promise<FabriTagResponse> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/extract/fabri-tag', request)
            .then((response: any) => {
                resolve(response.data || { tags: [] as string[] });
            })
            .catch((error: any) => {
                console.error('Failed to generate tags:', error);
                reject(error);
            });
    });
}

// Sağlayıcıya özgü ek yapılandırma alanları (Azure api_version, LKEAP secret_key vb.).
// Mirrors internal/models/catalog.ExtraField.
export interface ModelProviderExtraFieldOption {
    label: string;
    labels?: Record<string, string>;
    value: string;
}

/**
 * Renames the primary credential input for vendors whose API does not take a
 * plain API key. LKEAP and Volcengine sign their rerank requests with a
 * CAM / IAM key pair, so the first field is a SecretId / Access Key ID.
 * The vendor declares the wording so the editor needs no vendor table.
 */
export interface ModelProviderCredentialLabel {
    label: string;
    labels?: Record<string, string>;
    placeholder?: string;
    placeholders?: Record<string, string>;
    hint?: string;
    hints?: Record<string, string>;
    // Backend model types ("KnowledgeQA", "Rerank", ...); empty = all.
    model_types?: string[];
    required?: boolean;
}

export interface ModelProviderExtraField {
    key: string;
    label: string;
    labels?: Record<string, string>;
    type: 'string' | 'number' | 'boolean' | 'select' | 'password' | string;
    required?: boolean;
    default?: string;
    placeholder?: string;
    placeholders?: Record<string, string>;
    options?: ModelProviderExtraFieldOption[];
    // Backend model types ("KnowledgeQA", "Embedding", "Rerank", "VLLM", "ASR"); empty = all.
    model_types?: string[];
    // Secret values are never echoed back; they are stored as the app_secret credential.
    secret?: boolean;
}

// Sağlayıcının yerleşik model katalog girdisi. Mirrors handler.ModelCatalogEntryDTO.
export interface ModelCatalogEntry {
    id: string;
    name: string;
    type: string;
    api?: string;
    reasoning?: boolean;
    input?: string[];
    context_window?: number;
    max_output_tokens?: number;
    dimension?: number;
    thinking_levels?: ReasoningEffortLevel[];
    cost?: Record<string, unknown>;
    // Vendor page these facts came from, for the "read the docs" link.
    source?: string;
}

export interface ModelProviderThinking {
    format: string;
    levels: ReasoningEffortLevel[];
}

// Model sağlayıcı bilgisi türü. Mirrors handler.ModelProviderDTO — ön uç hiçbir
// vendor table of its own; everything rendered for a vendor comes from here.
export interface ModelProviderOption {
    value: string;        // provider tanımlayıcısı
    label: string;        // Görünen ad (İngilizce / varsayılan)
    labels?: Record<string, string>;  // locale bazında görünen ad
    description: string;  // Açıklama
    descriptions?: Record<string, string>;
    website?: string;
    icon?: string;        // data:image/svg+xml;base64,... doğrudan <img src> ile kullanılabilir
    api?: string;
    auth?: string;
    requiresAuth?: boolean;
    defaultUrls: Record<string, string>;  // Model türüne göre ayrılmış varsayılan URL
    modelTypes: string[]; // Desteklenen model türleri
    extraFields?: ModelProviderExtraField[];
    credentialLabels?: ModelProviderCredentialLabel[];
    models?: ModelCatalogEntry[];
    thinking?: ModelProviderThinking;
    order?: number;
}

// Model sağlayıcı listesini al.
//
// Başarısız olduğunda reject et (artık boş dizi olarak yutma): tek çağıran stores/modelProviders,
// "Backend'de gerçekten sağlayıcı yok" ile "bu istek başarısız oldu" durumlarını ayırt etmelidir — hatayı boş liste olarak önbelleğe almak,
// sağlayıcı açılır listesinin kullanıcı sayfayı yenileyene kadar tüm oturum boyunca boş kalmasına neden olur.
export function listModelProviders(modelType?: string): Promise<ModelProviderOption[]> {
    const url = modelType
        ? `/api/v1/models/providers?model_type=${encodeURIComponent(modelType)}`
        : '/api/v1/models/providers';
    return get(url).then((response: any) => {
        const data = response?.data;
        // Reject rather than coerce: the store distinguishes "this vendor list
        // is genuinely empty" (cacheable) from "the request did not produce a
        // list" (retry on the next mount). Returning [] here would make a
        // malformed 200 look like the former and blank the vendor dropdown and
        // every card icon for the rest of the session.
        if (!Array.isArray(data)) {
            return Promise.reject(new Error('model providers response is not a list'));
        }
        return data as ModelProviderOption[];
    });
}

export interface ResolveModelCatalogParams {
    spec?: ModelSpecOverride;
    provider: string;
    model?: string;
    base_url?: string;
    model_type?: string;
    api?: string;
    thinking_control?: string;
    remote_model_name?: string;
    // Vendor-declared non-secret extra fields (Azure api_version, ...) are
    // forwarded by key so the preview resolves the same request the runtime
    // will make. The backend only accepts keys the vendor declares.
    [extraField: string]: string | ModelSpecOverride | undefined;
}

// Dizin ayrıştırma sonucu. Mirrors handler.ResolveModelCatalog response data.
export interface ResolvedModelCatalog {
    provider: string;
    api: string;
    // base_url and url are only returned to callers who may configure
    // integrations; a viewer gets the capability answer without the endpoint.
    base_url?: string;
    // url is the endpoint the row will actually call, and only vendors that
    // compute their own URL report one (Azure, whose api_version picks
    // between the v1 data plane and the dated deployments path).
    url?: string;
    remote_model: string;
    cataloged: boolean;
    model: Record<string, unknown>;
    capabilities: ModelCapabilities;
}

// Düzenleyicide anlık gösterim için modelin geçerli entegrasyon yapılandırmasını (protokol, düşünme seviyesi, bağlam penceresi vb.) ayrıştırır.
export function resolveModelCatalog(params: ResolveModelCatalogParams): Promise<ResolvedModelCatalog> {
    return new Promise((resolve, reject) => {
        post(`/api/v1/models/catalog/resolve`, params)
            .then((response: any) => {
                if (response?.success && response?.data) {
                    resolve(response.data as ResolvedModelCatalog);
                } else {
                    reject(new Error(response?.message || 'resolve failed'));
                }
            })
            .catch((error: any) => {
                reject(error?.error || error);
            });
    });
}
