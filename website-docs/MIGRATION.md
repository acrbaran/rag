# Eski docs belge taşıma kaydı

Sonraki ürün, dağıtım, API ve geliştirme belgeleri `website-docs/` altında birlikte yönetilir. Bu kayıt, bakımcıların taşıma kapsamını incelemesi içindir ve belge sitesinde yayımlanmaz. Bu sefer taşınmış, yinelenen veya güncelliğini yitirmiş 87 eski elle yazılmış belge silinmiştir; ikinci bir ana metin seti tutulmaz. Eski içerik Git geçmişinden izlenebilir; `docs/` yalnızca aşağıda listelenen mühendislik kaynaklarını korur.

## Yeni siteye eklenen içerik

| Eski belge (`docs` dizinine göre) | Yeni giriş | İşlem |
| --- | --- | --- |
| `QA.md`, `migration-troubleshooting.md` | [Sık sorulan sorular ve yükseltme sorun giderme](01-getting-started/05-troubleshooting.md) | Belirti tespiti ve kurtarma akışları korunur, özellik ayrıntıları mevcut bölümlere bağlanır; “başarısızlık mutlaka tam geri alma getirir” ve mekanik force ifadeleri düzeltilir |
| `paradedb-upgrade.md` | [ParadeDB yükseltmesi](01-getting-started/06-paradedb-upgrade.md) | Yedekleme, imaj ve eklenti SQL yükseltmesi, geri alma ve doğrulama girişleri korunur; geçmiş test sonuçları bu seferin doğrulaması sayılmaz |
| `sandbox-cluster.md`, `sandbox-docker-backend.md`, `sandbox-desktop.md`, `sandbox-protocol.md` | [Korumalı alan dağıtımı ve sorun giderme](06-development/04-sandbox-deployment.md) | Şablon, daemon, ağ geçidi, Redis, masaüstü ve anlık görüntü sınırları korunur; eski local seçeneğinin kaldırıldığı açıklanır, eski Docker uygulaması karşılaştırmaları ve doğrulanmamış üçüncü taraf dağıtım özellikleri çıkarılır |
| `browser-skill-integration.md`, `browser-skill-production.md` | [Yerel tarayıcı](05-clients/09-local-browser.md) | Eşleştirme, insan katılımı, görev kurtarma, eşlik eden derleme, proxy ve çoklu kopya gereksinimleri eklenir |
| `agent-prompt-assembly.md` | [Sohbet istemi birleştirme](06-development/05-agent-prompts.md) | Mevcut birleştirme girişleri, şablon kaydetme ve ileti sınırları korunur; tur bazlı inceleme kayıtları atlanır |
| `mcp-tool-directory.md` | [MCP entegrasyonu](03-features/08-mcp.md#mcp-tool-directory), [MCP API](04-api/02-api-agent-mcp.md) | Eski tam kayıt açıklaması değiştirilir; kalıcı dizin, özne yalıtımı, isteğe bağlı yükleme ve senkronizasyon arayüzleri eklenir; eski belgedeki refresh davranışına ilişkin çelişkili ifadeler düzeltilir |
| `worker-pool-governance.md` | [Eşzamansız görev kapasitesi](02-architecture/05-async-tasks.md#capacity-planning) | Yeni sitedeki daha güncel kuyruk topolojisi kullanılır; toplu yapılandırmanın kullanımdan kaldırılması, kapasite tahmini ve aşağı akış kota sınırları eklenir |
| `embed-subdomain.md`, `embed-secure-mode.md` | [Gömülü kanallar](03-features/13-embed-channel.md#embed-subdomain) | Güvenli modun ana gövdesi zaten kapsanmıştır; bağımsız alt alan adı, çalışma zamanı adresi ve proxy stratejisi eklenir; yalnızca Cookie/Header varlığını denetleyen sahte oturum açma doğrulama örnekleri taşınmaz |
| `wiki/集成扩展/飞书云盘数据源接入说明.md` (Entegrasyonlar / Feishu Drive veri kaynağı bağlantı kılavuzu) | [Feishu Drive entegrasyonu](03-features/24-feishu-drive.md) | Klasör yetkilendirmesi ve ayrıştırma modları eklendi; varsayılan export, blocks geri dönüşü ve görsel koşulları, eski imleçe dayalı silme algılama ve hata kurtarma sınırları güncel uygulamaya göre düzeltildi |
| `dev/opensearch-integration-test.md` | [Arama motoru: yerel entegrasyon testi](03-features/05-retrieval-engines.md#opensearch-local-testing) | Mevcut sayfaya birleştirilir; geliştirme kümesi, SSRF ve doğrulama akışları eklenir, mevcut kopya sayısının 0 iken fiilen 1'e geri döndüğü belirtilir; temizleme komutu, diğer geliştirme veri birimlerinin silinmesini önlemek için hedefli durdurma olarak değiştirilir |
| `cloud-image/README.md`, eski bulut sağlayıcısına özel dağıtım kılavuzu | [Geliştirme rehberi: bulut imajı betikleri](06-development/01-dev-guide.md#cloud-image-scripts) | Yalnızca betik sınırları ve ilk başlatma kurtarması korunur, ayrı bir dağıtım eğitimi oluşturulmaz; eski bulut platformu kotaları, fiyatları, inceleme süreleri ve sabit konsol yolları kullanılmaz |

## Yeni site tarafından kapsandı, tam metin artık kopyalanmayacak

Aşağıda konu bazında karşılaştırılmıştır; “kapsama”, etkin işlevlerin ve geliştirme giriş noktalarının zaten bir yere ait olduğu anlamına gelir, tüm eski örneklerin, ekran görüntülerinin veya işlev bazında açıklamaların korunacağı anlamına gelmez.

| Eski belge | Bakım konumu |
| --- | --- |
| `开发指南.md` (Geliştirme kılavuzu), `快速开发模式说明.md` (Hızlı geliştirme modu açıklaması) | [Geliştirme kılavuzu](06-development/01-dev-guide.md) |
| `LITE.md` | [Kurulum ve dağıtım](01-getting-started/02-installation.md); sürüm paketinin README dosyasının hâlâ ayrı bağımlılıkları vardır, aşağıya bakın |
| `CHUNKING.md` | [Parçalama](03-features/04-chunking.md) |
| `KnowledgeGraph.md`, `开启知识图谱功能.md` (Bilgi grafiği özelliğini etkinleştirme) | [Bilgi grafiği](03-features/09-knowledge-graph.md) |
| `使用其他向量数据库.md` (Başka vektör veritabanları kullanma) | [Arama motorları](03-features/05-retrieval-engines.md), [Genişletme noktaları](06-development/03-extension-points.md) |
| `添加新的网络搜索引擎.md` (Yeni web arama motoru ekleme) | [Web araması](03-features/11-web-search.md), [Genişletme noktaları](06-development/03-extension-points.md) |
| `数据源导入开发文档.md` (Veri kaynağı içe aktarma geliştirme belgesi) | [Veri kaynağı senkronizasyonu](03-features/10-datasource.md), [Genişletme noktaları](06-development/03-extension-points.md); bulut sürücü uygulaması için ayrıca yeni sayfa eklendi |
| `IM集成开发文档.md` (IM entegrasyonu geliştirme belgesi) | [IM entegrasyonu](03-features/12-im-integration.md), [Genişletme noktaları](06-development/03-extension-points.md); eski platform yönetim ekran görüntüleri/toplu izin listeleri doğrudan yeniden kullanılmadı |
| `OIDC认证调用流程.md` (OIDC kimlik doğrulama akışı), `RBAC说明.md` (RBAC açıklaması), `共享空间说明.md` (Paylaşılan alan açıklaması) | [Kimlik doğrulama ve yetkilendirme](03-features/01-tenant-auth.md), [Kimlik doğrulama API'si](04-api/02-api-auth.md), [Organizasyon API'si](04-api/02-api-org.md) |
| `BUILTIN_MODELS.md` | [Model yönetimi](03-features/06-models.md), [Platform yönetimi](03-features/20-platform-admin.md) |
| `BUILTIN_MCP_SERVICES.md`, `MCP功能使用说明.md` (MCP özelliği kullanım kılavuzu), `zh/mcp-approval.md` | [MCP entegrasyonu](03-features/08-mcp.md) |
| `Langfuse集成.md` (Langfuse entegrasyonu), `日志配置.md` (Günlük yapılandırması) | [Gözlemlenebilirlik](03-features/16-observability.md), [Yapılandırma başvurusu](01-getting-started/04-configuration.md) |
| `agent-skills.md`, `agent-tools-design.md` | [Beceriler ve sanal alan](03-features/22-skills-sandbox.md), [Agent motoru](03-features/07-agent.md), yeni sanal alan dağıtım sayfası; eski araç adları, local yapılandırması ve güncelliğini yitirmiş normal kullanıcı/salt okunur imaj sözleşmeleri taşınmayacak |
| `chat-steering.md` | [Oturum deneyimi](03-features/18-chat-experience.md), [Oturum API'si](04-api/02-api-chat.md) |
| `client-integration-upgrade-notes.md` | [Go SDK](05-clients/03-go-sdk.md), [Dosya erişimi](03-features/21-file-access.md) |
| `api/*.md` | [API genel bakışı](04-api/01-api-overview.md) ve aynı dizindeki her konu; eksik MCP metadata ve sanal alan masaüstü arayüzleri eklenecek, ikinci bir elle yazılmış API seti artık korunmayacak |
| `wiki/` içindeki diğer sayfalar | Yukarıdaki konuların özet kopyalarıdır, toplu olarak taşınmayacak; eski gezinme, geri bağlantılar ve grafik yeni siteye alınmayacak |

## Mevcut ürün belgelerine taşınmayacak

- `ROADMAP.md`: eski plan, uygulanmış yetenekleri içerir ve mevcut taahhüt olarak kullanılamaz; giriş noktası mevcut ürün tanıtımı olarak değiştirilecek, plan ancak yeniden doğrulandıktan sonra yazılmalıdır.
- `code-slimming-audit.md`: tek seferlik kod sadeleştirme denetimi, gövde metni silinecek ve yalnızca Git geçmişinde korunacak.
- `plans/2026-09-10-faq-enabled-filter-design.md`: geçmiş tasarım süreci, gövde metni silinecek; nihai arayüz FAQ/API bölümleri tarafından ele alınacak, tasarım süreci Git geçmişinden incelenebilir.
- `poc/docker-sandbox/`: bağımsız Go modülü ve eski Docker fizibilite doğrulaması, kullanıcı belgesi değildir; ileride geliştirme deneyleri dizinine taşınabilir, siteye karıştırılmayacak.

## docs dizininde tutulan mühendislik kaynakları

1. **Swagger üretim paketi**: `internal/router/router.go`, `github.com/acrbaran/rag/docs` paketini içe aktarır; `Makefile` içindeki `docs` hedefi çıktıyı bu dizine yazar. `docs.go` şu anda backend derlemesine katılır, `swagger_contract_test.go` ise JSON/YAML dosyalarını okur. Normal derleme ve CI bunları otomatik üretmediği için üç üretilmiş dosya ve testler şimdilik tutulur. İleride bağımsız bir üretim paketine taşınabilir ve içe aktarma, üretim yolu, testler ve lint istisnaları güncellenebilir; üretilen dosyalar commit edilmeyecekse önce sabit sürümlü üretim adımı tüm derleme, test ve yayın girişlerine bağlanmalıdır.
2. **Lite yayın README'si**: `scripts/package-lite.sh` ve `.github/workflows/release-lite.yml`, `docs/LITE.md` dosyasını çevrimdışı yayın paketine kopyalar; taşınırken çevrimdışı okumaya uygun bir README korunmalıdır, site içi göreli bağlantılar içeren uzun bir sayfayla doğrudan değiştirilemez.
3. **Görseller**: Yalnızca Helm chart ikonu ve tanıtım videosu tarafından kullanılan `docs/images/logo.png` tutuldu; eski README görselleri silindi. Yeni sitenin kendi görselleri `public/` içindedir ve eski dizine bağımlı değildir.
4. **Geçmiş deneyler**: `poc/docker-sandbox/` bağımsız Go modülü ve çalıştırma açıklaması tutuldu; bakımı yapılan ürün belgelerine ait değildir. Eski el yazımı metinler (API, Wiki kopyaları, yol haritası, denetim ve tasarım kayıtları dahil) silindi.

Ortam değişkeni örneklerindeki, ön yüz yardımı ve parçalama örneklerindeki, Helm kurulum ipuçlarındaki, örnek projelerdeki, kod yorumlarındaki ve CHANGELOG'daki tıklanabilir belge bağlantıları yeni siteye yönlendirildi. CHANGELOG'da eski sürümlerde hangi dosyaların eklendiğini anlatan geçmiş metinler olduğu gibi bırakıldı ve güncel belge girişi olarak kullanılmaz. `scripts/cloud-image/README.md`, yinelenen öğreticilerin bakımını sürdürmemek için yeni site girişine indirgendi.

## Bu seferki inceleme kapsamı

Taşıma, mevcut deponun rotaları, işleyicileri, sandbox/tarayıcı uygulamaları, veri kaynağı servisleri, taşıma SQL'leri ve dağıtım betikleri esas alınarak yapıldı. Site bağlantı kontrolü, Mermaid kontrolü ve belge derlemesi çalıştırıldı; bu belge düzenlemesinde üretim veritabanı yükseltmesi, bulut imajı temizliği, gerçek sandbox kümesi veya IM platformu entegrasyonu yapılmadı.

## Doğruluk ve gereklilik gözden geçirmesi

Bu seferki yeni bağımsız sayfalar altı türle sınırlı tutuldu: genel sorun giderme, ParadeDB mevcut veri yükseltmesi, Feishu Drive entegrasyonu, yerel tarayıcı, sandbox dağıtımı ve istem bakımı. Bunlar sırasıyla bölümler arası sorun gidermeyi, durumlu yükseltmeyi, yetkilendirme ve ayrıştırma seçimini, bağımsız istemci dağıtımını, sandbox operasyonunu ve geliştirme sözleşmesini tamamlar; mevcut sayfalarda yalnızca özet ve bağlantılar tutuldu.

- OpenSearch entegrasyon testi arama motorlarına, bulut imajı bakımı geliştirme kılavuzuna birleştirildi; az içerikli veya henüz gerçekten test edilmemiş bağımsız öğreticiler eklenmedi.
- İstem sayfası `prompts.go`, `observe.go`, `finalize.go` ve `config/agent_prompts.go` ile karşılaştırıldı; Agent eski bölümündeki tarih biçimi, MCP bahsetme kapsamı ve kapanış mesajı rolü de düzeltildi, yinelenen tarayıcı iç protokol açıklaması kaldırıldı.
- Feishu Drive, ortak `core/engine.go`, `core/shared.go` ve veri kaynağı servisiyle karşılaştırıldı; tam senkronizasyonun silmeleri tamamlayabileceği artık vaat edilmiyor. Üçüncü taraf izin başvuruları resmi arayüz açıklamalarına bırakıldı; bu seferki platformda doğrulanmamış izin listeleri sabitlenmedi.
- OpenSearch, sürücü yapılandırması ve indeks uygulamasıyla karşılaştırıldı; geçersiz sıfır kopya örneği kaldırıldı ve eski sayfadaki tek indeks boyutu açıklaması düzeltildi.
- Veritabanı yükseltme adımları üretim ve geliştirme Compose'unu ayırır; sandbox belgesi masaüstü host'u ile üç sunucu tarafı adlandırma yapılandırmasını ayırır; bulut imajı systemd sabit yolunu ve ilk başlatma işaretinin gerçek yazılma zamanını belirtir.

Yukarıdakiler kaynak kod ve yapılandırma karşılaştırmasıdır; gerçek platform entegrasyonu veya üretim yükseltmesi kabulüyle eşdeğer tutulmamalıdır.
