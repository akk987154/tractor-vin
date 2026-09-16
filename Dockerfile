# syntax=docker/dockerfile:1

# ---------- 构建阶段 ----------
# 使用当前受支持的 Go 版本。原先固定 golang:1.22-alpine，
# 而 chi v5.3.2 要求 go >= 1.23，旧镜像会触发工具链自动下载。
FROM golang:1.27-alpine AS builder

WORKDIR /app

# 先只复制依赖清单，让这一层能被 Docker 缓存。
# go.sum 必须一起复制：它是模块完整性的唯一凭据，
# 没有它就无法校验下载到的模块，也无法复现构建
# （原 Dockerfile 只复制 go.mod，且 go.sum 从未提交到仓库）。
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# CGO_ENABLED=0 产生静态链接的二进制，才能跑在 alpine 上
# -trimpath 去掉构建机上的绝对路径
# -ldflags="-s -w" 去掉符号表与调试信息
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/tractor-vin .

# ---------- 运行阶段 ----------
FROM alpine:3.23

# 只安装 ca-certificates；不引入额外工具以缩小攻击面。
# 同时创建非 root 用户。
RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 10001 app && \
    adduser -S -u 10001 -G app app

WORKDIR /app
COPY --from=builder /out/tractor-vin /usr/local/bin/tractor-vin

# 原 Dockerfile 没有任何 USER 指令，容器以 root 运行。
# 本程序不写任何文件（无 os.Create / WriteFile / OpenFile），降权是零成本的。
USER 10001:10001

EXPOSE 8080

# 原先虽然已经暴露 /health 接口却没有健康检查。
# alpine 自带 busybox wget，无需额外安装。
# 默认端口为 8080；若用 -port 改了端口，需要同步调整这里。
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["tractor-vin"]
CMD ["serve"]
