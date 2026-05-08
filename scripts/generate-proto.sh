#!/bin/bash

# 生成protobuf代码
echo "Generating protobuf code..."

# 检查buf是否安装
if ! command -v buf &> /dev/null; then
    echo "buf not found, please install it from https://buf.build/"
    exit 1
fi

# 生成Go代码
buf generate

echo "Protobuf code generation completed."