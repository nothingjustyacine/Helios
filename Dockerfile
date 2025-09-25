# 使用官方 Go 镜像作为构建阶段
FROM golang:1.23.7-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# 使用轻量级的 Alpine 镜像作为运行阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates sqlite

# 创建非 root 用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 创建数据目录并设置权限
RUN mkdir -p /data && \
    chown -R appuser:appgroup /app /data

# 注释掉非 root 用户，使用 root 运行
# USER appuser

# 暴露端口
EXPOSE 8080

# 设置环境变量默认值（用户需要在运行时覆盖）
ENV USERNAME=your_username
ENV PASSWORD=your_password
ENV SUBSCRIPTION_URL=https://your_subscription_url.com

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/login || exit 1

# 启动应用程序
CMD ["./main"]
