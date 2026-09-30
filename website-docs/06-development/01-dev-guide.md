# Geliştirme Kılavuzu

Geliştirme ortamı, bağımlı hizmetleri başlatmak için `docker-compose.dev.yml` kullanır ve Go arka ucunun, Web ön yüzünün ve docreader'ın bağımsız çalıştırılmasını destekler. Kod değişikliklerinden sonra ilgili modülde derleme ve testleri çalıştırın, ardından ilgili entegrasyon yollarını kontrol edin.

## Teknoloji Yığını ve Ortam Gereksinimleri {#teknoloji-yigini-ve-ortam-gereksinimleri}

Rethra, bağımsız olarak geliştirilebilen üç süreçten oluşur:

| Bileşen | Dizin | Dil / Çalışma Zamanı | Sürüm Gereksinimi (Kaynak) |
| --- | --- | --- | --- |
| Ana arka uç `app` | `cmd/server` + `internal/` | Go | **Go 1.26.0** (`go.mod` içindeki `go 1.26.0`), CGO gerektirir (DuckDB, sqlite-vec bağlamaları) |
| Belge ayrıştırma hizmeti `docreader` | `docreader/` | Python + gRPC | **Python >= 3.10.18** (`docreader/pyproject.toml` içindeki `requires-python`), bağımlılıklar **uv** ile yönetilir (depoda `uv.lock` bulunur, Docker içinde `uv sync --locked`) |
| Ön yüz `frontend` | `frontend/` | Node.js + Vue 3 | Node 22 serisi (`devDependencies`, `@tsconfig/node22` ve `@types/node ^22` içerir), Vite 7 + TypeScript ~6.0 + Vue 3.5 + TDesign, sürüm numarası `0.8.2` |

Ek olarak kurulması önerilen geliştirme araçları:

```bash
# Veritabanı migration CLI'si (scripts/migrate.sh buna bağlıdır)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Kod denetimi (make lint tarafından çağrılır)
# Kurulum için bkz. https://golangci-lint.run; depo kökünde .golangci.yml yapılandırması vardır

# Swagger dokümantasyonu üretimi (make docs tarafından çağrılır)
make install-swagger    # go install github.com/swaggo/swag/cmd/swag@latest

# Python bağımlılık yönetimi
pip install uv          # docreader bağımlılıkları uv sync ile kurar

# Docker + Docker Compose (v2 eklentisi veya bağımsız docker-compose olabilir; scripts/dev.sh otomatik algılar)
```

## Hızlı Başlangıç: Geliştirme Modu (Önerilen) {#hizli-baslangic-gelistirme-modu-onerilen}

Geliştirme modu, altyapıyı Docker içinde; `app` ve `frontend` bileşenlerini ise yerelde çalıştırır. Uygulama kodu değiştirildikten sonra bağımlılık imajlarını yeniden oluşturmaya gerek kalmadan süreçler doğrudan yeniden başlatılabilir. Giriş noktası `scripts/dev.sh` olup Makefile'daki `dev-*` hedefleri ilgili işlemleri kapsüller.

```bash
# 1. Ortam değişkenlerini hazırlayın: dev.sh önce .env dosyasını yükler (bulunması zorunludur), sonra .env.local ile üzerine yazar (isteğe bağlı)
cp .env.example .env

# 2. Altyapıyı başlatın (ParadeDB/Postgres + Redis + docreader, varsayılan olarak Langfuse da gelir)
make dev-start                      # ./scripts/dev.sh start ile aynıdır
make dev-start DEV_ARGS=--qdrant    # İsteğe bağlı profile ekler

# 3. Yeni bir terminalde backend'i yerelde çalıştırın (içeride go run -ldflags=... ./cmd/server çalışır)
make dev-app                        # ./scripts/dev.sh app ile aynıdır

# 4. Bir terminal daha açıp frontend'i yerelde çalıştırın (cd frontend && npm install && npm run dev)
make dev-frontend                   # ./scripts/dev.sh frontend ile aynıdır

# Diğer
make dev-status   # Konteyner durumunu gösterir
make dev-logs     # Günlükleri gösterir
make dev-stop     # Durdurur
make dev-restart  # Yeniden başlatır
```

Ön yüz dev server, `5173` portunu dinler (`frontend/vite.config.ts` içindeki `server.port: 5173`) ve `/api` ile `/files` isteklerini yerel arka uca (`DEV_PROXY_TARGET`) yönlendirir. `vite preview` (port `4173`), üretim derleme çıktılarıyla hizmet verir ve release imajına en yakın doğrulama ortamıdır.

### docker-compose.dev.yml Hizmet Listesi {#docker-compose-dev-yml-hizmet-listesi}

`docker-compose.dev.yml` yalnızca bağımlı hizmetleri içerir; `app`/`frontend` içermez. Varsayılan olarak başlatılan ve profile ile isteğe bağlı hizmetler aşağıdadır (profile, `dev.sh start` parametreleriyle etkinleştirilir):

| Hizmet | İmaj | Port (Varsayılan) | Başlatma Koşulu |
| --- | --- | --- | --- |
| `postgres` | `paradedb/paradedb:v0.22.6-pg17` (yerleşik pg_search/BM25) | `5432` | Varsayılan olarak başlatılır |
| `redis` | `redis:7.0-alpine` (`--requirepass`) | `6379` | Varsayılan olarak başlatılır |
| `docreader` | Yerelde `docker/Dockerfile.docreader` ile derlenir | `50051` (gRPC) | Varsayılan olarak başlar |
| `searxng` (+`searxng-init`) | `searxng/searxng:latest` | `127.0.0.1:8888` | `--searxng` / `--full` (compose profile `searxng`) |
| `minio` | `pgsty/minio:latest` | `9000` / konsol `9001` | `--minio` / `--full` |
| `qdrant` | `qdrant/qdrant:v1.16.2` | `6333` / `6334` | `--qdrant` / `--full` |
| `opensearch` | `opensearchproject/opensearch:3.3.2` (security kapalı, yalnızca HTTP) | `9200` | profile `opensearch` / `full` |
| `opensearch-dashboards` | `opensearchproject/opensearch-dashboards:3.3.0` | `5601` | profile `opensearch-ui` (gerektiğinde ayrı başlatılır) |
| `milvus` | `milvusdb/milvus:v2.6.11` (standalone, gömülü etcd) | `19530` / `9091` | profile `milvus` / `full` |
| `neo4j` | `neo4j:latest` (APOC eklentisi) | `7474` / `7687` | `--neo4j` / `--full` |
| `dex` | `dexidp/dex:latest` (OIDC test kimlik kaynağı, yapılandırma: `misc/dex-config.yaml`) | `5556` | `--dex` / `--full` |
| `langfuse-web` / `langfuse-worker` / `langfuse-clickhouse` / `langfuse-minio` / `langfuse-db-init` | Langfuse v3 kendi barındırılan yığını; dev postgres'i (ayrı `langfuse` veritabanı) ve redis'i (DB 1) yeniden kullanır | web `3000`, minio `9100/9101` | `--langfuse` (`dev.sh` varsayılan olarak etkinleştirir, `--no-langfuse` devre dışı bırakır) |
| `odl-hybrid` | Yerelde `docker/Dockerfile.odl-hybrid` ile derlenir (Docling PDF arka ucu) | `5002` | `--odl-hybrid` (imaj büyüktür, gerektiğinde) |
| `sandbox` | `rethra-sandbox` (Skills betik yürütme sanal alanı, yalnızca build/pull, sürekli çalışmaz) | - | profile `full` |

`dev.sh start` için isteğe bağlı parametreler: `--minio`, `--qdrant`, `--neo4j`, `--dex`, `--langfuse` (varsayılan olarak açık), `--no-langfuse`, `--odl-hybrid`, `--full` (odl-hybrid hariç tüm isteğe bağlı servisler). Makefile üzerinden parametre geçirme: `make dev-start DEV_ARGS=--odl-hybrid`.

### Docreader'ı yerelde ayrı çalıştırma {#docreader-i-yerelde-ayri-calistirma}

`dev-start`, docreader'ı varsayılan olarak bir kapsayıcıda çalıştırır; Python kodunda yerel hata ayıklama gerekirse:

```bash
cd docreader
uv sync                       # Bağımlılıkları uv.lock'a göre kurar (konteynerde uv sync --locked --no-dev)
uv run -m docreader.main      # gRPC servisini başlatır (Dockerfile CMD ile aynı), DOCREADER_GRPC_PORT üzerinde dinler (varsayılan 50051)
```

Docreader'ın çok sayıdaki ince ayar parametresi (PDF işleme DPI'ı, taranmış belge algılama, SSRF izin listesi, gRPC TLS vb.) `DOCREADER_*` ortam değişkenleriyle aktarılır; tam liste için `docker-compose.dev.yml` içindeki `docreader.environment` bölümüne bakın.

### Lite modu (sıfır harici bağımlılık) {#lite-modu-sifir-harici-bagimlilik}

Lite modu, SQLite'ı (+sqlite-vec) ve bellek içi kuyruğu tek bir ikili dosyada derler; tarayıcı üzerinden hızlı denemeler için uygundur:

```bash
make build-lite     # Önce frontend'i web/ dizinine derler, sonra Go'yu CGO ile derler (tags: sqlite_fts5); SKIP_FRONTEND=1 frontend'i atlar
make run-lite       # .env.lite gerektirir; Rethra-lite'ı derleyip başlatır
make package-lite   # tarball dağıtım paketi oluşturur (scripts/package-lite.sh)
```

## Makefile hedeflerine genel bakış {#makefile-hedeflerine-genel-bakis}

Aşağıdaki hedefler kök dizindeki `Makefile` içinde tanımlıdır; `make help` ayrıca bir Çince yardım içerir.

### Temel derleme ve çalıştırma {#temel-derleme-ve-calistirma}

| Hedef | İşlev |
| --- | --- |
| `build` | `go build -o Rethra ./cmd/server` |
| `run` | Önce `build`, ardından `./Rethra` çalıştırılır |
| `test` | `go test -v ./...` |
| `clean` | `go clean` çalıştırır ve ikili dosyayı siler |
| `build-prod` | Üretim derlemesi: CGO_ENABLED=1; `-ldflags "-w -s"`, Version/CommitID/BuildTime/GoVersion değerlerini (`internal/handler` paket değişkenleri) ekler ve protobuf `conflictPolicy=warn` ayarını yapar (qdrant/milvus proto çakışmalarını önler) |
| `fmt` | `go fmt ./...` |
| `lint` | `golangci-lint run` |
| `deps` | `go mod download` |
| `docs` | Swagger belgelerini oluşturmak için `swag init -g ./cmd/server/main.go -o ./docs --parseDependency --parseInternal` çalıştırır |
| `install-swagger` | `swag` CLI'ını kurar |

### Docker İmaj ve Hizmet Yönetimi {#docker-imaj-ve-hizmet-yonetimi}

| Hedef | İşlev |
| --- | --- |
| `docker-build-app` | `rethra-app` oluşturur (`docker/Dockerfile.app`, `scripts/get_version.sh` sürüm bilgisi eklenir) |
| `docker-build-docreader` | `rethra-docreader` oluşturur (`docker/Dockerfile.docreader`) |
| `docker-build-frontend` | Çok aşamalı olarak `rethra-ui` oluşturur (builder içinde `npm ci` + `npm run build`; ana makinede dist önceden oluşturulmasına gerek yoktur; `VITE_FRONTEND_COMMIT` otomatik olarak git'ten eklenir) |
| `docker-build-all` | Yukarıdaki üç imaj |
| `docker-run` | `.env` dosyasının mevcut olmasını sağlar (eksikse `.env.example` dosyasından kopyalar veya touch uygular), ardından `docker-compose up` çalıştırır |
| `docker-stop` / `docker-restart` | `docker-compose down` / `stop -t 60` + `up` |
| `start-all` / `stop-all` | `scripts/start_all.sh` (tüm hizmetleri tek komutla başlatır/durdurur) |
| `build-images` / `build-images-app` / `build-images-docreader` / `build-images-frontend` / `clean-images` | `scripts/build_images.sh` kaynak koddan imaj oluşturur/imajları temizler |
| `check-env` / `list-containers` / `pull-images` | `start_all.sh --check / --list / --pull` |
| `show-platform` | `uname -m` ile Docker oluşturma platformunu gösterir (amd64/arm64 otomatik algılanır) |
| `clean-db` | Üç Docker volume'u, `rethra_postgres-data` / `rethra_minio_data` / `rethra_redis_data`, siler (**verileri temizler**) |

### Veritabanı Geçişleri (ayrıntılar için «Veritabanı ve Geçişler» bölümüne bakın) {#veritabani-gecisleri-ayrintilar-icin-veritabani-ve-gecisler-bolumune-bakin}

| Hedef | İşlev |
| --- | --- |
| `migrate-up` / `migrate-down` | `scripts/migrate.sh up / down` |
| `migrate-version` | Mevcut geçiş sürümünü görüntüler |
| `migrate-create name=xxx` | Bir çift yeni geçiş dosyası oluşturur |
| `migrate-force version=N` | Sürümü zorla ayarlar (dirty state kurtarma) |
| `migrate-goto version=N` | Belirtilen sürüme geçiş yapar |

### Geliştirme Modu ve Lite {#gelistirme-modu-ve-lite}

| Hedef | İşlev |
| --- | --- |
| `dev-start` / `dev-stop` / `dev-restart` / `dev-logs` / `dev-status` | `scripts/dev.sh start/stop/restart/logs/status` (`DEV_ARGS` ile profile parametreleri desteklenir) |
| `dev-app` | Yerel `go run ./cmd/server` (sürüm ldflags ile) |
| `dev-frontend` | Yerel `npm run dev` |
| `build-lite` / `run-lite` / `package-lite` | Lite modu oluşturma/çalıştırma/paketleme (bkz. 2.3) |
| `download_spatial` | DuckDB spatial uzantısını indirmek için `go run cmd/download/duckdb/duckdb.go` çalıştırır (veri analizi araçları için) |

### Model Sağlayıcıları Dizini {#model-saglayicilari-dizini}

| Hedef | İşlev |
| --- | --- |
| `model-catalog-generate` | `python3 scripts/model-catalog/generate.py`; sağlayıcı kataloğunun meta verilerini ve protokol kapsamını yeniden üretir |
| `model-catalog-check` | Üretim sonucunun güncel olup olmadığını doğrular ve `go test ./internal/models/...` çalıştırır (CI de doğrular) |
| `model-catalog-diff` | models.dev çıktısındaki model meta verisi fark raporunu karşılaştırır; yalnızca manuel inceleme içindir, dosyalara geri yazmaz; tek bir sağlayıcıyı görmek için `VENDOR=deepseek` kullanılabilir |

Sağlayıcı entegrasyon yöntemleri için [Genişletme noktaları](03-extension-points.md) bölümüne bakın.

## Test sistemi {#test-sistemi}

### Go birim testleri (ana modül) {#go-birim-testleri-ana-modul}

```bash
make test          # go test -v ./...
# Veya pakete göre çalıştırın:
go test ./internal/infrastructure/chunker/...
go test -run TestXxx ./internal/application/service/...
```

Ana modül testleri, `go.mod` içinde görülebileceği üzere `go-sqlmock`, `miniredis` ve benzeri bellek içi taklitleri yaygın olarak kullanır; çoğu gerçek bir veritabanı olmadan çalışabilir. Bazı paketler CGO'ya bağlıdır (DuckDB/sqlite-vec).

### docreader testleri (Python) {#docreader-testleri-python}

Testler `docreader/tests/` altında bulunur ve standart kitaplıktaki `unittest` ile yazılmıştır (dosya içindeki `unittest.main()`); ayrıştırma yönlendirmesi, eşzamanlılık, EPUB/Excel/MHTML/PDF ayrıştırması, SSRF koruması ve benzerlerini kapsar:

```bash
cd docreader
uv sync
uv run python -m unittest discover -s tests -v      # Tümü
uv run python -m unittest tests.test_parser_routing  # Tek test
```

### Ön yüz testleri {#on-yuz-testleri}

`cd frontend && npm run type-check` (vue-tsc) ve `npm test` (`tsx --test`, Node test çalıştırıcısı).

## Kod standartları ve gönderim süreci {#kod-standartlari-ve-gonderim-sureci}

### Go kod standartları {#go-kod-standartlari}

Depo kökündeki `.golangci.yml` (golangci-lint v2 yapılandırma biçimi):

```yaml
version: 2
linters-settings:
  lll:
    line-length: 120
    tab-width: 4
linters:
  enable:
    - lll           # Satır genişliğini denetler (120 sütun)
    - govet
    - revive
formatters:
  enable:
    - gofmt
    - gofumpt
```

Göndermeden önce şunların çalıştırılması önerilir:

```bash
make fmt && make lint && make test
```

Biçimlendirme standardının **gofumpt** olduğuna dikkat edin (gofmt'ten daha katıdır); satır genişliği üst sınırı 120'dir.

Depoyla gelen Git hook'ları da kurulabilir (`./scripts/install-git-hooks.sh`, `core.hooksPath` değerini `scripts/git-hooks` konumuna yönlendirir): pre-commit boşluk karakterlerini denetler, otomatik gofmt uygular ve kuruluysa golangci-lint çalıştırır; pre-push, CI ile uyumlu olarak gofmt ile `go vet` / `go test` / `go build` çalıştırır. Geçici olarak atlamak için `SKIP_HOOKS=1`, yalnızca pre-push testlerini atlamak için `HOOK_SKIP_TEST=1` ayarlanabilir.

Üçüncü taraf bağımlılıklar eklenirken veya yükseltilirken ya da dağıtım çıktılarıyla gönderilen veri dosyaları değiştirilirken `THIRD_PARTY_NOTICES.md` ve `licenses/` eşzamanlı güncellenmeli, ayrıca `scripts/check-license-bundle.sh` ile kontrol edilmelidir (`app.yml` de aynı denetimi çalıştırır).

### CI ve gönderim süreci {#ci-ve-gonderim-sureci}

`.github/` altındaki gerçek yapılandırmalar:

| Dosya | Tetikleme yolları | İşlev |
| --- | --- | --- |
| `workflows/app.yml` | Kök modül Go kodu, `go.mod`, `config/`, `migrations/`, `scripts/`, `docker/Dockerfile.app`, lisans dosyaları, model kataloğu verileri | Ana modül denetimleri: gofmt biçim doğrulaması (yalnızca PR içindeki gönderimler için), `go vet`, `go test`, `go build ./cmd/server`; ayrıca üçüncü taraf lisans paketini, model sağlayıcı kataloğu üretim sonucunu ve Git hook testlerini doğrular |
| `workflows/go-lint.yml` / `go-lint-cache.yml` | PR / main | golangci-lint yalnızca PR'nin birleştirme tabanına göre eklenen yeni sorunları raporlar; `go-lint-cache.yml`, main üzerinde önbelleği önceden hazırlar |
| `workflows/frontend.yml` | `frontend/`, `scripts/build_frontend_dist.sh` | Node 24: `npm test` + `npm run type-check` + `npm run build`; ayrıca `frontend/Dockerfile` çok aşamalı imajı oluşturulur (gönderilmez) ve imaj içindeki gömülü sayfa ile MCP vekil rotaları doğrulanır (`scripts/test_embed_nginx.py`) |
| `workflows/docreader.yml` | `docreader/`, `testdata/`, `packages/`, ilgili Dockerfile'lar | uv ile bağımlılıkları kur → `compileall` → `unittest discover docreader/tests`; ardından docreader gRPC hizmetini başlatıp `go test ./docreader/client ./docreader/proto` çalıştır |
| `workflows/mcp-server.yml` | `mcp-server/` | Python 3.10-3.13 matris testleri; main'e birleştirildikten sonra `pyproject.toml` içindeki sürüm numarasıyla PyPI Trusted Publishing kullanılarak otomatik yayımlanır (sürüm zaten varsa yükleme atlanır, etiket eklemeye bağlı değildir) |
| `workflows/docker-image.yml` | — | Docker imajı oluşturma ve yayımlama |
| `workflows/release-lite.yml` | — | Lite sürümü yayımlama |
| `workflows/anydoc.yml` | `third_party/anydoc-go/`, `internal/infrastructure/docparser/` | Süreç içi Office ayrıştırma motoru oluşturma ve testleri |
| `pull_request_template.md` | — | PR şablonu |
| `ISSUE_TEMPLATE/` | — | Issue şablonu |
| `dependabot.yml` | — | Bağımlılık yükseltme botu |

Yola göre tetiklenen dört kontrol (app / frontend / docreader / mcp-server) ana modülleri kapsar, ancak önce yerelde çalıştırmak yine de daha fazla zaman kazandırır. Ön yüz için doğrudan `scripts/verify_frontend_pr.sh` kullanılabilir; bu betik CI ile aynı sırada `npm test` → `npm run type-check` → `npm run build` çalıştırır.

Gönderim akışı: fork / dal → yerelde `fmt + lint + test` → PR (şablona göre doldurulur) → ilgili yolların tetiklediği CI.

## Hata ayıklama ipuçları {#hata-ayiklama-ipuclari}

### Günlük seviyesi {#gunluk-seviyesi}

Günlük kaydı `internal/logger/logger.go` içinde uygulanır (logrus). Seviye, `LOG_LEVEL` ortam değişkeniyle denetlenir; değerler `debug` / `info` / `warn` (`warning`) / `error` / `fatal` olabilir. Ayarlanmadığında veya geçersiz olduğunda varsayılan **debug** kullanılır (`getLogLevelFromEnv()`). `LOG_PATH` çıktı yolunu denetler; ikisi de `main()` içinde `.env` yüklendikten sonra hemen etkili olur. docreader tarafı da `LOG_LEVEL` okur (compose içinde aktarılır).

Her istek, app ve docreader günlükleri boyunca `X-Request-ID` taşır (docreader'da `init_logging_request_id`); sorun araştırırken önce request id alın.

### GIN_MODE ve Swagger {#gin-mode-ve-swagger}

- `GIN_MODE=release` olduğunda Swagger UI devre dışı bırakılır (`internal/router/router.go`) ve embed channel'ın güvenlik davranışını etkiler. Yerel kaynak kodu geliştirmede `debug` olarak ayarlanabilir; Docker Compose'da bu değişken ayarlanmadığında varsayılan olarak `release` kullanılır.
- Swagger, **app arka ucu** tarafından sunulur. Varsayılan adres `http://localhost:8080/swagger/index.html` şeklindedir; Compose içinde `APP_PORT` değiştirilirse bu ana makine eşleme bağlantı noktası kullanılmalıdır. Ön yüz bağlantı noktası (varsayılan `80`) ve Vite geliştirme bağlantı noktası (varsayılan `5173`) için `/swagger/` vekili yapılandırılmamıştır.
- Yayımlanan imaj Swagger belgelerini zaten içerir; yalnızca arayüz açıklamaları değiştirildiğinde, belgelerin yeniden oluşturulması ve arka ucun derlenmesi gerektiğinde `make docs` çalıştırmak gerekir.

#### Docker Compose erişim adımları

Proje `.env` dosyasında `GIN_MODE=debug` ayarlayın, ardından ortam değişkeninin etkili olması için app kapsayıcısını yeniden oluşturun (`docker compose restart app` tek başına kapsayıcı ortamını güncellemez):

```bash
docker compose up -d --no-deps --force-recreate app
docker compose exec app printenv GIN_MODE
docker compose port app 8080
```

İlk kontrol çıktısının `debug` olduğunu doğrulayın, ardından ikinci kontrolün gösterdiği eşlenmiş bağlantı noktasıyla arka uca erişin. Örneğin çıktı `0.0.0.0:8080` ise yerel makineden `http://localhost:8080/swagger/index.html` adresine erişin; başka bir makineden erişirken `localhost` yerine dağıtım makinesinin adresini kullanın. Sorun giderme tamamlandıktan sonra `GIN_MODE` değerini `release` olarak geri yükleyin ve app kapsayıcısını yeniden oluşturun.

#### Açıldıktan sonra boş görünmesi veya yükleme hatası

Önce isteğin arka uca ulaşıp ulaşmadığını kontrol edin (aşağıda varsayılan arka uç bağlantı noktası olan `8080` örnek olarak kullanılmıştır):

```bash
curl -i http://localhost:8080/swagger/index.html
curl -i http://localhost:8080/swagger/doc.json
```

Normal koşullarda her ikisi de `200` döndürür: ilki `swagger-ui` içeren HTML, ikincisi ise `swagger` ve `paths` alanlarını içeren JSON'dur.

| Belirti | İnceleme yönü |
| --- | --- |
| Durum kodu `200`, ancak HTML normal bir ön uç sayfası (`<div id="app">` içerir); sayfa boş veya giriş sayfasına yönleniyor | İstek, ön ucun SPA geri dönüşüne düşmüştür. app arka uç eşleme portunu kullanın; giriş ön ucu bunun için `/swagger/` proxy'si eklemez. |
| Arka uç `401` veya `404` döndürüyor, Swagger UI yok | Erişim portunu ve kapsayıcı içindeki gerçek `GIN_MODE` değerini doğrulayın; `release` modu Swagger rotalarını kaydetmez, eşleşmeyen istekler kimlik doğrulama ara yazılımına girebilir. |
| Swagger UI görünüyor, ancak API tanımının yüklenemediğini bildiriyor | Aynı arka uçtaki `/swagger/doc.json` yanıtını ayrı olarak kontrol edin; özel bir ters proxy kullanıyorsanız, `/swagger/` yolunun tamamının ve statik kaynakların app'e iletildiğinden emin olun. |

### Veritabanı ve migration hata ayıklama {#veritabani-ve-migration-hata-ayiklama}

- `AUTO_MIGRATE=false`, başlangıçta otomatik geçişi kapatabilir; `AUTO_RECOVER_DIRTY` (varsayılan olarak açıktır, kapatmak için `false` ayarlanır) dirty state otomatik kurtarmayı kontrol eder (`internal/container/container.go`). Geçiş hataları yalnızca uyarı verir ve başlatmayı engellemez; başlangıç günlüklerindeki `Database migration failed` kaydına dikkat edin.
- Şema sürümünü hızlıca doğrulamak için `make migrate-version` kullanın.

### LLM çağrı zinciri gözlemi (Langfuse) {#llm-cagri-zinciri-gozlemi-langfuse}

`dev.sh start`, varsayılan olarak kendi barındırılan Langfuse'u (`http://localhost:3000`) başlatır. Yerel `go run` ile çalışan app'in dışa aktarması gerekir:

```bash
export LANGFUSE_HOST=http://localhost:3000
export LANGFUSE_PUBLIC_KEY=pk-lf-xxx
export LANGFUSE_SECRET_KEY=sk-lf-xxx
```

Böylece Langfuse UI içinde her oturumun model çağrısı trace'ini görüntüleyebilirsiniz (belge işleme span'i de `knowledge_processing_spans` tablosuna kaydedilir ve ön uçta görselleştirilir).

### pprof {#_6-5-pprof}

Mevcut kodda **yerleşik** `net/http/pprof` uç noktası yoktur (`internal/` ve `cmd/` altında pprof referansı bulunmaz). Performans profilleme gerekirse, geçici olarak `cmd/server/main.go` içine `import _ "net/http/pprof"` ekleyip bağımsız bir `http.ListenAndServe("localhost:6060", nil)` başlatabilir veya belirli paketleri profillemek için `go test -bench . -cpuprofile` kullanabilirsiniz.

### Parçalama stratejisi tanılama {#parcalama-stratejisi-tanilama}

Parçalama stratejisi geri dönüş sorunlarını incelemek için `LOG_LEVEL=debug` kullanarak `chunker: tier %s rejected` günlüklerini görüntüleyin.

### Bulut imajı bakım betikleri {#cloud-image-scripts}

`scripts/cloud-image/`, özel ve gözden çıkarılabilir bir Linux imaj oluşturma makinesinde dağıtım imajı hazırlamak için kullanılır. Normal dağıtımlar [Kurulum ve dağıtım](../01-getting-started/02-installation.md) kullanır; bu betiklerin çalıştırılması gerekmez.

- `prepare.sh`, `RETHRA_REF` ile eşleşen çalışma dosyalarını indirir, imajları çeker ve systemd hizmetini kurar. Bu referans imaj sürümünü ayarlamak için de kullanılır; karşılık gelen imaj etiketinin bulunduğu doğrulanmalıdır. Depoyla sağlanan systemd birimi sabit olarak `/opt/Rethra` kullanır; yalnızca betikteki dizin değişkenini değiştirmek kurulum konumunu taşımak için yeterli değildir.
- `cleanup.sh`, imaj oluşturma makinesindeki verileri, anahtarları, SSH yetkilerini, günlükleri ve Docker önbelleğini temizler ve makineyi kapatır; etkisi Rethra diziniyle sınırlı değildir, yalnızca incelenmiş özel oluşturma makinelerinde kullanılmalıdır.
- `firstboot.sh`, yeni örnekte anahtarlar oluşturur, `.env` dosyasını yazar ve Compose'u başlatır; tamamlandıktan sonra ilk başlatma hizmetini devre dışı bırakır, betiği kendi kendine silmez.

İmaj oluşturmadan önce betiklerin seçilen sürümle eşleştiği kontrol edilmelidir. Dağıtımdan önce başlatma, hizmet sağlığı ve anahtar bağımsızlığı yeni bir örnekte doğrulanmalıdır; bu açıklama, mevcut betiklerin belirli bir bulut platformunda dağıtım veya listeleme kabulünden geçtiği anlamına gelmez.

İlk başlatma sorunlarını gidermek için `journalctl -u rethra-firstboot`, `/var/log/rethra-firstboot.log` ve `/opt/Rethra` altındaki Compose durumunu inceleyin. `.firstboot.done`, anahtarlar oluşturulduktan sonra ve Compose başlatılmadan önce yazılır; işaretin bulunması hizmetin sağlıklı olduğu anlamına gelmez. Başlatma başarısız olursa oluşturulmuş `.env` ve işareti koruyun, nedeni düzelttikten sonra Compose'u yeniden başlatın; anahtarları yeniden oluşturmak için işareti silmek, yapılandırma ile önceden başlatılmış veritabanı arasında tutarsızlığa neden olabilir.
