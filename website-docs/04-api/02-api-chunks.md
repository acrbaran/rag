# API Referansı: Parçalar ve etiketler

Parçalar (chunk), aramanın en küçük birimidir; etiketler belgeleri sınıflandırmak için kullanılır. Her iki arayüz grubu da bilgi tabanı kaynaklarına aittir ve [Bilgi Tabanı ve Bilgi](./02-api-knowledge.md) ile aynı izin kurallarını paylaşır: okuma için Viewer+ ve üst KB üzerinde read izni gerekir (API key `retrieve`); yazma için "KB oluşturucusu OR Admin+" ve write izni gerekir (API key `ingest`); her ikisi de API key'in KB beyaz listesiyle sınırlıdır.

Genel kurallar (Base URL, kimlik doğrulama, hata kodları, sayfalama) için bkz. [API Genel Bakış](./01-api-overview.md).

## Parçalar (/api/v1/chunks)

Handler: `internal/handler/chunk.go`. Okuma: Viewer+ ve üst KB read (`retrieve`/full API key); yazma: KB oluşturucusu OR Admin+ ve üst KB write (`ingest`/full API key).

### GET /api/v1/chunks/:knowledge_id

Amaç: bilgi parçalarının listesi.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `page` | int | Hayır | Varsayılan 1 |
| `page_size` | int | Hayır | Varsayılan 10, üst sınır 100 |
| `chunk_type` | string | Hayır | Tekrarlanabilir, parça türüne göre filtreler |

Yanıt: 200 `{"success":true,"data":[Chunk],"total","page","page_size"}`

```bash
curl "$BASE/api/v1/chunks/k-1?page=1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/chunks/by-id/:id

Amaç: chunk ID ile tek bir parça almak (`knowledge_id` gerekmez).

Yanıt: 200 `{"success":true,"data":{Chunk}}`. `Chunk.source_locators`, parçanın özgün dosyadaki konumudur; yapı için [API Genel Bakış](01-api-overview.md) içindeki `source_locators` açıklamasına bakın.

```bash
curl $BASE/api/v1/chunks/by-id/c-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/chunks/:knowledge_id/:id

Amaç: Parça içeriğini veya etkinlik durumunu düzenlemek (`000078` migration sürümünden itibaren sürümlü iyimser güncelleme kullanılır).

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `content` | string | Hayır | Yeni içerik; baştaki ve sondaki boşluklar kaldırıldıktan sonra boş olamaz, uzunluk üst sınırı 200000 bayttır |
| `is_enabled` | bool | Hayır | Bu parçayı etkinleştirir/devre dışı bırakır |
| `expected_revision` | int | Hayır | İyimser eşzamanlılık denetimi için beklenen `content_revision` |

Kısıtlamalar ve yan etkiler:

- Yalnızca `text` türündeki parçalar düzenlenebilir, diğer türler 400 döndürür;
- `expected_revision` mevcut `content_revision` ile eşleşmezse **409** döndürülür (`Chunk was modified by another user; refresh and retry`);
- Düzenleme sırasında kaynak içerikte bulunmayan görsel URL'lerinin eklenmesine izin verilmez; Markdown görsellerinin silinmesi, ilgili OCR/caption alt parçalarını eşzamanlı olarak devre dışı bırakır;
- Düzenleme başarılı olduktan sonra `content_revision` +1 olur, eski sürüm `chunk_revisions` tablosuna yazılır, `index_status` sırasıyla `processing` → `ready` durumlarından geçer; arama dizininin yeniden oluşturulması başarısız olursa satır yine kaydedilir ancak `index_status = failed` olur; aynı içerik yeniden gönderilerek tekrar deneme tetiklenebilir;
- Alt parça düzenlemeleri, ofsetlere göre üst parça içeriğine geri yazılır (üst parçanın `source_content` değeri değiştirilemez kalır);
- İçerik veya etkinlik durumundaki değişiklik, bir belge özeti yenilemesini kuyruğa ekler.

Yanıt: 200 `{"success":true,"data":{Chunk},"summary_status":"pending","description":"..."}`

```bash
curl -X PUT $BASE/api/v1/chunks/k-1/c-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"content":"Düzeltilmiş içerik","expected_revision":2}'
```

### GET /api/v1/chunks/:knowledge_id/:id/revisions

Amaç: Parçanın geçmiş sürümlerinin listesi (`chunk_revisions` tablosu, revision değerine göre azalan sırada).

Yanıt: 200 `{"success":true,"data":[{ChunkRevision}]}`; her kayıt `revision`, `content`, `is_enabled`, `editor_id`, `edit_source`, `edited_at` içerir.

```bash
curl $BASE/api/v1/chunks/k-1/c-1/revisions -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/chunks/:knowledge_id/:id/revert

Amaç: Belirli bir geçmiş sürüme geri dönmek. Geri alma işlemi de yeni bir düzenlemedir: `content_revision` artmaya devam eder, mevcut içerik yeni bir geçmiş sürüm olarak kaydedilir.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `revision` | int | Evet | Hedef geçmiş sürüm numarası (negatif olmayan) |
| `expected_revision` | int | Hayır | İyimser kilit, anlamı yukarıdakiyle aynıdır; çakışmada 409 döner |

Yanıt: 200 `{"success":true,"data":{Chunk},"summary_status":"...","description":"..."}`

```bash
curl -X POST $BASE/api/v1/chunks/k-1/c-1/revert -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"revision":1}'
```

### DELETE /api/v1/chunks/:knowledge_id/:id

Amaç: Tek bir parçayı silmek.

Yanıt: 200 `{"success":true,"message":"Chunk deleted"}`

```bash
curl -X DELETE $BASE/api/v1/chunks/k-1/c-1 -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/chunks/:knowledge_id

Amaç: Bilgi altındaki tüm parçaları silmek.

Yanıt: 200 `{"success":true,"message":"All chunks under knowledge deleted"}`

```bash
curl -X DELETE $BASE/api/v1/chunks/k-1 -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/chunks/by-id/:id/questions

Amaç: Bu parça altındaki oluşturulmuş bir soruyu silmek. İstek gövdesi: `{"question_id":"..."}` (`binding:"required"`).

Yanıt: 200 `{"success":true,"message":"Generated question deleted"}`

```bash
curl -X DELETE $BASE/api/v1/chunks/by-id/c-1/questions -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"question_id":"q-1"}'
```

### PUT /api/v1/chunks/by-id/:id/questions

Amaç: Bu parça için oluşturulmuş bir soru eklemek veya değiştirmek. İstek gövdesi: `{"question":"...","question_id":"..."}`; `question` zorunludur; boş `question_id` yeni ekleme anlamına gelir.

Yanıt: 200 `{"success":true,"data":{GeneratedQuestion}}`

```bash
curl -X PUT $BASE/api/v1/chunks/by-id/c-1/questions -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"question_id":"q-1","question":"Rethra vektör deposunu nasıl yapılandırır?"}'
```

### POST /api/v1/chunks/by-id/:id/questions/regenerate

Amaç: Parçanın güncel içeriğine göre arama sorularını yeniden oluşturmak. İçerik düzenlendikten sonra mevcut sorular silinmez, bunun yerine "süresi geçmiş" olarak işaretlenir (revision güncel ana metinle eşleşmez); bu arayüzle yenilenebilir.

Yanıt: 200 `{"success":true,"data":[{GeneratedQuestion}]}`

```bash
curl -X POST $BASE/api/v1/chunks/by-id/c-1/questions/regenerate -H "Authorization: Bearer $TOKEN"
```

## Etiketler (/api/v1/knowledge-bases/:id/tags)

İşleyici: `internal/handler/tag.go`. Okuma: Viewer+ + KB okuma (API anahtarı `retrieve`/full); yazma: KB oluşturucusu VEYA Admin+ + KB yazma (API anahtarı `ingest`/full).

### GET /api/v1/knowledge-bases/:id/tags

Amaç: Etiket listesi. Sorgu parametreleri: `page`, `page_size`, `keyword` (tümü isteğe bağlı).

Yanıt: 200 `{"success":true,"data":[KnowledgeTag]}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/tags -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledge-bases/:id/tags

Amaç: Etiket oluşturmak.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet (`binding:"required"`) | Etiket adı |
| `color` | string | Hayır | Renk |
| `sort_order` | int | Hayır | Sıralama |

Yanıt: 200 `{"success":true,"data":{KnowledgeTag}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/tags -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Satış sonrası"}'
```

### PUT /api/v1/knowledge-bases/:id/tags/:tag_id

Amaç: Etiketi güncellemek (`tag_id` UUID veya tamsayı seq_id destekler). İstek gövdesi: `name`/`color`/`sort_order` (işaretçi alanlar, tümü isteğe bağlı).

Yanıt: 200 `{"success":true,"data":{KnowledgeTag}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/tags/t-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Satış sonrası destek"}'
```

### DELETE /api/v1/knowledge-bases/:id/tags/:tag_id

Amaç: Etiketi silmek. Sorgu parametreleri: `force` (bool, başvurulan etiketi zorla siler), `content_only` (bool, yalnızca etiket altındaki içeriği siler, etiketi korur). İstek gövdesi (isteğe bağlı): `{"exclude_ids":[int64]}`, silme sırasında korunacak FAQ öğelerinin seq_id değerlerini listeler.

İstek gövdesi biçimi geçersizse veya ID pozitif bir tamsayı değilse 400; ID yoksa 404; ID mevcut bilgi tabanına ait bir FAQ girdisi değilse 403 döner.

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE "$BASE/api/v1/knowledge-bases/kb-1/tags/t-1?force=true" -H "Authorization: Bearer $TOKEN"
```

## Uygulama referansı

Rota kaydı: `internal/router/routes_knowledge.go` içindeki `RegisterChunkRoutes`, `RegisterKnowledgeTagRoutes`.
