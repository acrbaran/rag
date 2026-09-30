# API Referansı: Kiracı (alan) ve üyeler

Çalışma alanlarını, üyeleri, davetleri, API Key'leri ve denetim günlüklerini yönetin. İşlemler mevcut etkin alan üzerinde uygulanır; alanlar arası erişim uç nokta izinlerine göre doğrulanır.

Tüm `/tenants/:id/*` rotaları grup düzeyinde `PathTenantMatch()` ile bağlanır (`internal/middleware/access.go`): URL'deki `:id`, mevcut etkin alanla eşit olmalıdır (alanlar arası süper yönetici istisnası); bu, başkalarının alanlarında yetkisiz işlem yapılmasını önler.

Kiracı `memory_config` yapılandırma alanı ve kişisel bellek uç noktaları için [uzun süreli bellek API'sine](02-api-memory.md) bakın. Alan yöneticisi yapılandırmayı güncellerken korunacak tam nesneyi göndermelidir.

## Alan yaşam döngüsü

### POST /api/v1/tenants

Amaç: Alan oluşturmak (kendi kendine yeni çalışma alanı açma; çağıran otomatik olarak Owner olur). İzin: Oturum açmış herhangi bir kullanıcı (alanı olmayabilir); API key yalnızca platform key olmalı ve `system_tenants_manage` iznine sahip olmalıdır. Handler: `internal/handler/tenant.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet (`binding:"required,min=1,max=128"`) | Alan adı |
| `description` | string | Hayır (`binding:"max=512"`) | Açıklama |

Alanlar arası süper yönetici, tam `types.Tenant` nesnesini (`storage_quota`, `status` vb. dahil) gönderebilir.

Yanıt: 201 `{"success":true,"data":{Tenant}}` (yapılandırma izin verirse `api_key` içerebilir). Kendi kendine oluşturma devre dışıysa 403 (code 2005), kota aşımında 429 döner.

```bash
curl -X POST $BASE/api/v1/tenants -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Alanım"}'
```

### GET /api/v1/tenants

Amaç: Erişebildiğim alanları listelemek. İzin: Oturum açmış kullanıcı; API key için `manage_tenant_settings` veya full-access gerekir. Handler: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"data":{"items":[TenantResponse]}}`

```bash
curl $BASE/api/v1/tenants -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/all

Amaç: Tüm alanları listelemek (alanlar arası süper yönetici). İzin: `CrossTenant()` (`CanAccessAllTenants` ve kümede `EnableCrossTenantAccess` etkin); platform key için `system_tenants_read|manage` gerekir. Handler: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"data":{"items":[TenantResponse]}}`

```bash
curl $BASE/api/v1/tenants/all -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/search

Amaç: Anahtar sözcüğe göre alan aramak (alanlar arası süper yönetici). İzin: Yukarıdakiyle aynı. Handler: `internal/handler/tenant.go`

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `keyword` | string | Hayır | Anahtar sözcük |
| `tenant_id` | string | Hayır | Tam alan ID'si |
| `page` / `page_size` | int | Hayır | Sayfalama (varsayılan 1/20, üst sınır 100) |

Yanıt: 200 `{"success":true,"data":{"items":[...],"total","page","page_size"}}`

```bash
curl "$BASE/api/v1/tenants/search?keyword=demo&page=1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/:id

Amaç: Alan ayrıntıları. İzin: Viewer+; platform key için `system_tenants_read|manage` gerekir. Handler: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"data":{TenantResponse}}`

```bash
curl $BASE/api/v1/tenants/1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/tenants/:id

Amaç: Alan yapılandırmasını güncellemek. İzin: Owner; platform key için `system_tenants_manage` gerekir. Handler: `internal/handler/tenant.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | *string | Hayır (`binding:"omitempty,min=1,max=128"`) | Yeni ad |
| `description` | *string | Hayır (`binding:"omitempty,max=512"`) | Yeni açıklama |

Yanıt: 200 `{"success":true,"data":{TenantResponse}}`

```bash
curl -X PUT $BASE/api/v1/tenants/1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Yeni ad"}'
```

### DELETE /api/v1/tenants/:id

Amaç: Alanı siler. Yetki: Owner; platform key için `system_tenants_manage` gereklidir. Alan kaydı ve tüm üye ilişkileri mantıksal olarak silinir, üyeler hemen erişimlerini kaybeder; alan içindeki bilgi tabanları, modeller ve diğer veriler hemen fiziksel olarak temizlenmez, silinen alanın kuyruğundaki Wiki görevleri artık modeli çağırmaz. Handler: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"message":"Workspace deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1 -H "Authorization: Bearer $TOKEN"
```

## Alan KV Yapılandırması

`:key`, alan kimliği değil yapılandırma anahtarıdır (alan kimlik doğrulama bağlamından alınır); isteğe bağlı değerler: `web-search-config`, `prompt-templates`, `parser-engine-config`, `storage-engine-config`, `chat-history-config`, `retrieval-config`, `memory-config`.

### GET /api/v1/tenants/kv/:key

Amaç: Alan düzeyindeki KV yapılandırmasını okur. Yetki: Viewer+; API key için `manage_tenant_settings` veya full-access gereklidir. Handler: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"data":{...ilgili yapılandırma nesnesi...}}`

```bash
curl $BASE/api/v1/tenants/kv/retrieval-config -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/tenants/kv/:key

Amaç: Alan düzeyindeki KV yapılandırmasını günceller. Yetki: Admin+; API key için `manage_tenant_settings` veya full-access gereklidir. İstek gövdesi: `:key` ile eşleşen yapılandırma JSON nesnesi. Handler: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"message":"Configuration updated"}`

```bash
curl -X PUT $BASE/api/v1/tenants/kv/web-search-config -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":true}'
```

## API Key ve API Öznesi

### GET /api/v1/tenants/:id/api-keys

Amaç: Alan API key'lerini listeler (maskeli gösterim). Yetki: Owner, yalnızca JWT (API key varsayılan olarak reddedilir). Handler: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"data":[{id,scope_type,name,api_key(maskeli),full_access,knowledge_base_ids,capabilities,last_used_at,expires_at,created_at}]}`

```bash
curl $BASE/api/v1/tenants/1/api-keys -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/api-keys

Amaç: Alan API key'i oluşturur (açık metin yalnızca bir kez döndürülür). Yetki: Owner, yalnızca JWT. Handler: `internal/handler/tenant.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet | key adı |
| `full_access` | bool | Hayır | Alan için tam yetkili key (varsayılan false) |
| `knowledge_base_ids` | []string | Hayır | KB beyaz listesi (scoped key) |
| `capabilities` | []string | Hayır | capability listesi (genel bakışa bakın) |
| `expires_at_unix` | *int64 | Hayır | Son kullanma zaman damgası |

Yanıt: 201 `{"success":true,"data":{...,"api_key":"<açık metin>","token":"<açık metin>"}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/api-keys -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"ingest-bot","capabilities":["ingest","retrieve"],"knowledge_base_ids":["kb-1"]}'
```

### PUT /api/v1/tenants/:id/api-keys/:key_id

Owner, yalnızca JWT. Mevcut bir Key'in name, full_access, knowledge_base_ids, capabilities ve expires_at_unix alanlarını günceller; yetkilendirme alanları tek bir alanı değiştiren PATCH olarak değil, tam yapılandırma olarak gönderilir. expires_at_unix atlanırsa veya null olursa mevcut son kullanma zamanı temizlenir. Yetkiler değiştirildikten sonra aynı token kullanılır; yeni yetkiler sonraki kimlik doğrulamada geçerli olur ve açık metin yeniden döndürülmez.

200 `{success,data:APIKeyResponse}` döndürür, Key maskelenir; geçersiz capability/bilgi tabanı kapsamı için 400, bulunamazsa 404 döndürülür.

```bash
curl -X PUT "$BASE/api/v1/tenants/1/api-keys/5" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"search-bot","full_access":false,"knowledge_base_ids":["kb-1"],"capabilities":["retrieve"]}'
```

### DELETE /api/v1/tenants/:id/api-keys/:key_id

Amaç: API anahtarını silmek. Yetki: Owner, yalnızca JWT. Yol parametresi: `key_id`.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1/api-keys/5 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/:id/api-principal-config

Amaç: API harici kullanıcı öznesi yapılandırmasını okumak. Yetki: Owner, yalnızca JWT. İşleyici: `internal/handler/tenant.go`

Yanıt: 200 `{"success":true,"data":{"mode":"tenant|direct|signed_token","direct_header_name","signed_token_header_name","require_direct_header","has_hmac_secret"}}`

```bash
curl $BASE/api/v1/tenants/1/api-principal-config -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/tenants/:id/api-principal-config

Amaç: API harici kullanıcı öznesi yapılandırmasını güncellemek. Yetki: Owner, yalnızca JWT.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `mode` | string | Evet | `tenant` / `direct` / `signed_token` |
| `require_direct_header` | bool | Hayır | direct modunda başlığın zorunlu olup olmadığı |
| `hmac_secret` | *string | Hayır | signed_token modu anahtarı (`***` gönderilirse mevcut değer korunur) |

Yanıt: 200, GET ile aynı.

```bash
curl -X PUT $BASE/api/v1/tenants/1/api-principal-config -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"mode":"signed_token","hmac_secret":"topsecret"}'
```

### POST /api/v1/tenants/:id/api-principal-test-token

Amaç: Test için harici kullanıcı JWT'si düzenlemek. Yetki: Owner, yalnızca JWT.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `external_user_id` | string | Evet | Harici kullanıcı ID'si (≤128 karakter) |
| `expires_in_seconds` | int | Hayır | 1-3600, varsayılan 900 |

Yanıt: 200 `{"success":true,"data":{"token","header_name","expires_in_seconds","expires_at_unix","external_user_id"}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/api-principal-test-token -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"external_user_id":"u-123"}'
```

## Üye yönetimi (/tenants/:id/members)

İşleyici: `internal/handler/tenant_member.go`. API anahtarı için `manage_members` veya full-access gerekir.

### GET /api/v1/tenants/:id/members

Amaç: Üye listesi. Yetki: Viewer+.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `q` | string | Hayır | E-posta/kullanıcı adı filtresi |
| `page` / `page_size` | int | Hayır | Sayfalama |

Yanıt: 200 `{"success":true,"data":{"members":[{user_id,email,username,avatar,role,status,invited_by,joined_at}],"total","page","page_size"}}`

```bash
curl $BASE/api/v1/tenants/1/members -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/members

Amaç: Doğrudan üye eklemek. Yetki: Owner.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `email` | string | Evet (`binding:"required,email"`) | Üye e-postası (önceden kayıtlı olmalıdır) |
| `role` | string | Evet (`binding:"required"`) | `owner/admin/contributor/viewer` |

Yanıt: 201 `{"success":true,"data":{üye nesnesi}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/members -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"b@ex.com","role":"contributor"}'
```

### PUT /api/v1/tenants/:id/members/:user_id

Amaç: Üye rolünü değiştirmek. Yetki: Owner. İstek gövdesi: `{"role":"admin"}` (`binding:"required"`).

Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/tenants/1/members/u-123 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"role":"admin"}'
```

### DELETE /api/v1/tenants/:id/members/:user_id

Amaç: Üyeyi kaldırmak. Yetki: Owner.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1/members/u-123 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/leave

Amaç: Alandan ayrılmak (herhangi bir üye kendisi ayrılabilir; servis katmanı alanı Owner olmadan bırakan ayrılmaları reddeder). Yetki: Viewer+, yalnızca JWT.

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/tenants/1/leave -H "Authorization: Bearer $TOKEN"
```

## Alan davetleri (/tenants/:id/invitations ve invite-links)

İşleyici: `internal/handler/tenant_invitation.go`, `internal/handler/tenant_invite_link.go`. API anahtarı için `manage_members` veya full-access gerekir.

### GET /api/v1/tenants/:id/invitations

Amaç: Alan davetleri listesi. Yetki: Viewer+.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `include_terminal` | bool | Hayır | Tamamlanmış davetleri dahil et |
| `page` / `page_size` | int | Hayır | Sayfalama |

Yanıt: 200 `{"success":true,"data":{"invitations":[{id,tenant_id,invitee_email,inviter_email,role,status,message,expires_at,is_share_link,accepted_count,...}],"total","page","page_size"}}`

```bash
curl $BASE/api/v1/tenants/1/invitations -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/invitations

Amaç: Üye davet etmek (davet edilen kişi `/me/invitations` içinde onayladıktan sonra kaydedilir). Yetki: Owner.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `email` | string | Evet (`binding:"required,email"`) | Davet edilen e-posta adresi |
| `role` | string | Evet (`binding:"required"`) | Verilecek rol |
| `message` | string | Hayır | Not |

Yanıt: 201 `{"success":true,"data":{TenantInvitationResponse}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/invitations -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"c@ex.com","role":"viewer"}'
```

### DELETE /api/v1/tenants/:id/invitations/:inv_id

Amaç: Daveti iptal etmek. Yetki: Owner.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1/invitations/12 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/invite-links

Amaç: Paylaşım bağlantısı oluşturmak (birden çok kez kullanılabilen kayıt davet bağlantısı). Yetki: Owner. İşleyici: `internal/handler/tenant_invite_link.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `role` | string | Evet (`binding:"required"`) | Bağlantının verdiği rol |
| `message` | string | Hayır | Ek not |

Yanıt: 201 `{"success":true,"data":{id,token,invite_url,role,status,expires_at,is_share_link:true,accepted_count}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/invite-links -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"role":"viewer"}'
```

## Denetim günlükleri

Handler: `internal/handler/audit_log.go`. İmleç tabanlı sayfalama.

### GET /api/v1/tenants/:id/audit-log

Amaç: Alan denetim günlükleri (reddedilen işlem kayıtları dahil). Yetki: Yönetici+, yalnızca JWT.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `after_id` | int | Hayır | İmleç (önceki yanıttaki `next_cursor`) |
| `limit` | int | Hayır | 1-100, varsayılan 50 |
| `action` | string | Hayır | İşleme göre filtrele (ör. `rbac.member_added`) |
| `outcome` | string | Hayır | `success` / `denied` |
| `actor` | string | Hayır | İşlemi yapanın user_id değerine göre filtrele |

Yanıt: 200 `{"success":true,"data":[AuditLog],"next_cursor":N}`

```bash
curl "$BASE/api/v1/tenants/1/audit-log?limit=50" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/activity

Amaç: Tek bir KB için etkinlik akışı (salt okunur denetim). Yetki: KB oluşturucusu VEYA Yönetici+ ve KB için okuma izni; yalnızca JWT. Sorgu parametreleri yukarıdakiyle aynıdır (`after_id/limit/action/outcome/actor`). `RegisterKnowledgeBaseActivityRoutes` içinde kaydedilir.

Yanıt: 200 `{"success":true,"data":[AuditLog],"next_cursor":N}`. `details`, işlem yüküdür; bu kayıt bir API Key tarafından tetiklendiyse `api_key_id` ve `api_key_name` içerir (adın anlık görüntüsü, düz metin Key içermez).

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/activity -H "Authorization: Bearer $TOKEN"
```

## Uygulama referansı

Rota kaydı: `internal/router/routes_auth_tenant.go` içindeki `RegisterTenantRoutes`. Handler: `internal/handler/tenant.go`, `internal/handler/tenant_member.go`, `internal/handler/tenant_invitation.go`, `internal/handler/tenant_invite_link.go`, `internal/handler/audit_log.go`.
