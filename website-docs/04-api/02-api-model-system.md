# API başvurusu: modeller ve başlatma

Modelleri yönetin, bağlantıları test edin, bilgi tabanını başlatın ve değerlendirme görevleri başlatın.

Sistem bilgisi ve sistem yönetimi (`/system`, `/system/admin`) arayüzleri için bkz. [Sistem ve platform yönetimi](./02-api-system.md).

## Modeller (/api/v1/models)

API key: `manage_models` ya da full-access.

### GET /api/v1/models/providers

Amaç: sağlayıcı kataloğu (ön yüz sağlayıcı açılır listesini, simgeleri, ek alanları ve model seçimini buna göre dinamik oluşturur). Yetki: Viewer+. Sorgu parametresi: `model_type` (isteğe bağlı: `chat/embedding/rerank/vllm/asr`; `KnowledgeQA` gibi arka uç değerleri de kabul edilir; bilinmeyen değer 400 döndürür). Belirtilirse yalnızca bu türü destekleyen sağlayıcılar ve bu türdeki modeller döndürülür. Handler: `internal/handler/model_catalog.go`

Yanıt: 200 `{"success":true,"data":[ModelProviderDTO]}`, her öğe:

| Alan | Açıklama |
| --- | --- |
| `value` / `label` / `labels` / `description` / `descriptions` / `website` | Sağlayıcı id'si, marka adı, dile göre ad ve açıklama |
| `icon` | `data:image/svg+xml;base64,...`; doğrudan `<img src>` içinde kullanılabilir |
| `api` / `auth` / `requiresAuth` | Varsayılan protokol (`openai-completions` vb.), kimlik doğrulama yöntemi, anahtar gerekip gerekmediği |
| `defaultUrls` / `modelTypes` | Model türüne göre varsayılan adresler ve desteklenen türler. `defaultUrls` yalnızca Admin+ (ya da full-access / `manage_tenant_settings` API key) için döndürülür, diğer çağıranlar için boştur |
| `extraFields` | Sağlayıcı ek yapılandırma alanı tanımları (`key,label,labels,type,required,default,placeholder,options,model_types,secret`); değerler `parameters.extra_config` içine kaydedilir |
| `credentialLabels` | Bazı model türlerinde kimlik bilgisi giriş kutusunun adı ve ipucu (örneğin Volcano Engine ve LKEAP rerank için API Key alanı aslında Access Key ID / SecretId'dir) |
| `models` | Yerleşik model kataloğu (`id,name,type,api,reasoning,input,context_window,max_output_tokens,dimension,thinking_levels,cost,source`); `source`, model parametrelerinin dayandığı sağlayıcı belgesinin bağlantısıdır |
| `thinking` | Sağlayıcı düzeyinde düşünme kodlaması özeti (`format`, `levels`) |
| `order` | Liste sıralama değeri |

```bash
curl "$BASE/api/v1/models/providers?model_type=chat" -H "Authorization: Bearer $TOKEN"
```

### GET|POST /api/v1/models/catalog/resolve

Amaç: sağlayıcı, model adı, `base_url` ve `extra_config` değerlerine göre geçerli bağlantı yapılandırmasını (protokol, düşünme düzeyi, bağlam) çözer; model düzenleyicide anlık gösterim için kullanılır. Yetki: Viewer+.

Parametreler (GET'te sorgu parametresi, POST'ta JSON istek gövdesi; alanlar aynıdır): `provider` (sağlayıcı ID'si), `model`, `base_url`, `model_type` (varsayılan `chat`), `api`, `thinking_control`, `remote_model_name` ve o sağlayıcının tanımladığı gizli olmayan ek alanlar (örneğin Azure için `api_version`). POST gövdesi ayrıca tek satırlık katalog geçersiz kılmasını önizlemek için `spec` nesnesi (modelin `parameters.spec` alanıyla aynı) içerebilir. Anahtar türündeki alanlar hiçbir durumda kabul edilmez.

Yanıt: 200 `{"success":true,"data":{provider,api,remote_model,cataloged,model,capabilities,base_url,url}}`; burada `capabilities` şudur: `{provider,api,cataloged,reasoning,thinking_levels,thinking_format,input,context_window,max_output_tokens,max_tokens_field}`. `base_url` ve `url` (gerçek istek adresi; yalnızca adresi kendisi hesaplayan sağlayıcılarda döner, örneğin Azure) yalnızca Admin+ (ya da full-access / `manage_tenant_settings` API key) için döndürülür. Çözülemezse 400 döner. Aynı `capabilities` yapısı uzak sohbet/görsel modellerin `ModelResponse.capabilities` alanında da döndürülür.

```bash
curl "$BASE/api/v1/models/catalog/resolve?provider=deepseek&model=deepseek-v4-pro" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/models

Amaç: model oluşturur. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet (`binding:"required"`) | Model adı |
| `display_name` | string | Hayır | Görünen ad |
| `type` | string | Evet (`binding:"required"`) | Model türü: `KnowledgeQA` / `Embedding` / `Rerank` / `VLLM` / `ASR` (olduğu gibi kaydedilir; `chat` gibi ön yüz yazımları kabul edilmez) |
| `source` | string | Evet (`binding:"required"`) | Kaynak (`local` / `remote`) |
| `description` | string | Hayır | Açıklama |
| `parameters` | object | Evet (`binding:"required"`) | Bağlantı parametreleri (`base_url`, `provider`, `extra_config`, `spec`, `context_window` vb.; alanlar için bkz. [Model yönetimi](../03-features/06-models.md#model-yapilandirma-alanlari)). Oluştururken doğrudan `api_key` / `app_secret` verilebilir; sonrasında credentials alt kaynağıyla değiştirilir |

`parameters` model kataloğuna göre doğrulanır (bilinmeyen protokol, hatalı compat anahtarı, geçersiz düşünme düzeyi 400 döndürür); `base_url` SSRF denetiminden geçer.

Yanıt: 201 `{"success":true,"data":{ModelResponse}}` (`id,name,type,source,parameters,is_default,is_builtin,status,credentials,capabilities,...`; yanıt anahtar içermez)

```bash
curl -X POST $BASE/api/v1/models -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"gpt-5.5","type":"KnowledgeQA","source":"remote","parameters":{"provider":"openai","base_url":"https://api.openai.com/v1","api_key":"sk-..."}}'
```

### GET /api/v1/models

Amaç: model listesi. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[ModelResponse]}`

```bash
curl $BASE/api/v1/models -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/models/:id

Amaç: model ayrıntıları. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{ModelResponse}}`

```bash
curl $BASE/api/v1/models/m-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/models/:id/debug

Amaç: kaydedilmiş modelde hata ayıklama (gerçek üst kaynak çağrısı yapar, ücret doğurur). Yetki: Admin+. form-data alanları: `input` (≤64KB), `options` (JSON kodlu hata ayıklama seçenekleri: `system_prompt`, `temperature` (0~2), `top_p`, `max_tokens` (1~8192), `thinking`, `reasoning_effort` (`off/auto/minimal/low/medium/high/xhigh/max`; ayarlanırsa `thinking` değerini geçersiz kılar)), `documents` (JSON dizisi, ≤100 öğe), `file` (isteğe bağlı).

Yanıt: 200 `{"success":true,"data":{"ok",elapsed_ms,request,raw_response,observations,error}}`

```bash
curl -X POST $BASE/api/v1/models/m-1/debug -H "Authorization: Bearer $TOKEN" -F 'input=Merhaba'
```

### PUT /api/v1/models/:id

Amaç: modeli günceller (yerleşik modeller hizmet katmanında SystemAdmin ile sınırlıdır). Yetki: Admin+ ya da SystemAdmin (`AdminOrSystemAdmin`). İstek gövdesi: `name`, `display_name` (işaretçi), `description`, `parameters` (kayıtlı anahtarlar korunur), `source`, `type` (hepsi isteğe bağlı).

Yanıt: 200 `{"success":true,"data":{ModelResponse}}`

```bash
curl -X PUT $BASE/api/v1/models/m-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"display_name":"GPT-4o mini"}'
```

### DELETE /api/v1/models/:id

Amaç: modeli siler. Yetki: Admin+.

Yanıt: 200 `{"success":true,"message":"Model deleted"}`

Model hâlâ geçerli alanın bilgi tabanları, Agent'ları ya da uzun süreli belleği tarafından kullanılıyorsa yanıt HTTP 400 olur; uyumluluk için message korunur, ayrıca `error.code=2300` ve `error.details` ilgili nesneleri ve kullanım yerlerini verir:

```json
{
  "success": false,
  "error": {
    "code": 2300,
    "message": "model is used by 2 knowledge base(s); reconfigure or remove those references before deleting",
    "details": {
      "knowledge_bases": [
        {"id": "kb-1", "name": "Product docs", "bindings": ["vlm_model"]},
        {"id": "kb-2", "name": "Engineering", "bindings": ["vlm_model"]}
      ],
      "agents": [],
      "long_term_memory": {"bindings": []},
      "knowledge_base_total": 2,
      "agent_total": 0
    }
  }
}
```

Bilgi tabanı bağlama değerleri: `embedding_model`, `summary_model`, `image_processing_model`, `vlm_model`, `asr_model`, `wiki_synthesis_model`, `auto_tag_model`; Agent bağlama değerleri: `chat_model`, `rerank_model`, `vlm_model`, `asr_model`, `query_understand_model`, `follow_up_model`; uzun süreli bellek bağlama değerleri: `embedding_model`, `extract_model`. Ayrıntılar nesnenin `id`, `name` ve birleştirilmiş `bindings` alanlarını, ayrıca `knowledge_base_total` / `agent_total` değerlerini içerir. Listelerin her biri en fazla 50 öğedir; silme koruması toplam sayıya göre çalışır.

```bash
curl -X DELETE $BASE/api/v1/models/m-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/models/:id/credentials

Amaç: model anahtarını ayarlar (anahtar ana PUT ile taşınmaz). Yetki: Admin+ ya da SystemAdmin. Handler: `internal/handler/model_credentials.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `api_key` | *string | Hayır | Yeni API Key |
| `app_secret` | *string | Hayır | Yeni App Secret (ikisi de atlanırsa yalnızca durum döndürülür) |

Yanıt: 200 `{"success":true,"data":{"fields":{"api_key":{"configured":bool},"app_secret":{"configured":bool}}}}`

```bash
curl -X PUT $BASE/api/v1/models/m-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"api_key":"sk-..."}'
```

### DELETE /api/v1/models/:id/credentials/:field

Amaç: bir anahtar alanını (`api_key` ya da `app_secret`) siler. Yetki: Admin+ ya da SystemAdmin.

Yanıt: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/models/m-1/credentials/api_key -H "Authorization: Bearer $TOKEN"
```

## Başlatma (/api/v1/initialization)

Handler: `internal/handler/initialization.go`. KB yapılandırma işlemleri: API key `manage_kbs` (yazma) / `retrieve` (okuma); model denetim işlemleri: `manage_models` (hepsi full-access ile de olur).

### GET /api/v1/initialization/config/:kbId

Amaç: KB'nin geçerli model/ayrıştırma yapılandırmasını okur. Yetki: Viewer+, KB read.

Model `baseUrl` yalnızca KB'nin ait olduğu alandaki Admin+ (ya da full-access / `manage_tenant_settings` API key) için döndürülür. Kuruluş paylaşımıyla erişen alanlar yalnızca kimlik bilgilerinin yapılandırılıp yapılandırılmadığını (`credentials.*`) görür; kaynak alanın model adreslerini ve bucket bilgilerini göremez.

Yanıt: 200 `{"success":true,"data":{"hasFiles",llm,embedding,rerank,multimodal,documentSplitting,nodeExtract,questionGeneration}}`

```bash
curl $BASE/api/v1/initialization/config/kb-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/initialization/initialize/:kbId

Amaç: KB'nin model ve ayrıştırma yapılandırmasını başlatır (ilk yapılandırma sihirbazı). Yetki: KB oluşturucusu VEYA Admin+, KB write.

Yalnızca KB'nin ait olduğu alan çağırabilir; kuruluş paylaşımıyla düzenleme yetkisi alan alanlar reddedilir (403). KB'ye zaten model bağlıysa bu arayüz o modellerin yapılandırmasını yerinde günceller; bu adım `PUT /models/:id` ile aynı yetkiyi (Admin+ ya da `manage_models` yeteneğine sahip API key) gerektirir, aksi hâlde 403 döner.

Başlıca alanlar (`InitializationRequest`):

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `llm.source` / `llm.modelName` | string | Evet | LLM kaynağı ve model adı |
| `llm.baseUrl` / `llm.apiKey` | string | Hayır | Bağlantı parametreleri |
| `embedding.source` / `embedding.modelName` | string | Evet | Embedding modeli |
| `embedding.baseUrl` / `embedding.apiKey` / `embedding.dimension` | — | Hayır | Bağlantı ve boyut |
| `rerank.enabled` + `rerank.modelName/baseUrl/apiKey` | — | Hayır | Rerank yapılandırması |
| `multimodal.enabled` + `multimodal.vlm.*` + `multimodal.storageType` + `multimodal.cos.*|minio.*` | — | Hayır | Çok kipli işleme ve görsel depolama |
| `documentSplitting.chunkSize` / `separators` | int / []string | Evet | Parçalama yapılandırması |
| `documentSplitting.chunkOverlap` | int | Hayır | Örtüşme |
| `nodeExtract.*` | — | Hayır | Grafik çıkarımı (enabled/text/tags/nodes/relations) |
| `questionGeneration.*` | — | Hayır | Soru üretimi (enabled/questionCount) |

Yanıt: 200 `{"success":true,"message":"Bilgi tabanı yapılandırması güncellendi","data":{"models":[Model],"knowledge_base":{KnowledgeBase}}}`

```bash
curl -X POST $BASE/api/v1/initialization/initialize/kb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"llm":{"source":"remote","modelName":"gpt-4o-mini"},"embedding":{"source":"remote","modelName":"text-embedding-3-small"},"documentSplitting":{"chunkSize":512,"separators":["\n\n"]}}'
```

### PUT /api/v1/initialization/config/:kbId

Amaç: KB model/parçalama yapılandırmasını günceller (`KBModelConfigRequest`: `llmModelId` zorunlu; `embeddingModelId`, `vlm_config`, `asr_config`, `documentSplitting.*`, `multimodal.enabled`, `storageProvider`, `storageBackendId`, `nodeExtract.*`, `questionGeneration.*` isteğe bağlı). Yetki: KB oluşturucusu VEYA Admin+, KB write.

Kuruluş paylaşımıyla erişildiğinde geçerli paylaşım yetkisinin admin olması gerekir; editor yalnızca içeriği düzenleyebilir, ayarları değiştiremez (403). Depolama bağlamasını (`storageBackendId` / `storageProvider`) yalnızca KB'nin ait olduğu alan değiştirebilir; başka alanlar geçerli değerden farklı bir değer gönderirse 403 döner.

Yanıt: 200 `{"success":true,"message":"Yapılandırma güncellendi"}`

```bash
curl -X PUT $BASE/api/v1/initialization/config/kb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"llmModelId":"m-1","embeddingModelId":"m-2"}'
```

### Model bağlantı denetimi (hepsi POST, yetki Admin+)

İstek gövdesi her zaman `ModelTestRequest` biçimindedir:

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `source` | string | Hayır | Varsayılan `remote` |
| `modelName` | string | Evet | Model adı |
| `baseUrl` / `apiKey` / `appSecret` | string | Hayır | Bağlantı parametreleri |
| `provider` / `interfaceType` | string | Hayır | Sağlayıcı/arayüz türü |
| `dimension` / `supportsDimensionOverride` | int / bool | Hayır | embedding boyutu; boyutun istekte belirtilip belirtilmeyeceği |
| `customHeaders` / `extraConfig` | map | Hayır | Uzantılar |
| `spec` | object | Hayır | Tek satırlık katalog geçersiz kılması; modelin `parameters.spec` alanıyla aynı |
| `modelId` | string | Hayır | Kayıtlı model ID'si: istekte eksik olan anahtarlar, `extraConfig` ve `spec` bu modelden tamamlanır |

| Uç nokta | Amaç | Yanıt data |
| --- | --- | --- |
| `POST /api/v1/initialization/remote/check` | LLM uzak bağlantı denetimi | `{available,message}` |
| `POST /api/v1/initialization/embedding/test` | Embedding testi | `{available,message,dimension}` |
| `POST /api/v1/initialization/rerank/check` | Rerank testi | `{available,message}` |
| `POST /api/v1/initialization/asr/check` | ASR testi | `{available,message}` |

```bash
curl -X POST $BASE/api/v1/initialization/remote/check -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"modelName":"gpt-4o-mini","baseUrl":"https://api.openai.com/v1","apiKey":"sk-..."}'
```

### POST /api/v1/initialization/multimodal/test

Amaç: çok kipli (VLM + görsel depolama) uçtan uca test. Yetki: Admin+. multipart alanları: `image` (zorunlu), `vlm_model`, `vlm_base_url` (zorunlu), `vlm_api_key`, `vlm_interface_type`, `storage_type` (`cos|minio`, zorunlu) ve ilgili `cos_*`/`minio_*` alanları, `chunk_size`, `chunk_overlap`, `separators`.

Yanıt: 200 `{"success":true,"data":{"success","caption","ocr","processing_time"}}`

```bash
curl -X POST $BASE/api/v1/initialization/multimodal/test -H "Authorization: Bearer $TOKEN" \
  -F 'image=@demo.png' -F 'vlm_model=qwen-vl' -F 'vlm_base_url=http://x' -F 'storage_type=minio'
```

### POST /api/v1/initialization/extract/text-relation

Amaç: metinden grafik çıkarımı testi. Yetki: Admin+. İstek gövdesi: `text` (zorunlu, ≤5000 karakter), `tags` (zorunlu, en az bir tane), `model_id` (zorunlu).

Yanıt: 200 `{"success":true,"data":{"nodes":[GraphNode],"relations":[GraphRelation]}}`

```bash
curl -X POST $BASE/api/v1/initialization/extract/text-relation -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"text":"Ahmet Tencent'"'"'te çalışıyor","tags":["Kişi","Şirket"],"model_id":"m-1"}'
```

### POST /api/v1/initialization/extract/fabri-tag

Amaç: örnek etiketler üretir. Yetki: Admin+. İstek gövdesi yoktur.

Yanıt: 200 `{"success":true,"data":{"tags":[...]}}`

```bash
curl -X POST $BASE/api/v1/initialization/extract/fabri-tag -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/initialization/extract/fabri-text

Amaç: etiketlere göre örnek metin üretir. Yetki: Admin+. İstek gövdesi: `{"tags":[...],"model_id":"m-1"}` (model_id zorunlu).

Yanıt: 200 `{"success":true,"data":{"text":"..."}}`

```bash
curl -X POST $BASE/api/v1/initialization/extract/fabri-text -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"model_id":"m-1","tags":["Kişi"]}'
```

## Değerlendirme (/api/v1/evaluation)

Handler: `internal/handler/evaluation.go`. API key: `run_evaluations`/full.

### POST /api/v1/evaluation

Amaç: değerlendirme görevi başlatır (LLM çağrıları yapar, ücret doğurur). Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `dataset_id` | string | Hayır | Veri kümesi ID'si |
| `knowledge_base_id` | string | Hayır | Hedef KB |
| `chat_id` | string | Hayır | Sohbet modeli ID'si |
| `rerank_id` | string | Hayır | Rerank modeli ID'si |

Yanıt: 200 `{"success":true,"data":{değerlendirme görevi}}`

```bash
curl -X POST $BASE/api/v1/evaluation -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"knowledge_base_id":"kb-1","chat_id":"m-1"}'
```

### GET /api/v1/evaluation

Amaç: değerlendirme sonucunu sorgular. Yetki: Viewer+. Sorgu parametresi: `task_id` (zorunlu).

Yanıt: 200 `{"success":true,"data":{değerlendirme sonucu}}`

```bash
curl "$BASE/api/v1/evaluation?task_id=task-1" -H "Authorization: Bearer $TOKEN"
```

## Uygulama başvurusu

Rota kaydı: `internal/router/router.go`, `RegisterModelRoutes`, `RegisterInitializationRoutes` ve `RegisterEvaluationRoutes` fonksiyonlarını çağırır (tanımları `internal/router/routes_infra.go` içindedir). Handler'lar: `internal/handler/model.go`, `internal/handler/model_catalog.go`, `internal/handler/model_credentials.go`, `internal/handler/initialization.go`, `internal/handler/evaluation.go`.
