# API başvurusu: bilgi tabanları ve bilgi

Bilgi tabanları oluşturun, belgeleri içe aktarın ve yönetin; işleme ilerlemesini sorgulayın, içeriği kopyalayın veya taşıyın.

Yetki özeti: okuma rotaları Viewer+ gerektirir ve KB için read izni olmalıdır (sahip olunan/kuruluşla paylaşılan/paylaşılan Agent görünürlüğü); yazma rotaları “KB oluşturucusu OR Admin+” ve write izni gerektirir. API anahtarı: okuma için `retrieve`, içerik yazma için `ingest`, KB yaşam döngüsü için `manage_kbs` gerekir (tümü full-access tarafından geçersiz kılınabilir) ve KB beyaz listesine tabidir.

Parçalama ve etiket arayüzleri (`/chunks`, `/knowledge-bases/:id/tags`) [Parçalama ve etiketler](./02-api-chunks.md) bölümündedir.

## Bilgi tabanı (/api/v1/knowledge-bases)

### POST /api/v1/knowledge-bases

Amaç: Bilgi tabanı oluşturur. Yetki: Contributor+; API anahtarı `manage_kbs`/full. İşleyici: `internal/handler/knowledgebase.go`

İstek gövdesi (`types.KnowledgeBase`):

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet | Ad |
| `description` | string | Hayır | Açıklama |
| `type` | string | Hayır | `document` (varsayılan) /`faq`/`wiki` |
| `embedding_model_id` | string | Hayır | Gömme modeli ID'si |
| `chunking_config` | object | Hayır | Parçalama yapılandırması (chunk_size/overlap/separators/strategy…) |
| `image_processing_config` | object | Hayır | Görüntü özniteliği inceleme yapılandırması: `model_id` / `image_attrs_enabled` / `image_actions` (`{ ocr: { on: [...], on_unobserved: bool } }`) |
| `storage_provider_config` | object | Hayır | Depolama yapılandırması |
| `vector_store_id` | string | Hayır | Vektör deposu bağlantısı (geçersizse code 2200/2201 döner) |
| `faq_config` / `wiki_config` / `extract_config` / `indexing_strategy` | object | Hayır | Türe bağlı yapılandırma |
| `summary_model_id` | string | Hayır | Özet modeli; ayrıca otomatik etiketler ve AI açıklaması için varsayılan model |
| `auto_tag_config` / `profile_config` | object | Hayır | Otomatik etiketler, AI bilgi tabanı açıklaması (yalnızca document türü, aşağıya bakın) |

Yanıt: 201 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Ürün belgeleri","type":"document"}'
```

### Otomatik etiket ve AI açıklama yapılandırması

Bilgi tabanı oluşturulurken `auto_tag_config`, `profile_config` üst düzeyde bulunur; güncelleme sırasında `config.auto_tag_config`, `config.profile_config` içine yerleştirilir. Her ikisi de yalnızca document bilgi tabanları tarafından desteklenir ve varsayılan olarak enabled=false'tır.

`auto_tag_config` (otomatik etiketler):

| Alan | Tür | Varsayılan değer | Açıklama |
| --- | --- | --- | --- |
| `enabled` | bool | false | Ayrıştırmadan sonra mevcut etiketlerden eşzamansız olarak seçer |
| `model_id` | string | Boş | Boş olduğunda bilgi tabanının `summary_model_id` değeri kullanılır |
| `max_tags` | int | 3 | Her makaleyle ilişkilendirilebilecek en fazla sayı, üst sınır 10 |
| `skip_if_tagged` | bool | true | Mevcut etiket varsa atlar; false ek etiketlere izin verir |

Etkinleştirildikten sonra yeni ayrıştırılan/yeniden ayrıştırılan belgeler için geçerlidir; tüm eski belgeleri otomatik olarak taramaz. Aday etiket veya kullanılabilir model olmadığında bilgi tabanına eklemeyi engellemez.

`profile_config` (AI bilgi tabanı açıklaması):

| Alan | Tür | Varsayılan değer | Açıklama |
| --- | --- | --- | --- |
| `enabled` | bool | false | Etkinleştirildiğinde, belge ekleme, silme, taşıma veya özet güncellemesi `generated_profile` değerini otomatik olarak yeniler |
| `model_id` | string | boş | Boş olduğunda bilgi tabanının `summary_model_id` değeri kullanılır |
| `custom_instructions` | string | boş | Oluşturma istemine eklenen tamamlayıcı gereksinimler |

`generated_profile` yalnızca okunabilir bir alandır ve sistem tarafından yazılır; elle yazılan `description` değerinin üzerine yazmaz. Ayrıca aşağıdaki `profile/generate` aracılığıyla hemen oluşturulabilir. Güncelleme örneği:

```bash
curl -X PUT "$BASE/api/v1/knowledge-bases/kb-1" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Ürün belgeleri","config":{"auto_tag_config":{"enabled":true,"max_tags":3,"skip_if_tagged":true}}}'
```

### GET /api/v1/knowledge-bases

Amaç: Bilgi tabanı listesi. Yetki: Viewer+; API anahtarı `retrieve`/full.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `agent_id` | string | Hayır | Belirli bir paylaşılan Agent'ın görebildiği KB'leri filtreler |
| `agent_source_tenant_id` | uint64 | Hayır | Aynı adlı Agent birden fazla alan tarafından paylaşıldığında, kaynak alanı belirtir; değer paylaşım ilişkisiyle doğrulanır, geçersiz değer doğrudan 400 döndürür |
| `creator` | string | Hayır | `mine` / `others` |

Yanıt: 200 `{"success":true,"data":[KnowledgeBase],"total","page","page_size"}`

```bash
curl $BASE/api/v1/knowledge-bases -H "X-API-Key: $API_KEY"
```

### GET /api/v1/knowledge-bases/:id

Amaç: Bilgi tabanı ayrıntıları (paylaşılan KB, `my_permission` içerir). Yetki: Viewer+, KB read. Sorgu parametresi: `agent_id` (isteğe bağlı).

Yanıt: 200 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id

Amaç: Bilgi tabanını güncelleme. Yetki: Oluşturan OR Admin+, KB write; API anahtarı `manage_kbs`/full.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `name` | string | Evet (`binding:"required"`) | Ad |
| `description` | string | Hayır | Açıklama |
| `config` | object | Hayır | Kısmi yapılandırma güncellemesi: `chunking_config`, `image_processing_config`, `faq_config`, `wiki_config`, `auto_tag_config`, `profile_config`, `indexing_strategy` |

Yanıt: 200 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Ürün belgeleri v2"}'
```

### DELETE /api/v1/knowledge-bases/:id

Amaç: Bilgi tabanını silme (sahip alan + Admin ile sınırlıdır; paylaşılan editor silemez). Yetki: Oluşturan OR Admin+, KB write; API anahtarı `manage_kbs`/full.

Yanıt: 200 `{"success":true,"message":"Knowledge base deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id/pin

Amaç: Sabitleme/sabitlemeyi kaldırma (kullanıcı bazında saklanır). Yetki: Viewer+, KB read. İstek gövdesi yoktur.

Yanıt: 200 `{"success":true,"data":{KnowledgeBase(is_pinned değiştirildi)}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/pin -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledge-bases/:id/hybrid-search (GET ile uyumlu)

Amaç: KB içindeki alt seviye geri çağırma (vektör+anahtar kelime) için kullanılır; varsayılan olarak rerank uygulanmaz ve geri çağırma puanı döndürülür; isteğe bağlı olarak rerank etkinleştirilebilir. Geri çağırmayı değerlendirme, önceden hesaplanmış vektör iletme gibi ham geri çağırmanın kontrol edilmesi gereken senaryolar için uygundur; genel aramalar için [`knowledge-search`](./02-api-chat.md) kullanın, yöntem seçimi için bkz. [Arama API'si nasıl seçilir](./01-api-overview.md#retrieval-api). Yetki: Viewer+, KB read; API anahtarı `retrieve`/full. JSON body ile GET yalnızca geriye dönük uyumluluk içindir (#1727), POST önerilir.

Sorgu parametresi: `resource_urls=handle|public` (`public`, sonuçlardaki `content` / `image_info` içindeki `resource://` değerlerini yüklenebilir doğrudan bağlantılarla değiştirir; ayrıntılar için bkz. [API Genel Bakış](./01-api-overview.md)).

İstek gövdesi (`types.SearchParams`):

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `query_text` | string | Koşullu zorunlu | Sorgu metni (`query_embedding` sağlanmadıkça; rerank etkin olduğunda zorunludur) |
| `query_embedding` | []float32 | Hayır | Önceden hesaplanmış vektör |
| `vector_threshold` / `keyword_threshold` | float64 | Hayır | Eşleşme eşiği |
| `match_count` | int | Hayır | Döndürülecek sonuç sayısı üst sınırı (varsayılan 50) |
| `disable_keywords_match` / `disable_vector_match` | bool | Hayır | Bir geri çağırma yolunu devre dışı bırakır |
| `knowledge_base_ids` | []string | Hayır | Tek aramada birden fazla bilgi tabanında arama yapar; yoldaki `:id` bunların içinde olmalıdır; bu bilgi tabanlarının embedding modelleri aynı olmalıdır, aksi halde 400 döner |
| `knowledge_ids` | []string | Hayır | Bilgi öğelerini sınırlar |
| `tag_ids` | []string | Hayır | Etiket filtresi (OR) |
| `only_recommended` | bool | Hayır | FAQ yalnızca önerilen öğeler |
| `skip_context_enrichment` | bool | Hayır | Üst blok/bağlam tamamlama işlemini atlar |
| `rerank` | object | Hayır | Gönderildiğinde rerank etkinleşir (`{}` alan yapılandırmasındaki modeli kullanır); alanlar için bkz. [rerank nesnesi](./01-api-overview.md#retrieval-api) |

Yanıt: 200 `{"success":true,"data":[SearchResult]}`; `rerank` ile birlikte ek olarak `meta.rerank` bulunur (bkz. [meta.rerank tanılama](./01-api-overview.md#retrieval-api)).

```bash
curl -X POST "$BASE/api/v1/knowledge-bases/kb-1/hybrid-search?resource_urls=public" -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query_text":"İade süreci","match_count":5}'

# Geri çağırma parametrelerini sabitle, ardından belirtilen modelle rerank yap
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/hybrid-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query_text":"İade süreci","vector_threshold":0.3,"match_count":5,"rerank":{"model_id":"rr-1","threshold":0.2}}'
```

### POST /api/v1/knowledge-bases/copy

Amaç: KB'ler arasında içerik kopyalamak için kullanılır (asenkron görev). Yetki: Contributor+; API anahtarı `manage_kbs`/full (kaynak/hedef KB izin listesi handler tarafından doğrulanır).

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `source_id` | string | Evet (`binding:"required"`) | Kaynak KB |
| `target_id` | string | Hayır | Hedef KB (boşsa otomatik oluşturulur) |
| `task_id` | string | Hayır | Özel görev ID'si |

Yanıt: 200 `{"success":true,"data":{"task_id","source_id","target_id","message"}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/copy -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"source_id":"kb-1"}'
```

### POST /api/v1/knowledge-bases/:id/duplicate

Amaç: KB kopyası oluşturmak için kullanılır (yalnızca ayarları kopyalar; içerik/indeks/paylaşım kopyalanmaz). Yetki: Contributor+, kaynak KB read; API anahtarı `manage_kbs`/full. İstek gövdesi yoktur.

Yanıt: 201 `{"success":true,"data":{"source_id","target_id","message","knowledge_base":{...}}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/duplicate -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledge-bases/:id/profile/generate

Amaç: Bilgi tabanının AI açıklamasını (`generated_profile`) hemen yeniden oluşturur; bir belge profili toplama ve bir küçük model çağrısını eşzamanlı olarak çalıştırır, elle yazılmış `description` değerini değiştirmez. Yetki: bilgi tabanını güncelleme ile aynıdır (oluşturan/Admin ve KB write); API anahtarı `manage_kbs`/full. İstek gövdesi yoktur. Yalnızca document türü için geçerlidir; model yapılandırılmamışsa 400 döner.

Yanıt: 200 `{"success":true,"data":{"gist","topics":[...],"typical_questions":[...],"stats":{"document_count",...},"status":"ready","model_id","generated_at"}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/profile/generate -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/copy/progress/:task_id

Amaç: Kopyalama ilerlemesini sorgulamak (görevler alana göre yalıtılır). Yetki: Viewer+; API anahtarı `retrieve`/`manage_kbs`/full.

Yanıt: 200 `{"success":true,"data":{status,progress,message,...}}`

```bash
curl $BASE/api/v1/knowledge-bases/copy/progress/task-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/move-targets

Amaç: Taşıma hedefi olarak kullanılabilecek KB'leri listelemek (aynı tür/aynı embedding). Yetki: Viewer+, KB okuma.

Yanıt: 200 `{"success":true,"data":[KnowledgeBase]}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/move-targets -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/files

Amaç: KB kapsamlı dosya vekili (paylaşılan KB içeriğindeki görselleri işler; bağlamdaki tenant, KB sahibine göre yeniden yazılmıştır). Yetki: Viewer+, KB okuma; KB kısıtlı anahtarları reddedilir, alan genelindeki `retrieve`/full anahtarlarına izin verilir. `serveKBScopedFiles` içinde kaydedilmiştir (`internal/router/files.go`).

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `file_path` | string | Evet | `provider://...` depolama yolu (`..` yasaktır) |

Yanıt: 200 dosya akışı (`Content-Type` uzantıya göre çıkarılır; `Cache-Control: private`).

```bash
curl "$BASE/api/v1/knowledge-bases/kb-1/files?file_path=local://1/exports/chart.png" \
  -H "Authorization: Bearer $TOKEN" -o chart.png
```

## Bilgi (KB içeriği, /api/v1/knowledge-bases/:id/knowledge ve /api/v1/knowledge)

### POST /api/v1/knowledge-bases/:id/knowledge/file

Amaç: Dosya yükleyerek bilgi oluşturmak. Yetki: KB oluşturucusu VEYA Admin+, KB yazma; API anahtarı `ingest`/full. Handler: `internal/handler/knowledge.go`

multipart/form-data alanları:

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `file` | file | Evet | Yüklenen dosya |
| `fileName` | string | Hayır | Görünen adı geçersiz kılar |
| `metadata` | JSON dizesi | Hayır | Özel meta veriler |
| `enable_multimodel` | bool | Hayır | Çok modlu işleme anahtarı |
| `tag_ids` | string | Hayır | Virgülle ayrılmış etiket ID'leri |
| `channel` | string | Hayır | Alım kanalı |
| `process_config` | JSON dizesi | Hayır | Ayrıştırma yapılandırması geçersiz kılmaları (KnowledgeProcessOverrides), aşağıdaki tabloya bakın |

`process_config` yaygın alanları (tümü isteğe bağlıdır; atlanırsa bilgi tabanı yapılandırması kullanılır):

| Alan | Tür | Varsayılan | Açıklama |
| --- | --- | --- | --- |
| `summary_enabled` | bool | true | Bu içe aktarmadaki belgeler için özet oluşturulup oluşturulmayacağı; kapatıldıktan sonra ayrıştırma, dizinleme ve diğer işlemler normal şekilde yürütülür |
| `parser_engine_rules` | []object | Bilgi tabanı yapılandırması | Dosya türüne göre ayrıştırma motorunu belirtir |
| `parser_engine_overrides` | map[string]string | Boş | `pdf_force_scanned` gibi motor parametreleri |
| `chunking_config` | object | Bilgi tabanı yapılandırması | Parçalama parametreleri |
| `enable_multimodel` / `vlm_config` / `asr_config` | - | Bilgi tabanı yapılandırması | Çok modlu ve ses tanıma |
| `question_generation_config` | object | Bilgi tabanı yapılandırması | Soru oluşturma |
| `graph_enabled` / `extract_config` | - | Bilgi tabanı yapılandırması | Grafik çıkarma |

Yanıt: 200 `{"success":true,"data":{Knowledge}}`; yinelenen dosyalar 409 döndürür ve `data` mevcut Knowledge olur. Silinmekte olan veya ayrıştırması başarısız olan aynı adlı dosyalar yinelenen sayılmaz.

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/file \
  -H "X-API-Key: $API_KEY" -F 'file=@./manual.pdf' -F 'enable_multimodel=true'
```

### POST /api/v1/knowledge-bases/:id/knowledge/url

Amaç: URL'den çekerek bilgi oluşturmak. Yetkiler/API anahtarı yukarıdakiyle aynıdır.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `url` | string | Evet (`binding:"required"`) | Çekme adresi |
| `file_name` / `file_type` / `title` | string | Hayır | Bilgileri geçersiz kılar |
| `enable_multimodel` | *bool | Hayır | Çok modlu anahtar |
| `tag_ids` | []string | Hayır | Etiketler |
| `channel` | string | Hayır | Kanal |
| `process_config` | object | Hayır | Ayrıştırma geçersiz kılma |

Yanıt: 201 `{"success":true,"data":{Knowledge}}`; yinelenen URL 409 döndürür.

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/url -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"url":"https://example.com/doc"}'
```

### POST /api/v1/knowledge-bases/:id/knowledge/manual

Amaç: Elle oluşturulan (Markdown) bilgi oluşturmak. Yetkiler/API anahtarı yukarıdakiyle aynıdır.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `title` | string | Hayır | Başlık |
| `content` | string | Hayır | Markdown içeriği |
| `status` | string | Hayır | `draft` / `publish` |
| `tag_ids` | []string | Hayır | Etiketler |
| `channel` | string | Hayır | Kanal |
| `process_config` | object | Hayır | Ayrıştırma geçersiz kılma |

Yanıt: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/manual -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"SSS özeti","content":"# İçerik","status":"publish"}'
```

### GET /api/v1/knowledge-bases/:id/knowledge

Amaç: KB altındaki bilgi listesi. Yetki: Viewer+, KB okuma; API anahtarı `retrieve`/full.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `page` / `page_size` | int | Hayır | Sayfalama (varsayılan 1/20) |
| `tag_ids` | string | Hayır | Virgülle ayrılmış etiketler (OR) |
| `keyword` | string | Hayır | Anahtar kelime |
| `file_type` | string | Hayır | Dosya türü filtresi |
| `parse_status` | string | Hayır | `pending/processing/completed/failed` |
| `source` | string | Hayır | Kanal veya `manual`/`url` |
| `start_time` / `end_time` | string | Hayır | RFC3339, `updated_at` temel alınarak filtrelenir |
| `folder_path` | string | Hayır | Klasöre göre filtreleme; boş dize bilgi tabanı kök dizinini belirtir, gönderilmezse klasöre göre filtrelenmez |
| `folder_recursive` | bool | Hayır | `folder_path` ile birlikte kullanılır; `true` olduğunda alt klasörlerdeki belgeleri içerir |
| `sort_by` | string | Hayır | Sıralama alanı: `updated_at`, `created_at` veya `file_name`; varsayılan `created_at` |
| `sort_order` | string | Hayır | Sıralama yönü: `asc` veya `desc`; varsayılan `desc` |

Sıralama parametreleri gönderilmediğinde `created_at desc` ile sıralanır; değer yukarıdaki aralığın dışındaysa 400 döndürülür. `updated_at` kullanıldığında yeniden ayrıştırma, düzenleme veya durum değişiklikleri sıralamayı etkiler; `file_name` kullanıldığında görüntülenen dosya adına göre büyük/küçük harf duyarsız sıralanır, dosya adı boşsa sırasıyla başlığa ve kaynağa geri dönülür. Aynı sıralama değerleri bilgi ID'sine göre sıralanarak sayfalama sonuçlarının kararlı olması sağlanır.

Yanıt: 200 `{"success":true,"data":[Knowledge],"total","page","page_size"}`

```bash
curl "$BASE/api/v1/knowledge-bases/kb-1/knowledge?page=1&parse_status=completed" -H "X-API-Key: $API_KEY"
```

### POST /api/v1/knowledge-bases/:id/knowledge/batch-download

Amaç: Aynı bilgi tabanındaki birden fazla belgenin özgün dosyalarını ZIP olarak paketleyip indirmek. İzinler tek dosya indirme ile aynıdır: Contributor+ ve KB write (kuruluş paylaşımındaki Viewer indiremez); API anahtarı `retrieve`/full.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `ids` | []string | Evet | Bilgi ID listesi, 1-200 adet |

Davranış:

- Özgün dosyaların toplamı 512 MiB'ı aşamaz; aşılırsa 400 döndürülür;
- Özgün dosyası olmayan öğeler (web sayfası içe aktarma gibi) atlanır; seçilen öğelerin hiçbirinde özgün dosya yoksa 400 döndürülür;
- ZIP içinde bilgi tabanı klasör yapısı korunur, aynı adlı dosyalara otomatik olarak sıra numarası eklenir;
- Herhangi bir ID mevcut değilse veya bu bilgi tabanına ait değilse 404 döndürülür, okuma başarısız olursa 500 döndürülür; eksik dosyalı bir arşiv oluşturulmaz;
- Aynı örnek aynı anda en fazla 4 toplu indirme işleyebilir; aşılırsa 429 döndürülür.

Yanıt: 200 `application/zip` dosya akışı; dosya adı `knowledge-files-20260923-150405.zip` biçimindedir.

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/batch-download \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"ids":["k-1","k-2"]}' -o knowledge-files.zip
```

### GET /api/v1/knowledge-bases/:id/knowledge/folders

Amaç: Bilgi tabanının klasör dizin ağacını almak. Tüm dizin yüklendiğinde dizin yapısı korunur (`000079` migration'ından itibaren `knowledges.folder_path` sütunu bulunur; geçmiş `file_name` içindeki yollar bu alana geri doldurulmuştur). İzinler: Viewer+ + KBAccessRead.

Yanıt: 200 `{"success":true,"data":[{FolderNode}]}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/knowledge/folders -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id/knowledge/folders

Amaç: Bir klasörü tüm alt dizinleriyle birlikte yeniden adlandırmak veya taşımak. Hedef yol mevcutsa iki klasör birleştirilir; kendi alt dizinine taşınmasına izin verilmez. Yetki: KB owner veya Admin+ + KBAccessWrite.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `from` | string | Evet | Eski yol |
| `to` | string | Evet | Yeni yol |

Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/knowledge/folders -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"from":"Tasarım belgeleri/Eski sürüm","to":"Arşiv/Tasarım belgeleri"}'
```

### DELETE /api/v1/knowledge-bases/:id/knowledge

Amaç: Tüm KB içeriğini temizlemek (yıkıcı). Yetki: Admin+, KB write; API key yalnızca full-access.

Yanıt: 200 `{"success":true,"message":"Knowledge base contents clear task submitted","data":{"deleted_count":N}}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1/knowledge -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/batch

Amaç: Bilgileri ID'ye göre toplu almak (KB'ler arası, handler erişimi kendisi doğrular). Yetki: Viewer+; API key `retrieve`/full.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `ids` | []string | Evet | Bilgi ID'si (parametre tekrarlanabilir veya virgülle ayrılabilir) |
| `kb_id` | string | Hayır | KB ile sınırlandırır |
| `agent_id` | string | Hayır | Paylaşılan Agent kapsamı |
| `agent_source_tenant_id` | uint64 | Hayır | Paylaşılan Agent için kaynak alan seçicisi, paylaşım ilişkisi doğrulamasıyla birlikte |

Yanıt: 200 `{"success":true,"data":[Knowledge]}`. `pending`/`processing`/`finalizing` durumundaki bilgiler ayrıca `last_activity_at` (RFC3339) içerir; satırdaki `updated_at` ile bu bilginin tüm span'lerindeki en son yazımdan daha yeni olanı alınır. 20 dakikadan uzun süredir ilerleme olmayan bilgiler ayrıca `stall_state` içerir: `queued`, asynq kuyruğunda veya Wiki kalıcı kuyruğunda hâlâ bekleyen görevler olduğunu (birikme) belirtir; `stalled`, onu ilerletecek görev kalmadığını (muhtemelen takıldı) belirtir. Belirleme, housekeeping'in birikme belirlemesiyle aynıdır; kuyruk tarafında tüm kuyruk bir kez taranır, tüm istekler paylaşır ve 60 saniye önbelleğe alınır. Algılama başarısız olursa `stall_state` döndürülmez; ön yüz bunu normal ayrıştırma sırasında gösterir.

```bash
curl "$BASE/api/v1/knowledge/batch?ids=k-1&ids=k-2" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id

Amaç: Bilgi ayrıntıları. Yetki: Viewer+, üst KB read.

Yanıt: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl $BASE/api/v1/knowledge/k-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id/stages ve GET /api/v1/knowledge/:id/spans

Amaç: Ayrıştırma aşamaları/trace (iki yol aynı handler'ı, `GetKnowledgeSpans`, kullanır). Yetki: Viewer+, üst KB read. Sorgu parametresi: `attempt` (int, 0=en güncel deneme).

Yanıt: 200 `{"success":true,"data":{"knowledge_id","attempt","latest_attempt","parse_status","current_stage","last_activity_at","stall_state","trace":{...},"last_error":{...}}}`

`last_activity_at` yalnızca ayrıştırma sürerken döndürülür; satırdaki `updated_at` ile bu attempt'teki her span'in en son yazımından daha yeni olanı alınır; `stall_state` anlamı yukarıdakiyle aynıdır. `current_stage`, hâlâ çalışan aşamadır; çalışan aşama olmadığında (örneğin `finalizing` durumunda, son işleme aşaması kapanmış ancak özet gibi alt görevler hâlâ çalışıyorken), hâlâ çalışan alt span'in ait olduğu aşama alınır. Housekeeping tarafından takılmış olarak belirlenen bilgilerin takıldığı konumdaki span'i `TASK_STALLED` ile başarısız olarak işaretlenir ve `last_error` öncelikle onu gösterir.

```bash
curl $BASE/api/v1/knowledge/k-1/spans -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/knowledge/:id

Amaç: Bilgiyi silmek (eşzamansız). Yetki: KB oluşturucusu VEYA Admin+, KB write; API key `ingest`/full.

Yanıt: 200 `{"success":true,"message":"Delete task submitted","data":{"task_id"}}`

```bash
curl -X DELETE $BASE/api/v1/knowledge/k-1 -H "X-API-Key: $API_KEY"
```

### PUT /api/v1/knowledge/:id

Amaç: Bilgi meta verisini güncellemek. Yetki yukarıdakiyle aynıdır. İstek gövdesi (`types.Knowledge` alt kümesi): `title`, `description`, `tags`, `custom_metadata` (hepsi isteğe bağlı). description atlanırsa mevcut özet korunur, açıkça boş dize verilirse özet temizlenir, boş olmayan değer manuel özet olarak kaydedilir; arayüzde belge içerik sayfasından düzenlenebilir.

`custom_metadata`, kullanıcı tarafından girilen açıklayıcı meta veridir (sistem içi kullanım için olan `metadata` ile ayrı saklanır, migration `000078`); doğrulama kuralları için bkz. `internal/application/service/knowledge.go`:

| Kısıtlama | Değer |
| --- | --- |
| Alan sayısı | ≤ 20 |
| Anahtar uzunluğu | 1-64 karakter, boş olamaz |
| Değer türü | string / number / boolean / null |
| Değer uzunluğu | ≤ 1000 karakter |

Tamamen üzerine yazarak güncelleme (iletilen nesne, mevcut nesnenin yerini alır). Meta veriler değiştiğinde ve bu belgenin zaten bir özeti varsa, otomatik olarak bir özet yenileme işlemi kuyruğa eklenir (`summary_status` `pending` durumuna geçer). Meta veri metni, özet oluşturma ve belge düzeyindeki model bağlamına katılır (`Knowledge.CustomMetadataText()`).

Yanıt: 200 `{"success":true,"message":"Knowledge updated successfully","data":{Knowledge}}`

```bash
curl -X PUT $BASE/api/v1/knowledge/k-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Yeni başlık","custom_metadata":{"Departman":"Ar-Ge merkezi","Gizlilik":"Dahili","Sürüm":3}}'
```

### POST /api/v1/knowledge/:id/regenerate-summary

Amaç: Parça içeriği veya özel meta veriler düzenlendikten sonra bu belgenin özetini yeniden oluşturmak. Yetki: KB owner veya Admin+ ve üst KB için write yetkisi.

Davranış ikiye ayrılır: Belgenin daha önce özeti yoksa (`summary_status` boş veya `none`) eşzamanlı olarak bir oluşturma işlemi tetiklenir; özeti zaten varsa bunun yerine bir yenileme görevi kuyruğa eklenir, `summary_status` `pending` durumuna geçer ve `knowledge_summary_refresh.go` tarafından eşzamansız olarak çalıştırılır.

Yanıt: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge/k-1/regenerate-summary -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge/manual/:id

Amaç: Manuel bilgi içeriğini güncellemek (`ManualKnowledgePayload` alt kümesi: `title/content/status/...`). Yetki yukarıdakiyle aynıdır.

Yanıt: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl -X PUT $BASE/api/v1/knowledge/manual/k-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"content":"# Güncellenmiş içerik","status":"publish"}'
```

### POST /api/v1/knowledge/:id/reparse

Amaç: Bilgiyi yeniden ayrıştırmak. Yetki yukarıdakiyle aynıdır. İstek gövdesi (isteğe bağlı): `{"process_config":{...}}`.

Yanıt: 200 `{"success":true,"message":"Reparse task submitted","data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge/k-1/reparse -H "X-API-Key: $API_KEY"
```

### POST /api/v1/knowledge/:id/cancel-parse

Amaç: Ayrıştırmayı iptal etmek. Yetki yukarıdakiyle aynıdır. İstek gövdesi yoktur.

Yanıt: 200 `{"success":true,"message":"Knowledge parse cancelled","data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge/k-1/cancel-parse -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id/download

Amaç: Orijinal kaynak dosyasını indirmek (önizlemeden daha katıdır: Contributor+ ve KB write; kuruluş tarafından paylaşılan Viewer kaynak dosyayı indiremez). API key `retrieve`/full.

Yanıt: 200 ikili akış (`application/octet-stream`).

```bash
curl -OJ $BASE/api/v1/knowledge/k-1/download -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id/preview

Amaç: Ayrıştırılmış dosya içeriğini önizlemek. Yetki: Viewer+, KB read.

Yanıt: 200 önizleme akışı (metin/HTML).

```bash
curl $BASE/api/v1/knowledge/k-1/preview -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge/image/:id/:chunk_id

Amaç: Bir parçanın görsel bilgilerini güncellemek (caption/OCR vb.). Yetki: KB oluşturucusu OR Admin+, KB write. Yol parametreleri: `id` bilgi ID'si, `chunk_id` parça ID'si. İstek gövdesi görsel bilgisi JSON'udur.

Yanıt: 200 `{"success":true,...}`

```bash
curl -X PUT $BASE/api/v1/knowledge/image/k-1/c-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"caption":"Mimari diyagramı"}'
```

### GET /api/v1/knowledge/search

Amaç: KB'ler arası dosya araması (oturum @dosya seçicisi). Yetki: Viewer+; API key `retrieve`/full.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `keyword` / `query` | string | Koşullu zorunlu | Anahtar kelime (ikisi eşdeğerdir); boşsa `recent=true` iletilmelidir, aksi halde 400 döner |
| `file_types` | string | Hayır | Virgülle ayrılmış uzantı filtresi, örneğin `csv,xlsx` |
| `offset` / `limit` | int | Hayır | Sayfalama; `limit` varsayılan olarak 20, aralık 1–100 |
| `recent` | bool | Hayır | Anahtar kelime boş olduğunda en son dosyaları döndürür |
| `agent_id` | string | Hayır | Paylaşılan Agent kapsamı |
| `agent_source_tenant_id` | uint64 | Hayır | Paylaşılan Agent için kaynak alan seçicisi, paylaşım ilişkisini doğrular |

Yanıt: 200 `{"success":true,"data":[Knowledge],"has_more":bool,"total":N}`

```bash
curl "$BASE/api/v1/knowledge/search?keyword=rapor&limit=20" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/move/progress/:task_id

Amaç: Taşıma görevi ilerlemesini sorgulamak. Yetki: Viewer+; API anahtarı `retrieve`/full.

Yanıt: 200 `{"success":true,"data":{MoveProgress}}`

```bash
curl $BASE/api/v1/knowledge/move/progress/task-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge/tags

Amaç: Bilgi etiketlerini toplu güncellemek. Yetki: Contributor+; API anahtarı `ingest`/full (KB izin listesi handler içinde doğrulanır).

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `updates` | map[string][]string | Evet (`binding:"required,min=1"`) | knowledge_id → tag_ids |
| `kb_id` | string | Hayır | KB ile sınırla |

Yanıt: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge/tags -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"updates":{"k-1":["t-1"]},"kb_id":"kb-1"}'
```

### POST /api/v1/knowledge/batch-reparse

Amaç: Toplu yeniden ayrıştırma. Yetki: Contributor+; API anahtarı `ingest`/full.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `kb_id` | string | Evet (`binding:"required"`) | KB kimliği |
| `ids` | []string | Evet (`binding:"required"`) | Bilgi kimliği listesi |
| `process_config` | object | Hayır | Ayrıştırma geçersiz kılma ayarları |

Yanıt: 200 `{"success":true,"message":"Batch reparse task submitted","data":{"task_id"}}`

```bash
curl -X POST $BASE/api/v1/knowledge/batch-reparse -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"kb_id":"kb-1","ids":["k-1","k-2"]}'
```

### POST /api/v1/knowledge/batch-delete

Amaç: Toplu silme (≤200 öğe). Yetki: Contributor+; API anahtarı `ingest`/full.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `kb_id` | string | Evet (`binding:"required"`) | KB kimliği |
| `ids` | []string | Evet (`binding:"required"`) | Bilgi kimliği listesi (≤200) |

Yanıt: 200 `{"success":true,"message":"Batch delete task submitted","data":{"task_id","deleted_count"}}`

```bash
curl -X POST $BASE/api/v1/knowledge/batch-delete -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"kb_id":"kb-1","ids":["k-1"]}'
```

### POST /api/v1/knowledge/folder

Amaç: Birden çok belgeyi belirtilen klasörde sınıflandırmak (yalnızca sınıflandırmayı değiştirir, bilgi tabanı üyeliğini değiştirmez ve yeniden ayrıştırmaz). Yetki: Contributor+ / API anahtarı `ingest`.

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `kb_id` | string | Evet | Bilgi tabanı ID'si |
| `knowledge_ids` | []string | Evet | Taşınacak belgeler |
| `folder_path` | string | Hayır | Hedef klasör; boş dize, bilgi tabanı kök dizinine geri taşımayı belirtir |

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/knowledge/folder -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"kb_id":"kb-1","knowledge_ids":["k-1","k-2"],"folder_path":"Tasarım belgeleri"}'
```

### POST /api/v1/knowledge/move

Kullanım: KB'ler arasında bilgi taşıma (eşzamansız). Yetki: Contributor+; API anahtarı `ingest`/full (kaynak ve hedef KB'nin her ikisi de izin listesinde olmalıdır).

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `knowledge_ids` | []string | Evet (`binding:"required,min=1"`) | Taşınacak bilgiler |
| `source_kb_id` | string | Evet (`binding:"required"`) | Kaynak KB |
| `target_kb_id` | string | Evet (`binding:"required"`) | Hedef KB |
| `mode` | string | Evet (`binding:"required,oneof=reuse_vectors reparse"`) | Vektörleri yeniden kullan veya yeniden ayrıştır |

Yanıt: 200 `{"success":true,"data":{"task_id","source_kb_id","target_kb_id","knowledge_count","message"}}`

```bash
curl -X POST $BASE/api/v1/knowledge/move -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"knowledge_ids":["k-1"],"source_kb_id":"kb-1","target_kb_id":"kb-2","mode":"reuse_vectors"}'
```

## Uygulama referansı

Rota kaydı: `internal/router/routes_knowledge.go` içindeki `RegisterKnowledgeBaseRoutes`, `RegisterKnowledgeRoutes`. Handler: `internal/handler/knowledgebase.go`, `internal/handler/knowledge.go`.
