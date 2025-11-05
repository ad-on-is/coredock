FROM golang:1.21-alpine AS corednsbuilder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION


# Install git and other build dependencies
RUN apk add --no-cache git make
WORKDIR /coredns
RUN git clone https://github.com/coredns/coredns.git . && \
  git checkout v1.11.1

RUN echo "fanout:github.com/networkservicemesh/fanout" >> plugin.cfg

RUN go generate
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} make


FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS coredockbuilder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION

WORKDIR /build
COPY . .


RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X main.Version=${VERSION}" -o coredock .

FROM alpine:latest

RUN apk --no-cache add curl vim ca-certificates

WORKDIR /app
COPY entrypoint.sh .
COPY --from=coredockbuilder build/coredock .
COPY --from=corednsbuilder /coredns/coredns .

RUN chmod +x entrypoint.sh coredock
ENTRYPOINT ["./entrypoint.sh"]
