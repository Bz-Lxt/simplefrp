# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:22.14.0-alpine3.21 AS frontend
WORKDIR /web
COPY frontend-user/package.json frontend-user/package-lock.json* ./
RUN npm install
COPY frontend-user/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.23.6-alpine3.21 AS gobuild
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/client ./cmd/client \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/demo ./cmd/demo

FROM alpine:3.21.3 AS server
RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 app
USER app
WORKDIR /app
COPY --from=gobuild /out/server /app/server
ENV TZ=Asia/Shanghai
EXPOSE 7000 8080 9090
ENTRYPOINT ["/app/server"]

FROM alpine:3.21.3 AS client
RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 app
USER app
WORKDIR /app
COPY --from=gobuild /out/client /app/client
ENV TZ=Asia/Shanghai
EXPOSE 9091
ENTRYPOINT ["/app/client"]

FROM alpine:3.21.3 AS demo
RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 app
USER app
WORKDIR /app
COPY --from=gobuild /out/demo /app/demo
COPY --from=frontend /web/dist /app/static
ENV TZ=Asia/Shanghai
ENV SIMPLEFRP_STATIC_DIR=/app/static
EXPOSE 8080
ENTRYPOINT ["/app/demo"]
