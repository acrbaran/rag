# API Başvurusu: Agent, MCP ve Beceriler

Agent'ları, MCP hizmetlerini ve bunların kimlik bilgilerini, becerileri ve kaynak favorilerini yönetir. Agent'ın araç kapsamı ve çağrı onayı yapılandırması bu arayüz grubu üzerinden yönetilir.

## Agent (/api/v1/agents)

Okuma: Viewer+ (API key `read_agents`/`manage_agents`/`chat`/full); yazma: oluşturan VEYA Admin+ (API key `manage_agents`/full); yerleşik Agent (`is_builtin=true`) her zaman Admin+.

### GET /api/v1/agents/placeholders

Amaç: istem yer tutucusu tanımları (`/:id` rotasından önce kaydedilmelidir). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{"all":{...},"system_prompt":{...},"agent_system_prompt":{...},"context_template":{...},"rewrite_system_prompt":{...},"rewrite_prompt":{...},"fallback_prompt":{...}}}`

```bash
curl $BASE/api/v1/agents/placeholders -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/agents/type-presets

Amaç: akıllı akıl yürütme Agent türü ön ayarları (rag-qa / wiki-qa / hybrid / custom vb.). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[{type,system_prompt,allowed_tools,kb_compatibility}]}`

```bash
curl $BASE/api/v1/agents/type-presets -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/agents

Amaç: özel Agent oluşturma. Yetki: Contributor+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet (`binding:"required"`) | Ad |
| `description` | string | Hayır | Açıklama |
| `avatar` | string | Hayır | Avatar/emoji |
| `config` | object | Hayır | Agent yapılandırması (`types.CustomAgentConfig`, aşağıya bakın) |

`config` ana alanları: `agent_mode` (`quick-answer`/`smart-reasoning`), `agent_type` (`rag-qa/wiki-qa/hybrid-rag-wiki/data-analysis/custom`), `system_prompt`, `model_id`, `temperature` (0-2, geçersizse code 2103 döner), `max_iterations` (1-20, geçersizse code 2102 döner), `allowed_tools` (akıllı akıl yürütmede en az biri zorunlu, code 2101), `mcp_selection_mode`/`mcp_services`, `skills_selection_mode`, `kb_selection_mode`/`knowledge_bases`, `web_search_enabled`, `question_suggestions` vb. (tam tanım için `internal/types/custom_agent.go` dosyasına bakın).

Yanıt: 201 `{"success":true,"data":{id,name,description,avatar,is_builtin,created_by,config,creator_name,...}}`

```bash
curl -X POST $BASE/api/v1/agents -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Satış Sonrası Asistanı","config":{"agent_mode":"quick-answer","kb_selection_mode":"selected","knowledge_bases":["kb-1"]}}'
```

### GET /api/v1/agents

Amaç: Agent listesi (yerleşikler dahil). Yetki: Viewer+. Sorgu parametresi: `creator` (`mine`/`others`, isteğe bağlı).

Yanıt: 200 `{"success":true,"data":[Agent],"disabled_own_agent_ids":[...]}`

```bash
curl $BASE/api/v1/agents -H "X-API-Key: $API_KEY"
```

### GET /api/v1/agents/:id

Amaç: Agent ayrıntıları. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{Agent}}`

```bash
curl $BASE/api/v1/agents/agent-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/agents/:id

Amaç: Agent güncelleme. Yetki: oluşturan VEYA Admin+. İstek gövdesi: `name/description/avatar/config` (hepsi isteğe bağlı). Agent bir organizasyonla paylaşılmışsa, bilgi tabanı kapsamına yeni eklenen bilgi tabanlarının çağıran tarafından paylaşılabilir olması gerekir (bilgi tabanını oluşturan veya Admin+; `all` olarak değiştirmek Admin+ gerektirir), aksi hâlde 403 döner.

Yanıt: 200 `{"success":true,"data":{Agent}}`

```bash
curl -X PUT $BASE/api/v1/agents/agent-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"description":"Güncellenmiş açıklama"}'
```

### DELETE /api/v1/agents/:id

Amaç: Agent silme. Yetki: oluşturan VEYA Admin+.

Yanıt: 200 `{"success":true,"message":"Agent deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/agents/agent-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/agents/:id/copy

Amaç: Agent kopyalama (kopya çağıran kullanıcıya ait olur). Yetki: Contributor+. İstek gövdesi yoktur.

Yanıt: 201 `{"success":true,"data":{yeni Agent}}`

```bash
curl -X POST $BASE/api/v1/agents/agent-1/copy -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/agents/:id/suggested-questions

Amaç: Agent başlangıç önerisi soruları (`/agents/:id/shares` ile çakışmaması için grubun dışında kaydedilir). Yetki: Viewer+; API key `read_agents`/`manage_agents`/`chat`/full.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `knowledge_base_ids` | string | Hayır | Virgülle ayrılmış KB'ler |
| `knowledge_ids` | string | Hayır | Virgülle ayrılmış bilgi ID'leri |
| `tag_scopes` | string | Hayır | JSON dizisi biçiminde etiket kapsamı |
| `limit` | int | Hayır | En fazla 30 |

Yanıt: 200 `{"success":true,"data":{"questions":[{question,source,knowledge_base_id}]}}`

```bash
curl "$BASE/api/v1/agents/agent-1/suggested-questions?limit=6" -H "X-API-Key: $API_KEY"
```

## MCP Hizmetleri (/api/v1/mcp-services)

Alan düzeyinde harici araç hizmeti entegrasyonu. Okuma: Viewer+; yazma/test/onay politikası: Admin+. API key: `manage_mcp_services`/full. Handler: `internal/handler/mcp_service.go`

### POST /api/v1/mcp-services

Amaç: MCP hizmeti oluşturma. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet | Ad |
| `description` | string | Hayır | Eski sürüm açıklaması, uyumluluk için korunur; yönetim arayüzü yalnızca `usage_instructions` alanını düzenler |
| `usage_instructions` | string | Hayır | Hizmetin amacı, uygun senaryoları ve temel kısıtları; model hizmeti ne zaman kullanacağına buna göre karar verir. Yönetim arayüzü ikinci adımda doldurulmasını ister |
| `enabled` | bool | Hayır | Etkin |
| `transport_type` | string | Evet | `sse` / `http-streamable`; `stdio` güvenlik nedeniyle reddedilir |
| `url` | *string | Hayır | Hizmet URL'si (SSE/HTTP) |
| `headers` | map[string]string | Hayır | HTTP başlıkları |
| `auth_config` | object | Hayır | `auth_type`(`api_key/bearer/oauth`), `api_key_header`, `custom_headers`, `scopes`, `auth_server_metadata_url` (anahtarlar credentials alt kaynağı üzerinden gönderilir) |
| `advanced_config` | object | Hayır | `{timeout,retry_count,retry_delay}`, varsayılan 30 saniye / 3 kez / 1 saniye; `timeout` 60 saniyeden büyükse Agent'ın bu hizmetin aracını tek seferde çağırırken beklediği süre de uzar |
| `stdio_config` / `env_vars` | object | Hayır | Yalnızca eski verilerle uyumluluk için korunur; stdio devre dışıdır, etkisi yoktur |

Yanıt: 200 `{"success":true,"data":{MCPServiceResponse}}` (`credentials:{api_key:{configured},token:{configured}}` içerir; araç kataloğu eşitlenmiş hizmetlerde ayrıca `catalog:{tool_count,stale,synced_at}` bulunur)

```bash
curl -X POST $BASE/api/v1/mcp-services -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"github","transport_type":"sse","url":"https://mcp.example.com/sse"}'
```

### GET /api/v1/mcp-services

Amaç: MCP hizmet listesi. Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":[MCPServiceResponse]}`

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `agent_id` | string | Hayır | `agent_source_tenant_id` ile birlikte verildiğinde, bu paylaşılan Agent'ın @ ile çağırabileceği MCP hizmetlerini listeler |
| `agent_source_tenant_id` | int | Hayır | Paylaşılan Agent'ın kaynak alan ID'si; sohbet isteğindeki aynı adlı parametreyle aynıdır |

İki parametre birlikte verildiğinde paylaşılan Agent yolu kullanılır: yalnızca bu Agent'ın kaynak alanında `selected` moduyla belirlenmiş ve etkin olan hizmetler döner (`all`/`none` modlarında boş liste döner). Her öğe yalnızca ID, ad, açıklama, kullanım talimatları, aktarım türü, etkinlik durumu ve araç kataloğu özetini içerir; URL, istek başlıkları, kimlik doğrulama yapılandırması gibi bağlantı ayrıntılarını içermez. Çağıranın bu Agent'ı kullanma yetkisi yoksa 403 döner. Yalnızca biri verildiğinde veya hiçbiri verilmediğinde, çağıranın kendi alanındaki hizmetler listelenir.

```bash
curl $BASE/api/v1/mcp-services -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id

Amaç: ayrıntılar. Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":{MCPServiceResponse}}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/mcp-services/:id

Amaç: kısmi güncelleme (map semantiği; `auth_config` içinde api_key/token gönderilemez). Yetki: Admin+. Alanlar oluşturmayla aynıdır (hepsi isteğe bağlı).

`usage_instructions` gönderildiğinde, baştaki ve sondaki boşluklar kırpıldıktan sonra boş olmayan bir dize olmalı ve en fazla 16000 karakter uzunluğunda olmalıdır. Yalnızca bağlantı veya etkinlik durumu değiştirilirken bu alan atlanabilir; bu durumda eski değer korunur.

Yanıt: 200 `{"success":true,"data":{MCPServiceResponse}}`

```bash
curl -X PUT $BASE/api/v1/mcp-services/mcp-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":false}'
```

### POST /api/v1/mcp-services/:id/usage-instructions/generate

Amaç: eşitlenmiş ve süresi dolmamış MCP araç kataloğuna göre kısa kullanım talimatları oluşturma. Yetki: Admin+; API key için `manage_mcp_services` veya full gerekir.

İstek: `{"language":"zh-CN"}`. `zh-CN`, `en-US`, `ja-JP`, `ko-KR`, `ru-RU` desteklenir; varsayılan Çincedir.

Öncelikle alanın varsayılan kullanılabilir sohbet modeli kullanılır, yoksa ilk kullanılabilir sohbet modeli kullanılır. Girdi; hizmet adını, sunucu tarafı açıklamasını ve etkin araçların adlarını ve açıklamalarını içerir; OAuth kataloğu geçerli kullanıcının yetkilendirme kapsamını kullanır. MCP'ye bağlanmaz, araç çağırmaz ve oluşturulan sonucu otomatik olarak kaydetmez.

Yanıt: 200 `{"success":true,"data":{"usage_instructions":"Uzak günlükleri modül ve zaman aralığına göre sorgular; sorgu ID'si varsa ilgili günlüğü okur."}}`. Hedef 2–3 cümlelik kısa bir açıklamadır, en fazla 500 karakter; kullanıcı düzenledikten sonra PUT ile kaydedebilir. Katalog eşitlenmemişse, süresi dolmuşsa, etkin araç yoksa veya kullanılabilir sohbet modeli yoksa 400 döner.

### DELETE /api/v1/mcp-services/:id

Amaç: silme. Yetki: Admin+. Yanıt: 200 `{"success":true,"message":"MCP service deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/mcp-services/mcp-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/mcp-services/:id/test

Amaç: bağlantı testi (harici hizmeti yoklar). Yetki: Admin+. Yanıt: 200 `{"success":true,"data":{"success","message","oauth_required","tools":[...],"resources":[...]}}`

```bash
curl -X POST $BASE/api/v1/mcp-services/mcp-1/test -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id/metadata

Amaç: kalıcı araç kataloğunu okuma, üst hizmete bağlanmaz. Yetki: Viewer+; OAuth kataloğu geçerli yetkilendirme sahibine göre yalıtılır.

Yanıt: 200 `{"success":true,"data":null}` eşitlenmemiş olduğunu gösterir; eşitlenmişse `data` katalog anlık görüntüsüdür ve sunucu bilgilerini, instructions, tools ve eşitleme zamanını içerir. Bağlantı yapılandırması değiştikten sonraki anlık görüntü `stale:true` olarak işaretlenir ve çalışma zamanı araçlarını yüklemek için kullanılamaz.

```bash
curl $BASE/api/v1/mcp-services/mcp-1/metadata -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/mcp-services/:id/metadata/refresh

Amaç: üst hizmete açıkça bağlanıp araç kataloğunu eksiksiz çekmek ve atomik olarak güncellemek. Statik kimlik doğrulamalı katalog için Admin+ gerekir; OAuth kullanıcıları kendi kataloglarını eşitleyebilir (Viewer+). API Key için MCP yönetim yetkisi gerekir.

Yanıt, güncellenmiş katalog anlık görüntüsüdür. Başarısız olursa eski anlık görüntü korunur; yenileme sırasında bağlantı değişirse 409, katalog geçersiz/çok büyükse veya üst hizmetle eşitleme başarısız olursa 400, meta veri deposu kullanılamıyorsa 503 döner. Elle yazılmış kullanım talimatlarının ve araç bazında etkinlik/onay politikalarının üzerine yazmaz.

```bash
curl -X POST $BASE/api/v1/mcp-services/mcp-1/metadata/refresh -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id/tools

Amaç: araç listesi. Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":[{name,description,inputSchema,require_approval}]}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/tools -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id/resources

Amaç: kaynak listesi. Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":[{uri,name,description,mimeType}]}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/resources -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/mcp-services/:id/credentials

Amaç: anahtar ayarlama (`api_key`/`token`, işaretçi alanlar; atlanırsa korunur). Yetki: Admin+. Handler: `internal/handler/mcp_credentials.go`

Yanıt: 200 `{"success":true,"data":{"fields":{"api_key":{"configured"},"token":{"configured"}}}}`

```bash
curl -X PUT $BASE/api/v1/mcp-services/mcp-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"token":"ghp_..."}'
```

### DELETE /api/v1/mcp-services/:id/credentials/:field

Amaç: kimlik bilgisi alanını silme (`api_key` veya `token`). Yetki: Admin+. Yanıt: 204.

```bash
curl -X DELETE $BASE/api/v1/mcp-services/mcp-1/credentials/token -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id/tool-approvals

Amaç: araç etkinleştirme/devre dışı bırakma ve elle onay politikası listesi. Yetki: Viewer+. Yanıt: 200 `{"success":true,"data":[{service_id,tool_name,require_approval,enabled,...}]}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/tool-approvals -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/mcp-services/:id/tool-approvals/:tool_name

Amaç: bir aracın enabled (etkinleştirme/devre dışı bırakma) ve require_approval (elle onay) değerlerini güncelleme. Yetki: Admin+. En az biri verilmelidir, atlanan alanlar eski değerini korur; kayıt yoksa varsayılan olarak etkindir ve onay gerektirmez.

Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/mcp-services/mcp-1/tool-approvals/create_issue \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"require_approval":true}'
```

## MCP OAuth

Handler: `internal/handler/mcp_oauth.go`

### GET /api/v1/mcp-oauth/callback

Amaç: üçüncü taraf OAuth yetkilendirme geri çağrısı (kimlik doğrulama gerektirmez, tek kullanımlık `state` parametresiyle doğrulanır; `/mcp-services` grubunun dışında kaydedilir). Sorgu parametreleri: `code`, `state`, `error`.

Yanıt: 302 ile ön yüze yönlendirir (başarılıysa `#mcp_oauth_result=success`, başarısızsa `#mcp_oauth_error=<code>`).

```bash
curl -i "$BASE/api/v1/mcp-oauth/callback?code=xxx&state=yyy"
```

### POST /api/v1/mcp-services/:id/oauth/authorize-url

Amaç: kullanıcı düzeyinde yetkilendirme URL'si oluşturma. Yetki: Viewer+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `redirect_uri` | string | Evet | Arka uç geri çağrı URL'si (mutlak adres) |
| `frontend_redirect` | string | Hayır | Geri çağrıdan sonra ön yüzün yönleneceği adres (varsayılan `/`) |

Yanıt: 200 `{"success":true,"data":{"authorization_url","authorization_attempt"}}`

```bash
curl -X POST $BASE/api/v1/mcp-services/mcp-1/oauth/authorize-url -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"redirect_uri":"'$BASE'/api/v1/mcp-oauth/callback"}'
```

### GET /api/v1/mcp-services/:id/oauth/status

Amaç: kullanıcının kendi yetkilendirme durumunu sorgulama. Yetki: Viewer+. Sorgu parametresi: `authorization_attempt` (isteğe bağlı).

Yanıt: 200 `{"success":true,"data":{"authorized","state":"authorized|pending","refresh_available","expires_at"}}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/oauth/status -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/mcp-services/:id/oauth/token

Amaç: kullanıcının kendi OAuth token'ını iptal etme. Yetki: Viewer+. Yanıt: 204.

```bash
curl -X DELETE $BASE/api/v1/mcp-services/mcp-1/oauth/token -H "Authorization: Bearer $TOKEN"
```

## MCP Server Uç Noktaları (/api/v1/mcp-endpoints) {#mcp-server-uc-noktalari-api-v1-mcp-endpoints}

Geçerli alanın dışarıya yayımladığı MCP uç noktalarını yönetir; harici MCP istemcileri `/mcp/:endpoint_id` adresine bağlanır. Okuma: Viewer+; yazma: Admin+. API key: `manage_channels`/full. Handler: `internal/handler/mcp_endpoint.go`. Amaç ve araç açıklamaları için [MCP entegrasyonu](../03-features/08-mcp.md#harici-istemcilerin-cagirmasi-icin) sayfasına bakın.

| Yöntem | Yol | Açıklama |
| --- | --- | --- |
| GET | `/mcp-endpoints` | Uç nokta listesi (token içermez) |
| GET | `/mcp-endpoints/tools` | Araç kataloğu: `{groups,tools:[{name,group,destructive}],default_tools}` |
| POST | `/mcp-endpoints` | Oluşturma; 201, yanıt tek seferlik `token` içerir |
| GET | `/mcp-endpoints/:endpoint_id` | Ayrıntılar |
| PUT | `/mcp-endpoints/:endpoint_id` | Kısmi güncelleme, atlanan alanlar eski değerini korur |
| DELETE | `/mcp-endpoints/:endpoint_id` | Silme; bu uç noktayı kullanan istemciler hemen geçersiz olur |
| POST | `/mcp-endpoints/:endpoint_id/rotate-token` | Token yenileme; yanıt yeni `token` içerir, eski token hemen geçersiz olur |

İstek alanları (oluşturma ve güncellemede aynıdır, hepsi isteğe bağlı):

| Alan | Tür | Açıklama |
| --- | --- | --- |
| `name` | string | Ad, oluştururken zorunlu |
| `description` | string | Açıklama |
| `enabled` | bool | Varsayılan true; devre dışı bırakıldıktan sonra bağlantılar 403 döner |
| `knowledge_base_ids` | string[] | Erişilebilir bilgi tabanları; boş dizi alandaki tümü anlamına gelir |
| `tools` | string[] | Açılan araçlar, en az bir tane; oluştururken atlanırsa `default_tools` (tüm salt okunur araçlar) kullanılır |
| `default_agent_id` | string | `ask` tarafından kullanılan Agent; boşsa yerleşik hızlı yanıt kullanılır; dahili yerleşik Agent'lar seçilemez |
| `rate_limit_per_minute` | int | Dakika başına araç çağrısı üst sınırı; 0 veya atlanırsa 60, en fazla 6000 |

Yanıttaki `data` şudur: `{id,tenant_id,name,description,enabled,token_hint,knowledge_base_ids,tools,default_agent_id,rate_limit_per_minute,path,last_used_at,created_at,updated_at}`; oluşturma ve token yenilemede ayrıca `token` bulunur. Kısıtlı API Key ile çağrıldığında uç noktanın bilgi tabanları ve araçlarının gerektirdiği yetkiler bu Key'in kapsamını aşamaz, aksi hâlde 403 döner.

```bash
curl -X POST $BASE/api/v1/mcp-endpoints -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ürün Belgesi Asistanı","knowledge_base_ids":["kb-1"],"tools":["search_knowledge","read_document","ask"]}'
```

## Agent Çalışma Zamanı Etkileşimi (/api/v1/agent)

Sohbet sırasındaki elle onay ve OAuth sürdürme; yetkilerin tümü Viewer+ (bağlamı yalnızca oturumu başlatan kişi bilir), API key varsayılan olarak reddedilir.

### POST /api/v1/agent/tool-approvals/:pending_id

Amaç: onay bekleyen araç çağrısı hakkında karar verme. Handler: `internal/handler/mcp_service.go` içindeki `ResolveToolApproval`.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `decision` | string | Evet (`binding:"required"`) | `approve` / `reject` |
| `modified_args` | JSON | Hayır | Değiştirilmiş araç parametreleri |
| `reason` | string | Hayır | Gerekçe |

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/agent/tool-approvals/p-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"decision":"approve"}'
```

### POST /api/v1/agent/mcp-oauth-resolutions/:pending_id

Amaç: MCP OAuth nedeniyle duraklatılmış Agent çalışmasını sürdürme. Handler: `internal/handler/mcp_oauth.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `service_id` | string | Evet (`binding:"required"`) | MCP hizmet ID'si |
| `decision` | string | Hayır | `authorize` (varsayılan) / `cancel` |

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/agent/mcp-oauth-resolutions/p-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"service_id":"mcp-1"}'
```

### POST /api/v1/agent/mcp-oauth-resolutions/:pending_id/cancel

Amaç: duraklatılmış OAuth akışını iptal etme. İstek gövdesi yoktur.

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/agent/mcp-oauth-resolutions/p-1/cancel -H "Authorization: Bearer $TOKEN"
```

## Beceriler, Sanal Alan ve Kişisel Değişkenler

`GET /api/v1/skills?sandbox_config_id=...` belirtilen yapılandırmada kullanılabilir becerilerin ad/açıklamalarını ve skills_available değerini döndürür; `agent_id` + `agent_source_tenant_id` verildiğinde paylaşılan Agent'ın kaynak alanına ve sanal alan yapılandırmasına göre döndürür, ayrıntılar için [Sanal Alan ve Beceriler API](02-api-sandbox-skills.md#sanal-alandaki-yetenekler) sayfasına bakın. Katalog kaydı, kurulum, şablonlar, ilerleme, dosyalar ve kişisel değişkenlerle ilgili tüm arayüzler için [Sanal Alan ve Beceriler API](02-api-sandbox-skills.md) sayfasına bakın.

Agent config'e `sandbox_config_id` eklenir; skills_selection_mode ve selected_skills ile birlikte kullanılabilir becerileri belirler. shell/dosya araçları arka uç yeteneklerine göre kaydedilir; eski read_skill / execute_skill_script artık kaydedilmez.

## Uzun Süreli Bellek

Agent config'teki `memory_enabled` nil ise alan ayarı devralınır; false ise bu Agent için bellek okuma/yazma devre dışı kalır. Kişisel yönetim, konu/belge tercihleri, dışa aktarma ve anında düzenleme için [Uzun Süreli Bellek API](02-api-memory.md), kullanım adımları için [Oturumlar Arası Uzun Süreli Bellek](../03-features/23-memory.md) sayfasına bakın.

## Kullanıcı Favorileri (/api/v1/user/favorites)

Kullanıcı bazında saklanır (kaynağı oluşturan bazında değil); yetkilerin tümü Viewer+, yalnızca JWT (API key varsayılan olarak reddedilir). Handler: `internal/handler/user_resource_favorite.go`

### GET /api/v1/user/favorites

Amaç: favori listesi. Sorgu parametresi: `type` (zorunlu, `kb` veya `agent`).

Yanıt: 200 `{"success":true,"data":[{type,id,created_at}]}`

```bash
curl "$BASE/api/v1/user/favorites?type=kb" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/user/favorites

Amaç: favori ekleme. İstek gövdesi: `{"type":"kb|agent","id":"<kaynak ID>"}` (hepsi zorunlu).

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/user/favorites -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"type":"kb","id":"kb-1"}'
```

### DELETE /api/v1/user/favorites/:type/:id

Amaç: favoriden çıkarma. Yol parametreleri: `type`, `id`.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/user/favorites/kb/kb-1 -H "Authorization: Bearer $TOKEN"
```

## Uygulama Başvurusu

Rota kaydı: `internal/router/router.go` tarafından çağrılır; `RegisterCustomAgentRoutes`, `RegisterSkillRoutes`, `RegisterUserFavoriteRoutes` `routes_agent.go` içinde, `RegisterMCPServiceRoutes` (MCP OAuth ve `/agent` çalışma zamanı etkileşimi dahil) `routes_infra.go` içinde, `RegisterMCPEndpointRoutes` ve herkese açık `/mcp/:endpoint_id` `routes_mcp_endpoint.go` içinde tanımlanır. Handler: `internal/handler/custom_agent.go`, `internal/handler/mcp_service.go`, `internal/handler/mcp_credentials.go`, `internal/handler/mcp_oauth.go`, `internal/handler/mcp_endpoint.go`, `internal/handler/skill_handler.go`, `internal/handler/user_resource_favorite.go`.
