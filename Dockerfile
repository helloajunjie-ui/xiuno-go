FROM golang:latest AS builder

WORKDIR /build

# 安装 Node.js 和 npm
RUN apt-get update && apt-get install -y nodejs npm && rm -rf /var/lib/apt/lists/*

# 复制 Go 模块文件
COPY Xiuno/go.mod Xiuno/go.sum ./Xiuno/
WORKDIR /build/Xiuno
RUN go mod download

# 复制所有源码（包括 xiuno-ui）
WORKDIR /build
COPY xiuno-ui ./xiuno-ui/
COPY Xiuno ./Xiuno/

# 构建前端
WORKDIR /build/xiuno-ui
RUN npm install && npm run build

# 复制前端产物到 ui/dist
WORKDIR /build/Xiuno
RUN mkdir -p ui/dist && cp -r /build/xiuno-ui/dist/* ui/dist/

# 编译 Go 二进制（CGO_ENABLED=0 保证静态链接）
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o xiuno-server ./cmd/xiuno/

# 运行阶段
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/Xiuno/xiuno-server .
COPY Xiuno/xiuno.json .

RUN mkdir -p upload/attach upload/avatar upload/forum log

EXPOSE 8080

CMD ["./xiuno-server"]
