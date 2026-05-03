ARG COREDNS_IMAGE
FROM ${COREDNS_IMAGE} AS corednsbuilder

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS coredockbuilder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION

WORKDIR /build
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod,id=coredock-gomod \
    --mount=type=cache,target=/root/.cache/go-build,id=coredock-gobuild-${TARGETARCH} \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X main.Version=${VERSION}" -o coredock .

FROM alpine:latest

RUN apk --no-cache add curl coredns

WORKDIR /app
COPY entrypoint.sh .
COPY --from=corednsbuilder /coredns/coredns .
COPY --from=coredockbuilder /build/coredock .

RUN chmod +x entrypoint.sh coredock
ENTRYPOINT ["./entrypoint.sh"]
