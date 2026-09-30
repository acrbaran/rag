#!/bin/bash
# Bu betik, Rethra'nın tüm Docker görüntülerini kaynak koddan oluşturmak için kullanılır

# Renkleri ayarla
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # Renksiz

# Proje kök dizinini al (betiğin bulunduğu dizinin bir üst seviyesi)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# BuildKit'i etkinleştir
export DOCKER_BUILDKIT=1

# Sürüm bilgisi
VERSION="1.0.0"
SCRIPT_NAME=$(basename "$0")

# Yardım bilgisini göster
show_help() {
    echo -e "${GREEN}Rethra imaj oluşturma betiği v${VERSION}${NC}"
    echo -e "${GREEN}Kullanım:${NC} $0 [seçenekler]"
    echo "Seçenekler:"
    echo "  -h, --help     Yardım bilgilerini göster"
    echo "  -a, --all      Tüm imajları oluştur (varsayılan)"
    echo "  -p, --app      Yalnızca uygulama imajını oluştur"
    echo "  -d, --docreader Yalnızca belge okuyucu imajını oluştur"
    echo "  -f, --frontend Yalnızca ön uç imajını oluştur"
    echo "  -s, --sandbox  Yalnızca korumalı alan imajını oluştur"
    echo "  -c, --clean    Tüm yerel imajları temizle"
    echo "  -v, --version  Sürüm bilgilerini göster"
    exit 0
}

# Sürüm bilgisini göster
show_version() {
    echo -e "${GREEN}Rethra İmaj Oluşturma Betiği v${VERSION}${NC}"
    exit 0
}

# Günlük fonksiyonu
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Docker'ın yüklü olup olmadığını kontrol et
check_docker() {
    log_info "Docker ortamı kontrol ediliyor..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker kurulu değil, lütfen önce Docker'ı kurun"
        return 1
    fi
    
    # Docker hizmetinin çalışma durumunu kontrol et
    if ! docker info &> /dev/null; then
        log_error "Docker hizmeti çalışmıyor, lütfen Docker hizmetini başlatın"
        return 1
    fi
    
    log_success "Docker ortamı kontrolü başarılı"
    return 0
}

# Platformu algıla
check_platform() {
    log_info "Sistem platformu bilgileri algılanıyor..."
    if [ "$(uname -m)" = "x86_64" ]; then
        export PLATFORM="linux/amd64"
        export TARGETARCH="amd64"
    elif [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then
        export PLATFORM="linux/arm64"
        export TARGETARCH="arm64"
    else
        log_warning "Tanınmayan platform türü: $(uname -m), varsayılan platform linux/amd64 kullanılacak"
        export PLATFORM="linux/amd64"
        export TARGETARCH="amd64"
    fi
    log_info "Geçerli platform: $PLATFORM"
    log_info "Geçerli mimari: $TARGETARCH"
}

# Sürüm bilgisini al
get_version_info() {
    # Sürüm numarasını VERSION dosyasından al
    if [ -f "VERSION" ]; then
        VERSION=$(cat VERSION | tr -d '\n\r')
    else
        VERSION="unknown"
    fi
    
    # Commit ID'sini al
    if command -v git >/dev/null 2>&1; then
        COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    else
        COMMIT_ID="unknown"
    fi
    
    # Derleme zamanını al
    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
    
    # Go sürümünü al
    if command -v go >/dev/null 2>&1; then
        GO_VERSION=$(go version 2>/dev/null || echo "unknown")
    else
        GO_VERSION="unknown"
    fi
    
    log_info "Sürüm bilgisi: $VERSION"
    log_info "Commit ID: $COMMIT_ID"
    log_info "Oluşturma zamanı: $BUILD_TIME"
    log_info "Go sürümü: $GO_VERSION"
}

# Uygulama imajını derle
build_app_image() {
    log_info "Uygulama imajı oluşturuluyor (rethra-app)..."
    
    cd "$PROJECT_ROOT"
    
    # Sürüm bilgisini al
    get_version_info
    
    docker build \
        --platform $PLATFORM \
        --build-arg GOPRIVATE_ARG=${GOPRIVATE:-""} \
        --build-arg GOPROXY_ARG=${GOPROXY:-"https://goproxy.cn,direct"} \
        --build-arg GOSUMDB_ARG=${GOSUMDB:-"off"} \
        --build-arg VERSION_ARG="$VERSION" \
        --build-arg COMMIT_ID_ARG="$COMMIT_ID" \
        --build-arg BUILD_TIME_ARG="$BUILD_TIME" \
        --build-arg GO_VERSION_ARG="$GO_VERSION" \
        --build-arg WITH_ANYDOC=${WITH_ANYDOC:-1} \
        -f docker/Dockerfile.app \
        -t rethra-app:latest \
        .
    
    if [ $? -eq 0 ]; then
        log_success "Uygulama imajı başarıyla oluşturuldu"
        return 0
    else
        log_error "Uygulama imajı oluşturulamadı"
        return 1
    fi
}

# Belge okuyucu imajını derle
build_docreader_image() {
    log_info "Belge okuyucu imajı oluşturuluyor (rethra-docreader)..."
    
    cd "$PROJECT_ROOT"
    
    docker build \
        --platform $PLATFORM \
        --build-arg PLATFORM=$PLATFORM \
        --build-arg TARGETARCH=$TARGETARCH \
        --build-arg APT_MIRROR=${APT_MIRROR:-} \
        -f docker/Dockerfile.docreader \
        -t rethra-docreader:latest \
        .
    
    if [ $? -eq 0 ]; then
        log_success "Belge okuyucu imajı başarıyla oluşturuldu"
        return 0
    else
        log_error "Belge okuyucu imajı oluşturulamadı"
        return 1
    fi
}

# Ön uç imajını derle (çok aşamalı: npm builder içinde çalışır, ana makinede dist ön derlemesi gerekmez)
build_frontend_image() {
    log_info "Ön uç imajı oluşturuluyor (rethra-ui)..."
    
    cd "$PROJECT_ROOT"
    
    # Sürüm bilgisini al (ön uç commit hash'ini eklemek için)
    get_version_info

    docker build \
        --platform $PLATFORM \
        --build-arg VITE_FRONTEND_COMMIT="$COMMIT_ID" \
        ${NPM_REGISTRY:+--build-arg NPM_REGISTRY="$NPM_REGISTRY"} \
        ${NODE_MAX_OLD_SPACE_SIZE:+--build-arg NODE_MAX_OLD_SPACE_SIZE="$NODE_MAX_OLD_SPACE_SIZE"} \
        -f frontend/Dockerfile \
        -t rethra-ui:latest \
        frontend/
    
    if [ $? -eq 0 ]; then
        log_success "Ön uç imajı başarıyla oluşturuldu"
        return 0
    else
        log_error "Ön uç imajı oluşturulamadı"
        return 1
    fi
}

# Sandbox imajını derle
build_sandbox_image() {
    log_info "Korumalı alan imajı oluşturuluyor (rethra-sandbox)..."

    cd "$PROJECT_ROOT"

    # Aynı zamanda main etiketini de ekle: varsayılan imaj main'i takip eder (sandbox.go içindeki DefaultDockerImage'a bakın),
    # ve Docker arka ucu yalnızca yerelde eksikse çeker; yerelde bu etiketi eklememek, derlemeyi boşa yapmak demektir.
    docker build \
        --platform $PLATFORM \
        --build-arg TARGETPLATFORM=$PLATFORM \
        -f docker/Dockerfile.sandbox \
        --target sandbox \
        -t rethra-sandbox:latest \
        -t rethra-sandbox:main \
        .

    if [ $? -ne 0 ]; then
        log_error "Korumalı alan imajı oluşturulamadı"
        return 1
    fi

    # Cube şablonu doğrudan imajdan derler ve :49983/health ile durum kontrolü yapar; envd olmadan kaçınılmaz olarak başarısız olur,
    # bu nedenle Cube, envd eklenmiş varyant imajı kullanır. Ayrıntılar için website-docs/06-development/04-sandbox-deployment.md dosyasına bakın。
    # linux/amd64 sabitlendi: envd kaynak imajı cubesandbox-base arm64 yayımlamıyor.
    log_info "Korumalı alan imajının Cube varyantı oluşturuluyor (rethra-sandbox:main-cube)..."

    docker build \
        --platform linux/amd64 \
        --build-arg TARGETPLATFORM=linux/amd64 \
        --build-arg TARGETARCH=amd64 \
        -f docker/Dockerfile.sandbox \
        --target cube \
        -t rethra-sandbox:latest-cube \
        -t rethra-sandbox:main-cube \
        .

    if [ $? -ne 0 ]; then
        log_error "Korumalı alan imajının Cube varyantı oluşturulamadı"
        return 1
    fi

    # Desktop variant: XFCE + x11vnc + websockify. Tagged for E2B template
    # builds; the Docker backend does not consume this image yet.
    log_info "Korumalı alan imajının masaüstü varyantı oluşturuluyor (rethra-sandbox:main-desktop)..."

    docker build \
        --platform $PLATFORM \
        --build-arg TARGETPLATFORM=$PLATFORM \
        -f docker/Dockerfile.sandbox \
        --target desktop \
        -t rethra-sandbox:latest-desktop \
        -t rethra-sandbox:main-desktop \
        .

    if [ $? -ne 0 ]; then
        log_error "Korumalı alan imajının masaüstü varyantı oluşturulamadı"
        return 1
    fi

    log_info "Korumalı alan imajının masaüstü Cube varyantı oluşturuluyor (rethra-sandbox:main-desktop-cube)..."

    docker build \
        --platform linux/amd64 \
        --build-arg TARGETPLATFORM=linux/amd64 \
        --build-arg TARGETARCH=amd64 \
        -f docker/Dockerfile.sandbox \
        --target desktop-cube \
        -t rethra-sandbox:latest-desktop-cube \
        -t rethra-sandbox:main-desktop-cube \
        .

    if [ $? -eq 0 ]; then
        log_success "Korumalı alan imajı başarıyla oluşturuldu"
        return 0
    else
        log_error "Korumalı alan imajının masaüstü Cube varyantı oluşturulamadı"
        return 1
    fi
}

# Tüm imajları oluştur
build_all_images() {
    log_info "Tüm imajların oluşturulması başlatılıyor..."

    local app_result=0
    local docreader_result=0
    local frontend_result=0
    local sandbox_result=0

    # Uygulama imajını oluştur
    build_app_image
    app_result=$?

    # Belge okuyucu imajını oluştur
    build_docreader_image
    docreader_result=$?

    # Ön yüz imajını oluştur
    build_frontend_image
    frontend_result=$?

    # Korumalı alan imajını oluştur
    build_sandbox_image
    sandbox_result=$?

    # Oluşturma sonuçlarını göster
    echo ""
    log_info "=== Oluşturma sonuçları ==="
    if [ $app_result -eq 0 ]; then
        log_success "✓ Uygulama imajı başarıyla oluşturuldu"
    else
        log_error "✗ Uygulama imajı oluşturulamadı"
    fi

    if [ $docreader_result -eq 0 ]; then
        log_success "✓ Belge okuyucu imajı başarıyla oluşturuldu"
    else
        log_error "✗ Belge okuyucu imajı oluşturulamadı"
    fi

    if [ $frontend_result -eq 0 ]; then
        log_success "✓ Ön uç imajı başarıyla oluşturuldu"
    else
        log_error "✗ Ön uç imajı oluşturulamadı"
    fi

    if [ $sandbox_result -eq 0 ]; then
        log_success "✓ Korumalı alan imajı başarıyla oluşturuldu"
    else
        log_error "✗ Korumalı alan imajı oluşturulamadı"
    fi

    if [ $app_result -eq 0 ] && [ $docreader_result -eq 0 ] && [ $frontend_result -eq 0 ] && [ $sandbox_result -eq 0 ]; then
        log_success "Tüm imajlar oluşturuldu!"
        return 0
    else
        log_error "Bazı imajlar oluşturulamadı"
        return 1
    fi
}

# Yerel imajları temizle
clean_images() {
    log_info "Yerel Rethra imajları temizleniyor..."
    
    # İlgili kapsayıcıları durdur
    log_info "İlgili kapsayıcılar durduruluyor..."
    docker stop $(docker ps -q --filter "ancestor=rethra-app:latest" 2>/dev/null) 2>/dev/null || true
    docker stop $(docker ps -q --filter "ancestor=rethra-docreader:latest" 2>/dev/null) 2>/dev/null || true
    docker stop $(docker ps -q --filter "ancestor=rethra-ui:latest" 2>/dev/null) 2>/dev/null || true
    
    # İlgili kapsayıcıları sil
    log_info "İlgili kapsayıcılar siliniyor..."
    docker rm $(docker ps -aq --filter "ancestor=rethra-app:latest" 2>/dev/null) 2>/dev/null || true
    docker rm $(docker ps -aq --filter "ancestor=rethra-docreader:latest" 2>/dev/null) 2>/dev/null || true
    docker rm $(docker ps -aq --filter "ancestor=rethra-ui:latest" 2>/dev/null) 2>/dev/null || true
    
    # İmajı sil
    log_info "Yerel imajlar siliniyor..."
    docker rmi rethra-app:latest 2>/dev/null || true
    docker rmi rethra-docreader:latest 2>/dev/null || true
    docker rmi rethra-ui:latest 2>/dev/null || true
    docker rmi rethra-sandbox:latest 2>/dev/null || true
    docker rmi rethra-sandbox:latest-cube 2>/dev/null || true
    docker rmi rethra-sandbox:latest-desktop 2>/dev/null || true
    docker rmi rethra-sandbox:latest-desktop-cube 2>/dev/null || true
    docker rmi rethra-sandbox:main 2>/dev/null || true
    docker rmi rethra-sandbox:main-cube 2>/dev/null || true
    docker rmi rethra-sandbox:main-desktop 2>/dev/null || true
    docker rmi rethra-sandbox:main-desktop-cube 2>/dev/null || true
    
    docker image prune -f
    
    log_success "İmaj temizleme tamamlandı"
    return 0
}

# Komut satırı parametrelerini ayrıştır
BUILD_ALL=false
BUILD_APP=false
BUILD_DOCREADER=false
BUILD_FRONTEND=false
BUILD_SANDBOX=false
CLEAN_IMAGES=false

# Parametre olmadığında varsayılan olarak tüm imajları oluştur
if [ $# -eq 0 ]; then
    BUILD_ALL=true
fi

while [ "$1" != "" ]; do
    case $1 in
        -h | --help )       show_help
                            ;;
        -a | --all )        BUILD_ALL=true
                            ;;
        -p | --app )        BUILD_APP=true
                            ;;
        -d | --docreader )  BUILD_DOCREADER=true
                            ;;
        -f | --frontend )   BUILD_FRONTEND=true
                            ;;
        -s | --sandbox )    BUILD_SANDBOX=true
                            ;;
        -c | --clean )      CLEAN_IMAGES=true
                            ;;
        -v | --version )    show_version
                            ;;
        * )                 log_error "Bilinmeyen seçenek: $1"
                            show_help
                            ;;
    esac
    shift
done

# Docker ortamını kontrol et
check_docker
if [ $? -ne 0 ]; then
    exit 1
fi

# Platformu algıla
check_platform

# Temizleme işlemini gerçekleştir
if [ "$CLEAN_IMAGES" = true ]; then
    clean_images
    exit $?
fi

# Oluşturma işlemini gerçekleştir
if [ "$BUILD_ALL" = true ]; then
    build_all_images
    exit $?
fi

if [ "$BUILD_APP" = true ]; then
    build_app_image
    exit $?
fi

if [ "$BUILD_DOCREADER" = true ]; then
    build_docreader_image
    exit $?
fi

if [ "$BUILD_FRONTEND" = true ]; then
    build_frontend_image
    exit $?
fi

if [ "$BUILD_SANDBOX" = true ]; then
    build_sandbox_image
    exit $?
fi

exit 0
