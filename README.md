# 🚜 TractorVIN — 拖拉机序列号解码器

**TractorVIN** decodes tractor serial numbers for John Deere, Kubota, Massey Ferguson, Case IH, and New Holland. Available as a CLI tool and REST API.

---

## 中文

### 功能

- 🔢 支持 John Deere、Kubota、Massey Ferguson、Case IH、New Holland 五大品牌序列号解码
- 🖥️ CLI 命令行工具 + 🌐 REST API 服务
- 📋 输出品牌、型号、生产年份、工厂、发动机系列等详细信息
- 🚀 Go 语言编写，单二进制文件，零依赖部署

### 安装

```bash
go build -o tractor-vin .
```

### CLI 使用

```bash
./tractor-vin decode 1RW8136PABCD12345   # 解码序列号
./tractor-vin brands                     # 列出支持的品牌
./tractor-vin serve --port 8080          # 启动 API 服务
```

### API 使用

```bash
# 启动服务
./tractor-vin serve

# 解码
curl -X POST http://localhost:8080/decode \
  -H "Content-Type: application/json" \
  -d '{"serial":"1RW8136PABCD12345"}'

# 品牌列表
curl http://localhost:8080/brands
```

### 支持的品牌

| 品牌 | 序列号格式 | 解码信息 |
|------|-----------|---------|
| John Deere | 13/17位 PIN | 工厂、年份、发动机系列 |
| Kubota | 字母+数字 | 型号系列、年份范围、产地 |
| Massey Ferguson | 9/17位 | 系列、工厂、年份 |
| Case IH | 8-17位 | 型号、发动机、产地 |
| New Holland | 9-17位 | 系列、工厂、年份 |

---

## English

### Features

- 🔢 Decode serial numbers for 5 major tractor brands
- 🖥️ CLI tool + 🌐 REST API
- 📋 Returns brand, model, year, factory, engine family
- 🚀 Single Go binary, zero dependencies

### Quick Start

```bash
go build -o tractor-vin .
./tractor-vin decode 1RW8136PABCD12345
./tractor-vin brands
./tractor-vin serve
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /decode | Decode a serial number |
| GET | /brands | List supported brands |
| GET | /health | Health check |

### Docker

```bash
docker build -t tractor-vin .
docker run -p 8080:8080 tractor-vin serve
```

### Tech Stack

Go 1.22 · chi router · net/http
