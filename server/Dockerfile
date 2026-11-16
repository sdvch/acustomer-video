FROM golang:1.26-alpine AS builder
# 允许构建时覆盖模块代理（国内 ECS 直连 proxy.golang.org 经常超时）
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20
# 国内构建时把 alpine 源换成镜像站，否则 apk 下载 ffmpeg 依赖会长时间卡住
ARG APK_MIRROR=""
RUN if [ -n "$APK_MIRROR" ]; then \
        sed -i "s|dl-cdn.alpinelinux.org|${APK_MIRROR}|g" /etc/apk/repositories; \
    fi \
    && apk add --no-cache ffmpeg ca-certificates tzdata \
    && mkdir -p /app/storage /app/migrations /app/assets/fonts
WORKDIR /app
COPY --from=builder /server /app/server
COPY migrations /app/migrations
COPY assets /app/assets
ENV TZ=Asia/Shanghai
EXPOSE 18080
CMD ["/app/server"]
