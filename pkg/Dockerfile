FROM docker.io/library/golang:alpine AS source

WORKDIR $GOPATH/src/github.com/christiantragesser/dispatch
ADD go.mod .
ADD go.sum .
ADD main.go .
COPY dispatch ./dispatch
COPY status ./status
COPY tuiaction ./tuiaction
COPY tuicreate ./tuicreate

RUN go get -d -v

FROM source as linux-build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' -a \
    -o /go/bin/dispatch-linux-amd64 .

FROM gcr.io/distroless/static AS publish
COPY --from=linux-build /go/bin/dispatch-linux-amd64 /usr/local/bin/dispatch

CMD [ "dispatch" ]