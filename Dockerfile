FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION

WORKDIR /build
COPY . .


RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X main.Version=${VERSION}" -o coredock .

FROM alpine:latest

RUN apk --no-cache add curl coredns vim

WORKDIR /app
COPY entrypoint.sh .
COPY --from=builder build/coredock .

RUN chmod +x entrypoint.sh coredock
ENTRYPOINT ["./entrypoint.sh"]
