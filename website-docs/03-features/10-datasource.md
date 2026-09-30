# Veri Kaynağı İçe Aktarma (Data Source)

Veri kaynakları Feishu, Notion, Confluence, Yuque gibi platformlardaki içeriği sürekli olarak bilgi tabanına eşitler. Bağlantı kurulduktan sonra yeni ve değiştirilen içerik plana göre alınabilir; kaynakta silinen içerik eşitleme yapılandırmasına göre işlenir.

Veri kaynakları bilgi tabanında yapılandırılır. Hedef bilgi tabanının düzenleme ayarlarını açın, «Veri kaynağı» sekmesine girin, yeni bir bağlantı oluşturup kimlik bilgilerini girin, ardından eşitleme kapsamını ve periyodunu seçin. İlk eşitleme içeriğin tamamını alır, sonraki eşitlemeler bağlayıcının yeteneğine göre artımlı olarak güncellenir.

<Screenshot
  src="/screenshots/datasource-sync.png"
  caption="Veri kaynağı: bağlantı listesi ve eşitleme durumu"
  hint="Yapılandırılmış veri kaynaklarını (tür, hedef bilgi tabanı, son eşitleme zamanı, durum) ve eşitleme günlüğü girişini gösterir." />

Bağlayıcı harici içeriği okur, zamanlayıcı eşitlemeyi tetikler, hizmet katmanı değişiklik karşılaştırmasını ve bilgi tabanına aktarmayı tamamlar.

## Bağlantı Kurma ve Eşitleme

1. Alan yöneticisi olarak hedef bilgi tabanının «Veri kaynağı» ayarlarını açın.
2. Bir bağlayıcı seçin, kimlik bilgilerini girip bağlantıyı test edin ve gereken kaynakların listelenebildiğini doğrulayın.
3. Eşitleme kapsamını ve periyodunu seçin, kaydettikten sonra ilk eşitlemeyi başlatın.
4. Eşitleme durumunu ve günlüklerini inceleyerek eklenen, güncellenen, atlanan ve başarısız içeriklerin beklendiği gibi olduğunu doğrulayın.

Kimlik bilgilerini güncellemek için yapılandırmanın tamamı gönderilmelidir; sistem bunu çevrimiçi doğrular. Veri kaynağını duraklatmak sonraki planlı eşitlemeleri durdurur; devam ettirildiğinde zamanlama yeniden kaydedilir.

`sync_schedule` saniyeyi de içeren altı alanlı cron kullanır; örneğin `0 0 */6 * * *` her altı saatte bir tetikleme anlamına gelir; boş dize yalnızca elle eşitleme demektir. Veri kaynağı oluşturulurken, düzenlenirken ve devam ettirilirken ifade doğrulanır; geçersiz değerler HTTP 400 döndürür ve kayıtlı yapılandırmayı veya mevcut zamanlanmış görevleri değiştirmez. Geçmişte geçersiz ifade kaydedilmiş bir veri kaynağını devam ettirmeden önce ifadeyi geçerli bir değerle değiştirin veya temizleyin.

## Bağlayıcı Seçme

| Bağlayıcı | Tür kimliği | Eşitlenen nesne |
| --- | --- | --- |
| Feishu / Lark bilgi tabanı | `feishu` / `lark` | Wiki alanları ve düğümleri |
| Feishu Drive / Lark Drive | `feishu_drive` / `lark_drive` | Belirtilen drive klasörleri, bkz. [Feishu Drive entegrasyonu](24-feishu-drive.md) |
| Notion | `notion` | Sayfalar ve veritabanları |
| Confluence | `confluence` | Server / Data Center veya Cloud alanlarındaki sayfalar |
| Yuque | `yuque` | Bilgi tabanı belgeleri |
| DingTalk Docs | `dingtalk` | Bilgi tabanları, klasörler ve çevrimiçi belgeler |
| Tencent IMA | `ima` | Bilgi tabanındaki dosyalar ve notlar |
| GitLab | `gitlab` | Deponun belirtilen dal/etiketi altındaki dizinler |
| RSS / Atom | `rss` | Abonelik kaynağı makaleleri |

Her bağlayıcının desteklediği biçimler, kimlik doğrulama ve silme algılama için başvuru bölümüne bakın.

## Değişiklikleri ve Hataları İnceleme

İlk eşitleme seçilen kapsamdaki içeriği alır, sonraki eşitlemeler bağlayıcı imleci ve değişiklik bilgilerine göre güncellenir. Silme algılama bağlayıcıya ve eşitleme yapılandırmasına bağlıdır; RSS'nin doğal olarak eski girdileri düşürmesi kaynak belgenin silinmesi sayılmaz. Eşitleme başarısız olursa önce günlükteki kimlik bilgisi, kaynak görünürlüğü veya ayrıştırma hatalarını inceleyin, ardından bağlantıyı test edip yeniden deneyin.

## Bağlayıcı ve Arayüz Başvurusu

### Bağlayıcı Uygulama Ayrıntıları

#### Bağlayıcı Yetenek Karşılaştırması

| | Feishu / Lark | Notion | Yuque | RSS / Atom |
| --- | --- | --- | --- | --- |
| Kaynak kod dizini | `internal/datasource/connector/feishu/wiki/` (ortak `feishu/core/`) | `connector/notion/` | `connector/yuque/` | `connector/rss/` |
| Tür kimliği | `feishu` / `lark` | `notion` | `yuque` | `rss` |
| Kimlik doğrulama | Kurum içi uygulama `app_id` + `app_secret` (tenant_access_token) | Internal Integration Token (`api_key`) | Kişisel/ekip Token'ı (`api_token`, `X-Auth-Token` başlığı) | Kimlik doğrulama yok veya özel istek başlıkları (`auth_headers`) |
| Kimlik bilgisi alanları | `app_id`, `app_secret`, `base_url` (isteğe bağlı geçersiz kılma) | `api_key` (`base_url` Settings üzerinden) | `api_token`, `base_url` (özel dağıtımda isteğe bağlı) | `auth_headers` (isteğe bağlı, kimlik bilgisidir); `feed_urls` Settings'e aittir |
| Kaynak modeli | Wiki alanı → düğüm ağacı (tembel yükleme, `spaceID:nodeToken` bileşik ID) | Sayfa/veritabanı tam ağacı (parent ilişkisiyle tek seferde döner) | Bilgi tabanı (book/repo) düz listesi | Her feed URL'si için bir kaynak (düz) |
| İçerik biçimi | Dışa aktarma API'si → `.docx`/`.xlsx` dosyası; drive dosyaları olduğu gibi indirilir | Block → Markdown; veritabanı Markdown tablosuna dönüştürülür; ekler indirilir | `body` özgün Markdown (`.md`) | Readability ile tam metin çıkarma → HTML→Markdown |
| Artımlı mekanizma | İçeriğin `obj_edit_time` değeri karşılaştırılır (cursor: `SpaceNodeTimes`) | Sayfa/kaydın `last_edited_time` değeri karşılaştırılır (cursor: `PageEditTimes`) | Belgenin `content_updated_at` değeri karşılaştırılır (cursor: `BookDocTimes`) | feed sinyal parmak izi + içerik SHA-256 parmak izi ile çift katmanlı karşılaştırma |
| Silme algılama | Desteklenir (imleçte var, mevcut ağaçta yok → `IsDeleted`; listeleme kısmen başarısız olursa silme algılama atlanır) | Desteklenir ("kaynakta silindi" ile "kullanıcı seçimi kaldırdı" ayrılır, ikincisi silme olarak bildirilmez) | Desteklenir | Desteklenmez (feed eski girdileri doğal olarak kaydırarak düşürür) |
| Akışlı, devam ettirilebilir eşitleme | Evet (`StreamingConnector`, her 50 düğümde veya 30 saniyede bir checkpoint) | Hayır | Hayır | Hayır |
| Hız sınırına karşı önlem | 429'da `Retry-After` okunur + üstel geri çekilme (2s/4s/8s, en fazla 3 yeniden deneme); 5xx yeniden denenir | — | Her `GetDocDetail` arasında 300ms (kişisel token yaklaşık 100 istek/5dk) | — |
| Kısmi hata | Tek belge başarısız olursa hata metadata'lı yer tutucu girdi üretilir, eşitleme sürer | Tek sayfa hatası günlüğe yazılıp atlanır | Tek belge başarısız olursa yer tutucu girdi üretilir | Tek feed hatası → `PartialFetchError`; yalnızca tümü başarısız olursa fail sayılır |

#### Feishu / Lark (`connector/feishu/wiki/`)

Drive uygulama izinleri, klasör yetkilendirmesi, kaynak seçimi ve `FEISHU_DOCX_PARSE_MODE` tercihleri için [Feishu Drive entegrasyonu](24-feishu-drive.md) belgesine bakın. Varsayılan export ve blocks modlarında görsel ve eklerin anlamı farklıdır; eşitlemeli silme açıldığında mevcut veri kaynağına karşılık gelen bilgi girdileri silinir.

Feishu ve Lark (uluslararası sürüm open.larksuite.com) iki yalıtılmış bulutta dağıtılan aynı üründür; Wiki/docx/drive API'leri tamamen aynıdır, bu nedenle **aynı bağlayıcı kodunu paylaşırlar**. Bulut, `feishu/core/region.go` içindeki `Region` yapısıyla seçilir (`RegionFeishu` / `RegionLark`; sırasıyla `feishu` / `lark` türlerine ve `open.feishu.cn` / `open.larksuite.com` API alan adlarına karşılık gelir). `base_url` kimlik bilgisi alanı bunu açıkça geçersiz kılabilir (geçmişte feishu bağlayıcısını larksuite'e yönlendiren mevcut veri kaynaklarıyla uyumluluk için).

- **Kimlik doğrulama** (`core/client.go`): `POST /open-apis/auth/v3/tenant_access_token/internal` ile tenant_access_token alınır; karşılıklı dışlama kilidiyle önbelleğe alınır ve süresi dolunca yenilenir.
- **Kaynak listeleme** (`ListResources`): Üç düzeyli tembel yükleme: `parentID==""` Wiki alanlarını; `parentID==spaceID` alanın üst düzey düğümlerini; `parentID=="spaceID:nodeToken"` o düğümün alt düğümlerini listeler. Eski sürümler tüm ağacı önceden özyinelemeli olarak dolaşıyordu ve büyük Wiki'lerde zaman aşımı oluşuyordu (issue #1672); artık özyineleme yalnızca eşitleme sırasında yapılır. `ResolveResourceAncestors`, `GetWikiNode` çağrısının `parent_node_token` değeriyle düzey düzey yukarı çıkarak derin seçimleri O(depth) maliyetle geri gösterir.
- **İçerik alma** (`fetchNodeContent`) `obj_type` değerine göre dağıtılır:
  - `docx`/`doc` → eşzamansız dışa aktarma API'si (`POST /drive/v1/export_tasks`) ile `.docx` dışa aktarılır;
  - `sheet`/`bitable` → `.xlsx` dışa aktarılır;
  - `file` → drive özgün dosyası indirilir (PDF/Word/görsel vb.);
  - `mindnote`/`slides` → **atlanır** (içerik okuma API'si yok); `fetchTally` ile `discovered/fetched/failed/skipped_unsupported by_type` özet günlüğü üretilerek "13 belge bulundu ama neden yalnızca 3'ü eşitlendi" sorusu açıklanır (issue #2136).
- **Artımlı mantık**: İmleç `FeishuCursor.SpaceNodeTimes` (`resourceID → nodeToken → editTime`). Değişiklik kararı `obj_edit_time` (belge içeriğinin düzenlenme zamanı) ile verilir, `node_edit_time` ile **değil** (o yalnızca başlık değişikliği/konum taşımayı yansıtır). Alınamayan düğümlerde **imleç ilerletilmez** (eski editTime korunur, bir sonraki seferde prev != current olacağı için yeniden denenir); böylece anlık dışa aktarma hataları belgenin kalıcı olarak atlanmasına yol açmaz.
- **FetchStream**: Tam/artımlı yolları birleştirir (cursor==nil tam eşitleme demektir); her `FeishuStreamCheckpointInterval = 50` düğüm işlendiğinde veya son checkpoint'ten bu yana `FeishuStreamCheckpointMaxInterval = 30s` geçtiğinde imleç diske yazılır. İkinci koşul, "az sayıda belge var ama her birinin dışa aktarımı çok yavaş (hız sınırına takılmış)" olduğu için 2 saatlik zaman aşımından önce hiç checkpoint alınamayan durumu güvenceye alır.
- **Hata sınıflandırma** (`feishuFailure`): Ham hataları kararlı i18n kodlarına sınıflandırır (`feishu_auth_or_permission` / `feishu_rate_limited` / `feishu_timeout` / `feishu_server_unavailable` / `feishu_api_error`(+code) / `sync_failed`); ön uç bunları yerelleştirerek gösterir. Ham status/body/log_id yalnızca sunucu günlüğünde kalır.

#### GitLab (`connector/gitlab/`)

Veri kaynağında GitLab'ı seçin, credentials.base_url ve access_token alanlarını doldurun, ardından projeyi, dalı veya etiketi ve dizini seçin. Token seçilen projenin deposunu okuyabilmelidir; özel projelerin görünürlüğü GitLab kimlik bilgisine bağlıdır.

`config.settings.projects` boş olmayan bir dizidir; her öğe metin türünde project_id, isteğe bağlı ref ve paths içerir. ref boş bırakılırsa varsayılan dal kullanılır; paths boş bırakılırsa tüm depo seçilir, dizinler göreli yol ve eğik çizgi (/) ile yazılır. Zamanlanmış eşitlemeyi kaydetmeden önce kimlik bilgilerini doğrulayıp kaynaklara göz atın.

Akışlı eşitleme checkpoint'ten devam etmeyi destekler; artımlı eşitleme dosyaları depo commit farklarıyla günceller, kaynaktaki silmeler sync_deletions ayarına göre işlenir. Seçilen depo dosyaları yine Rethra'nın dosya türü, boyut ve ayrıştırma motoru denetimlerinden geçer; her kod veya ikili dosya doğrudan içe aktarılamaz. Farklı yollarda içeriği tamamen aynı olan dosyalar (örneğin alt dizinlerdeki README şablonları) ayrı bilgi girdileri olarak korunur; yinelenme kararı yalnızca aynı veri kaynağı ve aynı dosya yolu içinde geçerlidir.

```json
{"credentials":{"base_url":"https://gitlab.example.com","access_token":"<token>"},"settings":{"projects":[{"project_id":"123","ref":"main","paths":["docs"]}]}}
```

#### Tencent IMA (`connector/ima/`)

credentials.client_id ve api_key alanlarını doldurun; base_url isteğe bağlıdır, varsayılanı `https://ima.qq.com`. Kaynak listesi bu kimlik bilgisiyle görülebilen IMA bilgi tabanlarıdır (düz liste, dizinler açılmaz); seçim config.resource_ids olarak kaydedilir. Eşitleme sırasında seçilen bilgi tabanlarındaki tüm alt klasörler özyinelemeli olarak dolaşılır. Kaynakta yetkilendirme başarısız olursa veya kaynak görünmüyorsa önce IMA kimlik bilgilerini ve bilgi tabanı erişim iznini kontrol edin.

İndirilebilir dosyalar belge ayrıştırmaya girer, web sayfası türleri URL üzerinden alınır; notların metni note OpenAPI ile okunur. AI oturumları ve video ayrıştırmaları için kullanılabilir bir metin okuma girişi olmadığından atlanır. Tam ve artımlı eşitleme desteklenir: bilgi tabanı, üst dizin ve başlığa göre kararlı bir kimlik oluşturulur; aynı adlı dosya değiştirildiğinde media_id değişir ve güncelleme tetiklenir; silme algılaması yalnızca tam listeleme başarılı olduktan sonra yapılır.

```json
{"credentials":{"client_id":"<client-id>","api_key":"<api-key>"},"resource_ids":["<resource-id-from-tree>"]}
```

Feishu/Lark eşitleme kayıtlarının güncellenme zamanı içerik düzenleme zamanından alınır; böylece yalnızca Wiki düğüm işlem zamanına bakılarak metin değişikliklerinin kaçırılması önlenir. GitLab ve IMA silme algılamayı destekler; RSS'nin doğal kaydırmayla eski girdileri düşürmesi silme sayılmaz.

#### Notion (`connector/notion/`)

- **Kimlik doğrulama**: Internal Integration Token (kimlik bilgisi alanı `api_key`), API sürümü `NotionAPIVersion = "2026-03-11"`, varsayılan `https://api.notion.com` (`Settings.base_url` ile geçersiz kılınabilir).
- **Kaynak listeleme**: Search API tüm görünür sayfaları ve veritabanlarını tek seferde çeker ve `ParentID` içeren tam ağacı döndürür (bu nedenle `parentID != ""` olan tembel yükleme istekleri doğrudan boş döner; `ResolveResourceAncestors` için de ek işlem gerekmez). `resolveParentID`, 2025-09-03+ API'sinin `data_source` nesnesini işler: onun `parent` değeri veritabanı kapsayıcısını gösterir, gerçek çalışma alanı konumu için `database_parent` değerine bakılmalıdır.
- **Alma**: `fetchPage` sayfaları özyinelemeli işler: `GetBlockChildrenAll` blokları çeker → `BlocksToMarkdown` (`markdown.go`) Markdown'a dönüştürür; `file_upload` türündeki dosya blokları önce `ResolveBlock` ile geçici indirme URL'sine çevrilir; **ekler** (PDF vb.; görseller hariç, çünkü görseller Markdown içinde `![](url)` olarak satır içindedir) ayrı girdiler olarak indirilip içe aktarılır; `child_page` / `child_database` blokları özyinelemeli olarak derinleştirilir. Veritabanının iki biçimi vardır: tüm veritabanı tek bir Markdown tablosu olarak işlenir (`buildDatabaseItem`, her kaydın blok içeriğini ek olarak içerir); veritabanı kaydı tek başına göründüğünde "özellik listesi + blok içeriği" olarak işlenir (`buildRecordItem`). Özellik çıkarma işlevi `propertyToString`, genel olarak `type` zincirini izler ve 22 özellik türünün tümünü kapsar; özellik adları alfabetik sıralanarak artımlı karşılaştırmanın belirleyici olması sağlanır.
- **Artımlı mantık**: İlk eşitlemede (imleç boş) doğrudan `FetchAll` çağrılır ve dönen girdilerin `UpdatedAt` değerleriyle imleç oluşturulur; sonraki eşitlemelerde `discoverAllResources`, Search API + BFS ile seçili köklerin altındaki tüm alt öğeleri belirler ve sayfa sayfa `last_edited_time` karşılaştırır. Veritabanları `fetchDatabaseIncremental` yolunu izler: herhangi bir kayıt değişirse tüm tablo yeniden oluşturulur.
- **Silme ile seçim kaldırmanın ayrılması**: Kaynakta kaybolan sayfalar `IsDeleted` olarak bildirilir; hâlâ görünür olan ama kullanıcı bir üst öğenin seçimini kaldırdığı için artık erişilemeyen sayfalar excluded kümesine girer ve **silinmiş olarak yanlış bildirilmez**. `computeExcludedSet` aynı zamanda "kullanıcının hiç görmediği yeni sayfaların" dışlanmamasını sağlar; seçili üst düğüm yeni alt sayfaları otomatik olarak kapsamaya devam eder.

#### Confluence (`connector/confluence/`)

Confluence Server / Data Center ve Confluence Cloud desteklenir; ikisi de HTTP Basic kimlik doğrulaması kullanır.

| Alan | Zorunlu | Açıklama |
| --- | --- | --- |
| `edition` | Hayır | `server` (varsayılan, Server / Data Center) veya `cloud` |
| `base_url` | Evet | Confluence adresi; protokol yoksa `https://` eklenir; Cloud için `https://<site>.atlassian.net` girildiğinde `/wiki` otomatik eklenir |
| `username` | Evet | Server/DC kullanıcı adı; Cloud için Atlassian hesabı e-postası |
| `password` | Server/DC için zorunlu | Server/DC hesap parolası |
| `api_token` | Cloud için zorunlu | Atlassian hesabının API tokens sayfasında oluşturulur |

`edition`, `base_url` ve `username` gizli bilgi değildir; arayüz bunları düzenleme sırasında geri gösterebilmek için Settings'e de yazar. `password` / `api_token` yalnızca şifrelenmiş kimlik bilgisi olarak saklanır.

<Screenshot
  src="/screenshots/datasource-confluence.png"
  caption="Confluence veri kaynağı: sürüm seçimi ve kimlik bilgisi girişi"
  hint="Veri kaynağı düzenleme penceresinde Confluence seçilidir; «Confluence sürümü» açılır menüsü (Server / Data Center, Cloud), adres, kullanıcı adı ve Cloud API token alanları ile bağlantı testinden sonraki alan seçim listesi gösterilir." />

- **Kapsam**: Kaynak listesi kimlik bilgisiyle görülebilen alanlardır (düz liste); en az bir alan seçilmelidir. Alandaki tüm yayımlanmış sayfalar eşitlenir; ekler ve blog yazıları içe aktarılmaz.
- **İçerik**: Sayfanın işlenmiş HTML'i Markdown'a dönüştürülür (her sayfa bir `.md` girdisi); gövde boşsa yalnızca başlık tutulur. Girdi metadata'sı `space_key`, `space_name`, `page_id` içerir; yazar varsa `creator` de eklenir.
- **Artımlı**: Sayfa sürüm numarasına göre karşılaştırılır; sürümü değişmeyen sayfalar atlanır. Akışlı eşitleme uygulanmıştır, her sayfa içe aktarıldıktan sonra imleç kaydedilir; tam eşitlemenin ara imleci önceki sayfa temel çizgisini de korur, böylece görev yeniden denendiğinde kaldığı yerden devam edebilir ve silme mutabakatı sürdürülebilir.
- **Silme koruması**: Seçilen alan erişilemezse, sayfa listesi boşken önceki seferde içerik varsa veya bu seferde kaybolan sayfalar en az 20 adet ve önceki sayfa sayısının %80'inden fazlaysa tüm eşitleme hata verir ve silme yapılmaz. Eşitlemeli silme açıksa diğer durumlarda kaybolan sayfalar silinmiş olarak işlenir.
- **Hatalar**: Tek sayfa hatasında hata bilgisi içeren bir yer tutucu girdi üretilir ve devam edilir; hata kodları `confluence_auth_or_permission`, `confluence_not_found`, `confluence_rate_limited`, `confluence_server_unavailable`, `confluence_api_error`, `confluence_sync_failed` şeklindedir; ham yanıtlar yalnızca sunucu günlüğüne yazılır.

```json
{"credentials":{"edition":"cloud","base_url":"https://team.atlassian.net","username":"me@example.com","api_token":"<token>"},"resource_ids":["<space-id>"]}
```

#### Yuque (`connector/yuque/`)

- **Kimlik doğrulama**: Kişisel Token (Yuque ayarları → Token) veya ekip Token'ı; kimlik bilgisi alanı `api_token` (istek başlığı `X-Auth-Token`) + isteğe bağlı `base_url` (kurumsal özel alan adı; scheme yoksa `https://` otomatik eklenir).
- **Kaynak listeleme**: `GET /api/v2/user` ile token kimliği belirlenir: `type=="Group"` ise ekip token'ıdır ve doğrudan ekip repo'ları listelenir; aksi halde kişisel repo'lar + katılınan grupların repo'ları listelenir (kullanıcı hiçbir gruba katılmamışsa Yuque 404 döndürür, bu boş olarak işlenir). Çıktı düz bir `book` kaynak listesidir ve ExternalID'ye göre kararlı biçimde sıralanır.
- **Alma** (`walk`, tam/artımlı ortak): `ListBookDocs` belgeleri listeler → `type != "Doc"` olanları (Sheet/Thread/Board/Table atlanır) ve `status != "1"` olanları (taslaklar atlanır) filtreler → hız sınırından kaçınmak için her `GetDocDetail` arasında 300ms beklenir → `format` `markdown`/`lake` ise `body` özgün Markdown olarak içe aktarılır (html gibi diğer biçimler savunmacı biçimde atlanır ve `skip_reason` kaydedilir).
- **Artımlı mantık**: İmleç `yuqueCursor.BookDocTimes` (`bookID → docID → content_updated_at`); aynıysa atlanır. Silme algılama: imleçte var, mevcut listede yok → `IsDeleted`.
- **Dizin yapısı (`settings` ayarları)**: Varsayılan olarak tüm belgeler bilgi tabanının kök dizinine düz yerleştirilir. Bunu iki anahtar kontrol eder:
  - **`folder_mode`** (varsayılan `none`): `none` çıplak `<başlık>.md` üretir ve geçmiş davranışla bayt bayt aynıdır; `toc` kitabın içindekiler hiyerarşisine göre `<kitap adı>/<grup>/…/<başlık>.md` yolunu oluşturur. Ön uç **yeni** veri kaynağı oluştururken `toc` yazar, mevcut veri kaynakları `none` olarak kalır; bu nedenle yükseltme mevcut kaynakların davranışını değiştirmez.
  - **`toc_only`** (varsayılan `false`): Yalnızca kitabın içindekiler düğüm listesinde yer alan belgeler içe aktarılır. **Yalnızca `folder_mode=toc` olduğunda etkilidir**; içindekiler yalnızca bu modda çekilir, düz modda bu anahtarın etkisi yoktur. Ölçüt "içindekilerde yer alıyor mu"dur, "yolu var mı" değildir; içindekilere bağlı ama hiçbir gruba konmamış belgeler de korunur. **Bu bir kabul filtresidir, temizleyici değildir**: filtrelenen belgeler yalnızca içe aktarılmaz, bilgi tabanına önceden eşitlenmiş kopyalar olduğu gibi kalır ve silinmiş olarak bildirilmez. Yuque Web arayüzü "API ile oluşturulmuş, içindekilere hiç bağlanmamış" belgeleri göstermez; bu tür belgeler silinmiş sayılsaydı içerikleri hem kaynakta hem bilgi tabanında erişilemez olurdu.
- **Dizin sahipliği**: `toc` açıldıktan sonra dizinler bağlayıcı tarafından yönetilir; bilgi tabanında elle düzenlenen dizinler, ilgili belge bir sonraki eşitlendiğinde Yuque'deki yapıya geri döndürülür.
- **Dikkat / sınırlamalar**:
  - **Dizin modunu değiştirdikten sonra bir kez tam eşitleme çalıştırın**: Mevcut bir veri kaynağında `folder_mode` değiştirildikten sonra artımlı eşitleme yalnızca içeriği değişen belgeleri taşır; içe aktarılmış belgelerin tümünün yeni moda göre yeniden yerleşmesi için elle bir tam eşitleme tetikleyin.
  - **Belgeyi yalnızca Yuque içindekilerinde taşımak hemen yansımaz**: Artımlı eşitleme değişikliği `content_updated_at` ile belirler; yalnızca içindekiler konumu değişip içeriği değişmeyen belgeler bir sonraki içerik değişikliğinde veya tam eşitlemede yeni dizine taşınır.
  - **`toc_only` içe aktarılmış belgeleri kaldırmaz**: Açıldıktan sonra filtrelenen belgeler artık eşitlenmez, bilgi tabanındaki mevcut kopyalar olduğu gibi kalır; daha sonra Yuque'de silinseler bile silme eşitlenmez. Gerekirse bilgi tabanında elle temizleyin.

#### DingTalk Docs (`connector/dingtalk/`)

- **Kimlik doğrulama**: Kurum içi uygulamanın Client ID'si, Client Secret'ı ve hedef bilgi tabanına erişim izni olan işlem yapan kişinin Union ID'si; `Wiki.Workspace.Read`, `Wiki.Node.Read`, `Storage.File.Read` izinleri açıldıktan sonra uygulama yayımlanır.
- **Kapsam**: Bilgi tabanı, klasör veya tek bir `ALIDOC/adoc` çevrimiçi belge seçilir ve herkese açık Wiki / Blocks API'si ile Markdown'a dönüştürülür. Şu anda DingTalk tabloları veya normal yüklenmiş ekler içe aktarılmaz; eşzamansız dışa aktarma geri çağrılarına da bağımlı değildir.
- **Eşitleme**: Belgenin `modifiedTimestamp` (milisaniye) değerine göre artımlı okunur, yoksa `modifiedTime` değerine geri dönülür; çakışan seçimler birleştirilir. Tam eşitleme de önceki imleçle karşılaştırılarak silme mutabakatı yapar; böylece `sync_mode=full` silmeleri kaçırmaz. Dizin dolaşımı eksik kalırsa silme ertelenir.
- **Gövde**: Herkese açık Blocks API yalnızca belge kökünün altındaki birinci düzey blokları döndürür; vurgulu blok gibi kapsayıcıların yanıtında `children` varsa işlenmeye devam edilir, yoksa metadata'da `nested_blocks_unavailable` işaretlenir; böylece eksik gövde tam başarı gibi gösterilmez.
- **Doğrulama**: Bağlantı testi bilgi tabanlarını listeler, kök düğüm listesini yoklar ve kökte çevrimiçi belge varsa Blocks'u deneme amaçlı okur; böylece `Wiki.Node.Read` / `Storage.File.Read` eksikliği erkenden fark edilir.
- **Hata ve kurtarma**: Geçersizleşen kaynaklar diğer kapsamları engellemez; başarısız kapsamlar ve gövdesi okunamayan belgeler yeniden denenebilmesi için eski sürümlerini korur. Herhangi bir kapsam tam olarak taranamazsa silme ertelenir ve doğrulanacak kayıtlar tutulur. Geçersizleşen tekil seçimler için izinleri kontrol edin veya yeniden seçin.
- **Silme anahtarı**: Yalnızca eşitlemeli silme açıksa, kaynakta silindiği doğrulanan yerel bilgi girdileri kaldırılır; erişilemeyen kaynaklar doğrudan silinmiş sayılmaz.

#### RSS / Atom (`connector/rss/`)

- **Yapılandırma**: `feed_urls` (satır sonu/virgülle ayrılır, çoklu girdilerde yinelenenler kaldırılır) **Settings** içinde saklanır (gizli değildir, arayüzde doğrudan düzenlenebilir); `auth_headers` (her satırda bir `Name: Value`; yalnızca feed isteklerine eklenir, üçüncü taraf makale sayfalarına asla gönderilmez) **Credentials** içinde şifrelenerek saklanır. `HasConfiguredCredentials` RSS için özel durum uygular: yalnızca `auth_headers` varsa kimlik bilgisi yapılandırılmış sayılır.
- **Alma**: `gofeed` RSS/Atom/JSON feed'lerini ayrıştırır; girdide bağlantı varsa özgün sayfa alınıp readability çıkarıcısından geçirilir, başarılı olursa tam metin esas alınır, başarısız olursa feed'in kendi içeriğine (`content:encoded`/`description`) geri dönülür; HTML, `html-to-markdown/v2` ile Markdown'a dönüştürülür. Girdi ID'si `GUID > Link > Title` sırasındaki ilk boş olmayan değerdir.
- **Artımlı mantık**: Çift katmanlı parmak izi: önce feed tarafı sinyal parmak izi karşılaştırılır (`feedSignalFingerprint`; değişmemişse özgün sayfa bile alınmaz); ardından alınan içeriğin SHA-256 parmak izi karşılaştırılır. **Silme eşitlemesi desteklenmez** (feed eski girdileri doğal olarak düşürür).
- **Kısmi hata**: Tek bir feed alınamaz/ayrıştırılamazsa eski imleç kullanılır (`copyFeedCursor`) ve diğer feed'lerle devam edilir; sonunda `datasource.PartialFetchError` bildirilir (SyncLog `partial` kaydeder); yalnızca tüm feed'ler başarısız olursa genel hata verilir.

### Veri Kaynağı Yaşam Döngüsü ve REST API

Rotalar `internal/router/routes_infra.go` içindeki `RegisterDataSourceRoutes` ile kaydedilir (liste, ayrıntı ve eşitleme günlükleri Viewer+, diğer işlemler Admin+; API Key için `manage_datasources` veya full-access gerekir):

| Yöntem ve yol | Yetki | Açıklama |
| --- | --- | --- |
| `GET /api/v1/datasource/types` | Viewer | Kullanılabilir bağlayıcı metadata listesi (`ListAvailableConnectors`, Priority'ye göre sıralı) |
| `POST /api/v1/datasource/validate-credentials` | Admin | Ham kimlik bilgileriyle bağlantıyı test eder (veritabanına yazmaz); oluşturma sihirbazındaki "Bağlantıyı test et" düğmesi için |
| `POST /api/v1/datasource` | Admin | Veri kaynağı oluşturur (KB'nin kiracıya ait olduğu doğrulanır → bağlayıcı türü doğrulanır → çevrimiçi Validate → veritabanına yazılır → cron kaydedilir) |
| `GET /api/v1/datasource?kb_id=` | Viewer | Bilgi tabanına göre veri kaynaklarını listeler (en son SyncLog ile birlikte) |
| `GET /api/v1/datasource/:id` | Viewer | Ayrıntı |
| `PUT /api/v1/datasource/:id` | Admin | Güncelleme (kimlik bilgisi alanları yok sayılır; yapılandırma gerçekten değişmişse ve kimlik bilgisi varsa çevrimiçi doğrulama tetiklenir; cron da güncellenir) |
| `DELETE /api/v1/datasource/:id` | Admin | Yumuşak silme + cron kaldırma + pending/running durumundaki SyncLog kayıtlarını iptal etme |
| `PUT /api/v1/datasource/:id/credentials` | Admin | Kimlik bilgilerini atomik olarak değiştirir (bkz. [Kimlik bilgisi şifreli saklama](#kimlik-bilgisi-sifreli-saklama)) |
| `DELETE /api/v1/datasource/:id/credentials/:field` | Admin | Kimlik bilgilerini temizler (field yalnızca `credentials` kabul eder) |
| `POST /api/v1/datasource/:id/validate` | Admin | Kayıtlı veri kaynağı için bağlantı testi yapar; başarısızlıkta `status=error` ayarlanır, başarıda error durumu temizlenir |
| `GET /api/v1/datasource/:id/resources?parent_id=` | Admin | Harici sistemdeki seçilebilir kaynakları listeler (parent_id tembel yüklemeyle genişletmeyi destekler) |
| `POST /api/v1/datasource/:id/resource-ancestors` | Admin | Seçili kaynakların üst öğe zincirini çözer (düzenlemede derindeki seçimleri geri göstermek için) |
| `POST /api/v1/datasource/:id/sync` | Admin | Senkronizasyonu elle tetikler (SyncLog oluşturur + Asynq görevini kuyruğa ekler) |
| `POST /api/v1/datasource/:id/pause` / `resume` | Admin | Duraklatma/sürdürme (cron da aynı anda kaldırılır/yeniden bağlanır) |
| `GET /api/v1/datasource/:id/logs`, `GET /api/v1/datasource/logs/:log_id` | Viewer | Senkronizasyon geçmişi |

Tüm `:id` yolları önce `getOwnedDataSource` → `getOwnedKnowledgeBase` üzerinden **kiracı yalıtımı** denetiminden geçer (veri kaynağının ait olduğu KB geçerli kiracıya ait olmalı ve API Key için KB yetki denetimini geçmelidir).

Yaşam döngüsü durum geçişleri:

```mermaid
flowchart LR
    A["Oluştur<br/>POST /datasource"] --> B["Yetkilendir<br/>PUT /:id/credentials<br/>(AES-256-GCM ile şifreli kayıt + çevrimiçi Validate)"]
    B --> C["Kaynak seç<br/>GET /:id/resources<br/>(ResourceIDs Config içine yazılır)"]
    C --> D["active<br/>(cron zamanlaması / elle senkronizasyon)"]
    D -- "Senkronizasyon başarısız" --> E["error"]
    E -- "validate başarılı / senkronizasyon başarılı" --> D
    D -- "POST /:id/pause" --> F["paused"]
    F -- "POST /:id/resume" --> D
    F -- "Elle senkronizasyona yine izin verilir" --> D
    D -- "DELETE /:id" --> G["Yumuşak silme<br/>(cron kaldırılır + tamamlanmamış SyncLog iptal edilir)"]
```

## Senkronizasyon ve Depolama Başvurusu

### Senkronizasyon Zamanlaması (internal/datasource/scheduler.go)

`Scheduler`, `robfig/cron` (`cron.WithSeconds()`, saniye düzeyinde 6 alanlı ifade desteği) üzerine kuruludur ve `SyncSchedule` yapılandırılmış her active veri kaynağı için bir cron entry tutar; servis başlarken `Start()` tüm active veri kaynaklarını DB'den yükleyip toplu olarak kaydeder.

robfig/cron **mutlak duvar saati zamanına** göre tetiklendiği için (ör. `0 0 * * * *` her zaman saat başında tetiklenir), çok örnekli dağıtımlarda tüm örnekler aynı anda tetiklenir. Yinelenenleri ayıklamak için iki katmanlı mekanizma kullanılır:

1. **DB katmanında çakışma önleme**: `syncLogRepo.HasRunningSync` — önceki senkronizasyon hâlâ running durumundaysa bu çalıştırma atlanır (senkronizasyon süresi cron aralığını aştığında üst üste çalışmayı önler).
2. **Redis katmanında örnekler arası tekilleştirme**: Belirlenimci `asynq.TaskID = "dssync:<dsID>:<yyyyMMddHHmm>"` (dakikaya göre kesilir). Aynı dakika içinde tüm örnekler aynı TaskID'yi üretir; Redis yalnızca birinin kuyruğa eklenmesini garanti eder, diğerleri `asynq.ErrTaskIDConflict` alır ve ilgili SyncLog `canceled` olarak işaretlenir ("deduplicated: another instance enqueued first").

Kuyruğa ekleme parametreleri: kuyruk `types.QueueSync`, `MaxRetry(5)`, `Timeout(2*time.Hour)`. Görev türü `types.TypeDataSourceSync` (`"datasource:sync"`) olup `internal/router/task.go` içindeki `mux.HandleFunc(types.TypeDataSourceSync, params.DataSourceService.ProcessSync)` tarafından tüketilir.

### Senkronizasyon Yürütme ve Bilgi Aktarımı (datasource_service.go)

`ProcessSync` bir Asynq görev işleyicisidir; tam akış için aşağıdaki sıralama diyagramına bakın. Önemli noktalar:

- **Savunmacı iptal**: Veri kaynağı veya bilgi tabanı silinmişse SyncLog `canceled` yapılır ve nil döndürülür (yeniden denenmez).
- **İki çekme yolu**: Bağlayıcı `StreamingConnector` uyguluyorsa `processSyncStreaming` (akış) kullanılır; aksi halde `ForceFull || SyncMode==full` ise `FetchAll`, değilse `ParseSyncCursor()` imleciyle `FetchIncremental` (toplu) kullanılır.
- **Akış yolunun imleç stratejisi** (`streamStartCursor`): Kullanıcının tetiklediği tam senkronizasyon **ilk denemede** imleci atıp her şeyi çeker; Asynq **yeniden denemeleri** (attempt > 0) ve tüm artımlı senkronizasyonlar son checkpoint'ten devam eder. `FullStreamingConnector` uygulayan bağlayıcılar (şu an Confluence) tam senkronizasyonda `FetchFullStream` kullanır: tüm öğeleri yeniden çeker ve eski imleci silme mutabakatı için taban çizgisi olarak korur.
- **Aktarım çekirdeği `applyFetchedItem` → `ingestItem`**:
  - `IsDeleted=true` ve `sync_deletions=true` olduğunda kiracı, bilgi tabanı, veri kaynağı ID'si ve external_id ile ilgili bilgi bulunur ve gerçekten silinir; senkronize silme kapalıysa mevcut bilgi korunur. Silme yeteneği ayrıca bağlayıcının güvenilir silme algılaması sağlayıp sağlamadığına bağlıdır;
  - `Content` baytları varsa → `multipart.FileHeader` olarak sarılıp `KnowledgeService.CreateKnowledgeFromFile` (tam belge ayrıştırma hattı) kullanılır; yalnızca `URL` varsa → `CreateKnowledgeFromURL` ile Rethra indirip ayrıştırır;
  - **Güncelleme = önce sil, sonra oluştur**: metadata `external_id` ile mevcut bilgi girdisi bulunursa önce `DeleteKnowledge` çağrılır, sonra yeniden oluşturulur ve Updated olarak sayılır;
  - Yinelenen dosyalar (`DuplicateKnowledgeError`) Skipped olarak sayılır, hata sayılmaz;
  - Her girdiye otomatik olarak metadata eklenir: `external_id`, `source_resource_id`, `datasource_id` ve bağlayıcının eklediği metadata. Kaynak zaman bilgisi sağlıyorsa UTC RFC3339 biçiminde `source_created_at` / `source_updated_at` de saklanır; bunlar kaynak belgenin zamanını gösterir ve Rethra'nın created_at/updated_at alanlarından ayrıdır.
- **Otomatik etiketleme**: `resolveAutoTagIDs` veri kaynağı adına göre hedef KB'de bir etiketi FindOrCreate ile bulur ya da oluşturur ve tüm senkronize girdilere ekler; böylece KB'de kaynak kolayca tanınır. Etiketleme hatası senkronizasyonu durdurmaz.
- **Sonuç durumu**: Tüm girdiler başarısızsa → `failed` (`allFetchedItemsFailedError`); RSS'te bazı feed'ler başarısızsa (`PartialFetchError`) veya akış yolunda başarısız belge varsa → `partial`; diğer durumlarda → `success`. Hata örnekleri `SyncItemError` biçiminde en fazla 100 adet saklanır.
- Çekme başarısız olsa bile bağlayıcı yeni bir imleç döndürdüyse (ör. RSS) imleç yine kalıcı hale getirilir; böylece geçici bir arızadan sonra her şeyi yeniden çekmek gerekmez.

```mermaid
sequenceDiagram
    autonumber
    participant U as "Kullanıcı / Cron Scheduler"
    participant H as "DataSourceHandler"
    participant S as "DataSourceService"
    participant Q as "Asynq (QueueSync)"
    participant C as "Connector (ör. Feishu)"
    participant EXT as "Harici sistem API"
    participant K as "KnowledgeService"
    participant DB as "PostgreSQL"

    U->>H: POST /datasource/:id/sync (veya cron tetikler)
    H->>S: ManualSync(dsID)
    S->>DB: SyncLog oluştur(status=running)
    S->>Q: Enqueue(datasource:sync, MaxRetry=5, Timeout=2h)
    Q-->>S: ProcessSync(payload)
    S->>DB: DataSource / SyncLog yükle / KB varlığını doğrula
    S->>S: ParseConfig() kimlik bilgilerini çözer
    alt "StreamingConnector (Feishu/Lark, GitLab, Confluence)"
        S->>C: FetchStream(config, cursor, handler)
        loop "Wiki düğümlerini gez"
            C->>EXT: ListWikiNodesRecursive / ExportAndDownload
            EXT-->>C: Belge içeriği (.docx/.xlsx/özgün dosya)
            C->>S: handler.Emit(item)
            S->>K: CreateKnowledgeFromFile (önce sil sonra oluştur=güncelleme)
            C->>S: handler.Checkpoint(cursor) her 50 düğümde veya 30 sn'de
            S->>DB: LastSyncCursor + SyncLog ilerlemesini kalıcı yap
        end
        C-->>S: Son cursor
    else "Toplu bağlayıcılar (Notion/Yuque/DingTalk/IMA/RSS)"
        S->>C: FetchAll veya FetchIncremental(cursor)
        C->>EXT: Liste + değişen içeriği çek
        EXT-->>C: Belge / Markdown
        C-->>S: []FetchedItem + nextCursor
        loop "Her girdi"
            S->>K: applyFetchedItem → ingestItem
        end
    end
    S->>DB: SyncLog(success/partial/failed) + DataSource(LastSyncAt/Cursor/Result) güncelle
    S->>DB: Denetim günlüğü kaydet (recordKBActivity)
```

### Kimlik Bilgisi Şifreli Saklama {#kimlik-bilgisi-sifreli-saklama}

Kimlik bilgileri yazma, okuma ve API yanıtı aşamalarında ayrı ayrı ele alınır:

**1. Yazarken şifreleme — `DataSourceConfig.ToJSON()`** (`internal/types/datasource.go`):

```go
// SYSTEM_AES_KEY yapılandırıldığında, Credentials içindeki her dize değeri serileştirmeden önce
// AES-256-GCM ile şifrelenir. Bu, kimlik bilgilerinin DB'ye girdiği tek yazma yoludur (GORM'un JSON
// türü baytları olduğu gibi geçirir); bu yüzden burada şifrelemek DataSource.Config'in baştan sona şifreli saklanmasını sağlar.
if key := utils.GetAESKey(); key != nil && len(out.Credentials) > 0 {
    ...
    if enc, err := utils.EncryptAESGCM(s, key); err == nil { encCreds[k] = enc }
}
```

**2. Okurken şifre çözme — `DataSource.ParseConfig()`**: Üç durumu şeffaf biçimde ele alır: boş dize olduğu gibi döner; `enc:v1:` öneki olmayan eski düz metin olduğu gibi döner (geçiş gerekmez); şifreli metin `SYSTEM_AES_KEY` ile çözülür. Şifre çözme başarısız olursa (anahtar kayıp ya da döndürülmüş) ilgili kimlik bilgisi alanı boşaltılır ve arayüz "Kimlik bilgisi yapılandırılmadı" gösterir. Kullanıcının bilgileri yeniden girmesi yeterlidir; veri kaynağının diğer yapılandırmaları kaybolmaz.

**3. Bağımsız kimlik bilgisi alt kaynağı — `internal/handler/datasource_credentials.go`**: Kimlik bilgileri normal `PUT /datasource/:id` üzerinden gitmez; ayrı bir `/credentials` alt kaynağı kullanılır ve bütün olarak atomik değiştirilir. Böylece bağlayıcı tam kimlik bilgisini tek seferde alır:

- `PUT /api/v1/datasource/:id/credentials` — kimlik bilgisi map'ini bütün olarak değiştirir, ardından çevrimiçi doğrulama için hemen bağlayıcının `Validate` yöntemini çağırır (kimlik bilgisi geçersizse hemen hata döner)
- `DELETE /api/v1/datasource/:id/credentials/credentials` — tümünü temizler
- Yanıtta **şifreli veya düz metin asla geri gönderilmez**, yalnızca `{"credentials": {"configured": true/false}}` döner; liste ve ayrıntı uç noktaları `dto.NewDataSourceResponse` ile serileştirilirken `Credentials` yapı gereği çıkarılır

Normal güncelleme uç noktası `UpdateDataSource` (`datasource_service.go`) **veritabanında kayıtlı kimlik bilgilerini zorla korur**; istek gövdesinde credentials olsa bile yok sayılır ve uyarı günlüğü yazılır. Ayrıca `StripNonSecretCredentials`, yanlışlıkla credentials içine konmuş gizli olmayan alanları çıkarır (şu an yalnızca RSS'in `feed_urls` alanı; bu alan `Settings` içine aittir).

### Güvenlik Kısıtlamaları (internal/datasource/httpclient.go ve errors.go)

`httpclient.go` tüm bağlayıcıların ortak kullandığı iki SSRF koruma giriş noktası sağlar:

```go
// ValidateConnectorBaseURL, bağlayıcı base_url değerini SSRF politikasına göre doğrular (boş değer geçer; varsayılanı çağıran taraf uygular)
func ValidateConnectorBaseURL(rawURL string) error {
    ...
    if err := utils.ValidateURLForSSRF(url); err != nil { ... }
}

// NewConnectorHTTPClient, yönlendirmelerde ve bağlantı kurma anında SSRF koruması olan bir HTTP istemcisi döndürür
func NewConnectorHTTPClient(timeout time.Duration) *http.Client {
    cfg := utils.DefaultSSRFSafeHTTPClientConfig()
    cfg.Timeout = timeout
    return utils.NewSSRFSafeHTTPClient(cfg)
}
```

Alttaki `internal/utils/security.go` özel ağ, geri döngü (loopback), link-local gibi hedefleri reddeder ve yalnızca ilk URL'yi değil **her yönlendirmede ve gerçek bağlantı kurulurken** yeniden doğrular. Böylece kötü amaçlı feed'ler veya özel base_url değerleri Rethra'yı iç ağ servislerine yönlendiremez. Her bağlayıcının `parseXXXConfig` işlevi base_url için `ValidateConnectorBaseURL` çağırır.

`errors.go` modül düzeyindeki gözcü hataları (`ErrConnectorNotFound`, `ErrDataSourceInvalid`, `ErrInvalidCredentials`, `ErrSyncFailed` vb.) ve `PartialFetchError` türünü tanımlar (kaynakların bir kısmı başarılı, bir kısmı başarısız; çağıran taraf elde edilen girdileri işlemeli, imleci kalıcı hale getirmeli ve `Details` bilgisini kullanıcıya partial durumuyla göstermelidir).

### Çekirdek Soyutlama: Connector Arayüzü

Tüm bağlayıcılar `internal/datasource/connector.go` içindeki `Connector` arayüzünü uygulamalıdır:

```go
type Connector interface {
    // Type, bağlayıcı türü tanımlayıcısını döndürür (ör. "feishu", "notion")
    Type() string
    // Validate, harici API'yi gerçekten çağırarak yapılandırma ve kimlik bilgilerinin geçerliliğini doğrular
    Validate(ctx context.Context, config *types.DataSourceConfig) error
    // ListResources, senkronize edilebilir kaynakları (belge, alan, klasör vb.) listeler.
    // parentID hiyerarşik kaynakların tembel yüklenmesini destekler: ""=en üst düzey; boş değilse=o kaynağın doğrudan alt düğümleri
    ListResources(ctx context.Context, config *types.DataSourceConfig, parentID string) ([]types.Resource, error)
    // ResolveResourceAncestors, seçili kaynakların üst öğe zincirini çözer; tembel yüklenen seçicide derindeki seçimleri geri göstermek için kullanılır
    ResolveResourceAncestors(ctx context.Context, config *types.DataSourceConfig, resourceIDs []string) ([]string, error)
    // FetchAll, belirtilen kaynakları tam olarak senkronize eder
    FetchAll(ctx context.Context, config *types.DataSourceConfig, resourceIDs []string) ([]types.FetchedItem, error)
    // FetchIncremental, imlece göre artımlı senkronizasyon yapar; değişen öğeleri ve sonraki senkronizasyon için yeni imleci döndürür
    FetchIncremental(ctx context.Context, config *types.DataSourceConfig, cursor *types.SyncCursor) ([]types.FetchedItem, *types.SyncCursor, error)
}
```

#### İsteğe Bağlı Uzantı: StreamingConnector (Akışlı, Sürdürülebilir Senkronizasyon)

`StreamingConnector` öğeleri tek tek çekmeyi, aktarmayı ve imleci kaydetmeyi destekler. Büyük ölçekli senkronizasyonlar için uygundur ve tüm içeriği bir kerede önbelleğe almanın bellek maliyetini azaltır:

```go
type StreamHandler interface {
    // Emit, çekilen bir öğeyi tek başına aktarır; hata döndürürse tüm akış durdurulur
    Emit(ctx context.Context, item types.FetchedItem) error
    // Checkpoint, imleç anlık görüntüsünü eşzamanlı olarak kalıcı yapar (artım değil, tam ve sürdürülebilir bir anlık görüntü olmalıdır)
    Checkpoint(ctx context.Context, cursor *types.SyncCursor) error
}

type StreamingConnector interface {
    Connector
    FetchStream(ctx context.Context, config *types.DataSourceConfig,
        cursor *types.SyncCursor, h StreamHandler) (*types.SyncCursor, error)
}
```

Görev zaman aşımına uğradıktan sonra işlem en son checkpoint'ten sürdürülebilir. Asynq senkronizasyon görevinin zaman aşımı 2 saattir; akış yolu girdi girdi ilerler ve tüm dosya gövdelerini önbelleğe almaz. Feishu/Lark (bilgi tabanı ve bulut sürücü), GitLab ve Confluence bağlayıcıları `StreamingConnector` uygular. Confluence ayrıca `FullStreamingConnector` uygular, böylece tam senkronizasyonda da silme mutabakatı yapabilir. DingTalk şu an toplu `FetchAll`/`FetchIncremental`/`FetchAllFromCursor` kullanır; çok büyük bilgi tabanlarında tüm Markdown içeriğini bir kerede yüklememek için artımlı mod önerilir.

#### ConnectorRegistry: Kayıt ve Arama

`ConnectorRegistry` basit bir `map[string]Connector` kayıt defteridir. Gerçek kayıt `internal/container/container.go` içindeki `initConnectorRegistry()` işlevinde yapılır:

```go
registry.Register(wiki.NewConnector(core.RegionFeishu))             // feishu
registry.Register(wiki.NewConnector(core.RegionLark))               // lark (uluslararası sürüm; aynı uygulama, farklı Region)
registry.Register(drive.NewDriveConnector(core.RegionFeishuDrive))  // feishu_drive
registry.Register(drive.NewDriveConnector(core.RegionLarkDrive))    // lark_drive
registry.Register(notionConnector.NewConnector())                   // notion
registry.Register(confluenceConnector.NewConnector())               // confluence
registry.Register(yuqueConnector.NewConnector())                    // yuque
registry.Register(dingtalkConnector.NewConnector())                 // dingtalk
registry.Register(imaConnector.NewConnector())                      // ima
registry.Register(rssConnector.NewConnector())                      // rss
registry.Register(gitlabConnector.NewConnector())                   // gitlab
```

> Not: `connector.go` içindeki `ConnectorMetadataRegistry` henüz uygulanmamış bağlayıcıları da içerir (GitHub, Google Drive, OneDrive, Web Crawler, Slack, IMAP vb.). Şu an gerçekten kayıtlı ve kullanılabilir türler: `feishu`, `lark`, `feishu_drive`, `lark_drive`, `notion`, `confluence`, `yuque`, `dingtalk`, `ima`, `rss`, `gitlab`. Kayıtlı olmayan türler veri kaynağı oluşturulurken `connectorRegistry.Get()` tarafından `ErrConnectorNotFound` ile reddedilir.

### Veri Modeli (internal/types/datasource.go)

| Yapı | Açıklama |
| --- | --- |
| `DataSource` | Veri kaynağı yapılandırma varlığı (`data_sources` tablosu). Önemli alanlar: `Type` (bağlayıcı türü), `Config` (JSONB, şifreli kimlik bilgilerini içerir), `SyncSchedule` (cron ifadesi), `SyncMode` (`incremental`/`full`), `Status` (`active`/`paused`/`error`/`deleted`), `ConflictStrategy`, `SyncDeletions`, `LastSyncCursor` (artımlı imleç JSONB), `LastSyncAt`, `LastSyncResult`, `SyncLogRetentionDays` |
| `SyncLog` | Tek bir senkronizasyon çalıştırma kaydı (`sync_logs` tablosu). Durumlar: `running`/`success`/`partial`/`failed`/`canceled`; sayaçlar: `ItemsTotal/Created/Updated/Deleted/Skipped/Failed`; `Result` alanı `SyncResult` JSON'unu saklar |
| `DataSourceConfig` | Şifresi çözülmüş yapılandırma yapısı: `Type` + `Credentials map[string]interface{}` + `ResourceIDs []string` (seçili kaynaklar) + `Settings map[string]interface{}` (gizli olmayan yapılandırma) |
| `Resource` | Harici sistemdeki seçilebilir kaynak: `ExternalID`, `Name`, `Type`, `URL`, `ParentID`, `HasChildren`, `ModifiedAt`, `Metadata` |
| `FetchedItem` | Çekilen tek bir belge: `ExternalID`, `Title`, `Content []byte`, `ContentType`, `FileName`, `URL`, `UpdatedAt`, `Metadata`, `IsDeleted`, `SourceResourceID` |
| `SyncCursor` | Artımlı imleç: `LastSyncTime` + `ConnectorCursor map[string]interface{}` (bağlayıcıya özel yapı) |
| `SyncResult` | Senkronizasyon sonuç özeti + `Errors []SyncItemError` (hata örnekleri, üst sınır 100; bkz. `maxSyncResultErrors`) |
| `SyncItemError` | Kullanıcıya dönük hata örneği: kararlı i18n `Code` + ara değer `Params` + yedek `Message`; ham API durum kodu ve yanıt gövdesi yalnızca sunucu günlüklerinde kalır |
| `DataSourceSyncPayload` | Asynq görev yükü: `DataSourceID`, `TenantID`, `SyncLogID`, `ForceFull`, `Trigger` (`manual`/`schedule`) |

## Başvuru

- Bağlayıcı geliştirme kılavuzu (kodla birlikte güncellenir): `internal/datasource/CONNECTOR_IMPLEMENTATION_GUIDE.md`
- Modül açıklaması (kodla birlikte güncellenir): `internal/datasource/README.md`

## Uygulama Başvurusu

- Bağlayıcı çatısı ve uygulamaları: `internal/datasource/` (`connector.go`, `scheduler.go`, `httpclient.go`, `errors.go`, `connector/` altındaki uygulamalar)
- HTTP arayüz katmanı: `internal/handler/datasource.go`, `internal/handler/datasource_credentials.go`
- İş servisi katmanı: `internal/application/service/datasource_service.go`
- Veri modeli: `internal/types/datasource.go`
