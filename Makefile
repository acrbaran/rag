.PHONY: help build run test clean docker-build-app docker-build-docreader docker-build-frontend docker-build-all docker-run migrate-up migrate-down docker-restart docker-stop start-all stop-all build-images build-images-app build-images-docreader build-images-frontend clean-images check-env list-containers pull-images show-platform dev-start dev-stop dev-restart dev-logs dev-status dev-app dev-frontend docs install-swagger build-lite run-lite package-lite anydoc-lib build-anydoc

# Show help
help:
	@echo "Rethra Makefile Yardımı"
	@echo ""
	@echo "Temel komutlar:"
	@echo "  build             Uygulamayı derle"
	@echo "  run               Uygulamayı çalıştır"
	@echo "  test              Testleri çalıştır"
	@echo "  anydoc-lib        anydoc statik kütüphanesini derle (Rust araç zinciri gerekir)"
	@echo "  build-anydoc      anydoc ayrıştırma motoruyla uygulamayı derle"
	@echo "  clean             Derleme dosyalarını temizle"
	@echo ""
	@echo "Docker komutları:"
	@echo "  docker-build-app       Uygulama Docker imajını oluşturur (rethra-app)"
	@echo "  docker-build-docreader Belge okuyucu imajını oluşturur (rethra-docreader)"
	@echo "  docker-build-frontend  Ön uç imajını oluşturur (rethra-ui)"
	@echo "  docker-build-all       Tüm Docker imajlarını oluşturur"
	@echo "  docker-run            Docker kapsayıcısını çalıştırır"
	@echo "  docker-stop           Docker kapsayıcısını durdurur"
	@echo "  docker-restart        Docker kapsayıcısını yeniden başlatır"
	@echo ""
	@echo "Hizmet yönetimi:"
	@echo "  start-all         Tüm hizmetleri başlatır"
	@echo "  stop-all          Tüm hizmetleri durdurur"
	@echo ""
	@echo "İmaj oluşturma:"
	@echo "  build-images      Tüm imajları kaynak koddan oluşturur"
	@echo "  build-images-app  Uygulama imajını kaynak koddan oluşturur"
	@echo "  build-images-docreader Belge okuyucu imajını kaynak koddan oluşturur"
	@echo "  build-images-frontend  Ön uç imajını kaynak koddan oluşturur"
	@echo "  clean-images      Yerel imajları temizler"
	@echo ""
	@echo "Veritabanı:"
	@echo "  migrate-up        Veritabanı geçişlerini uygular"
	@echo "  migrate-down      Veritabanı geçişlerini geri alır"
	@echo ""
	@echo "Geliştirme araçları:"
	@echo "  fmt               Kodu biçimlendirir"
	@echo "  lint              Kod denetimi yapar"
	@echo "  deps              Bağımlılıkları yükler"
	@echo "  docs              Swagger API belgelerini oluşturur"
	@echo "  install-swagger   swag aracını yükle"
	@echo ""
	@echo "Model sağlayıcı kataloğu:"
	@echo "  model-catalog-check   sağlayıcı kataloğunu doğrula (değişmezler + eski ve yeni davranış karşılaştırması + sağlayıcı testleri)"
	@echo "  model-catalog-diff    models.dev ile karşılaştır, model meta veri fark raporunu çıktıla (manuel inceleme gerekir)"
	@echo "                        İsteğe bağlı: make model-catalog-diff VENDOR=deepseek"
	@echo ""
	@echo "Ortam kontrolü:"
	@echo "  check-env         ortam yapılandırmasını kontrol et"
	@echo "  list-containers   çalışan kapsayıcıları listele"
	@echo "  pull-images       en güncel imajları çek"
	@echo "  show-platform     geçerli derleme platformunu göster"
	@echo ""
	@echo "Geliştirme modu (önerilir):"
	@echo "  dev-start         geliştirme ortamı altyapısını başlat (yalnızca bağımlı hizmetleri başlatır)"
	@echo "                    İsteğe bağlı: make dev-start DEV_ARGS=--odl-hybrid"
	@echo "  dev-stop          geliştirme ortamını durdur"
	@echo "  dev-restart       geliştirme ortamını yeniden başlat"
	@echo "  dev-logs          geliştirme ortamı günlüklerini görüntüle"
	@echo "  dev-status        geliştirme ortamı durumunu görüntüle"
	@echo "  dev-app           arka uç uygulamasını başlat (yerelde çalışır, önce dev-start çalıştırılmalıdır)"
	@echo "                    make anydoc-lib yapıldığında anydoc motorunu otomatik olarak bağla"
	@echo "  dev-frontend      ön yüzü başlat (yerelde çalışır, önce dev-start çalıştırılmalıdır)"
	@echo ""
	@echo "Lite modu (sıfır harici bağımlılık):"
	@echo "  build-lite        Lite sürümü derle (önce ön yüzü web/ dizinine, sonra Go'yu derler; SKIP_FRONTEND=1 ön yüzü atlar)"
	@echo "  run-lite          Lite sürümünü derle ve başlat"
	@echo "  package-lite      Lite dağıtım paketini derle ve paketle (tarball)"

# Go related variables
BINARY_NAME=Rethra
MAIN_PATH=./cmd/server

# Docker related variables
DOCKER_IMAGE=rethra-app
DOCKER_TAG=latest

# Platform detection
ifeq ($(shell uname -m),x86_64)
    PLATFORM=linux/amd64
else ifeq ($(shell uname -m),aarch64)
    PLATFORM=linux/arm64
else ifeq ($(shell uname -m),arm64)
    PLATFORM=linux/arm64
else
    PLATFORM=linux/amd64
endif

# Build the application
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Build the anydoc static archive (Rust) that the `anydoc` build tag links.
# Override the platform with TARGET=<rust-target-triple>.
anydoc-lib:
	./scripts/build-anydoc-lib.sh

# Build the application with the in-process anydoc parser engine linked in.
build-anydoc: anydoc-lib
	go build -tags anydoc -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Generate reviewed metadata + protocol overrides, then verify every model.
.PHONY: model-catalog-generate model-catalog-check
model-catalog-generate:
	python3 scripts/model-catalog/generate.py

model-catalog-check:
	python3 scripts/model-catalog/generate.py --check
	go test ./internal/models/...

# Vendor catalog: report where our model metadata differs from models.dev.
# Development aid only — nothing is fetched at runtime and nothing is written
# automatically; review each line against the vendor's own documentation.
.PHONY: model-catalog-diff
model-catalog-diff:
	@python3 scripts/model_catalog_diff.py $(if $(VENDOR),--vendor $(VENDOR),)

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Build Docker image
docker-build-app:
	@echo "Sürüm bilgisi alınıyor..."
	@eval $$(./scripts/get_version.sh env); \
	./scripts/get_version.sh info; \
	docker build --platform $(PLATFORM) \
		--build-arg VERSION_ARG="$$VERSION" \
		--build-arg COMMIT_ID_ARG="$$COMMIT_ID" \
		--build-arg BUILD_TIME_ARG="$$BUILD_TIME" \
		--build-arg GO_VERSION_ARG="$$GO_VERSION" \
		--build-arg WITH_ANYDOC=$${WITH_ANYDOC:-1} \
		-f docker/Dockerfile.app -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Build docreader Docker image
docker-build-docreader:
	docker build --platform $(PLATFORM) -f docker/Dockerfile.docreader -t rethra-docreader:latest .

# Build frontend Docker image (multi-stage: npm runs inside the builder stage)
docker-build-frontend:
	@eval $$(./scripts/get_version.sh env); \
	docker build --platform $(PLATFORM) \
		--build-arg VITE_FRONTEND_COMMIT="$$COMMIT_ID" \
		-f frontend/Dockerfile -t rethra-ui:latest frontend/

# Build all Docker images
docker-build-all: docker-build-app docker-build-docreader docker-build-frontend

# Docker kapsayıcısını çalıştır (geleneksel yöntem)
# Touch .env if missing — docker-compose.yml's `env_file: [.env]` is required
# for ${ENV} interpolation in builtin_models.yaml and would otherwise refuse
# to parse on fresh clones. `start-all` handles this via check_env_file; this
# direct path needs its own guard.
docker-run:
	@[ -f .env ] || ([ -f .env.example ] && cp .env.example .env || touch .env)
	docker-compose up

# Tüm hizmetleri yeni betikle başlat
start-all:
	./scripts/start_all.sh

# Yalnızca Docker kapsayıcısını yeni betikle başlat
start-docker:
	./scripts/start_all.sh --docker

# Tüm hizmetleri yeni betikle durdur
stop-all:
	./scripts/start_all.sh --stop

# Docker kapsayıcısını durdur (geleneksel yöntem)
docker-stop:
	docker-compose down

# Kaynak koddan imaj oluşturmayla ilgili komutlar
build-images:
	./scripts/build_images.sh

build-images-app:
	./scripts/build_images.sh --app

build-images-docreader:
	./scripts/build_images.sh --docreader

build-images-frontend:
	./scripts/build_images.sh --frontend

clean-images:
	./scripts/build_images.sh --clean

# Restart Docker container (stop, start)
docker-restart:
	@[ -f .env ] || ([ -f .env.example ] && cp .env.example .env || touch .env)
	docker-compose stop -t 60
	docker-compose up

# Database migrations
migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

migrate-version:
	./scripts/migrate.sh version

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required"; \
		echo "Usage: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	./scripts/migrate.sh create $(name)

migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-force version=4"; \
		exit 1; \
	fi
	./scripts/migrate.sh force $(version)

migrate-goto:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-goto version=3"; \
		exit 1; \
	fi
	./scripts/migrate.sh goto $(version)

# Generate API documentation (Swagger)
docs:
	@echo "Swagger API belgeleri oluşturuluyor..."
	swag init -g $(MAIN_PATH)/main.go -o ./docs --parseDependency --parseInternal
	@echo "Belgeler ./docs dizinine oluşturuldu"
	@echo "Belgeleri görüntülemek için hizmeti başlattıktan sonra http://localhost:8080/swagger/index.html adresini ziyaret edin"

# Install swagger tool
install-swagger:
	go install github.com/swaggo/swag/cmd/swag@latest

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download

# Build for production
# google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn for qdrant milvus proto conflict
# GO_BUILD_TAGS adds optional build tags, e.g. GO_BUILD_TAGS=anydoc to link the
# in-process office document parser (run `make anydoc-lib` first).
build-prod:
	VERSION=$${VERSION:-$$(git describe --tags --abbrev=0 2>/dev/null || tr -d '\n\r' < VERSION 2>/dev/null)}; \
	VERSION=$${VERSION:-unknown}; \
	COMMIT_ID=$${COMMIT_ID:-unknown}; \
	CGO_ENABLED=1 \
	CGO_CFLAGS="-Wno-deprecated-declarations" \
	CGO_LDFLAGS="$$(if [ "$$(uname)" = 'Darwin' ]; then echo '-Wl,-no_warn_duplicate_libraries'; fi)" \
	BUILD_TIME=$${BUILD_TIME:-unknown}; \
	GO_VERSION=$${GO_VERSION:-unknown}; \
	LDFLAGS="-X 'github.com/acrbaran/rag/internal/handler.Version=$$VERSION' -X 'github.com/acrbaran/rag/internal/handler.Edition=standard' -X 'github.com/acrbaran/rag/internal/handler.CommitID=$$COMMIT_ID' -X 'github.com/acrbaran/rag/internal/handler.BuildTime=$$BUILD_TIME' -X 'github.com/acrbaran/rag/internal/handler.GoVersion=$$GO_VERSION' -X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"; \
	go build -tags "$(GO_BUILD_TAGS)" -ldflags="-w -s $$LDFLAGS" -o $(BINARY_NAME) $(MAIN_PATH)

# Build Lite version (single binary, SQLite + in-memory queue)
# Önce ön yüz web/ dizinine derlenir, ardından Go ikili dosyası oluşturulur; SKIP_FRONTEND=1 ön yüzü atlayabilir
build-lite:
	@if [ -f frontend/package.json ] && [ "$${SKIP_FRONTEND:-}" != "1" ]; then \
		echo ">> Building frontend for Lite..."; \
		(cd frontend && npm ci --prefer-offline && npm run build) && \
		rm -rf web && cp -r frontend/dist web; \
	elif [ "$${SKIP_FRONTEND:-}" = "1" ]; then \
		echo ">> Skipping frontend (SKIP_FRONTEND=1)"; \
	else \
		echo ">> No frontend/package.json, skipping frontend"; \
	fi
	export EDITION=lite; \
	eval "$$(./scripts/get_version.sh env)"; \
	LDFLAGS="$$(./scripts/get_version.sh ldflags) -X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"; \
	CGO_ENABLED=1 \
	CGO_CFLAGS="-Wno-deprecated-declarations" \
	CGO_LDFLAGS="$$(if [ "$$(uname)" = 'Darwin' ]; then echo '-Wl,-no_warn_duplicate_libraries'; fi)" \
	go build -tags "sqlite_fts5" -ldflags="-w -s $$LDFLAGS" -o $(BINARY_NAME)-lite $(MAIN_PATH)

# Run Lite version with .env.lite defaults
run-lite: build-lite
	@if [ ! -f .env.lite ]; then echo "Error: .env.lite not found"; exit 1; fi
	@set -a && . ./.env.lite && set +a && ./$(BINARY_NAME)-lite

# Package Lite version into distributable tarball
package-lite:
	./scripts/package-lite.sh

# Package Mac App

download_spatial:
	go run cmd/download/duckdb/duckdb.go

clean-db:
	@echo "Cleaning database..."
	@if [ $$(docker volume ls -q -f name=rethra_postgres-data) ]; then \
		docker volume rm rethra_postgres-data; \
	fi
	@if [ $$(docker volume ls -q -f name=rethra_minio_data) ]; then \
		docker volume rm rethra_minio_data; \
	fi
	@if [ $$(docker volume ls -q -f name=rethra_redis_data) ]; then \
		docker volume rm rethra_redis_data; \
	fi

# Environment check
check-env:
	./scripts/start_all.sh --check

# List containers
list-containers:
	./scripts/start_all.sh --list

# Pull latest images
pull-images:
	./scripts/start_all.sh --pull

# Show current platform
show-platform:
	@echo "Geçerli sistem mimarisi: $(shell uname -m)"
	@echo "Docker derleme platformu: $(PLATFORM)"

# Development mode commands
dev-start:
	./scripts/dev.sh start $(DEV_ARGS)

dev-stop:
	./scripts/dev.sh stop

dev-restart:
	./scripts/dev.sh restart

dev-logs:
	./scripts/dev.sh logs

dev-status:
	./scripts/dev.sh status

dev-app:
	./scripts/dev.sh app

dev-frontend:
	./scripts/dev.sh frontend
