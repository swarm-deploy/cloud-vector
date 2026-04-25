FROM golang:1.24.7-alpine AS proxy

WORKDIR /app

COPY ./go.* ./

RUN go mod download

COPY ./ .

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=${APP_VERSION}' -X 'main.BuildDate=${BUILD_TIME}'" -o /go/bin/proxy /app/cmd/main.go

##############
## Base Image
##############

FROM alpine:3.20

WORKDIR /app

RUN wget -O vector.tar.gz https://github.com/vectordotdev/vector/releases/download/v0.55.0/vector-0.55.0-x86_64-unknown-linux-musl.tar.gz && \
    tar -xvf vector.tar.gz && \
    cp vector-x86_64-unknown-linux-musl/bin/vector /usr/bin/vector && \
    cp -R vector-x86_64-unknown-linux-musl/etc/systemd /etc/systemd && \
    rm vector.tar.gz

COPY vector.toml /etc/vector/vector.toml

COPY --from=proxy /go/bin/proxy /go/bin/proxy

COPY entrypoint.sh /app/entrypoint.sh

LABEL org.swarm-deploy.service.type=monitoring

ENTRYPOINT ["/app/entrypoint.sh"]
