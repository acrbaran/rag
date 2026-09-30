# API referansı: organizasyon ve paylaşım

Kuruluş üyelerini, bilgi tabanlarını ve ajanların paylaşım ilişkilerini yönetin. Kuruluşlar çalışma alanlarını üye birimleri olarak kullanır.

Kuruluş (Organization), üye birimleri olarak “alanları (tenant)” kullanır. Kuruluş grubu rotaları için API key politikası `manage_spaces` veya full-access olmalıdır; KB/Ajan paylaşım yönetimi yalnızca full-access key ile kullanılabilir.

## Kuruluş Yönetimi (/api/v1/organizations)

### POST /api/v1/organizations

Amaç: Kuruluş oluşturmak. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet | Kuruluş adı |
| `description` | string | Hayır | Açıklama |
| `avatar` | string | Hayır | Avatar URL'si |
| `searchable` | bool | Hayır | Aramayla bulunabilir olup olmadığı |
| `require_approval` | bool | Hayır | Katılım için onay gerekip gerekmediği |
| `member_limit` | int | Hayır | Üye alan sayısı üst sınırı |
| `invite_code_validity_days` | int | Hayır | Davet kodu geçerlilik süresi (gün) |

Yanıt: 201 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X POST $BASE/api/v1/organizations -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Ar-Ge organizasyonu"}'
```

### GET /api/v1/organizations

Amaç: Üyesi olduğum kuruluşları listelemek. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{"organizations":[...],"total":N,"resource_counts":{"knowledge_bases":{"by_organization":{}},"agents":{"by_organization":{}}}}}`

```bash
curl $BASE/api/v1/organizations -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/preview/:code

Amaç: Davet koduyla kuruluşu önizlemek (katılmadan). Yetki: Viewer+. Yol parametresi: `code` davet kodu.

Yanıt: 200 `{"success":true,"data":{id,name,description,avatar,member_count,share_count,agent_share_count,is_already_member,require_approval,created_at}}`

```bash
curl $BASE/api/v1/organizations/preview/ABC123 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/join

Amaç: Davet koduyla kuruluşa katılmak. Yetki: Admin+. İstek gövdesi: `{"invite_code":"..."}` (zorunlu).

Yanıt: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X POST $BASE/api/v1/organizations/join -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"invite_code":"ABC123"}'
```

### POST /api/v1/organizations/join-request

Amaç: Katılım başvurusu göndermek (onay gerektiren kuruluşlar için). Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `invite_code` | string | Evet | Davet kodu |
| `message` | string | Hayır | Başvuru notu |
| `role` | string | Hayır | Beklenen rol |

Yanıt: 200 `{"success":true,"data":{JoinRequest}}`

```bash
curl -X POST $BASE/api/v1/organizations/join-request -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"invite_code":"ABC123","message":"Katılma talebi"}'
```

### GET /api/v1/organizations/search

Amaç: keşfedilebilir (searchable) organizasyonları aramak. Yetki: Viewer+.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `q` | string | Hayır | Anahtar kelime |
| `limit` | int | Hayır | Varsayılan 20, üst sınır 100 |

Yanıt: 200 `{"success":true,"data":[SearchableOrganization],"total":N}`

```bash
curl "$BASE/api/v1/organizations/search?q=Ar-Ge" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/join-by-id

Amaç: organizasyon ID'sine göre keşfedilebilir bir organizasyona katılmak (davet kodu gerekmez). Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `organization_id` | string | Evet | Hedef organizasyon ID'si |
| `message` | string | Hayır | Ek not |
| `role` | string | Hayır | İstenen rol |

Yanıt: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X POST $BASE/api/v1/organizations/join-by-id -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"organization_id":"org-1"}'
```

### GET /api/v1/organizations/:id

Amaç: organizasyon ayrıntıları. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl $BASE/api/v1/organizations/org-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/organizations/:id

Amaç: organizasyonu güncellemek (servis katmanı, çağıranın alanının organizasyon admin'i olduğunu doğrular; owner ile sınırlı değildir). Yetki: Admin+. İstek gövdesi alanları oluşturma ile aynıdır (tümü isteğe bağlıdır).

Yanıt: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X PUT $BASE/api/v1/organizations/org-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"description":"Güncellenmiş açıklama"}'
```

### DELETE /api/v1/organizations/:id

Amaç: organizasyonu silmek. Yetki: Admin+ (servis katmanı organizasyon owner'ı olmayı gerektirir).

Yanıt: 200 `{"success":true,"message":"Organization deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/organizations/org-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/:id/leave

Amaç: bu alanın organizasyondan ayrılması. Yetki: Admin+. İstek gövdesi yoktur. Bu alanın organizasyona paylaştığı bilgi tabanları ve Agent'lar da birlikte geri alınır.

Yanıt: 200 `{"success":true,"message":"Left organization successfully"}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/leave -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/:id/request-upgrade

Amaç: bu alanın organizasyon içindeki rolünün yükseltilmesi için başvurmak. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `requested_role` | string | Evet | İstenen organizasyon rolü (`viewer/editor/admin`) |
| `message` | string | Hayır | Ek not |

Yanıt: 200 `{"success":true,"data":{JoinRequest}}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/request-upgrade -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"requested_role":"editor"}'
```

### POST /api/v1/organizations/:id/invite-code

Amaç: organizasyon davet kodu oluşturmak. Yetki: Admin+ (servis katmanı organizasyon admin'i olmayı gerektirir). İstek gövdesi yoktur.

Yanıt: 200 `{"success":true,"data":{"invite_code":"..."}}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/invite-code -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/search-tenants

Amaç: Alan kimliğine göre davet edilebilir alanları çözümlemek. Yetki: Admin+; çağıranın alanı kuruluş yöneticisi olmalıdır. v0.8.2'den itibaren yalnızca tam alan kimlikleri kabul edilir, artık alan adına göre alanlar arası arama yapılmaz ve `limit` parametresi kaldırılmıştır.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `q` | string | Evet | Tam alan kimliği |

Yanıt: 200 `{"success":true,"data":[{"tenant_id","tenant_name"}]}`. `q` geçerli bir kimlik değilse, alan mevcut değilse veya zaten kuruluş üyesiyse boş dizi döner; aksi durumda tek aday döner.

```bash
curl "$BASE/api/v1/organizations/org-1/search-tenants?q=10002" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/search-users

Amaç: Kullanımdan kaldırılmış takma ad; davranışı `search-tenants` ile aynıdır. Yetki: Admin+. Parametreler yukarıdaki gibidir.

```bash
curl "$BASE/api/v1/organizations/org-1/search-users?q=10002" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/:id/invite

Amaç: Bir alanı doğrudan kuruluşa davet etmek. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `tenant_id` | uint64 | İkisinden biri | Hedef alan kimliği (önerilir) |
| `user_id` | string | İkisinden biri | Uyumluluk yolu: kullanıcı kimliği (alanına çözülür) |
| `representative_user_id` | string | Hayır | Yok sayılır, yalnızca uyumluluk için korunur: doğrudan eklenen alanlara temsilci kullanıcı atanmaz; böylece davet eden taraf, karşı alanın hangi kullanıcısının bilgilerinin üye listesinde görüneceğini belirleyemez |
| `role` | string | Evet | Kuruluş içindeki rol |

Yanıt: 200 `{"success":true,"message":"Member added successfully"}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/invite -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"tenant_id":2,"role":"viewer"}'
```

### GET /api/v1/organizations/:id/members

Amaç: Kuruluş üyeleri (alanlar) listesi. Yetki: Viewer+. `email` yalnızca çağıranın kendi alanına ait satırda döner; diğer alanlar için yalnızca kullanıcı adı ve avatar döner.

Yanıt: 200 `{"success":true,"data":{"members":[{id,user_id,representative_user_id,role,tenant_id,tenant_name,username,email,avatar,joined_at}],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/members -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/organizations/:id/members/:tenant_id

Amaç: Üye alanın kuruluş rolünü değiştirmek. Yetki: Admin+. Yol parametresi `tenant_id`, üye alan kimliğidir. İstek gövdesi: `{"role":"editor"}` (zorunlu, `viewer/editor/admin`).

Yanıt: 200 `{"success":true,"message":"Member role updated successfully"}`

```bash
curl -X PUT $BASE/api/v1/organizations/org-1/members/2 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"role":"editor"}'
```

### DELETE /api/v1/organizations/:id/members/:tenant_id

Amaç: Üye alanı kaldırmak (kendini kaldırma dahil). Yetki: Admin+. Kaldırılan alanın bu kuruluşla paylaştığı bilgi tabanları ve Agent'lar da birlikte iptal edilir.

Yanıt: 200 `{"success":true,"message":"Member removed successfully"}`

```bash
curl -X DELETE $BASE/api/v1/organizations/org-1/members/2 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/join-requests

Amaç: Katılım başvurusu kuyruğu. Yetki: Admin+.

Yanıt: 200 `{"success":true,"data":{"requests":[{id,user_id,username,email,message,request_type,prev_role,requested_role,status,created_at,reviewed_at}],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/join-requests -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/organizations/:id/join-requests/:request_id/review

Amaç: Katılım/yükseltme başvurularını onaylamak. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `approved` | bool | Evet | Onayla/Reddet |
| `message` | string | Hayır | Onay görüşü |
| `role` | string | Hayır | Onaylandığında verilecek rol |

Yanıt: 200 `{"success":true,"message":"Review completed"}`

```bash
curl -X PUT $BASE/api/v1/organizations/org-1/join-requests/req-1/review \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"approved":true}'
```

### GET /api/v1/organizations/:id/shares

Amaç: Bu kuruluşa paylaşılan KB listesini görüntülemek. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{"shares":[KnowledgeBaseShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/shares -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/agent-shares

Amaç: Bu kuruluşa paylaşılan Agent listesini görüntülemek. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{"shares":[AgentShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/agent-shares -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/shared-knowledge-bases

Amaç: Kuruluş alanı görünümü: kuruluş içindeki tüm paylaşılan KB'ler (kendiminkiler dahil). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[... is_mine, source_from_agent işaretleri dahil ...],"total":N}`

```bash
curl $BASE/api/v1/organizations/org-1/shared-knowledge-bases -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/shared-agents

Amaç: Kuruluş alanı görünümü: kuruluş içindeki tüm paylaşılan Agent'lar. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[SharedAgentInfo],"total":N}`

```bash
curl $BASE/api/v1/organizations/org-1/shared-agents -H "Authorization: Bearer $TOKEN"
```

## KB Paylaşımı (/api/v1/knowledge-bases/:id/shares)

API anahtarı: yalnızca full-access. İşleyici: `internal/handler/organization.go`

### POST /api/v1/knowledge-bases/:id/shares

Amaç: KB'yi kuruluşa paylaşmak. Yetki: KB oluşturucusu OR Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `organization_id` | string | Evet | Hedef kuruluş |
| `permission` | string | Evet | Paylaşım izni (kuruluş rolü anlamları, örneğin `viewer/editor`) |

Yanıt: 201 `{"success":true,"data":{KBShare}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/shares -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"organization_id":"org-1","permission":"viewer"}'
```

### GET /api/v1/knowledge-bases/:id/shares

Amaç: Bu KB'nin paylaşım listesini görüntülemek. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{"shares":[KnowledgeBaseShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/shares -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id/shares/:share_id

Amaç: Paylaşım iznini değiştirmek. Yetki: KB oluşturucusu OR Admin+. İstek gövdesi: `{"permission":"editor"}` (zorunlu).

Servis katmanı kuralları (paylaşımı iptal etme ile ortak):

- Asıl paylaşımı yapan kişi, KB'nin ait olduğu alanda işlem yapmalı ve alan rolü Contributor+ olmalıdır;
- KB'nin ait olduğu alanın Admin+ kullanıcıları, bu alandaki tüm paylaşımları yönetebilir;
- Hedef kuruluşta rolü admin olan alanların Admin+ kullanıcıları yalnızca izni **azaltabilir** veya paylaşımı iptal edebilir; izni mevcut değerin üzerine yükseltemezler.

`share_id`, yoldaki KB'ye ait olmalıdır; aksi hâlde 404 döner.

Yanıt: 200 `{"success":true,"message":"Share permission updated successfully"}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/shares/s-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"permission":"editor"}'
```

### DELETE /api/v1/knowledge-bases/:id/shares/:share_id

Amaç: paylaşımı iptal etmek. Yetki: KB oluşturucusu VEYA Admin+; servis katmanı kuralları yukarıdakiyle aynıdır. `share_id`, yoldaki KB'ye ait olmalıdır; aksi halde 404.

Yanıt: 200 `{"success":true,"message":"Share removed successfully"}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1/shares/s-1 -H "Authorization: Bearer $TOKEN"
```

## Agent Paylaşımı (/api/v1/agents/:id/shares)

API anahtarı: yalnızca full-access. Handler: `internal/handler/organization.go`

### POST /api/v1/agents/:id/shares

Amaç: Agent'ı organizasyonla paylaşmak. Yetki: Agent oluşturucusu VEYA Admin+. İstek gövdesi KB paylaşımıyla aynıdır (`organization_id` + `permission`, zorunlu). Yerleşik ajanlar paylaşılamaz (400): her alanda aynı ID'ye sahip yerleşik bir ajan bulunur ve paylaşım sonrası alıcı bunları ayırt edemez. Agent'ın bilgi tabanı kapsamı organizasyon üyelerine açılır; bu nedenle çağıranın içindeki her bilgi tabanını doğrudan paylaşma yetkisi olmalıdır (bilgi tabanı oluşturucusu veya Admin+). `kb_selection_mode: all` yalnızca Admin+ tarafından paylaşılabilir; aksi halde 403.

Yanıt: 201 `{"success":true,"data":{AgentShare}}`

```bash
curl -X POST $BASE/api/v1/agents/agent-1/shares -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"organization_id":"org-1","permission":"viewer"}'
```

### GET /api/v1/agents/:id/shares

Amaç: Bu Agent'ın paylaşım listesini görüntülemek. Yetki: Agent oluşturucusu VEYA Admin+.

Yanıt: 200 `{"success":true,"data":{"shares":[AgentShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/agents/agent-1/shares -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/agents/:id/shares/:share_id

Amaç: Agent paylaşımını iptal etmek. Yetki: Agent oluşturucusu VEYA Admin+; servis katmanı kuralları KB paylaşım iptaliyle aynıdır. `share_id`, yoldaki Agent'a ait olmalıdır; aksi halde 404.

Yanıt: 200 `{"success":true,"message":"Share removed successfully"}`

```bash
curl -X DELETE $BASE/api/v1/agents/agent-1/shares/s-1 -H "Authorization: Bearer $TOKEN"
```

## Paylaşılan Kaynaklar Birleşik Görünümü

### GET /api/v1/shared-knowledge-bases

Amaç: Organizasyon aracılığıyla benimle paylaşılan KB'leri listelemek (sahip tarafındaki vektör veritabanı metadatası kaldırılır). Yetki: Viewer+; API anahtarı `manage_spaces` veya full-access gerektirir.

Yanıt: 200 `{"success":true,"data":[...],"total":N}`

```bash
curl $BASE/api/v1/shared-knowledge-bases -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/shared-agents

Amaç: Organizasyon aracılığıyla benimle paylaşılan Agent'ları listelemek. Yetki: Viewer+; API anahtarı yukarıdakiyle aynıdır.

Yanıt: 200 `{"success":true,"data":[SharedAgentInfo],"total":N}`. `SharedAgentInfo`, `source_tenant_id` (kaynak alan), `org_name`, `shared_by_username`, `permission` ve `web_search_ready` içerir — yalnızca "kaynak alandaki çevrimiçi aramanın kullanılabilir olup olmadığı" boole değeri döndürülür; kaynak alanın provider yapılandırması gönderilmez (yapılandırma sızıntısına yol açar) ve alıcı alanın provider ID'siyle karşılaştırma yapılmaz (yanlış kullanılabilir değil bildirimi oluşturur).

Paylaşılan Agent ile diğer arayüzler çağrılırken, aynı adlı Agent birden fazla alan tarafından paylaşılmışsa kaynak alanı belirtmek için `agent_source_tenant_id` verilebilir; bu değer paylaşım ilişkileriyle tek tek doğrulanır. Geçersiz veya yetkisiz olduğunda doğrudan hata verilir, başka bir kaynağa sessizce geri dönülmez.

```bash
curl $BASE/api/v1/shared-agents -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/shared-agents/disabled

Amaç: "Bu alanda belirli bir paylaşılan Agent'ı devre dışı bırak" ayarını yapmak (tüm alanın oturum açılır listesini etkiler). Yetki: Admin+; API anahtarı yukarıdakiyle aynıdır.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `agent_id` | string | Evet (`binding:"required"`) | Paylaşılan Agent ID'si |
| `disabled` | bool | Hayır | Devre dışı olup olmadığı (varsayılan false) |

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/shared-agents/disabled -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"agent_id":"agent-1","disabled":true}'
```

## Uygulama Referansı

Rota kaydı: `internal/router/routes_agent.go` içindeki `RegisterOrganizationRoutes`. Handler: `internal/handler/organization.go`.
