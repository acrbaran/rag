# API referansı: uzun süreli bellek

Geçerli çağıranın uzun süreli belleğini, konularını ve belge tercihlerini तथा alan düzeyindeki bellek yapılandırmasını yönetir. Yollar `/api/v1` önekini kullanır.

Kişisel uç noktaların tümü Viewer+ gerektirir; API Key full-access olmalıdır. Kapsam kimlik bilgisinden belirlenir, rastgele bir `subject_id` kabul edilmez. Örneklerdeki `$BASE` hizmet adresi, `$TOKEN` ise mevcut kullanıcının Bearer jetonudur.

## Alan yapılandırması ve istek anahtarı

Alan yapılandırması `GET/PUT /tenants/kv/memory-config` kullanır; kiracı adı/açıklaması güncelleme uç noktaları kullanılmaz. Okuma Viewer+, yazma Admin+ gerektirir; API Key için manage_tenant_settings veya full-access gerekir. Yanıt `{success,data:MemoryConfig}` biçimindedir; PUT yapılandırma nesnesini doğrudan iletir. Kişisel `PUT /memory/settings`, alan anahtarının yerine geçemez.

```bash
curl -X PUT "$BASE/api/v1/tenants/kv/memory-config" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"enabled":true,"write_mode":"explicit_only","max_items":200}'
```

`memory_config` alanları: `enabled`, `write_mode` (explicit_only/auto), `extract_model_id`, `max_items`, `extract_delay_seconds`, `extract_min_interval_seconds`, `extract_instructions`, `interest_threshold`, `embedding_model_id`, `vector_recall`, `retrieval_conditioning`. Anlamları için bkz. [uzun süreli bellek](../03-features/23-memory.md). Güncelleme sırasında korunması gereken tam yapılandırma nesnesini gönderin.

`CustomAgentConfig.memory_enabled` atlandığında alan ayarı devralınır; false bu akıllı aracının belleği kullanmasını engeller. IM/Embed bağlı akıllı aracının yapılandırmasını kullanır; mevcut kanal yapısında ayrı bir memory_enabled alanı yoktur.

## Kişisel ayarlar

| Yöntem | Yol | İstek / Yanıt |
| --- | --- | --- |
| GET | `/memory/settings` | `{success,data:{workspace_enabled,user_enabled,effective,write_mode,item_count,max_items}}` |
| PUT | `/memory/settings` | `{"enabled":true}`, enabled zorunludur; güncellenmiş settings döner |

`effective`, alan ve kişisel anahtarların birleştirilmiş sonucunu ifade eder; tek bir sohbet ayrıca akıllı araç anahtarına da tabidir.

```bash
curl -X PUT "$BASE/api/v1/memory/settings" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"enabled":true}'
```

## Öğeler

| Yöntem | Yol | İstek / Yanıt |
| --- | --- | --- |
| GET | `/memory/items` | İsteğe bağlı status, limit, offset; `{success,data:[MemoryItem],total}` |
| POST | `/memory/items` | `{kind,content,importance}`; 200 `{success,data:MemoryItem}` |
| PUT | `/memory/items/:id` | `{content,importance}`; 200 `{success,data:MemoryItem}` |
| DELETE | `/memory/items/:id` | 200 `{"success":true}` |
| POST | `/memory/items/:id/confirm` | 200 `{success,data:MemoryItem}`; çıkarım geçersiz olmuşsa veya dayanağı değiştirilmişse 409 döner |
| POST | `/memory/items/:id/reject` | 200 `{"success":true}` |
| DELETE | `/memory/items` | Mevcut kimliği temizler; 200 `{success,removed}` |

`status` active, pending, superseded, archived olabilir; atlanırsa filtreleme yapılmaz. `limit` varsayılan olarak 50'dir, geçerli aralık 1–200'dür, sınır aşılırsa 50'ye döner; `offset` varsayılan olarak 0'dır, negatif değerler sıfırlanır. kind profile/preference/fact/task/interest olur; içerik en fazla 300 karakterlik kısa bir bellektir, importance önem sıralaması için kullanılır.

MemoryItem; `id`, `kind`, `content`, `topic`, `importance`, `origin`, `status`, `source_session_id`, `source_message_id`, `expires_at`, `superseded_by` ile oluşturulma/değiştirilme zamanlarını içerir. pending isteme dahil edilmez; düzenleme sonrasında el ile bakım olarak işlenir.

Mevcut belleği değiştirmek için kullanılan pending çıkarımlarında, onaydan önce eski öğe yürürlükte kalır; onay, aynı işlem içinde çıkarımı etkinleştirir ve eski öğeyi değiştirir. Geçersiz olmuş, süresi dolmuş veya dayandığı içerik değiştirilmiş/silinmiş çıkarımlar onaylanamaz; 409 döner ve istemci listeyi yenilemelidir. İçerik neredeyse tamamen kimlik bilgileri gibi hassas bilgilerden oluşuyorsa, ekleme veya düzenleme 400 döner.

```bash
curl -X POST "$BASE/api/v1/memory/items" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"kind":"preference","content":"Önce sonucu ver, ardından gerekçeyi açıkla","importance":3}'

curl "$BASE/api/v1/memory/items?status=pending&limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN"

curl -X POST "$BASE/api/v1/memory/items/item-1/confirm" \
  -H "Authorization: Bearer $TOKEN"
```

## Konu ve belge tercihleri

| Yöntem | Yol | Açıklama |
| --- | --- | --- |
| GET | `/memory/topics` | İzlenen, henüz yükseltilmemiş konular; limit/offset öğelerle aynı, data ve total döner |
| POST | `/memory/topics/:id/promote` | Uzun vadeli ilgi olarak manuel yükseltme; `{success,data:MemoryItem}` döner |
| DELETE | `/memory/topics/:id` | Bu konunun izlenmesini durdurur; success döner |
| GET | `/memory/documents` | Belge tercihleri; limit/offset öğelerle aynı, data ve total döner |
| DELETE | `/memory/documents/:id` | Bu tercih kaydını siler; success döner, bilgi tabanı belgesini silmez |

Topic alanları id/topic/aliases/hits/threshold/last_seen_at içerir; Document alanları id/knowledge_id/knowledge_base_id/title/hits/last_used_at içerir. Silme işlemi tercih kaydı id değerini kullanır.

```bash
curl "$BASE/api/v1/memory/topics" -H "Authorization: Bearer $TOKEN"
curl -X POST "$BASE/api/v1/memory/topics/topic-1/promote" \
  -H "Authorization: Bearer $TOKEN"
curl -X DELETE "$BASE/api/v1/memory/documents/affinity-1" \
  -H "Authorization: Bearer $TOKEN"
```

## Dışa aktarma ve anında düzenleme

`GET /memory/export`, indirme dosya adı `rethra-memories.json` ile `{success,total,truncated,data}` döner. En fazla 20,000 kayıt dışa aktarılır; sınıra ulaşıldığında truncated kontrol edilir.

`POST /memory/consolidate`, `{success,data:{merged,demoted,expired,reviewed,candidates,skipped?}}` döner; benzer anlamlı öğeleri hemen birleştirir ve süresi dolanları arşivler. Değişiklik olmadığında skipped nedeni açıklar.

```bash
curl "$BASE/api/v1/memory/export" -H "Authorization: Bearer $TOKEN" \
  -o rethra-memories.json
curl -X POST "$BASE/api/v1/memory/consolidate" -H "Authorization: Bearer $TOKEN"
```

Parametre geçersizse, içerik hassas bilgi içeriyorsa veya bellek etkin değilse 400 döner; mevcut kimliğe ait öğe bulunamazsa 404 döner; onay çakışmasında 409 döner; kimlik doğrulama/izin koşulları karşılanmazsa 401/403 döner. Arayüzde, yöneticilerin başkalarının belleklerini okuması için subject parametresi yoktur. Uygulama: `internal/handler/memory.go`, `internal/router/routes_memory.go`.
