FROM alpine:latest

RUN apk --no-cache add curl coredns

WORKDIR /app
COPY Corefile .
COPY entrypoint.sh .
COPY build/coredock .

RUN chmod +x entrypoint.sh coredock
ENTRYPOINT ["./entrypoint.sh"]
