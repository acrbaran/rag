#!/bin/bash
# Bu betik, docker-compose hizmetlerini gerektiğinde başlatmak/durdurmak için kullanılır

# Rengi ayarla
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # Renk yok

# Proje kök dizinini al (betiğin bulunduğu dizinin bir üst seviyesi)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Sürüm bilgisi
VERSION="1.0.1" # Sürüm güncellemesi
SCRIPT_NAME=$(basename "$0")

# Yardım bilgisini göster
show_help() {
    printf "%b\n" "${GREEN}Rethra başlatma betiği v${VERSION}${NC}"
    printf "%b\n" "${GREEN}Kullanım:${NC} $0 [seçenekler]"
    echo "Seçenekler:"
    echo "  -h, --help     Yardım bilgilerini göster"
    echo "  -d, --docker   Docker kapsayıcı hizmetlerini başlat"
    echo "  -a, --all      Tüm hizmetleri başlat (varsayılan)"
    echo "  -s, --stop     Tüm hizmetleri durdur"
    echo "  -c, --check    Ortamı kontrol et ve sorunları tanıla"
    echo "  -r, --restart  Belirtilen kapsayıcıyı yeniden oluştur ve yeniden başlat"
    echo "  -l, --list     Çalışan tüm kapsayıcıları listele"
    echo "  -p, --pull     Üçüncü taraf hizmet imajlarını çek"
    echo "  --no-pull      Üçüncü taraf hizmet imajlarını önceden çekme"
    echo "  -v, --version  Sürüm bilgilerini göster"
    exit 0
}

# Sürüm bilgisini göster
show_version() {
    printf "%b\n" "${GREEN}Rethra başlatma betiği v${VERSION}${NC}"
    exit 0
}

# Günlük işlevi
log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

# Inject git short hash into the frontend image when composing from source.
# docker-compose.yml interpolates VITE_FRONTEND_COMMIT; the build context is
# frontend/ (no .git), so Vite cannot discover the commit on its own.
export_frontend_build_args() {
    if [ -n "${VITE_FRONTEND_COMMIT:-}" ]; then
        export VITE_FRONTEND_COMMIT
        return 0
    fi
    # shellcheck source=/dev/null
    eval "$("$PROJECT_ROOT/scripts/get_version.sh" env)"
    export VITE_FRONTEND_COMMIT="${COMMIT_ID:-unknown}"
    log_info "VITE_FRONTEND_COMMIT=${VITE_FRONTEND_COMMIT}"
}

# Kullanılabilir Docker Compose komutunu seç (önce docker compose, ardından docker-compose)
DOCKER_COMPOSE_BIN=""
DOCKER_COMPOSE_SUBCMD=""

detect_compose_cmd() {
	# Docker Compose eklentisini öncelikli kullan
	if docker compose version &> /dev/null; then
		DOCKER_COMPOSE_BIN="docker"
		DOCKER_COMPOSE_SUBCMD="compose"
		return 0
	fi

	# docker-compose (v1) sürümüne geri dön
	if command -v docker-compose &> /dev/null; then
		if docker-compose version &> /dev/null; then
			DOCKER_COMPOSE_BIN="docker-compose"
			DOCKER_COMPOSE_SUBCMD=""
			return 0
		fi
	fi

	# Hiçbiri kullanılamıyor
	return 1
}

# .env dosyasını kontrol et ve oluştur
check_env_file() {
    log_info "Ortam değişkeni yapılandırması kontrol ediliyor..."
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        log_warning ".env dosyası yok, şablondan oluşturulacak"
        if [ -f "$PROJECT_ROOT/.env.example" ]; then
            cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
            log_success ".env dosyası .env.example dosyasından oluşturuldu"
        else
            log_error ".env.example şablon dosyası bulunamadı, .env dosyası oluşturulamıyor"
            return 1
        fi
    else
        log_info ".env dosyası zaten mevcut"
    fi
    
    # Gerekli ortam değişkenlerinin ayarlanıp ayarlanmadığını kontrol et
    source "$PROJECT_ROOT/.env"
    local missing_vars=()
    
    # Temel değişkenleri kontrol et
    if [ -z "$DB_DRIVER" ]; then missing_vars+=("DB_DRIVER"); fi
    if [ -z "$STORAGE_TYPE" ]; then missing_vars+=("STORAGE_TYPE"); fi
    
    return 0
}

# Docker'ın kurulu olup olmadığını kontrol et
check_docker() {
    log_info "Docker ortamı kontrol ediliyor..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker kurulu değil, lütfen önce Docker'ı kurun"
        return 1
    fi
    
	# Kullanılabilir Docker Compose komutunu kontrol et ve seç
	if detect_compose_cmd; then
		if [ "$DOCKER_COMPOSE_BIN" = "docker" ]; then
			log_info "Docker Compose eklentisi algılandı (docker compose)"
		else
			log_info "docker-compose algılandı (v1)"
		fi
	else
		log_error "Docker Compose algılanmadı (ne docker compose ne de docker-compose mevcut). Lütfen bunlardan birini yükleyin."
		return 1
	fi
    
    # Docker hizmetinin çalışma durumunu kontrol et
    if ! docker info &> /dev/null; then
        log_error "Docker hizmeti çalışmıyor, lütfen Docker hizmetini başlatın"
        return 1
    fi
    
    log_success "Docker ortamı denetimi başarılı"
    return 0
}

check_platform() {
     # Mevcut sistem platformunu algıla
    log_info "Sistem platformu bilgileri algılanıyor..."
    if [ "$(uname -m)" = "x86_64" ]; then
        export PLATFORM="linux/amd64"
    elif [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then
        export PLATFORM="linux/arm64"
    else
        log_warning "Tanınmayan platform türü: $(uname -m), varsayılan platform linux/amd64 kullanılacak"
        export PLATFORM="linux/amd64"
    fi
    log_info "Mevcut platform: $PLATFORM"
}

# Sandbox imajını oluştur (yalnızca oluşturur, başlatmaz; Agent Skills çalıştırması için gereklidir)
ensure_sandbox_image() {
    local sandbox_image="rethra-sandbox:${RETHRA_VERSION:-latest}"

    # Sandbox imajının yerelde mevcut olup olmadığını kontrol et
    if docker image inspect "$sandbox_image" &> /dev/null; then
        log_success "Korumalı alan imajı hazır: $sandbox_image"
        return 0
    fi

    log_info "Korumalı alan imajı ($sandbox_image) algılanmadı, arka planda oluşturuluyor..."
    log_info "Agent Skills özelliği bu imaja bağlıdır; ilk çalıştırmadan önce oluşturmanın tamamlanması gerekir"

    # Ana akışı engellemeden arka planda oluştur
    (
        if PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD --profile sandbox build sandbox; then
            log_success "Korumalı alan imajı oluşturma tamamlandı: $sandbox_image"
        else
            log_warning "Korumalı alan imajı oluşturulamadı, Agent Skills özelliği kullanılamayabilir"
            log_warning "Daha sonra elle oluşturabilirsiniz: $DOCKER_COMPOSE_BIN $DOCKER_COMPOSE_SUBCMD --profile sandbox build sandbox"
        fi
    ) &

    return 0
}

# Docker konteynerini başlat
start_docker() {
    log_info "Docker kapsayıcıları başlatılıyor..."
    
    # Docker ortamını kontrol et
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # .env dosyasını kontrol et
    check_env_file
    
    # .env dosyasını oku
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}
    
    check_platform
    
	# Proje kök dizinine geçip docker-compose komutunu çalıştır
    cd "$PROJECT_ROOT"

    export_frontend_build_args
    
    # Temel hizmetleri başlat
    log_info "Çekirdek hizmet kapsayıcıları başlatılıyor..."
	# Tespit edilen Compose komutuyla başlat
	log_info "Rethra imajı yerel kaynak kodundan oluşturuluyor ve başlatılıyor..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up --build -d
    if [ $? -ne 0 ]; then
        log_error "Docker kapsayıcıları başlatılamadı"
        return 1
    fi
    
    log_success "Tüm Docker kapsayıcıları başarıyla başlatıldı"

    # Konteyner durumunu göster
    log_info "Mevcut kapsayıcı durumu:"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps

    # Sandbox imajını oluştur (Agent Skills çalıştırmak için gereklidir; yalnızca oluşturur, başlatmaz)
    ensure_sandbox_image

    return 0
}

# Docker konteynerlerini durdur
stop_docker() {
    log_info "Docker kapsayıcıları durduruluyor..."
    
    # Docker ortamını kontrol et
    check_docker
    if [ $? -ne 0 ]; then
        # Kontrol başarısız olsa bile, her ihtimale karşı durdurmayı dene
        log_warning "Docker ortamı denetimi başarısız oldu, kapsayıcılar yine de durdurulmaya çalışılacak..."
    fi
    
    # Proje kök dizinine geçip docker-compose komutunu çalıştır
    cd "$PROJECT_ROOT"
    
    # Tüm konteynerleri durdur
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD down --remove-orphans
    if [ $? -ne 0 ]; then
        log_error "Docker kapsayıcıları durdurulamadı"
        return 1
    fi
    
    log_success "Tüm Docker kapsayıcıları durduruldu"
    return 0
}

# Çalışmakta olan tüm konteynerleri listele
list_containers() {
    log_info "Çalışan tüm kapsayıcılar listeleniyor..."
    
    # Docker ortamını kontrol et
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Proje kök dizinine geçip docker-compose komutunu çalıştır
    cd "$PROJECT_ROOT"
    
    # Tüm konteynerleri listele
    printf "%b\n" "${BLUE}Şu anda çalışan kapsayıcılar:${NC}"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | sort
    
    return 0
}

# Üçüncü taraf Docker imajlarını çek
pull_images() {
    log_info "Üçüncü taraf Docker imajları çekiliyor..."
    
    # Docker ortamını kontrol et
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # .env dosyasını kontrol et
    check_env_file
    
    # .env dosyasını oku
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}
    
    check_platform
    
    # Proje kök dizinine geçip docker-compose komutunu çalıştır
    cd "$PROJECT_ROOT"
    
    # Rethra imajı yerel kaynak kodundan oluşturulur
    log_info "Üçüncü taraf hizmet imajları çekiliyor..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD pull --ignore-buildable
    if [ $? -ne 0 ]; then
        log_error "İmaj çekilemedi"
        return 1
    fi

    log_success "Üçüncü taraf imajları başarıyla çekildi"
    
    # Çekilen imaj bilgilerini göster
    log_info "Çekilen imajlar:"
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}\t{{.Size}}" | head -10
    
    return 0
}

# Belirtilen konteyneri yeniden başlat
restart_container() {
    local container_name="$1"
    
    if [ -z "$container_name" ]; then
        log_error "Kapsayıcı adı belirtilmedi"
        echo "Kullanılabilir kapsayıcılar:"
        list_containers
        return 1
    fi
    
    log_info "Kapsayıcı yeniden oluşturuluyor ve yeniden başlatılıyor: $container_name"
    
    # Docker ortamını kontrol et
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    check_platform
    
    # Proje kök dizinine geçip docker-compose komutunu çalıştır
    cd "$PROJECT_ROOT"

    export_frontend_build_args
    
    # Kapsayıcının var olup olmadığını kontrol et
	if ! "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | grep -q "^$container_name$"; then
        log_error "'$container_name' kapsayıcısı mevcut değil veya çalışmıyor"
        echo "Kullanılabilir kapsayıcılar:"
        list_containers
        return 1
    fi
    
    # Kapsayıcıyı derle ve yeniden başlat
    log_info "'$container_name' kapsayıcısı yeniden oluşturuluyor..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD build "$container_name"
    if [ $? -ne 0 ]; then
        log_error "'$container_name' kapsayıcısı oluşturulamadı"
        return 1
    fi
    
    log_info "'$container_name' kapsayıcısı yeniden başlatılıyor..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up -d --no-deps "$container_name"
    if [ $? -ne 0 ]; then
        log_error "'$container_name' kapsayıcısı yeniden başlatılamadı"
        return 1
    fi
    
    log_success "'$container_name' kapsayıcısı başarıyla yeniden oluşturuldu ve yeniden başlatıldı"
    return 0
}

# Sistem ortamını kontrol et
check_environment() {
    log_info "Ortam denetimi başlatılıyor..."
    
    # İşletim sistemini kontrol et
    OS=$(uname)
    log_info "İşletim sistemi: $OS"
    
    # Docker'ı kontrol et
    check_docker
    
    # .env dosyasını kontrol et
    check_env_file
    
    # Sandbox imajını kontrol et
    log_info "Korumalı alan imajı denetleniyor..."
    local sandbox_image="rethra-sandbox:${RETHRA_VERSION:-latest}"
    if docker image inspect "$sandbox_image" &> /dev/null; then
        log_success "Korumalı alan imajı hazır: $sandbox_image"
    else
        log_warning "Korumalı alan imajı bulunamadı: $sandbox_image (Agent Skills özelliği bu imajı gerektirir)"
        log_info "Şu komutla oluşturulabilir: docker compose --profile sandbox build sandbox"
    fi

    # Disk alanını kontrol et
    log_info "Disk alanı denetleniyor..."
    df -h | grep -E "(Filesystem|/$)"
    
    # Belleği kontrol et
    log_info "Bellek kullanımı kontrol ediliyor..."
    if [ "$OS" = "Darwin" ]; then
        vm_stat | perl -ne '/page size of (\d+)/ and $size=$1; /Pages free:\s*(\d+)/ and print "Free Memory: ", $1 * $size / 1048576, " MB\n"'
    else
        free -h | grep -E "(total|Mem:)"
    fi
    
    # CPU'yu kontrol et
    log_info "CPU bilgisi:"
    if [ "$OS" = "Darwin" ]; then
        sysctl -n machdep.cpu.brand_string
        echo "CPU çekirdek sayısı: $(sysctl -n hw.ncpu)"
    else
        grep "model name" /proc/cpuinfo | head -1
        echo "CPU çekirdek sayısı: $(nproc)"
    fi
    
    # Kapsayıcı durumunu kontrol et
    log_info "Konteyner durumu kontrol ediliyor..."
    if docker info &> /dev/null; then
        docker ps -a
    else
        log_warning "Konteyner durumu alınamadı, Docker çalışmıyor olabilir"
    fi
    
    log_success "Ortam kontrolü tamamlandı"
    return 0
}

# Komut satırı parametrelerini ayrıştır
START_DOCKER=false
STOP_SERVICES=false
CHECK_ENVIRONMENT=false
LIST_CONTAINERS=false
RESTART_CONTAINER=false
PULL_IMAGES=false
NO_PULL=false
CONTAINER_NAME=""

# Parametre olmadığında varsayılan olarak tüm hizmetleri başlat
if [ $# -eq 0 ]; then
    START_DOCKER=true
fi

while [ "$1" != "" ]; do
    case $1 in
        -h | --help )       show_help
                            ;;
        -d | --docker )     START_DOCKER=true
                            ;;
        -a | --all )                                START_DOCKER=true
                            ;;
        -s | --stop )       STOP_SERVICES=true
                            ;;
        -c | --check )      CHECK_ENVIRONMENT=true
                            ;;
        -l | --list )       LIST_CONTAINERS=true
                            ;;
        -p | --pull )       PULL_IMAGES=true
                            ;;
        --no-pull )         NO_PULL=true
                                                    START_DOCKER=true
                            ;;
        -r | --restart )    RESTART_CONTAINER=true
                            CONTAINER_NAME="$2"
                            shift
                            ;;
        -v | --version )    show_version
                            ;;
        * )                 log_error "Bilinmeyen seçenek: $1"
                            show_help
                            ;;
    esac
    shift
done

# Ortam kontrollerini gerçekleştir
if [ "$CHECK_ENVIRONMENT" = true ]; then
    check_environment
    exit $?
fi

# Tüm kapsayıcıları listele
if [ "$LIST_CONTAINERS" = true ]; then
    list_containers
    exit $?
fi

# En güncel imajı çek
if [ "$PULL_IMAGES" = true ]; then
    pull_images
    exit $?
fi

# Belirtilen kapsayıcıyı yeniden başlat
if [ "$RESTART_CONTAINER" = true ]; then
    restart_container "$CONTAINER_NAME"
    exit $?
fi

# Hizmet işlemini gerçekleştir
if [ "$STOP_SERVICES" = true ]; then
    stop_docker
elif [ "$START_DOCKER" = true ]; then
    start_docker
    result=$?
    if [ "$result" -eq 0 ]; then
        log_success "Docker konteyneri başlatıldı, aşağıdaki adreslerden erişebilirsiniz:"
        printf "%b\n" "${GREEN}  - Ön yüz arayüzü: http://localhost:${FRONTEND_PORT:-80}${NC}"
        printf "%b\n" "${GREEN}  - API uç noktası: http://localhost:${APP_PORT:-8080}${NC}"
        log_info "Konteyner günlükleri sürekli olarak gösteriliyor (günlüklerden çıkmak için Ctrl+C tuşlarına basın, konteyner durmaz)..."
        "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD logs app docreader postgres --since=10s -f
    fi
    exit "$result"
fi
