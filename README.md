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

# 响应示例（实际输出）
# {"success":true,"data":{
#   "brand":"John Deere",
#   "model":"W813系列",
#   "series":"",
#   "year":"2011",
#   "factory":"美国爱荷华州安克尼",
#   "engineFamily":"PowerTech E 1.6-2.9L",
#   "productionNumber":"CD12345",
#   "country":"美国",
#   "metadata":{"WMI":"1RW"}
# }}

# 失败时返回 HTTP 400 与错误说明
curl -X POST http://localhost:8080/decode \
  -H "Content-Type: application/json" \
  -d '{"serial":"1ZZZZ"}'
# {"success":false,"error":"无法识别该序列号格式。支持的品牌: ..."}

# 品牌列表
curl http://localhost:8080/brands
# {"brands":["John Deere","Kubota","Massey Ferguson","New Holland","Case IH"]}

# 健康检查
curl http://localhost:8080/health
```

### 响应格式说明

所有响应都是 `{"success": bool, ...}` 信封结构：

- 成功：`{"success": true, "data": {...}}`
- 失败：`{"success": false, "error": "..."}`，同时返回 HTTP 400

注意 `year` 是字符串（例如 `"2011"`）而不是数字，`factory` / `country` 是中文描述。

### 服务参数

```bash
./tractor-vin serve                      # 默认监听 0.0.0.0:8080
./tractor-vin serve -host 127.0.0.1      # 只允许本机访问
./tractor-vin serve -port 9000
```

默认绑定 `0.0.0.0` 是为了配合容器部署。如果直接在本机运行且不希望同网段可访问，
请显式指定 `-host 127.0.0.1`。

服务端已配置 `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` / `IdleTimeout`，
并对请求体（4 KiB）与序列号长度（32 字符）设置了上限。

### 各品牌解码规则

这个表记录的是解码器的实际实现方式。最复杂的是 Kubota（没有统一标准），最简单的是 John Deere（17 位 PIN 码接近汽车 VIN 标准）：

| 品牌 | 序列号格式 | 解码方法 | 解码字段 |
|------|-----------|---------|---------|
| **John Deere** | 13/17位 PIN | WMI + 工厂码 + 年份码查表 | 工厂、年份、发动机系列、产地 |
| **Kubota** | 字母+数字混合 | 型号前缀匹配 + 年份范围推断 | 型号系列、年份范围、产地 |
| **Massey Ferguson** | 9-17位 | 型号前缀 + 工厂码 + 序列段 | 系列、工厂、年份 |
| **Case IH** | 8-17位 | 型号前缀 + 数字型号映射 | 型号、发动机、产地 |
| **New Holland** | 9-17位 | T-series 前缀 + Boomer/Workmaster 分支 | 系列、工厂、年份 |

### 已知限制

这些是当前实现的真实短板，写在这里以免把输出当作权威结论使用：

1. **品牌判定依赖尝试顺序，而不是真正的 WMI 识别。**
   各品牌的正则存在大面积重叠 —— `^[A-Z0-9]{17}$` 同时属于 John Deere、
   Massey Ferguson 和 New Holland，而它又是 Case IH `^[A-Z0-9]{8,17}$` 的子集。
   实际结果完全取决于 `decoder.go` 中 `decoders` 切片的顺序，
   因此一个 17 位字母数字串总会被判为 John Deere。

2. **不校验 VIN 校验位。** 标准 VIN 的第 9 位是 mod-11 校验位，本实现未做校验，
   所以一个号码打错一位仍会返回一个看起来很确定的错误结果。

3. **年份码表只覆盖 2010–2026，且年份字母实际以 30 年为周期循环。**
   同一个字母在 1980 / 2010 / 2040 年是同一个值。表外的年份会返回空字符串，
   而落在表内的老机器会被报成一个并不正确的近年份。

4. **型号提取是启发式的。** 例如 `1RW8136PABCD12345` 会得到 `W813系列`，
   这只是把序列号的第 3–6 位当作型号段，并不代表真实型号。

5. **Kubota 的"年份"是区间**（如 `1990-1995`），而不是确切年份。

以上任何一条要真正解决都需要引入各厂商的 WMI 前缀表和校验位算法，
属于路线图里尚未完成的工作。

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

### v0.5 — 正确性与服务端加固 (当前)
- **删除 John Deere 的"部分匹配"兜底分支**。该分支只要输入满足
  `len >= 5 && HasPrefix(serial, "1")` 就放行，而 John Deere 排在品牌派发
  列表首位，导致 `1ZZZZ` 这类无意义输入被返回为 John Deere（全字段为空），
  且 Kubota / Massey Ferguson / New Holland / Case IH 几乎永远不可达
- **修复 Kubota 型号解析的非确定性**。原实现 `for prefix := range kubotaModelPrefix`
  遍历 map，而键之间互为前缀（"L" 是 "L35" 的前缀，"M" 是 "M5"/"MX" 的前缀，
  "B" 是 "BX" 的前缀），map 迭代顺序随机，同一序列号可能得到不同型号。
  这正是 v0.4 声称已在 decoder.go 修好的那类 bug，当时漏掉了品牌文件内部
- HTTP 服务加超时（ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout）。
  原实现用 `http.ListenAndServe`，四个超时全为 0，存在 Slowloris 风险
- 请求体限制 4 KiB（`http.MaxBytesReader`，超出返回 413），
  序列号长度限制 32 字符（API 层与解码器层各校验一次）
- 响应统一设置 `Content-Type: application/json` 与 `X-Content-Type-Options: nosniff`，
  原先由 Go 嗅探成 text/plain
- 新增 `-host` 参数（默认 0.0.0.0 以配合容器），
  并修正启动日志——原先无论绑定到哪都打印 "localhost"
- `api.Serve` 返回 error 而不是在库代码里 `log.Fatal`/`os.Exit`
- 抽出 `newRouter()` 使路由可被 httptest 直接测试
- 移除 CORS 中误导性的 `Authorization` 允许头（本服务没有鉴权）
- **补交 go.sum**（此前从未提交），Dockerfile 随之 `COPY go.mod go.sum`
- chi v5.1.0 → v5.3.2（修复 Host Header 注入导致的重定向问题与 RealIP 的
  X-Forwarded-For 伪造问题）；go.mod 的 go 指令随之升到 1.23
- Dockerfile：基础镜像升到 golang:1.27-alpine / alpine:3.23（原 1.22 已不满足
  chi 的版本要求），新增非 root USER、HEALTHCHECK、.dockerignore、
  `go mod verify` 与 `-trimpath -ldflags="-s -w"`
- 新增 20 个单元测试（decoder 10 个 + api 10 个），覆盖上述全部回归点，
  含"任何输入都不得 panic"与"结果必须稳定"两类不变式

### v0.4 — 解码器策略模式重构 + CORS
- 所有品牌解码器统一 `Decoder` 接口
- 新增 Case IH 和 New Holland 解码器
- chi 中间件栈（日志 + 恢复 + RealIP + CORS）
- `DecodedInfo` 字段名跟 TractorCompare 对齐
- Docker 多阶段构建
- 注：本次改动同时把品牌派发从 map 改为有序切片。但 v0.5 发现品牌文件
  内部仍有多处 map 迭代，且 John Deere 的兜底分支使派发顺序形同虚设

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
