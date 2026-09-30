# Değişiklik günlüğü

Projedeki tüm önemli değişiklikler bu dosyada kayıt altına alınır.

Biçim [Keep a Changelog](https://keepachangelog.com/tr-TR/1.0.0/) temel alınarak hazırlanmıştır
ve bu proje [Anlamsal Sürümleme](https://semver.org/lang/tr/) kurallarına uyar.

## [Unreleased]

### Eklenenler
- `create_knowledge_from_text` aracı eklendi: el ile yazılmış Markdown metninden bilgi kaydı oluşturur, mevcut `/knowledge-bases/{id}/knowledge/manual` arayüzünü çağırır ve #323'teki "metin" bölümünü tamamlar. Varsayılan `status="publish"` olduğundan oluşturulduktan hemen sonra ayrıştırma/indeksleme akışına girer ve aranabilir olur; yalnızca kaydedip indekslememek için `status="draft"` verilebilir.
- `update_knowledge_from_text` aracı eklendi: mevcut `PUT /knowledge/manual/{id}` arayüzüyle el ile yazılmış Markdown bilgisini günceller; varsayılan olarak yeniden indeksler, taslak olarak da kaydedilebilir. #2378'i tamamlar.
- README araç listesine mevcut `create_knowledge_from_file` eklendi.

## [1.1.1] - 2026-07-30

### Değişenler
- Resmî PyPI paket adı **`rethra-mcp`** olarak değiştirildi (acrbaran/rag deposundaki CI tarafından Trusted Publishing ile yayımlanır).
- Proje URL'si resmî [acrbaran/rag](https://github.com/acrbaran/rag) deposunu (`mcp-server/` dizini) gösterir.

### Düzeltmeler
- HTTP taşıması `stateless_http=True` davranışına geri döndü; geçişten önceki `StreamableHTTPSessionManager(stateless=True)` davranışıyla aynıdır.
- SSE taşıması `/sse/messages/` mesaj uç noktasını geri kazandı; geçişten önceki rotayla aynıdır.
- `RethraClient` iş parçacığına özgü `requests.Session` kullanır; böylece MCP 2.x iş parçacığı havuzunda eşzamanlı çağrılarda Session yarış durumu oluşmaz.
- Dosya yükleme (`create_knowledge_from_file`) `RETHRA_VERIFY_SSL` ayarına uyar.

### Notlar
- Araç çalıştırması başarısız olduğunda MCPServer 2.x `CallToolResult(isError=True)` (`ToolError`) döndürür;
  eski düşük seviyeli API'deki gibi başarılı yanıtın metin bloğunda `"Error executing …"` önekiyle dönmez.
  Yalnızca `content[0].text` değerini ayrıştıran istemciler genellikle farkı hissetmez; `isError` bayrağına dayanan entegrasyonlar MCP spesifikasyonuna daha uygun davranır.

## [1.1.0] - 2026-07-30

### Düzeltmeler
- MCP Python SDK 2.x altında sunucunun başlangıçta çökmesi (`AttributeError: 'Server' object has no attribute 'list_tools'`) düzeltildi.
  SDK 2.0, düşük seviyeli `Server` sınıfının dekoratör API'sini (`@app.list_tools()` / `@app.call_tool()` / `app.get_capabilities()`) kaldırdı;
  yayımlanmış paket `uvx` ile başlatıldığında en son 2.x sürümüne çözümleniyor ve bağlantı kapanıyordu.

### Değişenler
- MCP sunucu uygulaması düşük seviyeli `Server` API'sinden üst seviyeli `MCPServer` API'sine taşındı (mcp 2.x; eski adıyla FastMCP).
  - 28 araç `@mcp.tool()` fonksiyon imzası biçiminde yeniden yazıldı: girdi parametreleri tip açıklamalarıyla tanımlanır (şemayı çatı otomatik üretir),
    açıklamalar docstring'den gelir, düz Python dönüş değerlerini çatı serileştirir.
  - Taşıma katmanı `run_stdio_async()` / `sse_app()` / `streamable_http_app()` kullanır; kimlik doğrulama yine `MCPAuthMiddleware` ile sarılır.
  - `RethraClient` iş mantığı (REST/SSE çağrıları, resolve_*, wiki) değişmedi.
- Bağımlılık üst sınırı `mcp>=2,<3` olarak ayarlandı (yayın paketi ve geliştirme ortamı aynıdır; üst sınır olmadığı için gelecekteki kırıcı sürümlere çözümlenme tekrar yaşanmaz).

### Notlar
- Bu sürüm çalışma ortamında `mcp>=2` gerektirir. Geçici olarak eski SDK gerekiyorsa başlatma komutuna `--with "mcp<2"` eklenebilir, ancak bu sürüme yükseltmeniz önerilir.

## [1.0.1] - 2026-07-28

### Düzeltmeler
- Giriş betiklerinin (`run_server.py`, `main.py`, `run.py`) tanılama çıktısı stderr'e yönlendirildi; böylece MCP stdio protokol akışı bozulup istemcinin başlatılamaması önlendi
- Wheel paketine `upload_paths.py` dahil edilmediği için oluşan `ModuleNotFoundError` düzeltildi
- `__init__.py` mutlak içe aktarmaya geçirildi; unittest/pytest test toplarken oluşan paket içe aktarma hatası düzeltildi

### Değişenler
- PyPI dağıtım paket adı `rethra-mcp` olarak birleştirildi (komut satırı girişleri `rethra-mcp-server` / `rethra-server` değişmedi)
- CI workflow'u eklendi (`.github/workflows/mcp-server.yml`); yayın etiketi biçimi `mcp-server-v*`

## [1.0.0] - 2024-01-XX

### Eklenenler
- İlk sürüm yayımlandı
- Rethra MCP Server temel işlevleri
- Tam Rethra API entegrasyonu
- Alan yönetimi araçları
- Bilgi tabanı yönetimi araçları
- Bilgi yönetimi araçları
- Model yönetimi araçları
- Oturum yönetimi araçları
- Sohbet araçları
- Parça yönetimi araçları
- Birden çok başlatma yöntemi desteği
- Komut satırı parametresi desteği
- Ortam değişkeniyle yapılandırma
- Tam paket kurulumu desteği
- Geliştirme ve üretim modları
- Ayrıntılı belgeler ve kurulum kılavuzu

### Araç listesi
- `create_tenant` - Yeni alan oluşturur
- `list_tenants` - Tüm alanları listeler
- `create_knowledge_base` - Bilgi tabanı oluşturur
- `list_knowledge_bases` - Bilgi tabanlarını listeler
- `get_knowledge_base` - Bilgi tabanı ayrıntılarını getirir
- `delete_knowledge_base` - Bilgi tabanını siler
- `hybrid_search` - Hibrit arama
- `create_knowledge_from_url` - URL'den bilgi oluşturur
- `list_knowledge` - Bilgileri listeler
- `get_knowledge` - Bilgi ayrıntılarını getirir
- `delete_knowledge` - Bilgiyi siler
- `create_model` - Model oluşturur
- `list_models` - Modelleri listeler
- `get_model` - Model ayrıntılarını getirir
- `create_session` - Sohbet oturumu oluşturur
- `get_session` - Oturum ayrıntılarını getirir
- `list_sessions` - Oturumları listeler
- `delete_session` - Oturumu siler
- `chat` - Sohbet mesajı gönderir
- `list_chunks` - Bilgi parçalarını listeler
- `delete_chunk` - Bilgi parçasını siler

### Dosya yapısı
```
Rethra/mcp-server/
├── __init__.py              # Paket başlatma dosyası
├── main.py                  # Ana giriş noktası (önerilen)
├── run.py                   # Kolay başlatma betiği
├── run_server.py           # Özgün başlatma betiği
├── rethra_mcp_server.py   # MCP sunucu uygulaması
├── test_module.py          # Modül test betiği
├── requirements.txt        # Bağımlılık listesi
├── setup.py               # Kurulum betiği (geleneksel)
├── pyproject.toml         # Modern proje yapılandırması
├── MANIFEST.in            # Dahil edilen dosyalar listesi
├── LICENSE                # MIT lisansı
├── README.md              # Proje açıklaması
├── INSTALL.md             # Ayrıntılı kurulum kılavuzu
└── CHANGELOG.md           # Değişiklik günlüğü
```

### Başlatma yöntemleri
1. `python main.py` - Ana giriş noktası (önerilen)
2. `python run_server.py` - Özgün başlatma betiği
3. `python run.py` - Kolay başlatma betiği
4. `python rethra_mcp_server.py` - Doğrudan çalıştırma
5. `python -m rethra_mcp_server` - Modül olarak çalıştırma
6. `rethra-mcp-server` - Kurulumdan sonra komut satırı aracı
7. `rethra-server` - Kurulumdan sonra komut satırı aracı (takma ad)

### Teknik özellikler
- Model Context Protocol (MCP) 1.0.0+ tabanlı
- Asenkron G/Ç desteği
- Kapsamlı hata işleme
- Ayrıntılı günlük kaydı
- Ortam değişkeniyle yapılandırma
- Komut satırı parametresi desteği
- Birden çok kurulum yöntemi
- Geliştirme ve üretim modları
- Kapsamlı test kapsamı

### Bağımlılıklar
- Python 3.10+
- mcp >= 1.0.0
- requests >= 2.31.0

### Uyumluluk
- Windows, macOS ve Linux desteği
- Python 3.10-3.12 desteği
- Modern Python paket yönetim araçlarıyla uyumlu
