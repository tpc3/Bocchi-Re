FROM golang:alpine AS build
RUN apk add --no-cache git build-base

WORKDIR /go/src/Bocchi-Re

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=1 \
    CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN go build -tags musl -ldflags "-s -w" -o Bocchi-Re .

FROM alpine
RUN apk add --no-cache ca-certificates

COPY --from=build /go/src/Bocchi-Re/Bocchi-Re /bin/Bocchi-Re
WORKDIR /data
CMD ["/bin/Bocchi-Re"]