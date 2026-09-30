# API referansı: altyapı ve veri kaynakları

Vektör depolama, dosya depolama, web arama hizmetleri ve veri kaynaklarını kaydeder ve yönetir; bağlantı testi ile eşzamanlama işlemleri sunar.

Standart kural: okuma için Viewer+, yazma/bağlantı testi için Admin+ gerekir (kimlik bilgileri harici sistemde sorgulanır). API key yetenekleri: vektör veritabanı `manage_vector_stores`, depolama arka ucu `manage_storage_backends`, Web arama `manage_web_search`, veri kaynağı `manage_datasources` (hepsi full-access olabilir).

## Vektör depolama (/api/v1/vector-stores)

### GET /api/v1/vector-stores/types

Amaç: kullanılabilir motor türleri ve yapılandırma schema. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[tür tanımları]}`

```bash
curl $BASE/api/v1/vector-stores/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/vector-stores/test

Amaç: bağlantıyı ham yapılandırmayla test etmek (veritabanına kaydetmeden). Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `engine_type` | string | Evet (`binding:"required"`) | Motor türü |
| `connection_config` | object | Evet (`binding:"required"`) | Bağlantı yapılandırması |

Yanıt: 200 `{"success":true|false,"version":"...","error":"..."}`

```bash
curl -X POST $BASE/api/v1/vector-stores/test -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"engine_type":"qdrant","connection_config":{"addr":"qdrant:6334"}}'
```

### POST /api/v1/vector-stores

Amaç: vektör deposu yapılandırması oluşturmak. Yetki: Admin+. Alanlar: `name` (zorunlu), `engine_type` (zorunlu), `connection_config` (zorunlu), `index_config` (isteğe bağlı).

Yanıt: 201 `{"success":true,"data":{VectorStoreResponse}}` (`id,tenant_id,name,engine_type,connection_config,index_config,...`)

```bash
curl -X POST $BASE/api/v1/vector-stores -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"qdrant-main","engine_type":"qdrant","connection_config":{"addr":"qdrant:6334"}}'
```

### GET /api/v1/vector-stores

Amaç: vektör deposu listesi (ortam değişkenleriyle eklenen `__env_*` depoları önce gelir). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[VectorStoreResponse]}`

```bash
curl $BASE/api/v1/vector-stores -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/vector-stores/:id

Amaç: vektör deposu ayrıntıları (`__env_*` ID desteği). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{VectorStoreResponse}}`

```bash
curl $BASE/api/v1/vector-stores/vs-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/vector-stores/:id

Amaç: güncelleme (yalnızca yeniden adlandırma; env deposu değiştirilemez). Yetki: Admin+. İstek gövdesi: `{"name":"..."}` (`binding:"required"`).

Yanıt: 200 `{"success":true,"data":{VectorStoreResponse}}`

```bash
curl -X PUT $BASE/api/v1/vector-stores/vs-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"qdrant-prod"}'
```

### DELETE /api/v1/vector-stores/:id

Amaç: silme (env deposu silinemez). Yetki: Admin+.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/vector-stores/vs-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/vector-stores/:id/test

Amaç: kaydedilmiş/env vektör deposunu test etmek. Yetki: Admin+.

Yanıt: 200 `{"success":true|false,"version","error"}`

```bash
curl -X POST $BASE/api/v1/vector-stores/vs-1/test -H "Authorization: Bearer $TOKEN"
```

## Depolama arka uçları (/api/v1/storage-backends)

İstek gövdesi (Create/Update/TestRaw için ortak `storageBackendRequest`):

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet (`binding:"required"`) | Ad |
| `provider` | string | Evet (`binding:"required"`) | Sağlayıcı: `local`/`minio`/`cos`/`tos`/`s3`/`oss`/`ks3`/`obs`, `STORAGE_ALLOW_LIST` ile sınırlıdır |
| `config` | object | Hayır | Sağlayıcı yapılandırması; alanlar için bkz. [Depolama arka uçları](../03-features/19-storage-backends.md#baglanti-parametreleri) (yanıtta kimlik bilgileri maskelenir) |
| `status` | string | Hayır | `active` (varsayılan)/`disabled` |

### GET /api/v1/storage-backends/types

Amaç: `STORAGE_ALLOW_LIST` tarafından izin verilen provider adlarının listesi (ayarlanmadığında tümü döner). Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":["local","minio",...]}`

```bash
curl $BASE/api/v1/storage-backends/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/storage-backends/test

Amaç: ham yapılandırma bağlantısını test etmek. Yetki: Admin+. Yanıt: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/storage-backends/test -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"t","provider":"minio","config":{"endpoint":"minio:9000"}}'
```

### POST /api/v1/storage-backends

Amaç: depolama arka ucu oluşturmak. Yetki: Admin+. Yanıt: 201 `{"success":true,"data":{StorageBackend}}`

```bash
curl -X POST $BASE/api/v1/storage-backends -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"minio-main","provider":"minio","config":{"endpoint":"minio:9000"}}'
```

### GET /api/v1/storage-backends

Amaç: liste (`default_storage_backend_id` dahil). Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":[...],"default_storage_backend_id":"..."}`

```bash
curl $BASE/api/v1/storage-backends -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/storage-backends/:id

Amaç: ayrıntılar (kimlik bilgileri maskelenir). Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":{StorageBackend}}`

```bash
curl $BASE/api/v1/storage-backends/sb-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/storage-backends/:id

Amaç: güncelleme. Yetki: Admin+. Yanıt: 200 `{"success":true,"data":{StorageBackend}}`

```bash
curl -X PUT $BASE/api/v1/storage-backends/sb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"minio-prod","provider":"minio"}'
```

### DELETE /api/v1/storage-backends/:id

Amaç: silme. Yetki: Admin+. Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/storage-backends/sb-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/storage-backends/:id/test

Amaç: kaydedilmiş arka ucu test etme. Yetki: Admin+. Yanıt: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/storage-backends/sb-1/test -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/storage-backends/:id/default

Amaç: varsayılan arka uç olarak ayarlama. Yetki: Admin+. Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/storage-backends/sb-1/default -H "Authorization: Bearer $TOKEN"
```

## Web araması (/api/v1/web-search ve /api/v1/web-search-providers)

Şu anda Metaso, Exa, Bocha, Brave ve Serply dahil 14 arama sağlayıcısı kayıtlıdır. Her birinin api_key ve extra_config parametreleri için [web araması](../03-features/11-web-search.md) bölümüne bakın.

### GET /api/v1/web-search/providers

Amaç: yerleşik arama sağlayıcısı dizini (salt okunur). Yetki: Viewer+, yalnızca JWT (API key politikası belirtilmemiştir). Handler: `internal/handler/web_search.go`

Yanıt: 200 `{"success":true,"data":[...]}`

```bash
curl $BASE/api/v1/web-search/providers -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/web-search-providers/types

Amaç: sağlayıcı türleri ve parametre şeması. Yetki: Viewer+. Handler: `internal/handler/web_search_provider.go`

Yanıt: 200 `{"success":true,"data":[...]}`

```bash
curl $BASE/api/v1/web-search-providers/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/web-search-providers/test

Amaç: ham kimlik bilgisi testi (veritabanına kaydetmeden). Yetki: Admin+. İstek gövdesi: `provider` (`binding:"required"`), `parameters` (isteğe bağlı).

Yanıt: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/web-search-providers/test -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"provider":"tavily","parameters":{"api_key":"tvly-..."}}'
```

### POST /api/v1/web-search-providers

Amaç: sağlayıcı yapılandırması oluşturma. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet (`binding:"required"`) | Ad |
| `provider` | string | Evet (`binding:"required"`) | Tür (bing/tavily/google…) |
| `description` | string | Hayır | Açıklama |
| `parameters` | object | Hayır | Parametreler (api_key için credentials alt kaynağının kullanılması önerilir) |
| `is_default` | bool | Hayır | Varsayılan sağlayıcı |

Yanıt: 201 `{"success":true,"data":{WebSearchProviderResponse}}`

```bash
curl -X POST $BASE/api/v1/web-search-providers -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"tavily-main","provider":"tavily"}'
```

### GET /api/v1/web-search-providers

Amaç: sağlayıcı listesi. Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":[...]}`

```bash
curl $BASE/api/v1/web-search-providers -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/web-search-providers/:id

Amaç: ayrıntılar. Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":{...}}`

```bash
curl $BASE/api/v1/web-search-providers/wsp-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/web-search-providers/:id

Amaç: güncelleme (boş alanlar önceki değerlerini korur; APIKey korunur). Yetki: Admin+. İstek gövdesi: `name/description/parameters/is_default` (tümü isteğe bağlı).

Yanıt: 200 `{"success":true,"data":{...}}`

```bash
curl -X PUT $BASE/api/v1/web-search-providers/wsp-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"is_default":true}'
```

### DELETE /api/v1/web-search-providers/:id

Amaç: silme. Yetki: Admin+. Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/web-search-providers/wsp-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/web-search-providers/:id/credentials

Amaç: API anahtarını ayarlamak (`{"api_key":"..."}`; atlandığında durum döner). Yetki: Admin+. İşleyici: `internal/handler/web_search_provider_credentials.go`

Yanıt: 200 `{"success":true,"data":{"fields":{"api_key":{"configured":bool}}}}`

```bash
curl -X PUT $BASE/api/v1/web-search-providers/wsp-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"api_key":"tvly-..."}'
```

### DELETE /api/v1/web-search-providers/:id/credentials/:field

Amaç: Kimlik bilgisi alanını silmek (`field` yalnızca `api_key`). Yetki: Admin+. Yanıt: 204.

```bash
curl -X DELETE $BASE/api/v1/web-search-providers/wsp-1/credentials/api_key -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/web-search-providers/:id/test

Amaç: Kaydedilmiş sağlayıcıyı test etmek. Yetki: Admin+. Yanıt: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/web-search-providers/wsp-1/test -H "Authorization: Bearer $TOKEN"
```

## Veri kaynakları (/api/v1/datasource)

Harici içerik bağlayıcıları (Feishu/Notion/Yuque vb.); eşitleme görevleri KB'ye yazar. İşleyici: `internal/handler/datasource.go`. Bu grubun yanıtlarının çoğu ham nesne/dizidir (`success` sarmalaması yoktur).

Şu anda kayıtlı türler feishu, lark, feishu_drive, lark_drive, notion, confluence, yuque, dingtalk, ima, rss, gitlab'dır. Her bağlayıcının credentials, kaynak seçimi ve eşitleme sınırlamaları için bkz. [Veri kaynağı içe aktarma](../03-features/10-datasource.md). sync_deletions etkinleştirildiğinde bu veri kaynağına ait silinmiş bilgiler gerçekten silinir; source_created_at/source_updated_at bilgi metadata'sında saklanır.

### GET /api/v1/datasource/types

Amaç: Kullanılabilir bağlayıcı kataloğu. Yetki: Viewer+.

Yanıt: 200 `[{type,name,description,icon,priority,auth_type,capabilities}]`

```bash
curl $BASE/api/v1/datasource/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/validate-credentials

Amaç: Ham kimlik bilgilerini doğrulamak (“Bağlantıyı test et” düğmesi, veritabanına kaydetmez). Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `type` | string | Evet (`binding:"required"`) | Bağlayıcı türü |
| `credentials` | map | Evet (`binding:"required"`) | Kimlik bilgileri |

Yanıt: 200 `{"status":"connected"}`; başarısızlıkta 400 `{"error":"..."}`

```bash
curl -X POST $BASE/api/v1/datasource/validate-credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"type":"notion","credentials":{"api_key":"ntn_xxx"}}'
```

### POST /api/v1/datasource

Amaç: Veri kaynağı oluşturmak. Yetki: Admin+. İstek gövdesi (`types.DataSource`):

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `knowledge_base_id` | string | Evet | Hedef KB (bu alana ait olmalıdır) |
| `name` | string | Evet | Ad |
| `type` | string | Evet | Bağlayıcı türü |
| `config` | object | Evet | Kimlik bilgileri (şifreli depolama) + kaynak seçimi + ayarlar |
| `sync_schedule` | string | Hayır | cron ifadesi |
| `sync_mode` | string | Hayır | `incremental` (varsayılan) / `full` |
| `conflict_strategy` | string | Hayır | `overwrite` (varsayılan) / `skip` |
| `sync_deletions` | bool | Hayır | Varsayılan true |
| `sync_log_retention_days` | int | Hayır | Varsayılan 30 |

Yanıt: 201 `DataSourceResponse` (kimlik bilgileri çıkarılmıştır, bkz. `internal/handler/dto/datasource.go`).

```bash
curl -X POST $BASE/api/v1/datasource -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"knowledge_base_id":"kb-1","name":"notion senkronizasyonu","type":"notion","config":{}}'
```

### GET /api/v1/datasource

Amaç: veri kaynağı listesi. Yetki: Viewer+. Sorgu parametresi: `kb_id` (zorunlu).

Yanıt: 200 `[DataSourceResponse]`

```bash
curl "$BASE/api/v1/datasource?kb_id=kb-1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/:id

Amaç: ayrıntılar. Yetki: Viewer+. Yanıt: 200 `DataSourceResponse`; 404 `{"error":"data source not found"}`

```bash
curl $BASE/api/v1/datasource/ds-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/datasource/:id

Amaç: güncelleme (`id/tenant_id/knowledge_base_id` özgün değerlerine kilitlidir). Yetki: Admin+. İstek gövdesi oluşturma ile aynıdır.

PUT, veri kaynağı yapılandırmasını tamamen değiştirir ve kısmi güncellemeyi desteklemez: önce ayrıntıları GET ile alın, değiştirdikten sonra nesnenin tamamını (`name`, `type`, `config`, `sync_schedule`, `sync_mode`, `conflict_strategy`, `sync_deletions`, `sync_log_retention_days`, `status` vb.) geri gönderin. `sync_schedule` ve `sync_deletions` her zaman istek gövdesine göre yazılır: `sync_schedule` öğesinin atlanması onu boş dizeye temizler (yalnızca manuel senkronizasyon; mevcut zamanlanmış görevler de kaldırılır), `sync_deletions` öğesinin atlanması false anlamına gelir; diğer alanlar atlandığında özgün değerlerinin korunacağı garanti edilmez. Kimlik bilgileri bu uç nokta üzerinden değiştirilmez; istek gövdesindeki `config.credentials` yok sayılır ve kayıtlı kimlik bilgileri korunur.

Yanıt: 200 `DataSourceResponse`

```bash
curl -X PUT $BASE/api/v1/datasource/ds-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"notion senkronizasyonu v2","type":"notion","knowledge_base_id":"kb-1",
       "config":{"type":"notion","resource_ids":["page-1"]},
       "sync_schedule":"0 0 */6 * * *","sync_mode":"incremental","conflict_strategy":"overwrite",
       "sync_deletions":true,"sync_log_retention_days":30,"status":"active"}'
```

### DELETE /api/v1/datasource/:id

Amaç: silme. Yetki: Admin+. Yanıt: 204.

```bash
curl -X DELETE $BASE/api/v1/datasource/ds-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/datasource/:id/credentials

Amaç: kimlik bilgilerini tamamen değiştirme (veri kaynağı kimlik bilgileri, tek bir mantıksal `credentials` alanından oluşan atomik bir map'tir). Yetki: Admin+. İstek gövdesi: `{"credentials":{...}}` (boş olmayan map zorunludur). İşleyici: `internal/handler/datasource_credentials.go`

Yanıt: 200 `{"success":true,"data":{"fields":{"credentials":{"configured":bool}}}}`

```bash
curl -X PUT $BASE/api/v1/datasource/ds-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"credentials":{"api_key":"ntn_xxx"}}'
```

### DELETE /api/v1/datasource/:id/credentials/:field

Amaç: kimlik bilgilerini temizleme (`field`, `credentials` olmalıdır). Yetki: Admin+. Yanıt: 204.

```bash
curl -X DELETE $BASE/api/v1/datasource/ds-1/credentials/credentials -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/:id/validate

Amaç: kayıtlı veri kaynağı bağlantısını doğrulama. Yetki: Admin+. Yanıt: 200 `{"status":"connected"}`

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/validate -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/:id/resources

Amaç: harici kaynak ağacını görüntüleme (tembel yükleme). Yetki: Admin+. Sorgu parametresi: `parent_id` (isteğe bağlı, boş=üst seviye).

Yanıt: 200 `[{external_id,name,type,description,url,modified_at,parent_id,has_children,metadata}]`

```bash
curl "$BASE/api/v1/datasource/ds-1/resources?parent_id=" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/:id/resource-ancestors

Amaç: kaynak ata zincirini çözümleme (seçici genişletme). Yetki: Admin+. İstek gövdesi: `{"resource_ids":["..."]}` (zorunlu).

Yanıt: 200 `{"ancestors":[...]}`

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/resource-ancestors -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"resource_ids":["page-1"]}'
```

### POST /api/v1/datasource/:id/sync

Amaç: senkronizasyonu manuel olarak tetikleme. Yetki: Admin+. Yanıt: 200 `SyncLog` (`id,status,started_at,items_total,items_created,items_updated,items_deleted,items_failed,...`)

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/sync -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/:id/pause ve POST /api/v1/datasource/:id/resume

Amaç: zamanlanmış senkronizasyonu duraklatma / sürdürme. Yetki: Admin+.

Yanıt: 200 `{"status":"paused"}` / `{"status":"active"}`

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/pause -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/:id/logs

Amaç: senkronizasyon günlüğü listesi. Yetki: Viewer+. Sorgu parametreleri: `limit` (varsayılan 10, üst sınır 100), `offset` (varsayılan 0).

Yanıt: 200 `[SyncLog]`

```bash
curl "$BASE/api/v1/datasource/ds-1/logs?limit=10" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/logs/:log_id

Amaç: tek bir senkronizasyon günlüğü kaydı. Yetki: Viewer+. Yanıt: 200 `SyncLog`; 404 `{"error":"sync log not found"}`

```bash
curl $BASE/api/v1/datasource/logs/log-1 -H "Authorization: Bearer $TOKEN"
```

## Uygulama referansı

Rota kaydı: `internal/router/routes_infra.go` içindeki `RegisterVectorStoreRoutes`, `RegisterStorageBackendRoutes`, `RegisterWebSearchRoutes`, `RegisterWebSearchProviderRoutes`, `RegisterDataSourceRoutes`. İşleyiciler: `internal/handler/vectorstore.go`, `internal/handler/storagebackend.go`, `internal/handler/web_search.go`, `internal/handler/web_search_provider.go`, `internal/handler/web_search_provider_credentials.go`, `internal/handler/datasource.go`, `internal/handler/datasource_credentials.go`.
