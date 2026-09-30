#!/bin/bash
set -ex

# Dizini ayarla
PROTO_DIR="docreader/proto"
PYTHON_OUT="docreader/proto"
GO_OUT="docreader/proto"

# Python kodu oluştur
python3 -m grpc_tools.protoc -I${PROTO_DIR} \
    --python_out=${PYTHON_OUT} \
    --pyi_out=${PYTHON_OUT} \
    --grpc_python_out=${PYTHON_OUT} \
    ${PROTO_DIR}/docreader.proto

# Go kodu oluştur (yalnızca `protoc-gen-go` kullanılabilir olduğunda çalıştırılır)
if command -v protoc-gen-go &> /dev/null; then
    protoc -I${PROTO_DIR} --go_out=${GO_OUT} \
        --go_opt=paths=source_relative \
        --go-grpc_out=${GO_OUT} \
        --go-grpc_opt=paths=source_relative \
        ${PROTO_DIR}/docreader.proto
else
    echo "protoc-gen-go not found, skipping Go code generation"
fi

# Python içe aktarma sorununu düzelt (MacOS uyumlu sürüm)
if [ "$(uname)" == "Darwin" ]; then
    # MacOS sürümü
    sed -i '' 's/import docreader_pb2/from docreader.proto import docreader_pb2/g' ${PYTHON_OUT}/docreader_pb2_grpc.py
else
    # Linux sürümü
    sed -i 's/import docreader_pb2/from docreader.proto import docreader_pb2/g' ${PYTHON_OUT}/docreader_pb2_grpc.py
fi

echo "Proto files generated successfully!"