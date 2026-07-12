# ============================================================
# Stage 1: 构建前端 (Vue + Vite)
# ============================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build/frontend

# 先复制依赖文件，利用 Docker 缓存层
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

# 复制前端源码并构建
COPY frontend/ ./
RUN npm run build

# ============================================================
# Stage 2: 构建后端 (Go)
# ============================================================
FROM golang:1.26-alpine AS backend-builder

WORKDIR /build

# 复制 Go 模块文件
COPY go.mod go.sum ./
RUN go mod download

# 复制 Go 源码
COPY core/ ./core/
COPY model/ ./model/
COPY handler/ ./handler/
COPY cmd/ ./cmd/
COPY plugin/ ./plugin/
COPY ui/ ./ui/

# 复制前端构建产物到 ui/dist/
COPY --from=frontend-builder /build/frontend/dist/ ./ui/dist/

# 编译 Go 二进制（CGO_ENABLED=0 保证静态链接，兼容 alpine）
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o xiuno-server ./cmd/xiuno/

# ============================================================
# Stage 3: 运行阶段 (最小 alpine 镜像)
# ============================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 复制二进制和配置
COPY --from=backend-builder /build/xiuno-server .
COPY xiuno.json .

# 创建运行时目录
RUN mkdir -p upload/attach upload/avatar upload/forum log

EXPOSE 8080

CMD ["./xiuno-server"]
