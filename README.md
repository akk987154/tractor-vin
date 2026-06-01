# 🚜 TractorVIN — 拖拉机序列号解码器

<!--
  这个工具最初只是为了在 TractorCompare 里加一个"输入序列号自动识别机型"的功能，
  结果做着做着就变成了一个独立的 CLI + API 项目。最难搞的是 Kubota——它的序列号没有
  统一的国际标准格式，不同年代、不同产地的编码规则都不一样。John Deere 反而是最规整的，
  17 位 PIN 码跟汽车 VIN 一个逻辑。

  品牌枚举和型号命名字段跟 TractorCompare 的数据层严格对齐，两边改 schema 的时候
  都互相通知——本来是打算做成同一个 monorepo 的，后来因为 Go ↔ TypeScript 的跨语言
  协作太麻烦才拆开了。
-->

[![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev)
[![chi](https://img.shields.io/badge/chi-v5-083b66?logo=go)](https://github.com/go-chi/chi)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ed?logo=docker)](https://www.docker.com)

**TractorVIN** 是 [TractorTools](https://github.com/seb/tractor-tools) 生态的识别层——相当于整个工具集的"入口"。拿到一台陌生拖拉机的序列号，先用它解码出品牌/型号/年份，然后：
- 去 [TractorCompare](../tractor-compare) 查详细规格和对比
- 去 [FarmCalc](../farm-calc) 算运营成本和折旧
- 去 [TractorLog](../tractor-log) 建档开始记录维护
- 去 [TractorWatch](../tractor-watch) 查二手市场行情

提供 CLI 命令行和 HTTP API 两种调用方式，单 Go 二进制，零依赖部署。

---

## 🗺️ 项目生态

| 项目 | 定位 | 与本项目的关系 |
|------|------|---------------|
| [TractorCompare](../tractor-compare) | 规格对比 | 品牌枚举 + 型号命名 + 系列分类严格对齐，解码结果可直接作为对比页的查询参数 |
| [FarmCalc](../farm-calc) | 成本/ROI 计算 | 解码出的年份 → 计算已使用年限 → 折旧计算 |
| [TractorLog](../tractor-log) | 维护日志 PWA | 解码结果可直接用于创建农机档案 |
| [TractorWatch](../tractor-watch) | 二手价格追踪 | 解码出的型号/年份用于精确搜索二手挂牌 |

---

## 中文

### 功能

- 🔢 **五大品牌解码** — John Deere、Kubota、Massey Ferguson、Case IH、New Holland
- 🖥️ **CLI 命令行** — 终端直接解码，支持管道和脚本集成
- 🌐 **REST API** — JSON 输入输出，方便其他工具调用（TractorCompare 和 TractorWatch 的集成端点就是对着这个 API 设计的）
- 📋 **解码信息** — 品牌、型号、系列、生产年份、工厂、发动机系列、生产序号、产地
- 🚀 **单二进制** — Go 编译，无运行时依赖，Alpine Docker 镜像不到 15MB
- 🐳 **Docker 支持** — 多阶段构建，提供 Dockerfile

### 安装

```bash
# 直接编译
go build -o tractor-vin .

# 或安装到 $GOPATH/bin
make install

# Docker
docker build -t tractor-vin .
```

### CLI 使用

```bash
# 解码序列号
./tractor-vin decode 1RW8136PABCD12345
# 输出：品牌、型号、系列、年份、工厂、发动机系列、产地

# 列出支持的品牌
./tractor-vin brands

# 启动 API 服务
./tractor-vin serve --port 8080
```

### API 使用

```bash
# 启动服务
./tractor-vin serve

# 解码
curl -X POST http://localhost:8080/decode \
  -H "Content-Type: application/json" \
  -d '{"serial":"1RW8136PABCD12345"}'

# 响应示例
# {
#   "serial": "1RW8136PABCD12345",
#   "brand": "John Deere",
#   "model": "5075E",
#   "series": "5E Series",
#   "year": 2018,
#   "factory": "Pune, India",
#   "engineFamily": "PowerTech 3029",
#   "productionNumber": "BCD12345",
#   "country": "India"
# }

# 品牌列表
curl http://localhost:8080/brands

# 健康检查
curl http://localhost:8080/health
```

### 各品牌解码规则

这个表记录的是解码器的实际实现方式。最复杂的是 Kubota（没有统一标准），最简单的是 John Deere（17 位 PIN 码接近汽车 VIN 标准）：

| 品牌 | 序列号格式 | 解码方法 | 解码字段 |
|------|-----------|---------|---------|
| **John Deere** | 13/17位 PIN | WMI + 工厂码 + 年份码查表 | 工厂、年份、发动机系列、产地 |
| **Kubota** | 字母+数字混合 | 型号前缀匹配 + 年份范围推断 | 型号系列、年份范围、产地 |
| **Massey Ferguson** | 9-17位 | 型号前缀 + 工厂码 + 序列段 | 系列、工厂、年份 |
| **Case IH** | 8-17位 | 型号前缀 + 数字型号映射 | 型号、发动机、产地 |
| **New Holland** | 9-17位 | T-series 前缀 + Boomer/Workmaster 分支 | 系列、工厂、年份 |

### 解码器架构

```
main.go                  # CLI 入口 (cobra-style: decode / brands / serve 三个子命令)
api/
  api.go                 # chi HTTP 路由 (POST /decode, GET /brands, GET /health) + CORS 中间件
decoder/
  decoder.go             # DecodedInfo 结构体 + Decoder 接口 + Decode() 调度器
  johndeere.go           # PIN 正则 + WMI 表 + 工厂/年份/发动机映射
  kubota.go              # 型号前缀匹配 + 年份范围推断
  masseyferguson.go      # 型号前缀 + 工厂码查表
  caseih.go              # 型号前缀 + 数字型号→名称映射
  newholland.go          # T-series 前缀 + Boomer/Workmaster 分支逻辑
```

解码器用策略模式——新增品牌只需要实现 `Decoder` 接口（`Decode(serial string) *DecodedInfo`）并在 `decoder.go` 的注册表中加一行。这个模式是在加了第三个品牌（Massey Ferguson）之后重构出来的，前两个品牌是硬编码在 main 里的。

`DecodedInfo` 结构体的字段名跟 [TractorCompare](../tractor-compare) 的 TypeScript 接口严格对齐——两边改了字段名会互相通知。这是拆成两个项目后定下的约定。

### 技术栈

Go 1.22 · chi/v5 · net/http · encoding/json

选 Go 的原因：其他四个工具全是前端/脚本语言（JS/Python），需要一个高性能、低资源占用的服务端组件来做 API——TractorCompare、TractorWatch、TractorLog 将来都可以调用这个 API。Go 的单一二进制部署也比 Node/Python 省心（不需要安装运行时）。

---

## English

### Features

- 🔢 **Five brand decoders** — John Deere, Kubota, Massey Ferguson, Case IH, New Holland
- 🖥️ **CLI tool** — Command-line decoding with pipe/script support
- 🌐 **REST API** — JSON in/out, designed for integration with sibling tools
- 📋 **Rich decode output** — Brand, model, series, year, factory, engine family, production number, country
- 🚀 **Single binary** — Zero runtime dependencies, <15MB Docker image
- 🐳 **Docker** — Multistage build included

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
| POST | /decode | Decode a serial number → full tractor info |
| GET | /brands | List supported brands |
| GET | /health | Health check |

### Docker

```bash
docker build -t tractor-vin .
docker run -p 8080:8080 tractor-vin serve
```

### Tech Stack

Go 1.22 · chi router · net/http

---

## 📋 迭代记录

<details>
<summary>点击展开</summary>

### v0.4 — 解码器策略模式重构 + CORS (当前)
- 所有品牌解码器统一 `Decoder` 接口
- 新增 Case IH 和 New Holland 解码器
- chi 中间件栈（日志 + 恢复 + RealIP + CORS）
- `DecodedInfo` 字段名跟 TractorCompare 对齐
- Docker 多阶段构建

### v0.3 — API 服务 + Kubota 解码
- 新增 HTTP API（chi 路由）
- Kubota 解码器（最复杂的品牌——序列号没有统一格式）
- 从 main.go 拆分出 decoder 和 api 包

### v0.2 — Massey Ferguson 解码
- 新增 Massey Ferguson 解码器
- CLI 支持 `brands` 子命令
- 解码逻辑从硬编码改为查表法

### v0.1 — 初始版本
- 仅 John Deere PIN 解码
- 纯 CLI，无 API
- 解码逻辑硬编码在 main.go

</details>

## 🗓️ 路线图

- [ ] [TractorCompare](../tractor-compare) 集成——对比页支持"输入序列号自动识别机型"
- [ ] [TractorLog](../tractor-log) 集成——"扫码/输入序列号快速建档"
- [ ] 更多品牌（Fendt、Claas、McCormick、Deutz-Fahr）
- [ ] VIN 校验位验证（check digit validation）
- [ ] 批量解码端点（POST /decode/batch）
- [ ] gRPC 端点（供内部微服务调用）
