FROM golang:alpine AS build
RUN apk add git gcc musl-dev
ENV CGO_ENABLED 1
ADD . /go/src/Bocchi-Re/
WORKDIR /go/src/Bocchi-Re
RUN go build .

FROM alpine
COPY --from=build /go/src/Bocchi-Re/Bocchi-Re /bin/Bocchi-Re
WORKDIR /data
CMD Bocchi-Re
