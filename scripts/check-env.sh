#!/bin/bash
# Geliştirme ortamı yapılandırmasını kontrol et

# Rengi ayarla
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # Renk yok

# Proje kök dizinini al
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[✓]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[✗]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[!]${NC} $1"
}

echo ""
printf "%b\n" "${GREEN}========================================${NC}"
printf "%b\n" "${GREEN}  Rethra geliştirme ortamı yapılandırma denetimi${NC}"
printf "%b\n" "${GREEN}========================================${NC}"
echo ""

cd "$PROJECT_ROOT"

# .env dosyasını kontrol et
log_info ".env dosyası kontrol ediliyor..."
if [ -f ".env" ]; then
    log_success ".env dosyası mevcut"
else
    log_error ".env dosyası bulunamadı"
    echo ""
    log_info "Çözüm:"
    echo "  1. Örnek dosyayı kopyalayın: cp .env.example .env"
    echo "  2. .env dosyasını düzenleyip gerekli ortam değişkenlerini yapılandırın"
    exit 1
fi

echo ""
log_info "Gerekli ortam değişkenleri kontrol ediliyor..."

# .env dosyasını yükle
set -a
source .env
set +a

# Gerekli ortam değişkenlerini kontrol et
errors=0

check_var() {
    local var_name=$1
    local var_value="${!var_name}"
    
    if [ -z "$var_value" ]; then
        log_error "$var_name ayarlanmamış"
        errors=$((errors + 1))
    else
        log_success "$var_name = $var_value"
    fi
}

# Veritabanı yapılandırması
log_info "Veritabanı yapılandırması:"
check_var "DB_DRIVER"
check_var "DB_HOST"
check_var "DB_PORT"
check_var "DB_USER"
check_var "DB_PASSWORD"
check_var "DB_NAME"

echo ""
log_info "Depolama yapılandırması:"
check_var "STORAGE_TYPE"

if [ "$STORAGE_TYPE" = "minio" ]; then
    check_var "MINIO_BUCKET_NAME"
fi

if [ "$STORAGE_TYPE" = "tos" ]; then
    check_var "TOS_ENDPOINT"
    check_var "TOS_REGION"
    check_var "TOS_ACCESS_KEY"
    check_var "TOS_SECRET_KEY"
    check_var "TOS_BUCKET_NAME"
fi

if [ "$STORAGE_TYPE" = "s3" ]; then
    check_var "S3_REGION"
    check_var "S3_BUCKET_NAME"
fi

echo ""
log_info "Redis yapılandırması:"
check_var "REDIS_ADDR"

echo ""
log_info "Model yapılandırması:"
if [ -n "$INIT_LLM_MODEL_NAME" ]; then
    log_success "INIT_LLM_MODEL_NAME = $INIT_LLM_MODEL_NAME"
else
    log_warning "INIT_LLM_MODEL_NAME ayarlanmamış (isteğe bağlı)"
fi

if [ -n "$INIT_EMBEDDING_MODEL_NAME" ]; then
    log_success "INIT_EMBEDDING_MODEL_NAME = $INIT_EMBEDDING_MODEL_NAME"
else
    log_warning "INIT_EMBEDDING_MODEL_NAME ayarlanmamış (isteğe bağlı)"
fi

# Go ortamını kontrol et
echo ""
log_info "Go ortamı kontrol ediliyor..."
if command -v go &> /dev/null; then
    go_version=$(go version)
    log_success "Go yüklü: $go_version"
else
    log_error "Go yüklü değil"
    errors=$((errors + 1))
fi

# Air'i kontrol et
if command -v air &> /dev/null; then
    log_success "Air yüklü (sıcak yeniden yükleme desteklenir)"
else
    log_warning "Air yüklü değil (isteğe bağlı, sıcak yeniden yükleme için)"
    log_info "Yükleme komutu: go install github.com/air-verse/air@latest"
fi

# npm'i kontrol et
echo ""
log_info "Node.js ortamı kontrol ediliyor..."
if command -v npm &> /dev/null; then
    npm_version=$(npm --version)
    log_success "npm yüklü: $npm_version"
else
    log_error "npm yüklü değil"
    errors=$((errors + 1))
fi

# Docker'ı kontrol et
echo ""
log_info "Docker ortamı kontrol ediliyor..."
if command -v docker &> /dev/null; then
    docker_version=$(docker --version)
    log_success "Docker yüklü: $docker_version"
    
    if docker info &> /dev/null; then
        log_success "Docker servisi çalışıyor"
    else
        log_error "Docker servisi çalışmıyor"
        errors=$((errors + 1))
    fi
else
    log_error "Docker yüklü değil"
    errors=$((errors + 1))
fi

# Docker Compose'u kontrol et
if docker compose version &> /dev/null; then
    compose_version=$(docker compose version)
    log_success "Docker Compose yüklü: $compose_version"
elif command -v docker-compose &> /dev/null; then
    compose_version=$(docker-compose --version)
    log_success "docker-compose yüklü: $compose_version"
else
    log_error "Docker Compose yüklü değil"
    errors=$((errors + 1))
fi

# Özet
echo ""
printf "%b\n" "${GREEN}========================================${NC}"
if [ $errors -eq 0 ]; then
    log_success "Tüm kontroller başarılı! Ortam yapılandırması normal"
    echo ""
    log_info "Sonraki adımlar:"
    echo "  1. Geliştirme ortamını başlatın: make dev-start"
    echo "  2. Arka ucu başlatın: make dev-app"
    echo "  3. Ön ucu başlatın: make dev-frontend"
else
    log_error "$errors sorun bulundu, lütfen düzelttikten sonra geliştirme ortamını başlatın"
    echo ""
    log_info "Sık karşılaşılan sorunlar:"
    echo "  - .env dosyası yoksa, lütfen .env.example dosyasını kopyalayın"
    echo "  - DB_DRIVER değerinin 'postgres' veya 'mysql' olarak ayarlandığından emin olun"
    echo "  - Docker servisinin çalıştığından emin olun"
fi
printf "%b\n" "${GREEN}========================================${NC}"
echo ""

exit $errors
