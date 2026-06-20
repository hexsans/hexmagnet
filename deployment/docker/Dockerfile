FROM --platform=$BUILDPLATFORM node:24-alpine AS webui-builder
WORKDIR /webui
COPY webui/package.json webui/package-lock.json ./
RUN npm ci --prefer-offline
COPY webui/ .
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG GOTAGS
ARG VERSION=dev
ARG TARGETOS TARGETARCH TARGETVARIANT

RUN apk add --no-cache git tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

COPY --from=webui-builder /webui/dist ./webui/dist

RUN --mount=type=cache,target=/root/.cache/go-build \
    export GOOS=$TARGETOS; \
    export GOARCH=$TARGETARCH; \
    [ "$GOARCH" = "amd64" ] && [ -n "$TARGETVARIANT" ] && export GOAMD64=$TARGETVARIANT; \
    [ "$GOARCH" = "arm" ] && [ "$TARGETVARIANT" = "v6" ] && export GOARM=6; \
    [ "$GOARCH" = "arm" ] && [ "$TARGETVARIANT" = "v7" ] && export GOARM=7; \
    go build -tags "${GOTAGS}" -ldflags "-s -w -X github.com/hexsans/hexmagnet/internal/version.GitTag=${VERSION}" -o /build/hexmagnet main.go

FROM alpine:latest AS runner

COPY --link --from=build /build/hexmagnet /usr/local/bin/hexmagnet

WORKDIR /app

ENTRYPOINT ["/usr/local/bin/hexmagnet"]
