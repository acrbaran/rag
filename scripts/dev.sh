#!/bin/bash
# Geliştirme ortamı başlatma betiği - yalnızca altyapıyı başlatır; app ve frontend yerelde manuel olarak çalıştırılmalıdır

# Rengi ayarla
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # Renk yok

# Proje kök dizinini al
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Günlük fonksiyonu
log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

# Kullanılabilir Docker Compose komutunu seç
DOCKER_COMPOSE_BIN=""
DOCKER_COMPOSE_SUBCMD=""

detect_compose_cmd() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_BIN="docker"
        DOCKER_COMPOSE_SUBCMD="compose"
        return 0
    fi
    if command -v docker-compose &> /dev/null; then
        if docker-compose version &> /dev/null; then
            DOCKER_COMPOSE_BIN="docker-compose"
            DOCKER_COMPOSE_SUBCMD=""
            return 0
        fi
    fi
    return 1
}

# Yardım bilgilerini göster
show_help() {
    printf "%b\n" "${GREEN}Rethra geliştirme ortamı betiği${NC}"
    echo "Kullanım: $0 [komut] [seçenek]"
    echo ""
    echo "Komutlar:"
    echo "  start      Temel altyapı hizmetlerini başlatır (postgres, redis, docreader, langfuse)"
    echo "  stop       Tüm hizmetleri durdurur"
    echo "  restart    Tüm hizmetleri yeniden başlatır"
    echo "  logs       Hizmet günlüklerini görüntüler"
    echo "  status     Hizmet durumunu görüntüler"
    echo "  app        Arka uç uygulamasını başlatır (yerel olarak çalışır)"
    echo "  frontend   Ön uç geliştirme sunucusunu başlatır (yerel olarak çalışır)"
    echo "  help       Bu yardım bilgisini görüntüler"
    echo ""
    echo "İsteğe bağlı Profiller (start komutu için):"
    echo "  --minio       MinIO nesne depolamasını başlatır"
    echo "  --qdrant      Qdrant vektör veritabanını başlatır"
    echo "  --neo4j       Neo4j grafik veritabanını başlatır"
    echo "  --dex         Dex'i başlat (OIDC kimlik doğrulama)"
    echo "  --langfuse    Langfuse'u başlat (varsayılan olarak açık)"
    echo "  --no-langfuse Langfuse'u başlatma"
    echo "  --odl-hybrid  OpenDataLoader hybrid'i başlat (Docling; imaj büyüktür, gerektiğinde etkinleştirin)"
    echo "  --full        Tüm isteğe bağlı hizmetleri başlatır (odl-hybrid hariçtir, ayrıca --odl-hybrid eklenmelidir)"
    echo ""
    echo "Örnekler:"
    echo "  $0 start                    # Temel hizmetleri başlatır"
    echo "  $0 start --qdrant           # Temel servisleri + Qdrant'ı başlat"
    echo "  $0 start --dex             # Temel servisleri + Dex'i başlat"
    echo "  $0 start --odl-hybrid       # Temel servisleri + OpenDataLoader hybrid'i başlat"
    echo "  $0 start --full             # Tüm hizmetleri başlatır"
    echo "  make dev-start DEV_ARGS=--odl-hybrid   # Yukarıdakiyle aynı (Makefile parametre aktarımı)"
    echo "  $0 app                      # Arka ucu başka bir terminalde başlatır"
    echo "  $0 frontend                 # Ön yüzü başka bir terminalde başlat"
}

# .env ve isteğe bağlı .env.local dosyasını yükle (ikincisi birincinin üzerine yazar)
# Okurken satır sonundaki \r karakterini kaldır; Windows tarzı (CRLF) satır sonlarıyla uyumlu olsun,
# aksi takdirde bash source kalan \r karakterini komut olarak yorumlar ve "...: $'\r': command not found" hatasına yol açar.
# Not: source <(sed ...) kullanılamaz — macOS ile gelen Bash 3.2, process substitution içindeki
# source işlemiyle değişkenleri geçerli shell'e aktaramaz; önce seek edilebilir geçici bir dosyaya yazıp ardından source kullanılmalıdır.
_source_env_file() {
    local src="$1"
    local tmp
    tmp="$(mktemp)" || return 1
    sed -e 's/\r$//' "$src" > "$tmp"
    set -a
    # shellcheck source=/dev/null
    source "$tmp"
    set +a
    rm -f "$tmp"
}

load_env_files() {
    if [ -f ".env" ]; then
        _source_env_file .env || return 1
    else
        return 1
    fi

    if [ -f ".env.local" ]; then
        log_info ".env.local geçersiz kılma yapılandırması yükleniyor..."
        _source_env_file .env.local || return 1
    fi
    return 0
}

# Docker'ı kontrol et
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker kurulu değil, lütfen önce Docker'ı kurun"
        return 1
    fi
    
    if ! detect_compose_cmd; then
        log_error "Docker Compose algılanmadı"
        return 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker hizmeti çalışmıyor"
        return 1
    fi
    
    return 0
}

# .env dosyasının hybrid modunu etkinleştirip etkinleştirmediğini kontrol et (--odl-hybrid başlatıldıktan sonra docreader'ı yeniden oluşturmak için)
_should_enable_odl_hybrid_from_env() {
    local hybrid="${DOCREADER_ODL_HYBRID:-off}"
    hybrid=$(printf '%s' "$hybrid" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')
    case "$hybrid" in
        off|"") return 1 ;;
        *) return 0 ;;
    esac
}

_enable_odl_hybrid_profile() {
    PROFILES="$PROFILES --profile odl-hybrid"
    ENABLED_SERVICES="$ENABLED_SERVICES odl-hybrid"
}

# odl-hybrid HTTP sağlık kontrolünün geçmesini bekle (compose başlatıldıktan sonra servis bağımlılıkları hâlâ çekiyor olabilir)
_wait_odl_hybrid_ready() {
    local port="${ODL_HYBRID_PORT:-5002}"
    local max_wait="${ODL_HYBRID_STARTUP_WAIT_SEC:-180}"
    local waited=0
    local interval=5

    if ! command -v curl &> /dev/null; then
        log_warning "curl yüklü değil, odl-hybrid hazır olma beklemesi atlanıyor; lütfen http://localhost:${port}/health adresini manuel olarak kontrol edin"
        return 0
    fi

    log_info "odl-hybrid'in hazır olması bekleniyor (en fazla ${max_wait}s; ilk seferde imaj derlenmeli: docker compose ... build odl-hybrid)..."
    while [ "$waited" -lt "$max_wait" ]; do
        if curl -sf "http://127.0.0.1:${port}/health" >/dev/null 2>&1; then
            log_success "odl-hybrid hazır (http://localhost:${port}/health)"
            return 0
        fi
        sleep "$interval"
        waited=$((waited + interval))
    done
    log_warning "odl-hybrid ${max_wait}s içinde hazır olmadı, lütfen şuna bakın: docker logs Rethra-odl-hybrid"
    return 1
}

# Altyapı hizmetlerini başlat
start_services() {
    log_info "Geliştirme ortamı altyapı hizmetleri başlatılıyor..."
    
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"
    
    # .env dosyasını kontrol et
    if [ ! -f ".env" ]; then
        log_error ".env dosyası mevcut değil, lütfen önce oluşturun"
        return 1
    fi

    load_env_files
    if [ $? -ne 0 ]; then
        log_error ".env dosyası mevcut değil, lütfen önce oluşturun"
        return 1
    fi

    if [ -n "${DEV_REMOTE_HOST:-}" ]; then
        log_warning "DEV_REMOTE_HOST=${DEV_REMOTE_HOST} yapılandırıldı, yerel Docker altyapısının başlatılması atlanıyor"
        log_info "Uzak hizmetler: PostgreSQL/Redis/DocReader/Langfuse → ${DEV_REMOTE_HOST}"
        log_info "Sonraki adımlar: make dev-app（yerel arka uç） veya make dev-frontend（frontend）"
        return 0
    fi
    
    # profile parametresini ayrıştır
    shift  # "start" komutunun kendisini kaldır
    # Varsayılan olarak altyapıyı (postgres / redis / docreader) + langfuse başlat,
    # Diğer isteğe bağlı hizmetleri gerektiğinde --minio / --qdrant / --neo4j / --dex / --full ile etkinleştir.
    PROFILES="--profile langfuse"
    ENABLED_SERVICES="langfuse"
    while [ $# -gt 0 ]; do
        case "$1" in
            --minio)
                PROFILES="$PROFILES --profile minio"
                ENABLED_SERVICES="$ENABLED_SERVICES minio"
                ;;
            --qdrant)
                PROFILES="$PROFILES --profile qdrant"
                ENABLED_SERVICES="$ENABLED_SERVICES qdrant"
                ;;
            --neo4j)
                PROFILES="$PROFILES --profile neo4j"
                ENABLED_SERVICES="$ENABLED_SERVICES neo4j"
                ;;
            --dex)
                PROFILES="$PROFILES --profile dex"
                ENABLED_SERVICES="$ENABLED_SERVICES dex"
                ;;
            --langfuse)
                PROFILES="$PROFILES --profile langfuse"
                ENABLED_SERVICES="$ENABLED_SERVICES langfuse"
                ;;
            --no-langfuse)
                PROFILES="${PROFILES//--profile langfuse/}"
                ENABLED_SERVICES="${ENABLED_SERVICES//langfuse/}"
                ;;
            --odl-hybrid)
                if [[ "$ENABLED_SERVICES" != *"odl-hybrid"* ]]; then
                    _enable_odl_hybrid_profile
                fi
                ;;
            --full)
                PROFILES="--profile full"
                ENABLED_SERVICES="minio qdrant neo4j dex"
                break
                ;;
            *)
                log_warning "Bilinmeyen parametre: $1"
                ;;
        esac
        shift
    done

    # Hizmetleri başlat (odl-hybrid ayrı olarak --build; docreader'ın her seferinde yeniden derlenmesini önler)
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml $PROFILES up -d
    local compose_rc=$?
    if [ "$compose_rc" -eq 0 ] && [[ "$ENABLED_SERVICES" == *"odl-hybrid"* ]]; then
        log_info "odl-hybrid imajı oluşturuluyor/güncelleniyor..."
        "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml $PROFILES up -d --build odl-hybrid
        _wait_odl_hybrid_ready || true
        # docreader'ın DOCREADER_ODL_HYBRID değerini okuması gerekir; .env az önce değiştirildiyse ortam değişkenlerini eklemek için yeniden derlemeyi zorla
        if _should_enable_odl_hybrid_from_env; then
            log_info "DOCREADER_ODL_HYBRID=${DOCREADER_ODL_HYBRID} uygulamak için docreader yeniden oluşturuluyor ..."
            "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml up -d --force-recreate docreader
        fi
    fi

    if [ "$compose_rc" -eq 0 ]; then
        log_success "Altyapı hizmetleri başlatıldı"
        echo ""
        log_info "Hizmet erişim adresleri:"
        echo "  - PostgreSQL:    localhost:5432"
        echo "  - Redis:         localhost:6379"
        echo "  - DocReader:     localhost:50051"
        
        # Etkinleştirilen profile göre ek hizmetleri göster
        if [[ "$ENABLED_SERVICES" == *"minio"* ]]; then
            echo "  - MinIO:         localhost:9000 (Console: localhost:9001)"
        fi
        if [[ "$ENABLED_SERVICES" == *"qdrant"* ]]; then
            echo "  - Qdrant:        localhost:6333 (gRPC: localhost:6334)"
        fi
        if [[ "$ENABLED_SERVICES" == *"neo4j"* ]]; then
            echo "  - Neo4j:         localhost:7474 (Bolt: localhost:7687)"
        fi
        if [[ "$ENABLED_SERVICES" == *"dex"* ]]; then
            echo "  - Dex:           localhost:5556"
        fi
        if [[ "$ENABLED_SERVICES" == *"langfuse"* ]]; then
            echo "  - Langfuse:      http://localhost:${LANGFUSE_WEB_PORT:-3000}"
        fi
        if [[ "$ENABLED_SERVICES" == *"odl-hybrid"* ]]; then
            echo "  - ODL Hybrid:    http://localhost:${ODL_HYBRID_PORT:-5002} (health: /health)"
            echo "                   docreader, DOCREADER_ODL_HYBRID=docling-fast gerektirir"
        fi
        
        echo ""
        log_info "Sonraki adımlar:"
        printf "%b\n" "${YELLOW}1. Yeni bir terminalde arka ucu çalıştırın:${NC} make dev-app"
        printf "%b\n" "${YELLOW}2. Ön yüzü yeni bir terminalde çalıştırın:${NC} make dev-frontend"
        return 0
    else
        log_error "Hizmet başlatılamadı"
        return 1
    fi
}

# Hizmetleri durdur
stop_services() {
    log_info "Geliştirme ortamı servisleri durduruluyor..."
    
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml down
    
    if [ $? -eq 0 ]; then
        log_success "Tüm servisler durduruldu"
        return 0
    else
        log_error "Servisler durdurulamadı"
        return 1
    fi
}

# Hizmetleri yeniden başlat
restart_services() {
    stop_services
    sleep 2
    start_services
}

# Günlükleri görüntüle
show_logs() {
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml logs -f
}

# Durumu görüntüle
show_status() {
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml ps
}

# Uzak geliştirme modunda altyapı portlarına erişilip erişilemediğini kontrol et
check_remote_dev_connectivity() {
    local host="${DEV_REMOTE_HOST:-}"
    if [ -z "$host" ]; then
        return 0
    fi

    local db_port="${DB_PORT:-5432}"
    local redis_port
    redis_port="${REDIS_ADDR#*:}"
    if [ "$redis_port" = "$REDIS_ADDR" ]; then
        redis_port=6379
    fi
    local docreader_port="${DOCREADER_PORT:-50051}"

    log_info "Uzak altyapı bağlantısı kontrol ediliyor (${host})..."
    local failed=0
    for spec in "PostgreSQL:${host}:${db_port}" "Redis:${host}:${redis_port}" "DocReader:${host}:${docreader_port}"; do
        local name="${spec%%:*}"
        local rest="${spec#*:}"
        local h="${rest%%:*}"
        local p="${rest##*:}"
        if command -v nc &> /dev/null; then
            if nc -z -w 3 "$h" "$p" 2>/dev/null; then
                log_success "${name} ${h}:${p} erişilebilir"
            else
                log_error "${name} ${h}:${p} erişilemez (no route / connection refused)"
                failed=1
            fi
        else
            log_warning "nc kurulu değil, ${name} bağlantı kontrolü atlanıyor"
        fi
    done

    if [ "$failed" -ne 0 ]; then
        echo ""
        log_error "Uzak geliştirme ortamına bağlanılamıyor ${host}"
        log_info "Sorun giderme önerileri:"
        echo "  1. Uzak makinedeki Docker kapsayıcılarının çalıştığını doğrulayın (postgres/redis/docreader)"
        echo "  2. Yerel makinenin ${host} ile aynı yerel ağda olduğunu doğrulayın (yerel makine: $(ipconfig getifaddr en0 2>/dev/null || echo 'bilinmiyor'))"
        echo "  3. Uzak makinede port eşlemelerini kontrol edin: docker ps --format 'table {{.Names}}\t{{.Ports}}'"
        echo "  4. Uzak güvenlik duvarının 5432/6379/50051 portlarına izin verip vermediğini kontrol edin"
        return 1
    fi
    return 0
}

# Host-platform path of the anydoc static archive (built by `make anydoc-lib`).
anydoc_host_archive() {
    case "$(uname -s)-$(uname -m)" in
        Darwin-arm64) echo "$PROJECT_ROOT/third_party/anydoc-go/lib/darwin_arm64/libanydoc_go.a" ;;
        Darwin-x86_64) echo "$PROJECT_ROOT/third_party/anydoc-go/lib/darwin_amd64/libanydoc_go.a" ;;
        Linux-x86_64)
            if [ -f "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_amd64_gnu/libanydoc_go.a" ]; then
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_amd64_gnu/libanydoc_go.a"
            else
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_amd64_musl/libanydoc_go.a"
            fi
            ;;
        Linux-aarch64)
            if [ -f "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_arm64_gnu/libanydoc_go.a" ]; then
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_arm64_gnu/libanydoc_go.a"
            else
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_arm64_musl/libanydoc_go.a"
            fi
            ;;
        *) echo "" ;;
    esac
}

# Enable the in-process anydoc engine when the archive is present, unless the
# caller already set GO_BUILD_TAGS (including empty, which opts out).
enable_anydoc_build_tag() {
    if [ -n "${GO_BUILD_TAGS+x}" ]; then
        export GO_BUILD_TAGS
        return
    fi
    local archive
    archive="$(anydoc_host_archive)"
    if [ -n "$archive" ] && [ -f "$archive" ]; then
        export GO_BUILD_TAGS=anydoc
        log_info "anydoc statik kitaplığı algılandı, -tags anydoc etkinleştirildi"
    else
        log_info "anydoc statik kitaplığı algılanmadı, ayrıştırma motoru kullanılamıyor. Gerektiğinde önce şunu çalıştırın: make anydoc-lib"
    fi
}

# Arka uç uygulamasını başlat (yerel)
start_app() {
    log_info "Arka uç uygulaması başlatılıyor (yerel geliştirme modu)..."
    
    cd "$PROJECT_ROOT"
    
    # Go'nun yüklü olup olmadığını kontrol et
    if ! command -v go &> /dev/null; then
        log_error "Go kurulu değil"
        return 1
    fi
    
    log_info "Ortam yapılandırması yükleniyor..."
    if ! load_env_files; then
        log_error ".env dosyası yok, önce yapılandırma dosyasını oluşturun"
        return 1
    fi
    
    # Yerel docker-compose.dev modu: kapsayıcı hizmet adlarını ana makinenin geri döngü adresine eşle
    # Uzak geliştirme modunda (DEV_REMOTE_HOST veya .env.local içinde adres ayarlanmışsa) .env/.env.local içindeki değerleri koru
    if [ -n "${DEV_REMOTE_HOST:-}" ]; then
        log_info "Uzak geliştirme modu: altyapı → ${DEV_REMOTE_HOST}"
        export DB_HOST="${DB_HOST:-$DEV_REMOTE_HOST}"
        export REDIS_ADDR="${REDIS_ADDR:-$DEV_REMOTE_HOST:6379}"
        export DOCREADER_ADDR="${DOCREADER_ADDR:-$DEV_REMOTE_HOST:50051}"
        export MINIO_ENDPOINT="${MINIO_ENDPOINT:-$DEV_REMOTE_HOST:9000}"
        export MILVUS_ADDRESS="${MILVUS_ADDRESS:-$DEV_REMOTE_HOST:19530}"
        export NEO4J_URI="${NEO4J_URI:-bolt://$DEV_REMOTE_HOST:7687}"
        export QDRANT_HOST="${QDRANT_HOST:-$DEV_REMOTE_HOST}"
        if [ -z "${LANGFUSE_HOST:-}" ] || [ "$LANGFUSE_HOST" = "http://langfuse-web:3000" ]; then
            export LANGFUSE_HOST="http://${DEV_REMOTE_HOST}:3000"
        fi
    else
        export DB_HOST=127.0.0.1
        export DOCREADER_ADDR=127.0.0.1:50051
        export MINIO_ENDPOINT=127.0.0.1:9000
        export REDIS_ADDR=127.0.0.1:6379
        export MILVUS_ADDRESS=127.0.0.1:19530
        export NEO4J_URI=bolt://127.0.0.1:7687
        export QDRANT_HOST=127.0.0.1
    fi
    export DOCREADER_TRANSPORT="${DOCREADER_TRANSPORT:-grpc}"

    if ! check_remote_dev_connectivity; then
        return 1
    fi

    # .env.example uses /data/files for the Docker app container, where a
    # volume is mounted at that path. When the backend runs directly on the
    # host via dev-app, /data is often read-only or missing, so use a repo-local
    # writable directory unless the developer explicitly configured another
    # local storage path.
    if [ -z "${LOCAL_STORAGE_BASE_DIR:-}" ] || [ "$LOCAL_STORAGE_BASE_DIR" = "/data/files" ]; then
        export LOCAL_STORAGE_BASE_DIR="$PROJECT_ROOT/.local-data/files"
    fi
    mkdir -p "$LOCAL_STORAGE_BASE_DIR"
    
    # Gerekli ortam değişkenlerinin ayarlandığından emin ol
    if [ -z "$DB_DRIVER" ]; then
        log_error "DB_DRIVER ortam değişkeni ayarlanmamış, lütfen .env dosyasını kontrol edin"
        return 1
    fi
    
    log_info "Ortam değişkenleri ayarlandı, uygulama başlatılıyor..."
    log_info "Veritabanı adresi: $DB_HOST:${DB_PORT:-5432}"
    
    export CGO_CFLAGS="-Wno-deprecated-declarations -Wno-gnu-folding-constant"
    if [[ "$(uname)" == "Darwin" ]]; then
      export CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries"
    fi

    enable_anydoc_build_tag

    # Air'in (sıcak yeniden yükleme aracı) yüklü olup olmadığını kontrol et
    if command -v air &> /dev/null; then
        log_success "Air algılandı, sıcak yeniden yükleme moduyla başlatılıyor..."
        log_info "Go kodu değiştirildikten sonra otomatik olarak yeniden derlenecek ve yeniden başlatılacak"
        air
    else
        log_info "Air algılanmadı, normal modda başlatılıyor"
        log_warning "İpucu: Air yüklemek, kod değişikliklerinden sonra otomatik yeniden başlatmayı sağlar"
        log_info "Yükleme komutu: go install github.com/air-verse/air@latest"
        LDFLAGS="$(./scripts/get_version.sh ldflags) -X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"
        go run -tags "${GO_BUILD_TAGS:-}" -ldflags="$LDFLAGS" ./cmd/server
    fi
}

# Ön ucu başlat (yerel)
start_frontend() {
    log_info "Ön uç geliştirme sunucusu başlatılıyor..."

    cd "$PROJECT_ROOT"
    if [ -f ".env" ] || [ -f ".env.local" ]; then
        load_env_files >/dev/null 2>&1 || true
    fi
    
    cd "$PROJECT_ROOT/frontend"
    
    # npm'in yüklü olup olmadığını kontrol et
    if ! command -v npm &> /dev/null; then
        log_error "npm yüklü değil"
        return 1
    fi
    
    # Bağımlılıkların yüklü olup olmadığını kontrol et
    if [ ! -d "node_modules" ]; then
        log_warning "node_modules mevcut değil, bağımlılıklar yükleniyor..."
        npm install
    fi
    
    log_info "Vite geliştirme sunucusu başlatılıyor..."
    log_info "Ön uç http://localhost:5173 adresinde çalışacak"
    log_info "Ön uç API vekil hedefi: ${VITE_DEV_PROXY_TARGET:-${FRONTEND_BACKEND_URL:-http://localhost:8080}}"
    
    # Geliştirme sunucusunu çalıştır
    npm run dev
}

# Komutu ayrıştır
CMD="${1:-help}"
case "$CMD" in
    start)
        start_services "$@"
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    logs)
        show_logs
        ;;
    status)
        show_status
        ;;
    app)
        start_app
        ;;
    frontend)
        start_frontend
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Bilinmeyen komut: $CMD"
        show_help
        exit 1
        ;;
esac
