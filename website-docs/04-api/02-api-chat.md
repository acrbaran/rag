# API referansı: oturumlar, mesajlar ve sohbet

Oturumlar oluşturur ve yönetir, mesajları ve geçici ekleri okur, ayrıca SSE üzerinden bilgi soru-cevap veya ajan yanıtları alır.

Oturumlar “kullanıcıya özel” kaynaklardır; handler içinde sahiplik doğrulaması zorunlu olarak uygulanır; rota katmanı Viewer+ içindir. API key: oturum/sohbet için `chat` capability (veya full-access) gerekir; mesaj araması için `message_history`; bilgi getirimi için `retrieve` gerekir.

## Oturumlar (/api/v1/sessions)

### POST /api/v1/sessions

Amaç: Oturum oluşturmak. Handler: `internal/handler/session/handler.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `title` | string | Hayır | Başlık |
| `description` | string | Hayır | Açıklama |

Yanıt: 201 `{"success":true,"data":{Session}}` (`id,title,description,tenant_id,user_id,is_pinned,last_request_state,created_at,...`)

```bash
curl -X POST $BASE/api/v1/sessions -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"title":"Yeni sohbet"}'
```

### GET /api/v1/sessions

Amaç: oturum listesi. Handler: `internal/handler/session/handler.go`

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `page` / `page_size` | int | Hayır | Sayfalama |
| `keyword` | string | Hayır | Başlıkta bulanık arama |
| `source` | string | Hayır | Kaynak filtresi (web/embed/api/feishu/wechat/slack/...) |
| `agent_id` | string | Hayır | Agent'e göre filtrele (IM oturumları) |

Yanıt: 200 `{"success":true,"data":[SessionListItem],"total","page","page_size"}`

```bash
curl "$BASE/api/v1/sessions?page=1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/:id

Amaç: oturum ayrıntıları.

Yanıt: 200 `{"success":true,"data":{Session}}`

```bash
curl $BASE/api/v1/sessions/s-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/sessions/:id

Amaç: oturumu güncelleme (başlık/açıklama/sabitleme). İstek gövdesi: `title`, `description`, `is_pinned` (hepsi isteğe bağlı).

Yanıt: 200 `{"success":true,"data":{Session}}`

```bash
curl -X PUT $BASE/api/v1/sessions/s-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"Yeniden adlandır"}'
```

### DELETE /api/v1/sessions/:id

Amaç: oturumu silme.

Yanıt: 200 `{"success":true,"message":"Session deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/sessions/s-1 -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/sessions/batch

Amaç: oturumları toplu silme. İstek gövdesi: `{"ids":["s-1"],"delete_all":false}` (iki seçenekten biri: `ids` veya `delete_all:true`).

Yanıt: 200 `{"success":true,"message":"Sessions deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/sessions/batch -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"ids":["s-1","s-2"]}'
```

### DELETE /api/v1/sessions/:id/messages

Amaç: oturum mesajlarını temizleme.

Yanıt: 200 `{"success":true,"message":"Session messages cleared successfully"}`

```bash
curl -X DELETE $BASE/api/v1/sessions/s-1/messages -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/sessions/:session_id/generate_title

Amaç: bağlam mesajlarına göre oturum başlığı oluşturma. Handler: `internal/handler/session/title.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `messages` | []Message | Evet (`binding:"required"`) | Bağlam olarak kullanılacak mesajlar |

Yanıt: 200 `{"success":true,"data":"Oluşturulan başlık"}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/generate_title -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"messages":[{"role":"user","content":"Ürünü tanıt"}]}'
```

### POST /api/v1/sessions/:session_id/stop

Amaç: oluşturulmakta olan yanıtı durdurma. Handler: `internal/handler/session/stream.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `message_id` | string | Evet (`binding:"required"`) | Asistan mesajı ID'si |

Yanıt: 200 `{"success":true,"message":"Generation stopped"}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/stop -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-1"}'
```

Durdurulduğunda, henüz ajana iletilmemiş ek mesajlar (aşağıdaki [çalışan yanıta mesaj ekleme](#steer) bölümüne bakın) birlikte atılır ve otomatik olarak yeni bir tur başlatılmaz.

### POST /api/v1/sessions/:session_id/fork

Amaç: Bir geçmiş mesajdan yeni bir oturum dallandırmak; kaynak oturum değişmeden kalır. Yalnızca oturum sahibi dallandırabilir. İşleyici: `internal/handler/session/fork.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `message_id` | string | Evet | Dallanma noktası. Kullanıcı mesajı: ondan önceki geçmiş kopyalanır, istemci genellikle bu soruyu giriş alanına önceden doldurur; asistan mesajı: bu yanıta kadarki geçmiş kopyalanır |
| `title` | string | Hayır | Yeni oturum başlığı, varsayılan olarak “kaynak başlık (dal)” |

Kopyalanan mesajlar özgün zaman çizelgesini ve oluşturulan dosyaları korur; yeni oturum `parent_session_id` ve `forked_from_message_id` kaydeder. Kaynak oturum bir sanal alana bağlıysa sistem sanal alanın anlık görüntüsünü alır; yeni oturum sanal alanı ilk kullandığında dallanma noktasına karşılık gelen turdaki çalışma alanı durumundan başlar. Çalışma alanı taşınamadığında dallanma yine de başarılı olur, ancak `degraded: true` ve şu neden döner:

| `reason` | Anlamı |
| --- | --- |
| `NO_CHECKPOINT` | Dallanma noktasından önceki turlarda çalışma alanı denetim noktası yok |
| `SANDBOX_REPLACED` | Denetim noktası, oturumun değiştirilmiş eski sanal alanına ait |
| `SANDBOX_GONE` | Kaynak oturumda şu anda anlık görüntüsü alınabilir bir sanal alan yok |
| `SNAPSHOT_UNSUPPORTED` | Sanal alan arka ucu anlık görüntüyü desteklemiyor veya anlık görüntü alma başarısız oldu |

Yanıt: 200 `{"success":true,"data":{"session_id":"yeni oturum ID","degraded":false}}`. Kaynak oturum oluşturma yapıyorsa veya dallanma noktası tamamlanmamış bir yanıtsa 409 (`code: FORK_SOURCE_BUSY`) döner; oturum ya da mesaj yoksa 404; dallanma noktası kullanıcı veya asistan mesajı değilse 400 döner.

```bash
curl -X POST $BASE/api/v1/sessions/s-1/fork -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-3"}'
```

### POST /api/v1/sessions/:session_id/rewind

Amaç: Geçerli oturumu yerinde belirli bir mesaja geri almak. Yalnızca oturum sahibi geri alabilir. İşleyici: `internal/handler/session/rewind.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `message_id` | string | Evet | Geri alma noktası. Kullanıcı mesajı: kendisi ve sonrasındaki mesajlar silinir; asistan mesajı: bu yanıt korunur, sonrasındaki mesajlar silinir |

Silinen mesajların oluşturulan dosyaları, takip sorusu önerileri ve sohbet geçmişi dizinleri de birlikte temizlenir. Oturum bir sanal alana bağlıysa sistem aynı anda `/workspace` dizinini korunan geçmişteki son turun denetim noktasına sıfırlar; çalışma alanı sıfırlama başarısız olursa tüm işlem başarısız olur ve mesajlar silinmez.

Yanıt: 200 `{"success":true,"data":{"deleted_messages":N,"workspace_reset":true,"reason":""}}`. `workspace_reset` false olduğunda, `reason` yalnızca konuşmanın geri alınma nedenini açıklar: `NO_SANDBOX` (oturum sanal alana bağlı değil) veya `NO_CHECKPOINT` (korunan geçmişte denetim noktası yok).

| Durum kodu | `code` | Anlamı |
| --- | --- | --- |
| 409 | `REWIND_SOURCE_BUSY` | Oturum oluşturma yapıyor veya geri alma noktası tamamlanmamış bir yanıt |
| 409 | `REWIND_NO_CHECKPOINT` | Sanal alan hâlâ mevcut, ancak korunan geçmişte kullanılabilir denetim noktası bulunamadı; dosyaların konuşmanın önüne geçmesini önlemek için reddedilir |
| 409 | `REWIND_SANDBOX_REPLACED` | Denetim noktası değiştirilmiş eski sanal alana ait |
| 404 | — | Oturum veya mesaj yok |
| 500 | — | Çalışma alanı sıfırlama başarısız oldu, yeniden denenebilir |

```bash
curl -X POST $BASE/api/v1/sessions/s-1/rewind -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-3"}'
```

### Çalışan yanıta mesaj ekleme (steer) {#steer}

Akıllı çıkarım modunda yanıt oluşturulurken, aynı oturumda `agent-chat` yeniden çağrılırsa 409 döner. Bu durumda yeni mesajı mevcut tura sıraya almak için aşağıdaki arayüz kullanılabilir. Yalnızca oturum sahibi çağırabilir; hızlı soru-cevap için enjekte edilebilir bir yürütme döngüsü yoktur, ekleme desteklenmez. İşleyici: `internal/handler/session/steer.go`

| Yöntem | Yol | Amaç |
| --- | --- | --- |
| POST | `/api/v1/sessions/:session_id/steer` | Bir mesaj ekler |
| GET | `/api/v1/sessions/:id/steer` | Mevcut turda henüz teslim edilmemiş sıradaki mesajları listeler; sayfa yenilendikten sonra kuyruğu geri yüklemek için kullanılır. `assistant_message_id` ve `items[]` (`steer_id`, `content`, `delivery`, `mentioned_items`) döner; çalışan tur yoksa `items` boştur |
| DELETE | `/api/v1/sessions/:id/steer/:steer_id` | Sıradaki bir mesajı geri çeker |
| POST | `/api/v1/sessions/:session_id/steer/:steer_id/inject` | Bir `after` mesajını `inject` olarak değiştirir |

POST istek gövdesi:

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `query` | string | Evet | Mesaj içeriği, en fazla 10000 karakter |
| `delivery` | string | Hayır | `after` (varsayılan): mevcut tur bittikten sonra sonraki turun sorusu olarak gönderilir; `inject`: mevcut turun bir sonraki yineleme sınırında (nihai yanıt verilmeden hemen önce dahil) etkene teslim edilir, etken buna göre ayarlama yaparak devam eder |
| `mentioned_items` | []object | Hayır | @bahsedilen ögeler, biçim sohbet isteğiyle aynıdır |
| `steer_id` | string | Hayır | İstemcinin oluşturduğu UUID, yeniden denemelerde yineleme engelleme için kullanılır; aynı ID farklı içerikle kullanılırsa 409 döner |
| `expected_assistant_message_id` | string | Hayır | İstemcinin gördüğü çalışan yardımcı mesajı ID'si; çalışma değiştiyse 409 döner, istemci yeniden denemelidir |
| `channel` | string | Hayır | Kaynak kanalı |

Yanıttaki `status`:

| `status` | Anlamı |
| --- | --- |
| `queued` | Sıraya alındı; `steer_id`, `delivery` ve `assistant_message_id` döner |
| `new_run` | Şu anda çalışan tur yok; istemci normal `agent-chat` çağrısına geçerek göndermelidir |
| `already_injected` | Bu mesaj etkene zaten teslim edildi (yeniden deneme, geri çekme veya enjeksiyona değiştirme sırasında görülebilir), geri çekilemez |
| `deleted` / `gone` | Yalnızca DELETE: geri çekildi (`removed`, bir mesajın gerçekten silinip silinmediğini belirtir) / şu anda çalışan tur yok |

Her turda aynı anda en fazla 10 teslim edilmemiş mesaj sıraya alınabilir; aşılırsa 400 döner. Mevcut tur normal şekilde bittiğinde, ilk teslim edilmemiş mesaj sunucu tarafından doğrudan sonraki turun sorusu olarak başlatılır; diğer mesajlar özgün teslim yöntemleriyle bu tura aktarılır. Bu tur için karşılık gelen bir `agent-chat` bağlantısı yoktur; `GET /steer` tarafından döndürülen `assistant_message_id` ile continue-stream çağrılarak alınabilir. Kullanıcı üretimi durdurduğunda, teslim edilmemiş tüm mesajlar atılır. Teslim edilmiş mesajlar oturum geçmişine yazılır ve istemciye `user_message_injected` olayıyla bildirilir. Çalışma durumu sorgusu başarısız olursa 503 döner, yeniden denenebilir.

```bash
curl -X POST $BASE/api/v1/sessions/s-1/steer -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"query":"Yalnızca 2024 verilerine bak","delivery":"inject"}'
```

### POST /api/v1/sessions/:session_id/pin ve DELETE /api/v1/sessions/:id/pin

Amaç: oturumu sabitlemek / sabitlemeyi kaldırmak. İstek gövdesi yoktur. İşleyici: `internal/handler/session/handler.go`

Yanıt: 200 `{"success":true,"is_pinned":true|false}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/pin -H "Authorization: Bearer $TOKEN"
curl -X DELETE $BASE/api/v1/sessions/s-1/pin -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/continue-stream/:session_id

Amaç: etkin akışın bağlantı kesintisinden sonra devamını sağlamak (geçmiş olayları yeniden oynatma + yeni ekleri 100ms aralıklarla yoklama). İşleyici: `internal/handler/session/stream.go`

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `message_id` | string | Evet | Devam ettirilecek asistan mesajı ID'si |

Yanıt: 200 SSE (`text/event-stream`, olay biçimi için genel bakıştaki “akış arayüzü protokolü” bölümüne bakın). Saklanan olaylar yeniden oynatılırken, aynı olay ID'si altındaki ardışık tamamlanmamış `answer`/`thinking`/`reflection` artışları tek bir karede birleştirilir; içerikleri olay ID'sine göre biriktiren istemcinin aldığı metin değişmez. Gerçek zamanlı iletim bölümü birleştirilmez.

```bash
curl -N "$BASE/api/v1/sessions/continue-stream/s-1?message_id=m-1" -H "Authorization: Bearer $TOKEN"
```

## Sanal alan grafik masaüstü {#sandbox-desktop}

Bu rotalar `internal/router/routes_chat.go` tarafından kaydedilir ve henüz Swagger'a eklenmemiştir. Yalnızca Cube/E2B masaüstü şablonları desteklenir; dağıtım ve vekil gereksinimleri için [sanal alan dağıtımı](../06-development/04-sandbox-deployment.md) bölümüne bakın.

### POST /api/v1/sessions/:session_id/sandbox/desktop-ticket

İki dakika geçerli, tek kullanımlık bir WebSocket bileti düzenler. Oturum sahibinin geçerli oturum açma Bearer access token'ı gerekir; yalnızca API Key kullanılamaz. JWT yalnızca bu POST isteğinin kimlik doğrulama başlığında bulunur.

```bash
curl -X POST "$BASE/api/v1/sessions/$SESSION_ID/sandbox/desktop-ticket" \
  -H "Authorization: Bearer $TOKEN"
```

Yanıt: 200 `{"success":true,"data":{"ticket":"<opaque-ticket>","expires_in":120}}`.

### GET /api/v1/sessions/:id/sandbox/desktop

WebSocket el sıkışması `?ticket=<opaque-ticket>` kullanır ve normal JWT ara katmanından geçmez. Bilet kullanıcıya, alana, oturuma ve özgün access token'a bağlıdır; tek kullanımda geçersiz olur. Kullanılmış, süresi dolmuş ve bilinmeyen biletlerin tümü reddedilir. Vekil günlükleri ticket sorgusunu kaydetmemelidir.

Her oturum için aynı anda yalnızca bir aktarma bağlantısına izin verilir. Sanal alan bağlı değilse, duraklatılmışsa, masaüstünü desteklemiyorsa veya başlatma başarısız olursa WebSocket upgrade önce tamamlanabilir, ardından `SANDBOX_NOT_BOUND`, `SANDBOX_PAUSED`, `DESKTOP_UNSUPPORTED`, `DESKTOP_START_FAILED` gibi close reason değerleriyle bağlantı kesilebilir; istemci kapatma nedenini okumalıdır.

### POST /api/v1/sessions/:session_id/sandbox/desktop/activity

Oturum sahibi klavye ve fare etkinliğini bildirir, yanıt 200 `{"success":true}`. Yalnızca sunucu tarafındaki RFB parser geriye düştüğünde süre uzatmak için kullanılır; parser normalken yok sayılır ve sanal alan ömrü yoklama yoluyla uzatılmamalıdır.

## Oturum ekleri (geçici belgeler)

Handler: `internal/handler/session/temporary_document.go`

### POST /api/v1/sessions/:session_id/attachments

Amaç: Oturum düzeyinde geçici belgeler yüklemek (eşzamansız ayrıştırma). multipart alanları: `file` (zorunlu), `agent_id` (isteğe bağlı, ayrıştırma motorunu/ASR modelini belirler), `parser_engine` (isteğe bağlı; paylaşılan ajan kullanılırken yok sayılır, ajanın ayrıştırma kuralları belirler).

Yanıt: 202 `{"success":true,"data":{TemporaryDocument}}` (`id,session_id,file_name,file_type,file_size,status(uploaded/processing/ready/failed),resource_ref,...`)

```bash
curl -X POST $BASE/api/v1/sessions/s-1/attachments -H "Authorization: Bearer $TOKEN" -F 'file=@notes.pdf'
```

### GET /api/v1/sessions/:id/attachments

Amaç: Ek listesi.

Yanıt: 200 `{"success":true,"data":[TemporaryDocument]}`

```bash
curl $BASE/api/v1/sessions/s-1/attachments -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/:id/attachments/:attachment_id

Amaç: Ek ayrıntıları (ayrıştırma durumu dahil).

Yanıt: 200 `{"success":true,"data":{TemporaryDocument}}`

```bash
curl $BASE/api/v1/sessions/s-1/attachments/a-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/:id/attachments/:attachment_id/preview

Amaç: Ekin özgün dosyasını önizlemek.

Yanıt: 200 dosya akışı (`Content-Disposition: inline|attachment`, `Cache-Control: private`).

```bash
curl $BASE/api/v1/sessions/s-1/attachments/a-1/preview -H "Authorization: Bearer $TOKEN" -o preview.pdf
```

### DELETE /api/v1/sessions/:id/attachments/:attachment_id

Amaç: Eki silmek.

Yanıt: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/sessions/s-1/attachments/a-1 -H "Authorization: Bearer $TOKEN"
```

## Yanıt önerileri (Suggestions)

Handler: `internal/handler/message_suggestion.go`

### GET /api/v1/sessions/:id/messages/:message_id/suggestions

Amaç: Belirli bir asistan mesajı için takip sorusu önerilerini okumak.

Yanıt: 200 `{"success":true,"data":{MessageSuggestionSet}}` (`status(generating/ready/suppressed/failed),questions:[{id,text,category,source,knowledge_base_ids}],allow_regenerate,...`)

```bash
curl $BASE/api/v1/sessions/s-1/messages/m-1/suggestions -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/sessions/:session_id/messages/:message_id/suggestions

Amaç: Önerilerin oluşturulmasını sağlamak (idempotent tetikleme). İstek gövdesi: `{"regenerate":true}` (isteğe bağlı, yeniden oluşturmayı zorlar).

Yanıt: 200 (hazır) veya 202 (oluşturuluyor) `{"success":true,"data":{MessageSuggestionSet|null}}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/messages/m-1/suggestions -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{}'
```

### POST /api/v1/sessions/:session_id/suggestion-events

Amaç: öneri etkileşim olaylarını raporlamak (izleme).

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `suggestion_set_id` | string | Evet (`binding:"required"`) | Öneri kümesi ID'si |
| `question_id` | string | Hayır | click/regenerate sırasında zorunlu |
| `event_type` | string | Evet (`binding:"required"`) | `impression/click/dismiss/regenerate` |

Yanıt: 204 İçerik Yok

```bash
curl -X POST $BASE/api/v1/sessions/s-1/suggestion-events -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"suggestion_set_id":"ss-1","event_type":"impression"}'
```

## Sohbet ve arama

İşleyici: `internal/handler/session/qa.go`. API anahtarı: sohbet için `chat`/full; `knowledge-search` için `retrieve`/full gerekir.

### POST /api/v1/knowledge-chat/:session_id

Amaç: bilgi tabanı soru-cevap işlemleri (SSE akışı).

İstek gövdesi (KnowledgeQA/AgentQA ortak):

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `query` | string | Evet (`binding:"required"`) | Kullanıcı sorusu; boşsa, yalnızca boşluk içeriyorsa, kontrol karakterleri veya geçersiz UTF-8 içeriyorsa 400 döner |
| `knowledge_base_ids` | []string | Hayır | Aranacak KB'ler |
| `knowledge_ids` | []string | Hayır | Bilgi dosyalarını sınırlar |
| `agent_enabled` | bool | Hayır | Agent modunun etkin olup olmadığı |
| `agent_id` | string | Hayır | Özel Agent ID'si |
| `agent_source_tenant_id` | uint64 | Hayır | Paylaşılan Agent'ın kaynak alanı; aynı adlı paylaşılan Agent birden fazla alandan geliyorsa ayırt etmek için kullanılır |
| `reasoning_effort` | string | Hayır | Bu turdaki düşünme yoğunluğu: `off`, `auto`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max` (`none`/`false`, `off` olarak; `true`/`on`, `auto` olarak değerlendirilir); belirtilmezse Agent yapılandırması kullanılır. Yalnızca bu tur için geçerlidir, Agent'ı değiştirmez; model seçilen seviyeyi desteklemiyorsa benzer bir seviyeye otomatik olarak ayarlanır. Geçersiz değer 400 döner |
| `web_search_enabled` | bool | Hayır | İnternet araması; yalnızca Agent'ın kendisinde internet araması etkinse çalışır |
| `local_browser_enabled` | bool | Hayır | Bu turda bağlı yerel tarayıcının kullanılmasına izin verir; yalnızca akıl yürütme Agent'larının `agent-chat` işlemi için kullanılabilir, aksi halde 400 döner. Bkz. [Yerel tarayıcı](../05-clients/09-local-browser.md) |
| `summary_model_id` | string | Hayır | Özet modeli; paylaşılan Agent kullanılırken yok sayılır ve her zaman Agent yapılandırmasındaki model kullanılır |
| `mcp_service_ids` | []string | Hayır | @ ile anılan MCP hizmetleri |
| `skill_names` | []string | Hayır | @ ile anılan beceriler |
| `tag_ids` | []string | Hayır | Etiket filtresi |
| `mentioned_items` | []object | Hayır | @bahsedilen öğeler (type/kb_id/kb_name/service_id/skill_name) |
| `disable_title` | bool | Hayır | Otomatik başlığı devre dışı bırak |
| `images` | []object | Hayır | Görseller (`data` base64 / `url` / `caption`) |
| `attachment_uploads` | []object | Hayır | Satır içi ekler (`data` base64, `file_name`, `file_size`) |
| `attachment_ids` | []string | Hayır | Yüklenmiş oturum eki kimlikleri |
| `channel` | string | Hayır | Kaynak kanalı |
| `suggestion_attribution` | object | Hayır | Tıklanan öneriye ait ilişkilendirme bilgileri |
| `question_origin` | object | Hayır | Kullanıcının seçtiği öneri sorusunun kaynağı: `knowledge_base_id`, isteğe bağlı `knowledge_id`. Yalnızca akıllı çıkarım modunda kullanılır; ajan önce bu kaynağı arar. Kaynak bu turdaki arama kapsamı dışındaysa yok sayılır, kapsam genişletilmez |

Seçilen `reasoning_effort`, oturumun `last_request_state.reasoning_effort` alanına yazılır ve oturum yeniden açıldığında arayüz buna göre geri yüklenir.

Yanıt: 200 SSE akışı, `event: message` + `data: StreamResponse` (genel bakışa bakın); `complete` olayıyla sona erer. Yanıt, modelin tek seferlik çıktı sınırı nedeniyle kesildiğinde `answer` olayı `data.truncated: true` içerir; kesilmeden önce üretilen içerik normal şekilde korunur.

```bash
curl -N -X POST $BASE/api/v1/knowledge-chat/s-1 -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"İade politikası nedir?","knowledge_base_ids":["kb-1"]}'
```

### POST /api/v1/agent-chat/:session_id

Kullanım: Ajan soru-cevap (SSE akışlı; `thinking/tool_call/tool_result/tool_approval_required/mcp_oauth_required` gibi olayları içerir). İstek gövdesi yukarıdakiyle aynıdır.

- Araç yürütme hataları `tool_result` olayıyla döner (`data.success: false`, neden `data.error` içindedir); ajan işlemeye devam eder. `error` olayı yalnızca tüm turun yürütülmesinin başarısız olduğunu belirtir.
- Ajan işlem sırasında ek mesaj aldığında `user_message_injected` gönderir; komut yürütülürken araç kartı çıktısı `command_output` ile güncellenir.
- Aynı oturumda akıllı çıkarım yanıtı hâlihazırda üretiliyorsa 409 döner; bu durumda mesaj eklemek için [steer arayüzü](#steer) kullanılmalıdır.

```bash
curl -N -X POST $BASE/api/v1/agent-chat/s-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"query":"Geçen çeyreğin verilerini analiz et","agent_id":"agent-1"}'
```

### POST /api/v1/knowledge-search

Kullanım: Oturumsuz bilgi arama (akışsız); harici sistemlerin arama sonuçlarını alması için tercih edilen arayüzdür. Ürün içi soru-cevapla aynı arama akışını kullanır (geri çağırma → rerank → birleştirme → kesme); sıralama, sayfa soru-cevaplarıyla tutarlıdır. `hybrid-search` ile nasıl seçileceği için bkz. [arama arayüzü nasıl seçilir](./01-api-overview.md#retrieval-api). Handler: `internal/handler/session/qa.go` içindeki `SearchKnowledge`.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `query` | string | Evet (`binding:"required"`) | Sorgu |
| `knowledge_base_id` | string | Hayır | Tek KB (eski sürümle uyumlu) |
| `knowledge_base_ids` | []string | Hayır | Çoklu KB |
| `knowledge_ids` | []string | Hayır | Dosyaları sınırla |
| `tag_ids` | []string | Hayır | Etiket filtreleme |
| `mentioned_items` | []object | Hayır | KB kapsamlı etiket bahsetmeleri |
| `vector_threshold` / `keyword_threshold` | float | Hayır | Geri çağırma eşikleri; belirtilmezse alan arama yapılandırması kullanılır (varsayılan 0.15 / 0.3) |
| `match_count` | int | Hayır | Dönen kayıt sayısı, üst sınır 200; belirtilmezse alan yapılandırmasındaki `rerank_top_k` kullanılır (varsayılan 10). Geri çağırma derinliği otomatik olarak bundan az olmayacak şekilde artırılır |
| `disable_keywords_match` / `disable_vector_match` | bool | Hayır | Bir geri çağırma yolunu kapatır; ikisi de `true` ise 400 döner |
| `rerank` | object | Hayır | rerank ayarlarını geçersiz kılar;`{"enabled":false}` rerank'i kapatır. Alanlar için bkz. [rerank nesnesi](./01-api-overview.md#retrieval-api). `rerank.top_k` aynı anda verildiğinde `match_count` yerine önceliklidir |

Atlanan alanlar alanın arama yapılandırmasını kullanır (`GET /tenants/kv/retrieval-config`); hiçbir yeni alan gönderilmezse davranış eskisiyle aynıdır.

Yanıt: 200 `{"success":true,"data":[SearchResult],"meta":{"rerank":{...}}}`. `SearchResult`, `id,content,knowledge_id,knowledge_title,score,chunk_type,knowledge_base_id,...` içerir; rerank uygulanmış sonuçların `metadata` alanında `model_score` ve `base_score` bulunur. `data` boşsa nedeni belirlemek için `meta.rerank.outcome` alanına bakın; bkz. [meta.rerank tanılama](./01-api-overview.md#retrieval-api).

```bash
curl -X POST $BASE/api/v1/knowledge-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query":"Dağıtım gereksinimleri","knowledge_base_ids":["kb-1"]}'

# Yalnızca vektör geri çağırma kullan, 5 sonuç al ve rerank'i kapat
curl -X POST $BASE/api/v1/knowledge-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"Dağıtım gereksinimleri","knowledge_base_ids":["kb-1"],"disable_keywords_match":true,"match_count":5,"rerank":{"enabled":false}}'
```

## Mesajlar (/api/v1/messages)

Handler: `internal/handler/message.go`

### POST /api/v1/messages/search

Amaç: sohbet geçmişinde arama. Yetki: Viewer+; API anahtarı `message_history`/full.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `query` | string | Evet (`binding:"required"`) | Sorgu |
| `mode` | string | Hayır | `keyword/vector/hybrid` (varsayılan hybrid) |
| `limit` | int | Hayır | Varsayılan 20 |
| `session_ids` | []string | Hayır | Oturumları sınırlar |

Yanıt: 200 `{"success":true,"data":{"total":N,"results":[{session_id,message_id,role,content,created_at,score}]}}`

```bash
curl -X POST $BASE/api/v1/messages/search -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"query":"Fiyat teklifi"}'
```

### GET /api/v1/messages/chat-history-stats

Amaç: sohbet geçmişi dizin istatistikleri. Yetki: Viewer+; API anahtarı `message_history`/full.

Yanıt: 200 `{"success":true,"data":{indexed_message_count,knowledge_base_size,last_indexed_at,...}}`

```bash
curl $BASE/api/v1/messages/chat-history-stats -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/messages/:session_id/load

Amaç: oturum mesajlarını yükleme (zaman imleciyle geriye doğru sayfalama). Yetki: Viewer+; API anahtarı `chat`/full.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `limit` | int | Hayır | Varsayılan 20 |
| `before_time` | string | Hayır | RFC3339/RFC3339Nano zaman damgası |

Yanıt: 200 `{"success":true,"data":[Message]}` (`id,session_id,role,content,is_completed,images,attachments,agent_steps,...`)

```bash
curl "$BASE/api/v1/messages/s-1/load?limit=20" -H "X-API-Key: $API_KEY"
```

### DELETE /api/v1/messages/:session_id/:id

Amaç: tek bir mesajı silme. Yetki: Viewer+ (handler oturum sahipliğini doğrular); API anahtarı `chat`/full.

Yanıt: 200 `{"success":true,"message":"Message deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/messages/s-1/m-1 -H "Authorization: Bearer $TOKEN"
```

## Oturum tarafından oluşturulan dosyalar

Aşağıdaki uç noktalar Viewer+ gerektirir; API Key `chat` veya `full-access` olmalı ve oturum sahipliği doğrulanmalıdır. Mevcut olmayan ya da erişilemeyen oturumlar 404 döndürür.

| Yöntem | Yol | Yanıt |
| --- | --- | --- |
| GET | `/api/v1/sessions/:id/artifacts` | 200 `{success:true,data:[Artifact]}`, oturum dosyalarını toplar |
| GET | `/api/v1/sessions/:id/messages/:message_id/artifacts` | Yukarıdakiyle aynı, yalnızca bu mesajın dosyaları |
| GET | `/api/v1/sessions/:id/messages/:message_id/artifacts/:index/download` | 200 dosya akışı, Content-Disposition: attachment |
| DELETE | `/api/v1/sessions/:id/messages/:message_id/artifacts/:index` | 200 `{success:true,data:{file_name,deleted}}`, bu dosyayı siler |

Artifact alanları: index, handle (isteğe bağlı resource:// referansı), file_name, file_type, file_size, source_path, mod_time, created_at. Yanıt, alttaki nesne depolama URL'sini döndürmez. İndirme index'i 0'dan başlar; karşılık gelen mesaj listesindeki indeks kullanılmalıdır, oturum özet indeksini doğrudan mesaj indirme adresine eklemek mümkün değildir. Geçersiz indeks 400, sınır dışı indeks veya mevcut olmayan dosya 404 döndürür.

```bash
curl "$BASE/api/v1/sessions/session-1/messages/message-1/artifacts" \
  -H "Authorization: Bearer $TOKEN"
curl "$BASE/api/v1/sessions/session-1/messages/message-1/artifacts/0/download" \
  -H "Authorization: Bearer $TOKEN" -o result.pdf
```

### Oluşturulan dosyaları silme

Silme, nesne depolamadaki baytları geri kazanır; **geri alınamaz**. İndirmeden farklı olarak silme yalnızca oturum sahibine açıktır: paylaşılan bir Agent aracılığıyla edinilen salt okunur erişim dosya indirebilir, ancak silemez — size ait olmayan oturumlar her zaman 404 olarak işlenir ("mevcut değil" ile "yetki yok" ayrımı yapılmaz; diğer oturum arayüzleriyle tutarlıdır). Silinen dosyalar 404 döndürür, tekrar silme de 404 döndürür.

Baytlar yalnızca başka hiçbir sahibi kalmadığında gerçekten geri kazanılır: aynı dosyanın bilgi tabanına kaydedilmesi, sonraki bir yanıtta yeniden referans verilmesi veya bulunduğu oturumun bir kopyasının çatallanması, dosyanın korunmasına neden olur. Geri kazanma hatası silme sonucunu etkilemez (arayüz yine 200 döndürür); dosya her yerdeki listelerden kaybolmuştur.

Silme sonrasında dosya liste arayüzlerinde ve yapıt kütüphanesinde artık görünmez, ancak mesajdaki **konumu korunur**: `index` indirme adresidir; sonraki dosyalar sırayla öne taşınırsa mevcut indirme bağlantıları yanlış dosyayı işaret eder. Aynı nedenle, sanal alandaki aynı adlı dosya bir sonraki toplamada yeniden kaydedilmez — mtime değeri kullanıcının silmesi nedeniyle değişmemiştir.

| Parametre | Açıklama |
| --- | --- |
| `all_versions` | Bu oturumdaki aynı `source_path` için tüm geçmiş sürümleriyle birlikte siler. Boole değeri, varsayılan `false` |

```bash
curl -X DELETE "$BASE/api/v1/sessions/session-1/messages/message-1/artifacts/0" \
  -H "Authorization: Bearer $TOKEN"
```

### Oturumlar arası yapıt listesi

`GET /api/v1/artifacts`, ana sayfa kenar çubuğundaki "Yapıtlar" sayfasında kullanılmak üzere mevcut kullanıcının web sohbetlerindeki tüm oluşturulmuş dosyaları listeler. Kapsam, oturum listesindeki `source=web` ile aynıdır: kullanıcının kendi oturumları ve geçmişte sahibi olmayan kiracı düzeyindeki web oturumları; IM kanalları, web bileşenleri (embed) ve API Key oturumları, IM oturumunun veritabanında sahibi olmasa bile, kesinlikle dahil edilmez. Aynı oturumda `source_path` değeri aynı olan dosyalar, aynı dosyanın birden çok sürümü sayılır; yalnızca en yeni sürüm döndürülür, `version_count` sürüm sayısını verir. Silinmiş oturum veya mesajlardaki dosyalar döndürülmez. Yetki gereksinimleri yukarıdaki tabloyla aynıdır.

| Parametre | Açıklama |
| --- | --- |
| `keyword` | Dosya adına göre filtreler, büyük/küçük harf duyarsızdır |
| `file_types` | Virgülle ayrılmış uzantılar; örneğin `.pdf,.pptx` (nokta atlanabilir) |
| `page` / `page_size` | Sayfalama; `page_size` en fazla 100, varsayılan 20 |

Yanıt `{success, data:[LibraryArtifact], total, page, page_size}` biçimindedir ve oluşturulma zamanına göre azalan sırada verilir. LibraryArtifact alanları: session_id, session_title, message_id, index, handle (isteğe bağlı), file_name, file_type, file_size, source_path, created_at, version_count. İndirme için bunlardaki session_id, message_id ve index kullanılarak yukarıdaki tablodaki indirme arayüzü çağrılır.

```bash
curl "$BASE/api/v1/artifacts?file_types=.pptx,.pdf&keyword=rapor&page=1&page_size=30" \
  -H "Authorization: Bearer $TOKEN"
```

`DELETE /api/v1/artifacts`, yapıt kütüphanesindeki bir dosyayı siler; konumlandırma için query parametreleri `session_id`, `message_id`, `index` kullanılır ve anlamı yukarıdaki oturum içi silmeyle aynıdır. Tek fark, `all_versions` belirtilmediğinde `true` kabul edilmesidir (açıkça değer gönderildiğinde iki arayüzün ayrıştırma kuralları aynıdır): yapıt kütüphanesindeki bir satır, belirli bir oluşturmayı değil bir dosyayı (`version_count` sürüm sayısını verir) temsil eder; yalnızca en yeni sürümü silmek satırın listede kalmasına ve önceki sürümün gösterilmesine yol açar. Yalnızca mevcut sürümü silmek için `all_versions=false` gönderilebilir.

```bash
curl -X DELETE "$BASE/api/v1/artifacts?session_id=session-1&message_id=message-1&index=0" \
  -H "Authorization: Bearer $TOKEN"
```

### Yanıtlardaki görsel ve dosya referansları

`GET /api/v1/sessions/:id/messages/:message_id/files?file_path=...`, mesaj düzeyinde yetkilendirme vekilidir. file_path, mesajın referans verdiği kaynak tanıtıcısını veya desteklenen depolama referansını alır; istemci URL kodlaması kullanmalıdır. Arka uç; mesaj erişim yetkisini, kaynağın mesaja bağlılığını ve bilgi tabanı/paylaşılan Agent için mevcut erişim yetkisini doğrular; rastgele dosya yollarına oturum ID'siyle erişilemez. Yetki geri alındıktan sonra eski mesaj referansları da reddedilir. Paylaşılan Agent, kuruluş paylaşımlı kütüphane yanıt görselleri ve mesaj yapıtları için uygundur; ayrıntılar için [Dosya erişimi](../03-features/21-file-access.md) bölümüne bakın.

### Tur başına kullanım

Mesajlar kalıcı usage değerini döndürür, Agent tamamlama olayı ise turn_usage taşır; bunlar bu turdaki her amaç için model çağrılarının toplu sonuçlarını içerir. Araç çağrılarının kendisi her zaman Token üretmez; kullanım, sağlayıcının döndürdüğü veya arka ucun topladığı kayıtlara dayanır. Alanlar için [Gözlemlenebilirlik](../03-features/16-observability.md) bölümüne bakın.

## Uygulama referansı

Rota kaydı: `internal/router/routes_chat.go` içindeki `RegisterSessionRoutes`, `RegisterChatRoutes`, `RegisterMessageRoutes` (`internal/router/router.go` tarafından çağrılır). Handler: `internal/handler/session/` (handler.go, qa.go, stream.go, title.go, temporary_document.go, fork.go, rewind.go, steer.go, artifact_*.go), `internal/handler/message.go`, `internal/handler/message_suggestion.go`.
