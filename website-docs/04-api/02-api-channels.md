# API başvurusu: IM, Embed ve dosya hizmetleri

IM ve web gömme kanallarını yönetir; kanal geri çağrıları, ziyaretçi oturumları ve dosya erişimi için arayüzler sunar. Yönetim tarafı, IM platformu geri çağrıları ve Embed ziyaretçileri kendi kimlik doğrulama yöntemlerini kullanır.

## IM geri çağrıları (genel kimlik doğrulama gerekmez)

### GET|POST /api/v1/im/callback/:channel_id

Amaç: IM platformlarının (Slack/Telegram/WeChat/QQBot vb.) olay geri çağrıları ve URL doğrulaması. Kimlik doğrulama ara katmanından önce kaydedilir ve her platformun kendi imza doğrulamasını kullanır; imza doğrulanamazsa 403, kanal yoksa 404 döner. Mesaj alınır alınmaz ACK gönderilir ve eşzamansız işlenir. Handler: `internal/handler/im.go`

Yanıt: 200 `{"success":true}` ya da platformun istediği ACK biçimi.

```bash
curl -X POST $BASE/api/v1/im/callback/ch-1 -H 'Content-Type: application/json' -d '{"event":"..."}'
```

## IM kanal yönetimi (kimlik doğrulama gerekir)

API key: `manage_channels`/full. IM kanalları harici bot kimlik bilgileri taşır: listeleme Viewer+, değiştirme/açıp kapatma/QR kodla giriş Admin+.

Yapılandırma örnekleri ve ağ gereksinimleri için bkz. [IM entegrasyonu](../03-features/12-im-integration.md). IM/Embed bellek tercihi bağlı Agent'ın config.memory_enabled değerinden gelir; mevcut kanal arayüzlerinde ayrı bir memory_enabled parametresi yoktur.

### POST /api/v1/agents/:id/im-channels

Amaç: Agent için IM kanalı oluşturur. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `platform` | string | Evet | `slack/telegram/wechat/qqbot` |
| `name` | string | Hayır | Görünen ad |
| `mode` | string | Hayır | `websocket` (varsayılan)/`webhook`/`longpoll` (wechat için longpoll zorunludur) |
| `output_mode` | string | Hayır | `stream` (varsayılan)/`full` (wechat için full zorunludur) |
| `locale` | string | Hayır | Yanıt dili: `en-US`/`tr-TR`; boş (varsayılan) ise kurulumun varsayılan dili kullanılır (`RETHRA_LANGUAGE`, ayarlanmamışsa `tr-TR`); diğer değerlerde 400 |
| `session_mode` | string | Hayır | `user` (varsayılan)/`thread` |
| `knowledge_base_id` | string | Hayır | Eklerin ayrıca kaydedileceği KB; bu alana ait olmalıdır, aksi halde 400 |
| `credentials` | object | Hayır | Platform kimlik bilgileri |
| `enabled` | bool | Hayır | Varsayılan true |

Yanıt: 200 `{"data":{IMChannel}}`; aynı kanal botu zaten varsa 409 döner.

```bash
curl -X POST $BASE/api/v1/agents/agent-1/im-channels -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"platform":"slack","name":"Slack Destek","locale":"en-US"}'
```

### GET /api/v1/agents/:id/im-channels

Amaç: Bir Agent'ın IM kanal listesi (özet). Yetki: Viewer+.

Yanıt: 200 `{"data":[IMChannel]}`

```bash
curl $BASE/api/v1/agents/agent-1/im-channels -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/im-channels

Amaç: Tüm alandaki IM kanallarının genel görünümü (kimlik bilgileri hariç). Yetki: Viewer+.

Yanıt: 200 `{"data":[IMChannel]}`

```bash
curl $BASE/api/v1/im-channels -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/im-channels/:id

Amaç: Kanalı günceller (kısmi güncelleme: `name/mode/output_mode/locale/session_mode/knowledge_base_id/credentials/enabled/agent_id` alanlarının tümü isteğe bağlıdır; `knowledge_base_id` boş dize verilirse ilişki kaldırılır, `locale` boş dize verilirse varsayılan dile dönülür). Yetki: Admin+.

Yanıt: 200 `{"data":{IMChannel}}`

```bash
curl -X PUT $BASE/api/v1/im-channels/ch-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":false}'
```

### DELETE /api/v1/im-channels/:id

Amaç: Kanalı siler. Yetki: Admin+.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/im-channels/ch-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/im-channels/:id/toggle

Amaç: Etkin/devre dışı durumunu değiştirir. Yetki: Admin+. İstek gövdesi yoktur.

Yanıt: 200 `{"data":{IMChannel}}`

```bash
curl -X POST $BASE/api/v1/im-channels/ch-1/toggle -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/wechat/qrcode

Amaç: WeChat giriş QR kodu üretir (kişisel WeChat hesabını alana bağlar). Yetki: Admin+. İstek gövdesi yoktur. Handler: `internal/handler/wechat_qrcode.go`

Yanıt: 200 `{"data":{"qrcode_url","qrcode"}}`

```bash
curl -X POST $BASE/api/v1/wechat/qrcode -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/wechat/qrcode/status

Amaç: QR kod tarama durumunu yoklar; onaylandıktan sonra kimlik bilgilerini döndürür. Yetki: Admin+. İstek gövdesi: `{"qrcode":"<tanımlayıcı>"}` (zorunlu).

Yanıt: 200 `{"data":{"status":"pending|scanned|confirmed|expired","credentials":{bot_token,ilink_bot_id,ilink_user_id,baseurl}}}`

```bash
curl -X POST $BASE/api/v1/wechat/qrcode/status -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"qrcode":"qr-1"}'
```

## Embed kanal yönetimi (kimlik doğrulama gerekir)

API key: `manage_channels`/full. Handler: `internal/handler/embed_channel.go`

### POST /api/v1/agents/:id/embed-channels

Amaç: Agent için web gömme kanalı oluşturur. Yetki: Admin+.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Hayır | Ad |
| `enabled` | bool | Hayır | Varsayılan true |
| `allowed_origins` | []string | Evet | En az bir kaynak (tam URL / `*.domain`; üretimde `*` yasaktır) |
| `welcome_message` | string | Hayır | Karşılama mesajı |
| `rate_limit_per_minute` | int | Hayır | IP başına/dakika, varsayılan 30 |
| `rate_limit_per_day` | int | Hayır | Kanal başına/gün, varsayılan 10000 |
| `primary_color` / `page_title` / `widget_position` | string | Hayır | Görünüm (position: varsayılan `bottom-right` ve diğer dört köşe) |
| `header_title_mode` | string | Hayır | `channel` (varsayılan)/`session` |
| `show_suggested_questions` | bool | Hayır | Varsayılan true |
| `allow_web_search` / `allow_file_upload` | bool | Hayır | Varsayılan false |
| `default_locale` | string | Hayır | `en-US`/`tr-TR`/boş (ziyaretçinin seçtiği dil kullanılır, yoksa `tr-TR`) |
| `webhook_url` / `webhook_secret` | string | Hayır | Ziyaretçi olayları webhook'u |
| `agent_id` | string | Hayır | Bağlanacak Agent |

Yanıt: 201 `{"success":true,"data":{publish_token içeren embedChannelResponse}}`

```bash
curl -X POST $BASE/api/v1/agents/agent-1/embed-channels -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Web sitesi destek","allowed_origins":["https://example.com"]}'
```

### GET /api/v1/agents/:id/embed-channels

Amaç: Bir Agent'ın gömme kanal listesi. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[embedChannelResponse]}`

```bash
curl $BASE/api/v1/agents/agent-1/embed-channels -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/embed-channels

Amaç: Tüm alandaki gömme kanalları listesi (publish token hariç). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":[embedChannelResponse]}`

```bash
curl $BASE/api/v1/embed-channels -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/embed-channels/:channel_id

Amaç: Kanal ayrıntıları (dağıtım kodunu kopyalamak için publish token dahil). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{embedChannelResponse}}`

```bash
curl $BASE/api/v1/embed-channels/ec-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/embed-channels/:channel_id

Amaç: Kanalı günceller (alanlar oluşturmayla aynıdır, tümü isteğe bağlıdır). Yetki: Admin+.

Yanıt: 200 `{"success":true,"data":{embedChannelResponse}}`

```bash
curl -X PUT $BASE/api/v1/embed-channels/ec-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":false}'
```

### DELETE /api/v1/embed-channels/:channel_id

Amaç: Kanalı siler. Yetki: Admin+.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/embed-channels/ec-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/embed-channels/:channel_id/rotate-token

Amaç: Publish token'ı yeniler (eski token geçersiz olur). Yetki: Admin+. İstek gövdesi yoktur.

Yanıt: 200 `{"success":true,"data":{yeni publish_token içeren embedChannelResponse}}`

```bash
curl -X POST $BASE/api/v1/embed-channels/ec-1/rotate-token -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/embed-channels/:channel_id/preview-session

Amaç: Yönetim tarafı önizlemesi için kısa ömürlü session token verir (publish token gerekmez). Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{"session_token","expires_in"}}`; kanal devre dışıysa 403.

```bash
curl -X POST $BASE/api/v1/embed-channels/ec-1/preview-session -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/embed-channels/:channel_id/stats

Amaç: Kanal kullanım istatistikleri. Yetki: Viewer+.

Yanıt: 200 `{"success":true,"data":{"session_count":N}}`

```bash
curl $BASE/api/v1/embed-channels/ec-1/stats -H "Authorization: Bearer $TOKEN"
```

## Embed gömme sayfası çerçeve politikası

### GET /api/v1/embed-frame-policy

Amaç: Nginx `auth_request` alt isteğinin, `/embed/:channel_id` sayfasını döndürmeden önce kanalın `Content-Security-Policy: frame-ancestors` değerini almasını sağlar. Token gerekmez; yalnızca politika başlığını döndürür, kanal yapılandırmasını döndürmez; yanıt `Cache-Control: no-store` içerir.

| İstek başlığı | Zorunlu | Açıklama |
| --- | --- | --- |
| `X-Embed-Page-URI` | Evet | Gömme sayfasının göreli URI'si, ör. `/embed/<channel_id>` |

Yanıt: CSP başlığıyla 204; kanal yoksa, devre dışıysa, yol geçersizse veya izin listesi boşsa 403.

```bash
curl -i $BASE/api/v1/embed-frame-policy -H "X-Embed-Page-URI: /embed/ec-1"
```

## Embed genel rotaları (/api/v1/embed/:channel_id, EmbedAuth)

Kimlik doğrulama: `Authorization: Embed <publish_token|session_token>`; oturum düzeyindeki işlemlere ayrıca `X-Embed-Session: <sig>` eklenir. Hız sınırı ve Origin doğrulaması için genel bakışa bakın. Handler: `internal/handler/embed_channel.go`; ara katman: `internal/middleware/embed_auth.go`

Aşağıda `$ET` Embed token başlığını temsil eder: `-H "Authorization: Embed $EMBED_TOKEN"`.

### POST /api/v1/embed/:channel_id/exchange

Amaç: Publish token'ı kısa ömürlü session token ile değiştirir (session token yeniden exchange edilemez). İstek gövdesi yoktur.

Yanıt: 200 `{"success":true,"data":{"session_token","expires_in"}}`

```bash
curl -X POST $BASE/api/v1/embed/ec-1/exchange -H "Authorization: Embed $PUBLISH_TOKEN"
```

### GET /api/v1/embed/:channel_id/config

Amaç: Kanalın herkese açık yapılandırması (anahtar içermez).

Yanıt: 200 `{"success":true,"data":{channel_id,name,display_title,knowledge_base_ids,agent_id,agent_name,welcome_message,primary_color,widget_position,allow_web_search,allow_file_upload,default_locale,...}}`

```bash
curl $BASE/api/v1/embed/ec-1/config -H "Authorization: Embed $EMBED_TOKEN"
```

### GET /api/v1/embed/:channel_id/suggested-questions

Amaç: Başlangıç için önerilen sorular. Sorgu parametresi: `limit` (≤12).

Yanıt: 200 `{"success":true,"data":{"questions":[...]}}`

```bash
curl "$BASE/api/v1/embed/ec-1/suggested-questions?limit=6" -H "Authorization: Embed $EMBED_TOKEN"
```

### GET /api/v1/embed/:channel_id/chunks/:chunk_id

Amaç: Alıntılanan parçayı görüntüler (içerik maskelenir; yetkisiz erişimde 403).

Yanıt: 200 `{"success":true,"data":{chunk}}`

```bash
curl $BASE/api/v1/embed/ec-1/chunks/c-1 -H "Authorization: Embed $EMBED_TOKEN"
```

### POST /api/v1/embed/:channel_id/sessions

Amaç: Ziyaretçi oturumu oluşturur; oturum ID'si ve imzalı tanıtıcı döndürür. İstek gövdesi yoktur.

Yanıt: 201 `{"success":true,"data":{"id":"<session_id>","sig":"<imza>"}}`

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions -H "Authorization: Embed $EMBED_TOKEN"
```

### POST /api/v1/embed/:channel_id/knowledge-chat/:session_id

Amaç: Ziyaretçi bilgi soru-cevabı (SSE; payload kanal kısıtlamalarına göre yeniden yazılıp KnowledgeQA'ya devredilir). `X-Embed-Session` gerekir. İstek gövdesi `/knowledge-chat` ile aynıdır (`query` zorunlu).

```bash
curl -N -X POST $BASE/api/v1/embed/ec-1/knowledge-chat/s-1 \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"query":"Çalışma saatleri?"}'
```

### POST /api/v1/embed/:channel_id/agent-chat/:session_id

Amaç: Ziyaretçi Agent soru-cevabı (SSE). `X-Embed-Session` gerekir. İstek gövdesi yukarıdakiyle aynıdır.

```bash
curl -N -X POST $BASE/api/v1/embed/ec-1/agent-chat/s-1 \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"query":"Sipariş vermeme yardım et"}'
```

### GET /api/v1/embed/:channel_id/messages/:session_id/load

Amaç: Ziyaretçi oturum mesajlarını yükler (`LoadMessages`'a devredilir, sorgu parametreleri `limit/before_time`). `X-Embed-Session` gerekir.

Yanıt: 200 `{"success":true,"data":[Message]}`

```bash
curl "$BASE/api/v1/embed/ec-1/messages/s-1/load?limit=20" \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG"
```

### POST /api/v1/embed/:channel_id/sessions/:session_id/stop

Amaç: Üretimi durdurur (StopSession'a devredilir; istek gövdesi `{"message_id":"..."}`). `X-Embed-Session` gerekir.

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/stop \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-1"}'
```

### GET|POST /api/v1/embed/:channel_id/sessions/:session_id/messages/:message_id/suggestions

Amaç: Mesaj önerilerini okur / üretimini tetikler (kanalda öneriler kapalıysa `suppressed` döner). `X-Embed-Session` gerekir.

Yanıt: 200 `{"success":true,"data":{"status","questions":[...]}}`

```bash
curl $BASE/api/v1/embed/ec-1/sessions/s-1/messages/m-1/suggestions \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG"
```

### POST /api/v1/embed/:channel_id/sessions/:session_id/suggestion-events

Amaç: Öneri etkileşim olaylarını bildirir (RecordEvent'e devredilir, alanlar kimlik doğrulamalı sürümle aynıdır). `X-Embed-Session` gerekir. Yanıt: 204.

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/suggestion-events \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"suggestion_set_id":"ss-1","event_type":"impression"}'
```

### POST /api/v1/embed/:channel_id/sessions/:session_id/events

Amaç: Ziyaretçi olaylarını kanal webhook'una iletir. `X-Embed-Session` gerekir.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `type` | string | Evet | `message_sent` / `message_received` |
| `query` / `content` | string | Hayır | Kullanıcı sorusu / bot yanıtı |

Yanıt: 200 `{"success":true}`; desteklenmeyen türde 400.

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/events \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"type":"message_sent","query":"Merhaba"}'
```

### MCP OAuth ve araç onayı (ziyaretçi tarafı)

Aşağıdaki rotaların tümü `X-Embed-Session` gerektirir ve ilgili kimlik doğrulamalı handler'a devredilir (`internal/handler/mcp_oauth.go`, `internal/handler/mcp_service.go`):

| Yöntem+yol | Amaç |
| --- | --- |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/mcp-oauth-resolutions/:pending_id` | OAuth nedeniyle duraklatılan çalışmayı sürdürür (gövde: `service_id` zorunlu, `decision` isteğe bağlı) |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/mcp-oauth-resolutions/:pending_id/cancel` | OAuth akışını iptal eder |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/mcp-services/:id/oauth/authorize-url` | Yetkilendirme URL'si üretir (gövde: `redirect_uri` zorunlu) |
| `GET /api/v1/embed/:channel_id/sessions/:session_id/mcp-services/:id/oauth/status` | Yetkilendirme durumunu sorgular |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/tool-approvals/:pending_id` | Araç onayı (gövde: `decision` zorunlu) |

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/tool-approvals/p-1 \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"decision":"approve"}'
```

### GET /api/v1/embed/:channel_id/files

Amaç: Ziyaretçi tarafı görsel vekili (bot yanıtlarına gömülü görseller; EmbedAuth kanal alanını enjekte eder, handler aynı alan yolunu zorunlu kılar). Sorgu parametresi: `file_path` (zorunlu).

Yanıt: 200 dosya akışı.

```bash
curl "$BASE/api/v1/embed/ec-1/files?file_path=local://1/exports/chart.png" \
  -H "Authorization: Embed $EMBED_TOKEN" -o chart.png
```

## Dosya hizmetleri

`internal/router/files.go` içinde uygulanmıştır (handler paketinde değil).

### GET /files

Amaç: Kimlik doğrulamalı birleşik dosya vekili (yerel/MinIO/COS/TOS vb.). Yetki: alanın kimliği doğrulanmış herhangi bir üyesi; API key KB ile sınırlı olmamalıdır (full-access veya tüm alanda retrieve, `middleware.AllowFileServeAPIKey()`); yol aynı alana ait olmalıdır (`ValidateStoragePathTenant`).

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `file_path` | string | Evet | `provider://...` yolu (`..` yasaktır; alanlar arası erişimde 403) |

Yanıt: 200 dosya akışı (`X-Content-Type-Options: nosniff`; izin listesinde olmayan türlerde `Content-Disposition: attachment` zorunludur).

```bash
curl "$BASE/files?file_path=local://1/docs/a.png" -H "Authorization: Bearer $TOKEN" -o a.png
```

### GET|HEAD /api/v1/files/presigned

Amaç: HMAC imzalı URL ile dosya erişimi (IM platformlarında gömülü görseller; kimlik doğrulama gerekmez, imza ve süre doğrulanır, imzaya `SYSTEM_AES_KEY` katılır).

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `file_path` | string | Evet | Depolama yolu |
| `tenant_id` | uint64 | Evet | Alan ID'si |
| `expires` | string | Evet | Unix sona erme zamanı |
| `sig` | string | Evet | HMAC imzası |

Yanıt: 200 dosya akışı (HEAD yalnızca başlıkları döndürür); imza geçersiz/süresi dolmuşsa 403.

```bash
curl "$BASE/api/v1/files/presigned?file_path=local://1/x.png&tenant_id=1&expires=1790000000&sig=abc" -o x.png
```

### GET /api/v1/files/presigned-preview

Amaç: Tanılama uç noktası: verilen yol için üretilecek ön imzalı HTTP URL'sini döndürür. Yetki: Admin+, API key açıkça reddedilir (`DenyAPIKeyPrincipal`). Sorgu parametresi: `file_path` (zorunlu).

`file_path` geçerli alana ait olmalıdır, tıpkı `/files` gibi: `resource://` tanıtıcılarında kaynağın bu alana ait olması, depolama yollarında kiracı bölümünün bu alanın ID'si olması gerekir; aksi halde 403 döner ve hiçbir URL imzalanmaz.

Yanıt: 200 `{"file_path","provider","url","rewritten":bool,"hint"}`

```bash
curl "$BASE/api/v1/files/presigned-preview?file_path=local://1/x.png" -H "Authorization: Bearer $TOKEN"
```

### GET|HEAD /r/:token

Amaç: Kısa ömürlü kaynak yetkilendirme URL'si (IM gibi kimlik doğrulama başlığı taşıyamayan istemciler için). Kimlik doğrulama gerekmez, token'ın kendisi yetki belgesidir; geçersiz/süresi dolmuşsa 404.

Yanıt: 200 dosya akışı (`Cache-Control: private, max-age=300`).

```bash
curl $BASE/r/abc123 -o file.png
```

## Uygulama başvurusu

Rota kaydı: `internal/router/routes_agent.go` içindeki `RegisterIMRoutes`, `RegisterIMChannelRoutes`, `RegisterEmbedChannelRoutes`, `RegisterEmbedPublicRoutes`; `internal/router/files.go` içindeki `serveFilesWithResources`, `servePresignedFiles`, `servePresignedPreview`, `serveResourceGrants`. Handler: `internal/handler/im.go`, `internal/handler/wechat_qrcode.go`, `internal/handler/embed_channel.go`.
