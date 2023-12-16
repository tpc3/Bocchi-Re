FROM golang:alpine AS build
ADD . /go/src/Bocchi-Re/
ENV CGO_ENABLED 0
WORKDIR /go/src/Bocchi-Re
RUN go build .

FROM alpine
COPY --from=build /go/src/Bocchi-Re/Bocchi-Re /bin/Bocchi-Re
WORKDIR /data
CMD Bocchi-Re
