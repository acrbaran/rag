# API başvurusu: FAQ ve Wiki

Bilgi tabanındaki FAQ girdilerini ve Wiki sayfalarını yönetir; içe aktarma, arama, düzenleme ve sürüm geri yüklemeyi destekler.

Her iki grup da KB içerik alt kaynaklarıdır: okuma için Viewer+ ve KB read gerekir (API key `retrieve`/full); yazma için “KB oluşturucusu OR Admin+” ve KB write gerekir (API key `ingest`/full) ve KB izin listesiyle sınırlıdır. Veri tabanları arası Wiki araması `POST /wiki-search`, yol üzerinde KB içermez; Viewer+ ve API key `retrieve`/full gerektirir; handler, her hedef bilgi tabanının allow-list değerini ve paylaşılan bilgi tabanı Viewer iznini doğrular.

## FAQ (/api/v1/knowledge-bases/:id/faq)

### GET /api/v1/knowledge-bases/:id/faq/entries

Kullanım: FAQ girdi listesi.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `page` / `page_size` | int | Hayır | Sayfalama |
| `tag_id` | int | Hayır | Eski tek etiketli seq_id |
| `tag_ids` | string | Hayır | Virgülle ayrılmış etiket UUID'leri |
| `keyword` | string | Hayır | Anahtar kelime |
| `search_field` | string | Hayır | `standard_question`/`similar_questions`/`answers` (varsayılan olarak tüm alanlar) |
| `sort_order` | string | Hayır | `asc` (varsayılan olarak güncelleme zamanına göre azalan sırada) |
| `is_enabled` | bool | Hayır | Etkinlik durumuna göre filtrele: `true` yalnızca etkin, `false` yalnızca devre dışı; gönderilmezse tümü döner; diğer değerler 400 döndürür |

Yanıt: 200 `{"success":true,"data":{sayfalanmış FAQEntry listesi}}`

```bash
curl "$BASE/api/v1/knowledge-bases/kb-1/faq/entries?page=1&is_enabled=false" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/faq/entries/export

Amaç: FAQ dışa aktarma. Sorgu parametreleri: `format` (`csv` varsayılan / `json`).

Yanıt: 200 dosya indirme (`text/csv` veya `application/json`).

```bash
curl -OJ "$BASE/api/v1/knowledge-bases/kb-1/faq/entries/export?format=csv" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/faq/entries/:entry_id

Amaç: FAQ kayıt ayrıntıları (`entry_id` bir tamsayı seq_id'dir).

Yanıt: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/faq/entries/12 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledge-bases/:id/faq/entries

Amaç: Toplu upsert / içe aktarma (eşzamansız görev). Handler metodu `UpsertEntries`.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `entries` | []FAQEntryPayload | Evet (`binding:"required"`) | Toplu kayıtlar |
| `mode` | string | Evet (`binding:"oneof=append replace"`) | Ekleme veya değiştirme |
| `knowledge_id` | string | Hayır | FAQ bilgi varlığı ID'si |
| `task_id` | string | Hayır | Özel görev ID'si; yalnızca harfler, rakamlar, `_`, `-` kabul edilir ve en fazla 128 karakter olabilir, aksi halde 400 döner |
| `dry_run` | bool | Hayır | Yalnızca doğrulama yapar, veritabanına yazmaz |

Yanıt: 200 `{"success":true,"data":{"task_id"}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/entries -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"mode":"append","entries":[{"standard_question":"Nasıl iade alırım?","answers":["Müşteri hizmetleriyle iletişime geçin"]}]}'
```

### POST /api/v1/knowledge-bases/:id/faq/entry

Amaç: Tek bir FAQ oluşturma. İstek gövdesi (`types.FAQEntryPayload`):

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `standard_question` | string | Evet (`binding:"required"`) | Standart soru |
| `similar_questions` | []string | Hayır | Benzer sorular |
| `negative_questions` | []string | Hayır | Negatif örnek sorular |
| `answers` | []string | Hayır | Yanıt listesi |
| `answer_strategy` | string | Hayır | `all` / `random` |
| `tag_id` | int64 | Hayır | Etiket seq_id |
| `tag_name` | string | Hayır | Etiket adı |
| `is_enabled` / `is_recommended` | *bool | Hayır | Etkin/önerilen |

Yanıt: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/entry -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"standard_question":"Nasıl iade alırım?","answers":["7 gün içinde iade edilebilir"]}'
```

### PUT /api/v1/knowledge-bases/:id/faq/entries/:entry_id

Amaç: Tek bir FAQ'yi güncellemek (istek gövdesi oluşturma ile aynıdır).

Yanıt: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/entries/12 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"standard_question":"Nasıl iade alırım?","answers":["30 gün içinde iade edilebilir"]}'
```

### POST /api/v1/knowledge-bases/:id/faq/entries/:entry_id/similar-questions

Amaç: Benzer sorular eklemek. İstek gövdesi: `{"similar_questions":["..."]}` (`binding:"required,min=1"`).

Yanıt: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/entries/12/similar-questions \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"similar_questions":["İade nasıl yapılır"]}'
```

### PUT /api/v1/knowledge-bases/:id/faq/entries/fields

Amaç: Öğe alanlarını toplu olarak güncellemek (`is_enabled`/`is_recommended`/`tag_id`).

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `by_id` | map[int64]object | Hayır | Öğe seq_id değerine göre güncelle |
| `by_tag` | map[int64]object | Hayır | Etikete göre toplu güncelle |
| `exclude_ids` | []int64 | Hayır | `by_tag` sırasında hariç tutulan öğeler |

Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/entries/fields -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"by_id":{"12":{"is_enabled":false}}}'
```

### PUT /api/v1/knowledge-bases/:id/faq/entries/tags

Amaç: Öğe etiketlerini toplu olarak değiştirmek. İstek gövdesi: `{"updates":{"<entry_id>":<tag_id|null>}}` (`binding:"required,min=1"`; null etiketi kaldırır).

Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/entries/tags -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"updates":{"12":3}}'
```

### DELETE /api/v1/knowledge-bases/:id/faq/entries

Amaç: Öğeleri toplu olarak silmek. İstek gövdesi: `{"ids":[int64]}` (`binding:"required,min=1"`).

Yanıt: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1/faq/entries -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"ids":[12,13]}'
```

### POST /api/v1/knowledge-bases/:id/faq/search

Amaç: FAQ araması (salt okunur anlamı; kapsamlı anahtar ile `retrieve` de çağrılabilir).

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `query_text` | string | Evet (`binding:"required"`) | Sorgu |
| `vector_threshold` | float64 | Hayır | Vektör eşiği, varsayılan 0.7 |
| `match_count` | int | Hayır | Varsayılan 10, üst sınır 50 |
| `first_priority_tag_ids` / `second_priority_tag_ids` | []int64 | Hayır | Etiket önceliği filtresi |
| `only_recommended` | bool | Hayır | Yalnızca önerilen girdiler |

Yanıt: 200 `{"success":true,"data":[FAQEntry(match_type/score içerir)]}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query_text":"İade"}'
```

### PUT /api/v1/knowledge-bases/:id/faq/import/last-result/display

Amaç: En son içe aktarma sonuç panelinin görüntülenme durumunu ayarlamak. İstek gövdesi: `{"display_status":"open|close"}` (`binding:"required,oneof=open close"`).

Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/import/last-result/display \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"display_status":"close"}'
```

### GET /api/v1/faq/import/progress/:task_id

Amaç: FAQ içe aktarma/dry-run ilerlemesini sorgulamak (görevler alana göre yalıtılır). Yetki: Viewer+; API anahtarı `retrieve`/`ingest`/full.

Yanıt: 200 `{"success":true,"data":{status,progress,failed_entries,...}}`

```bash
curl $BASE/api/v1/faq/import/progress/task-1 -H "X-API-Key: $API_KEY"
```

## Wiki (/api/v1/knowledgebase/:kb_id/wiki)

Bu grubun ön ekinin `/knowledgebase/:kb_id/wiki` olduğuna dikkat edin (tekil, tire yok). İşleyici: `internal/handler/wiki_page.go`. Bu grubun yanıtları çoğunlukla **ham nesnelerdir** (`success` sarmalayıcısı olmadan).

### GET /api/v1/knowledgebase/:kb_id/wiki/pages

Amaç: Wiki sayfası listesi.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `page_type` | string | Hayır | Virgülle ayrılmış türler |
| `status` | string | Hayır | Sayfa durumu |
| `query` | string | Hayır | Tam metin araması |
| `category_path` | string | Hayır | `/` ile ayrılmış yol filtresi |
| `folder_id` | string | Hayır | Kesin klasör filtresi (boş dize=kök) |
| `category_depth` | int | Hayır | Klasör derinliği |
| `page` / `page_size` | int | Hayır | Sayfalama (varsayılan 1/20) |
| `sort_by` / `sort_order` | string | Hayır | Sıralama alanları: `title`, `created_at`, `updated_at`, `page_type`, `wiki_path`, `sort_order`, `depth`; diğer değerler `updated_at` olarak ele alınır; varsayılan `updated_at` desc |

Yanıt: 200 `WikiPageListResponse`

```bash
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/pages?page=1" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/pages

Amaç: Sayfa oluşturmak. İstek gövdesi (`types.WikiPage`): `slug`, `title`, `content`, `folder_id`, `page_type` vb. (tümü isteğe bağlıdır; slug belirtilmezse otomatik oluşturulur).

Yanıt: 201 `WikiPage`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/pages -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"Mimari genel bakış","content":"# Genel bakış"}'
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/move-page

Amaç: Sayfayı klasöre taşımak. İstek gövdesi: `{"slug":"<sayfa slug>","folder_id":"<klasör ID|boş=kök>"}` (slug zorunludur).

Yanıt: 200 `WikiPage`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/move-page -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"slug":"overview","folder_id":"f-1"}'
```

### GET /api/v1/knowledgebase/:kb_id/wiki/pages/*slug

Amaç: Sayfayı almak (`*slug` joker yoludur).

Yanıt: 200 `WikiPage`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/pages/overview -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/pages/*slug

Amaç: Sayfayı güncellemek (istek gövdesi oluşturmayla aynıdır). Eski sürüm önce tam anlık görüntü olarak `wiki_page_revisions` içine kaydedilir, `version` artırılır ve `last_edit_source` `user` olarak işaretlenir (Agent aracı yazdığında `agent` olur).

Yanıt: 200 `WikiPage`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/pages/overview -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"content":"# Güncellenmiş genel bakış"}'
```

### GET /api/v1/knowledgebase/:kb_id/wiki/revisions/*slug

Amaç: Sayfa sürüm geçmişi (migration `000075`). Yetki: Viewer+ + KBAccessRead.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `version` | int | Hayır | Verildiğinde **bu sürümün tam metnini** döndürür (diff için); geçersiz veya < 1 ise 400, bulunamazsa 404 döner (yükseltmeden önce yazılan sürümlerde veya saklama politikası tarafından temizlenen anlık görüntülerde tam metin yoktur; bu normaldir) |
| `limit` | int | Hayır | Varsayılan 50, üst sınır 200; yalnızca liste modunda geçerlidir |
| `offset` | int | Hayır | Sayfalama kaydırması |

`version` olmadan geçmiş listesi (sürüm numarasına göre azalan, **gövde içermeyen**) ve sayfanın mevcut sürüm numarası döner; her kayıt `edit_source` (`pipeline` / `agent` / `user` / `revert`), `editor_id`, `edited_at` içerir.

Geçmiş saklama iki düzeyli sınıra sahiptir: 50 sürümlük yumuşak sınır yalnızca `pipeline` ve kaynağı boş olan anlık görüntüleri budar; 200 sürümlük kesin sınır tüm kaynaklara uygulanır, böylece manuel düzenlemeler işlem hattı tarafından silinmez.

```bash
# Geçmiş listesi
curl $BASE/api/v1/knowledgebase/kb-1/wiki/revisions/entity/acme-corp -H "Authorization: Bearer $TOKEN"
# 3. sürümün tam metnini al
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/revisions/entity/acme-corp?version=3" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/revert

Amaç: Sayfayı belirli bir geçmiş sürüme geri almak. Yetki: KB owner veya Admin+ + KBAccessWrite.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `slug` | string | Evet | Hedef sayfa |
| `version` | int | Evet | Hedef sürüm numarası (≥ 1) |

Geri alma **sürüm numarasını geri çekmez**: hedef sürümün içeriği yeni bir sürüm olarak yazılır, `last_edit_source` `revert` olarak kaydedilir; dolayısıyla geri alma işlemi de geri alınabilir. Mevcut sürüme geri alma 400 döndürür (genellikle ön uç geçmiş listesi güncel değildir).

Yanıt: 200 `WikiPage`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/revert -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"slug":"entity/acme-corp","version":3}'
```

### DELETE /api/v1/knowledgebase/:kb_id/wiki/pages/*slug

Amaç: Sayfayı silmek.

Yanıt: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/knowledgebase/kb-1/wiki/pages/overview -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/folders

Amaç: Dizin listesi. Sorgu parametreleri: `parent_id` (boş=kökte), `page_types` (virgülle ayrılmış).

Yanıt: 200 `WikiFolderListResponse`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/folders -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/folders

Amaç: Dizin oluşturmak.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet | Dizin adı |
| `parent_id` | string | Hayır | Üst dizin |

Yanıt: 201 `WikiFolder`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/folders -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Tasarım belgeleri"}'
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/folders/:folder_id

Amaç: Dizini yeniden adlandırmak/taşımak. İstek gövdesi: `name`, `parent_id`, `move_parent` (bool); tümü isteğe bağlıdır.

Yanıt: 200 `WikiFolder`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/folders/f-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Mimari tasarım"}'
```

### DELETE /api/v1/knowledgebase/:kb_id/wiki/folders/:folder_id

Amaç: dizini silmek.

Yanıt: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/knowledgebase/kb-1/wiki/folders/f-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/index

Amaç: Wiki dizin sayfası (türe göre gruplandırılmış pencere). Sorgu parametreleri: `types` (virgülle ayrılmış), `limit` (1-200, varsayılan 50), `cursor` (imleç).

Yanıt: 200 `WikiIndexResponse`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/index -H "Authorization: Bearer $TOKEN"
```

::: warning Kaldırıldı
`GET /api/v1/knowledgebase/:kb_id/wiki/log` (Wiki değişiklik günlüğü), `000077_remove_wiki_log` geçişiyle birlikte kaldırıldı ve `wiki_log_entries` tablosu silindi. Wiki değişiklikleri artık bilgi tabanı etkinlik akışına birleşik olarak yansıtılır; bunun yerine `GET /api/v1/knowledge-bases/:id/activity` kullanın.
:::

### GET /api/v1/knowledgebase/:kb_id/wiki/graph

Amaç: sayfa ilişki grafiği.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `mode` | string | Hayır | `overview` (varsayılan) / `ego` |
| `center` | string | Hayır | ego modu merkez slug değeri (ego için zorunlu) |
| `depth` | int | Hayır | 1-3, varsayılan 1 |
| `types` | string | Hayır | page_type filtresi |
| `limit` | int | Hayır | Varsayılan 500, üst sınır 2000 |

Yanıt: 200 `WikiGraphData`

```bash
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/graph?mode=overview" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/stats

Amaç: Wiki istatistikleri.

Yanıt: 200 `WikiStats`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/stats -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/search

Amaç: sayfa arama. Sorgu parametreleri: `q` (zorunlu), `limit` (varsayılan 10).

Eşleştirme anlamı depolama lehçesine göre değişir:

- Postgres: `q`, POSIX düzenli ifadesiyle (`~*`, büyük/küçük harfe duyarsız) eşleştirilir.
- SQLite / Lite: `LIKE` ile değişmez alt dize eşleştirmesi yapılır; `%` / `_` kaçırılır ve joker karakter olarak değerlendirilmez.

Yanıt: 200 `{"pages":[WikiPage]}`

```bash
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/search?q=dağıtım" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/wiki-search

Amaç: bilgi tabanları arasında Wiki sayfası araması (oturumsuz, karma olmayan getirme). İşleyici: `internal/handler/wiki_page.go` içindeki `SearchPagesAcross`. Algoritma, tek tabanlı `GET .../wiki/search` ile aynıdır (başlık > slug > özet > gövde); birden çok tabanda tek sorgudan sonra `match_rank` değerine göre genel olarak kesilir. API anahtarı: `retrieve`/full.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `query` | string | Evet (`binding:"required"`) | Tek tabanlı `q` ile aynı: Postgres için POSIX düzenli ifadesi (büyük/küçük harfe duyarsız); SQLite / Lite için değişmez `LIKE` alt dizesi |
| `knowledge_base_ids` | []string | Koşullu zorunlu | Çoklu taban; `knowledge_base_id` ile en az biri sağlanmalıdır, en fazla 32 adet |
| `knowledge_base_id` | string | Hayır | Tek depo uyumluluk alanı, ids içine birleştirildi |
| `limit` | int | Hayır | Genel top-N, varsayılan 10, en fazla 50 |

Hedef depolardan herhangi birinde wiki etkin değilse → 400. Depo yoksa veya okuma izni yoksa → 404. API Key beyaz liste yetkisi aşılırsa → 403. Geçersiz regex (Postgres) → 400.

Yanıt: 200 `{"success":true,"data":[WikiSearchHit]}`. Eşleşmeler yalnızca gezinme alanları + `match_snippet` içerir, gövde içermez; tam metin için `GET /knowledgebase/:kb_id/wiki/pages/{slug}` kullanılır.

| Yanıt alanı | Tür | Açıklama |
| --- | --- | --- |
| `id` | string | Sayfa ID'si |
| `knowledge_base_id` | string | Kaynak depo, depolar arası kullanımda okuma sayfası URL'sini oluşturmak için kullanılır |
| `slug` | string | Sayfa slug'ı |
| `title` | string | Başlık |
| `page_type` | string | Sayfa türü |
| `aliases` | []string | Takma adlar |
| `summary` | string | Özet |
| `match_snippet` | string | Gövdedeki ilk eşleşmenin çevresinde yaklaşık 60+eşleşme+60 karakter; gövdede eşleşme yoksa atlanır. snippet, `query` değerini yine Go regex ile derler; SQLite altında `query` regex meta karakterleri içerdiğinde depo içinde eşleşme bulunabilir, snippet boş olabilir |

`content`, ağaç/bağlantılar/meta veriler ve zaman damgaları arama sonuçlarında yer almaz. Tek depo için `GET .../wiki/search` hâlâ tam `WikiPage` döndürür.

```bash
curl -X POST $BASE/api/v1/wiki-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"dağıtım|yayın","knowledge_base_ids":["kb-1","kb-2"],"limit":10}'
```

### POST /api/v1/knowledgebase/:kb_id/wiki/rebuild-links

Amaç: sayfa içi karşılıklı bağlantıları yeniden oluşturmak. Yazma izni gerekir. İstek gövdesi yoktur.

Yanıt: 200 `{"message":"Links rebuilt successfully"}`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/rebuild-links -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/lint

Amaç: Wiki tutarlılık denetim raporu.

Yanıt: 200 `WikiLintReport`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/lint -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/auto-fix

Amaç: lint sorunlarını otomatik düzeltmek. Yazma izni gerekir. İstek gövdesi yoktur.

Yanıt: 200 `{"fixed":N,"message":"Auto-fixed N issues"}`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/auto-fix -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/issues

Amaç: sorun listesi. Sorgu parametreleri: `slug` (sayfaya göre filtreleme), `status` (`pending/ignored/resolved`).

Yanıt: 200 `[WikiPageIssue]`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/issues -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/issues/:issue_id/status

Amaç: sorun durumunu güncellemek. Yazma izni gerekir. İstek gövdesi: `{"status":"pending|ignored|resolved"}` (`binding:"required"`). `issue_id`, yoldaki bilgi deposuna ait olmalıdır; aksi hâlde 404.

Yanıt: 200 `{"message":"Issue status updated successfully"}`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/issues/i-1/status -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"resolved"}'
```

## Uygulama referansı

Yönlendirme kaydı: `internal/router/routes_knowledge.go` içindeki `RegisterFAQRoutes` ve `RegisterWikiPageRoutes`. İşleyiciler: `internal/handler/faq.go`, `internal/handler/wiki_page.go`.
