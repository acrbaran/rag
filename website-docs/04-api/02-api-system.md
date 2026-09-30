# API Referansı: Sistem ve Platform Yönetimi

Genel ayarlar, görev kuyruğu, platform API anahtarları, alanlar arası denetim ve kullanıcı parolası sıfırlama dahil olmak üzere dağıtım düzeyinde sistem bilgileri ve platform yönetimi arayüzleri sağlar. İşlev açıklamaları için bkz. [Platform Yönetimi ve Sistem Yöneticisi](../03-features/20-platform-admin.md).

`/system/admin/*` grubunun tamamında `SystemAdmin()` koruması bulunur; platform API anahtarları yeteneklere göre ayrılır (`system_settings_read/manage`, `system_runtime_read/manage`, `system_tenants_read/manage`, `system_audit_read`).

## Sistem Bilgileri (/api/v1/system)

Handler: `internal/handler/system.go`. API anahtarı: `manage_vector_stores`/tam yetki. Bu grubun yanıtları `{"code":0,"msg":"success","data":...}` sarmalayıcısını kullanır.

### GET /api/v1/system/capabilities

Viewer+; API Anahtarı okunabilir. `{code:0,data:{edition,capabilities}}` döndürür; her capability için supported/reason sağlanır. Ön yüz, menü girişlerini dağıtım sürümüne, gerçekten kaydedilmiş rotalara ve Docker anahtarlarına göre kontrol eder; menüyü gizlemek arka uç yetki doğrulamasının yerini tutmaz.

```bash
curl "$BASE/api/v1/system/capabilities" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/info

Amaç: Sistem sürümü ve motor bilgileri. Yetki: Viewer+.

Yanıt: 200 `{"code":0,"msg":"success","data":{version,edition,commit_id,build_time,go_version,keyword_index_engine,vector_store_engine,graph_database_engine,minio_enabled,db_version,started_at,uptime_seconds}}`

```bash
curl $BASE/api/v1/system/info -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/parser-engines

Amaç: Ayrıştırma motoru listesi ve DocReader bağlantı durumu. Yetki: Viewer+.

Yanıt: 200 `{"code":0,"msg":"success","data":[...],"docreader_addr","docreader_transport","connected"}`

```bash
curl $BASE/api/v1/system/parser-engines -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/parser-engines/check

Amaç: Verilen yapılandırmayla ayrıştırma motorunu yoklamak (`types.ParserEngineConfig` istek gövdesi). Yetki: Admin+.

Yanıt: 200, yukarıdakiyle aynı.

```bash
curl -X POST $BASE/api/v1/system/parser-engines/check -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{}'
```

### POST /api/v1/system/docreader/reconnect

Amaç: DocReader'ı yeniden bağlamak. Yetki: Admin+. İstek gövdesi: `{"addr":"host:port"}` (`binding:"required"`).

Yanıt: 200 `{"code":0,"msg":"bağlantı başarılı",...,"connected":true}`

```bash
curl -X POST $BASE/api/v1/system/docreader/reconnect -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"addr":"docreader:50051"}'
```

### GET /api/v1/system/storage-engine-status

Amaç: Nesne depolama motoru kullanılabilirliği. Yetki: Viewer+.

Yanıt: 200 `{"code":0,"msg":"success","data":{"engines":[{name,allowed,available,description}],"allowed_providers":[...],"minio_env_available":bool}}`

```bash
curl $BASE/api/v1/system/storage-engine-status -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/storage-engine-check

Amaç: Depolama yapılandırmasını doğrulamak (SSRF korumasından sonra yoklama). Yetki: Admin+. İstek gövdesi: `provider` (zorunlu, `minio/cos/tos/s3/oss/ks3/obs`) + ilgili `minio|cos|tos|s3|oss|ks3|obs` yapılandırma nesnesi.

Yanıt: 200 `{"code":0,"data":{"ok","message","bucket_created"}}`

```bash
curl -X POST $BASE/api/v1/system/storage-engine-check -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"provider":"minio","minio":{"endpoint":"minio:9000"}}'
```

## Sistem Yönetimi (/api/v1/system/admin, yalnızca SystemAdmin)

Grup düzeyinde `SystemAdmin()` koruması bağlanır (her zaman zorunludur, EnableRBAC'den etkilenmez); platform API anahtarı ilgili `system_*` capability gerektirir. Bu grubun okuma arayüzleri çoğunlukla ham satırlar/diziler döndürür (sarmalayıcı yoktur). Handler: `internal/handler/system.go`, `internal/handler/audit_log.go`.

### POST /api/v1/system/admin/promote

Amaç: SystemAdmin yetkisi vermek. İstek gövdesi: `user_id` (UUID, öncelikli) veya `email` (ikiden biri).

Yanıt: 200 `UserInfo` (ham nesne).

```bash
curl -X POST $BASE/api/v1/system/admin/promote -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"admin@ex.com"}'
```

### POST /api/v1/system/admin/revoke

Amaç: SystemAdmin yetkisini geri almak. İstek gövdesi: `{"user_id":"..."}` (`binding:"required"`).

Yanıt: 200 `UserInfo`

```bash
curl -X POST $BASE/api/v1/system/admin/revoke -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"user_id":"u-1"}'
```

### GET /api/v1/system/admin/list

Amaç: SystemAdmin listesi. Sorgu parametreleri: `offset` (varsayılan 0), `limit` (varsayılan 50, üst sınır 200).

Yanıt: 200 `{"total":N,"admins":[UserInfo]}`

```bash
curl $BASE/api/v1/system/admin/list -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/users/create

Yalnızca sistem yöneticileri; bu uç nokta platform API Key için açık değildir. İstek alanları: username (2–50 karakter), email (geçerli e-posta), password (isteğe bağlı veya null ise otomatik oluşturulur). Açıkça verilen boş dize yine de parola politikası doğrulamasından geçmelidir, otomatik oluşturma olarak değerlendirilmez.

| HTTP durumu | Yanıt ve anlamı |
| --- | --- |
| 201 | `{user:UserInfo,generated_password?}`, yeni oluşturuldu; parola yalnızca otomatik oluşturulduğunda döner |
| 200 | `{user:UserInfo}`, kimlik zaten mevcut, hesap veya parola değiştirilmez |
| 400 | Parametreler veya parola politikası karşılanmıyor |
| 409 | E-posta ve kullanıcı adı farklı kimliklere karşılık geliyor |

Bu, success/data sarmalaması olmayan ve idempotent alanı içermeyen ham yanıt nesnesidir. Yeni oluşturulan ve mevcut hesapları ayırmak için HTTP durumunu kullanın. Alan tahsisi auth.default_tenant_mode kuralını izler.

```bash
curl -i -X POST "$BASE/api/v1/system/admin/users/create" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com"}'
```

### POST /api/v1/system/admin/users/reset-password

Amaç: Kullanıcı parolasını sıfırlamak. İstek gövdesi: `email` (`binding:"required,email"`), `new_password` (`binding:"required"`).

Yanıt: 200 `{"message":"Password reset successfully"}`

```bash
curl -X POST $BASE/api/v1/system/admin/users/reset-password -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"a@ex.com","new_password":"newpass1"}'
```

### GET /api/v1/system/admin/api-keys

Amaç: Platform API key listesi (maskeli).

Yanıt: 200 `{"success":true,"data":[{id,name,api_key,capabilities,expires_at_unix,...}]}`

```bash
curl $BASE/api/v1/system/admin/api-keys -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/api-keys

Amaç: Platform API key oluşturmak (açık metin yalnızca bir kez döner). İstek gövdesi: `name` (boş olmamalı), `capabilities` (`system_*` listesi, zorunlu), `expires_at_unix` (isteğe bağlı, gelecekteki bir zaman olmalıdır).

Yanıt: 201 `{"success":true,"data":{...,"api_key":"<açık metin>","token":"<açık metin>"}}`

```bash
curl -X POST $BASE/api/v1/system/admin/api-keys -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"ops","capabilities":["system_tenants_read"]}'
```

### DELETE /api/v1/system/admin/api-keys/:key_id

Amaç: Platform API key silmek.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/system/admin/api-keys/3 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/settings ile GET /api/v1/system/admin/settings/:key

Amaç: Platform çalışma zamanı ayarları listesi / tekil ayar (platform key için `system_settings_read|manage` gerekir).

Yanıt: 200 `[SystemSetting]` / `SystemSetting` (ham, sarmalamasız; alanlar: `key,value,value_type,description,last_modified_by,last_modified_at`).

```bash
curl $BASE/api/v1/system/admin/settings -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/system/admin/settings/:key

Amaç: Ayarları güncellemek (platform key için `system_settings_manage` gerekir). İstek gövdesi: `{"value":<herhangi bir JSON, kayıt defteri türüne göre doğrulanır>}` (zorunlu).

Yanıt: 200 `SystemSetting`

```bash
curl -X PUT $BASE/api/v1/system/admin/settings/default_storage_quota -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"value":10737418240}'
```

### DELETE /api/v1/system/admin/settings/:key

Amaç: Ayarları varsayılan değerlere geri yüklemek.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/system/admin/settings/default_storage_quota -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/runtime/queues

Amaç: asynq kuyruk derinliği ve eşzamanlılık durumu (Lite modu `available:false` döner; platform key için `system_runtime_read|manage` gerekir).

Yanıt: 200 `{"available",upstream_concurrency,parse_concurrency,wiki_concurrency,pools,queues,model_limiter_available,models,timestamp}`

```bash
curl $BASE/api/v1/system/admin/runtime/queues -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/runtime/queues/:queue/tasks

Amaç: kuyruk görev listesi. Sorgu parametreleri: `state` (`pending/active/scheduled/retry/archived/completed`), `cursor`, `page_size` (varsayılan 20, üst sınır 100).

Yanıt: 200 `{"available","tasks":[RuntimeTaskInfo],"page_size","has_more","next_cursor"}`

```bash
curl "$BASE/api/v1/system/admin/runtime/queues/default/tasks?state=pending" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/runtime/queues/:queue/tasks/:task_id/actions/:action

Amaç: görev işlemleri (`action` ∈ `cancel/run_now/delete`; platform anahtarı için `system_runtime_manage` gereklidir).

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/system/admin/runtime/queues/default/tasks/t-1/actions/cancel \
  -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/system/admin/runtime/queues/:queue/archived

Amaç: arşivlenmiş görevleri temizleme.

Yanıt: 200 `{"success":true,"deleted":N}`

```bash
curl -X DELETE $BASE/api/v1/system/admin/runtime/queues/default/archived -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/tenants/apply-default-storage-quota

Amaç: geçerli varsayılan depolama kotasını toplu olarak tüm alanlara yazma (platform anahtarı için `system_tenants_manage` gereklidir). İstek gövdesi yoktur.

Yanıt: 200 `{"affected":N,"quota_bytes":N,"quota_gb":N}`

```bash
curl -X POST $BASE/api/v1/system/admin/tenants/apply-default-storage-quota -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/audit-log

Amaç: platform düzeyi denetim günlükleri (`tenant_id=0` satırları; platform anahtarı için `system_audit_read` gereklidir). Sorgu parametreleri alan denetimiyle aynıdır (`after_id/limit/action/outcome/actor`). İşleyici: `internal/handler/audit_log.go`

Yanıt: 200 `{"success":true,"data":[AuditLog],"next_cursor":N}`

```bash
curl $BASE/api/v1/system/admin/audit-log -H "Authorization: Bearer $TOKEN"
```

## Uygulama referansı

Rota kaydı: `internal/router/routes_auth_tenant.go` içindeki `RegisterSystemAdminRoutes` ve `RegisterSystemRoutes`. İşleyiciler: `internal/handler/system.go`, `internal/handler/audit_log.go`.


## Model kataloğu (/api/v1/system/admin/model-catalog)

Yalnızca sistem yöneticisi kullanıcı oturumları erişebilir; API Key için bu arayüz grubu açık değildir.

| Yöntem ve yol | İşlev |
| --- | --- |
| `GET /system/admin/model-catalog` | Geçerli `version`, `baseline`, yönetici `overlay`, en son 20 geçmiş sürümü ve `builtin` / `deployment` / `effective` kataloğunu döndürür |
| `POST /system/admin/model-catalog/preview` | Geçersiz kılma belgesini doğrular ve aday kataloğu döndürür (yalnızca `effective` ve normalleştirilmiş `overlay`; `history` / `builtin` / `deployment` için `null`); kalıcı olarak saklamaz ve yayımlamaz |
| `PUT /system/admin/model-catalog` | Doğrular, yeni sürümü kaydeder ve yayımlar; denetim yalnızca sürüm meta verilerini kaydeder |

Önizleme ve yayımlama aynı istek gövdesini kullanır:

```json
{
  "version": 0,
  "baseline": "GET yanıtında dönen dağıtım temel kimliği",
  "overlay": {
    "providers": {
      "openai": {
        "models": [{"id": "gpt-5", "context_window": 128000}]
      }
    }
  }
}
```

Yanıt, sarmalanmamış bir katalog durumu nesnesidir. Her sağlayıcı girdisi ayrıca `model_thinking_levels` (sohbet modeli id'sine göre, düşünme açıldığında seçilebilecek seviyeler; sağlayıcı eşlemesi ve protokol yetenekleri birleştirilmiştir) ve `vendor_thinking_levels` (ayrıca seviye yapılandırılmamış modellerin kullandığı sağlayıcı varsayılan seviyeleri) alanlarını içerir. Geçersiz belge 400 döndürür; sürüm eskiyse veya isteği alan örneğin dağıtım temeli uyuşmuyorsa 409 döner. Yayınlama önce kalıcı olarak kaydeder, sonra bu örneğe geçer; diğer örnekler yaklaşık 5 saniye içinde senkronize olur. Geri almak için geçmişteki `overlay`, güncel `version` / `baseline` ile birlikte yeniden yayımlanır. Kuralların ve sınırların tamamı için bkz. [Model yönetimi](../03-features/06-models.md#sistem-yoneticileri-model-katalogunu-yonetir).
