.PHONY: proto build run docker-build docker-run clean

# protobuf kodunu oluştur
proto:
	@echo "Generating protobuf code..."
	@sh ./scripts/generate_proto.sh

# Go istemcisini derle
build:
	@echo "Building Go client..."
	@go build -o bin/client ./src/client

# Python hizmetini çalıştır
run:
	@echo "Running Python server..."
	@python src/server/server.py

# Temizle
clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@find . -name "*.pyc" -delete
	@find . -name "__pycache__" -delete 