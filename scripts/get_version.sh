#!/bin/bash
# Birleşik sürüm bilgisi alma betiği
# Yerel derleme ve CI derleme ortamlarını destekler

# Varsayılan değerleri ayarla
VERSION="unknown"
EDITION="${EDITION:-standard}"
COMMIT_ID="unknown"
BUILD_TIME="unknown"
GO_VERSION="unknown"

# Sürüm numarasını al
if [ -f "VERSION" ]; then
    VERSION=$(cat VERSION | tr -d '\n\r')
fi

# commit ID'sini al
if [ -n "$GITHUB_SHA" ]; then
    # GitHub Actions ortamı
    COMMIT_ID="${GITHUB_SHA:0:7}"
elif command -v git >/dev/null 2>&1; then
    # Yerel ortam
    COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
fi

# Derleme zamanını al
if [ -n "$GITHUB_ACTIONS" ]; then
    # GitHub Actions ortamı, standart zaman biçimini kullan
    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
else
    # Yerel ortam
    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
fi

# Go sürümünü al
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version 2>/dev/null || echo "unknown")
fi

# Parametrelere göre farklı biçimlerde çıktı ver
case "${1:-env}" in
    "env")
        # Ortam değişkeni biçiminde çıktı ver; boşluk içeren değerleri kaçır
        echo "VERSION=$VERSION"
        echo "EDITION=$EDITION"
        echo "COMMIT_ID=$COMMIT_ID"
        echo "BUILD_TIME=\"$BUILD_TIME\""
        echo "GO_VERSION=\"$GO_VERSION\""
        ;;
    "json")
        # JSON biçimini çıktıla
        cat << EOF
{
  "version": "$VERSION",
  "edition": "$EDITION",
  "commit_id": "$COMMIT_ID",
  "build_time": "$BUILD_TIME",
  "go_version": "$GO_VERSION"
}
EOF
        ;;
    "docker-args")
        # Docker derleme bağımsız değişkeni biçimini çıktıla
        echo "--build-arg VERSION_ARG=$VERSION"
        echo "--build-arg COMMIT_ID_ARG=$COMMIT_ID"
        echo "--build-arg BUILD_TIME_ARG=$BUILD_TIME"
        echo "--build-arg GO_VERSION_ARG=$GO_VERSION"
        ;;
    "ldflags")
        # Go ldflags biçimini çıktıla
        echo "-X 'github.com/acrbaran/rag/internal/handler.Version=$VERSION' -X 'github.com/acrbaran/rag/internal/handler.Edition=$EDITION' -X 'github.com/acrbaran/rag/internal/handler.CommitID=$COMMIT_ID' -X 'github.com/acrbaran/rag/internal/handler.BuildTime=$BUILD_TIME' -X 'github.com/acrbaran/rag/internal/handler.GoVersion=$GO_VERSION'"
        ;;
    "info")
        # Bilgi biçimini çıktıla
        echo "Sürüm bilgisi: $VERSION"
        echo "Sürüm türü: $EDITION"
        echo "Commit ID: $COMMIT_ID"
        echo "Derleme zamanı: $BUILD_TIME"
        echo "Go sürümü: $GO_VERSION"
        ;;
    *)
        echo "Kullanım: $0 [env|json|docker-args|ldflags|info]"
        echo "  env        - Ortam değişkeni biçiminde çıktı verir (varsayılan)"
        echo "  json       - JSON biçiminde çıktı verir"
        echo "  docker-args - Docker derleme parametreleri biçiminde çıktı verir"
        echo "  ldflags    - Go ldflags biçiminde çıktı verir"
        echo "  info       - Bilgi biçiminde çıktı verir"
        exit 1
        ;;
esac
