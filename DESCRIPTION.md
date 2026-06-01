# TractorVIN

TractorVIN 是一个拖拉机序列号解码器，提供 CLI 和 REST API 两种使用方式，用于解析主流农机品牌的序列号信息。

## 核心内容

- 支持 John Deere、Kubota、Massey Ferguson、Case IH、New Holland 序列号解析
- 提供命令行工具和 HTTP API
- 输出品牌、型号、年份、工厂、发动机系列等信息
- 单二进制部署，依赖少，适合快速发布

## 技术栈

- Go
- chi router
- net/http

## 运行方式

```bash
go build -o tractor-vin .
./tractor-vin serve
```

## 适合上传到 GitHub 的描述

一个用于拖拉机序列号解析的 Go 工具，兼顾命令行和 API 服务场景。