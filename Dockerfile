FROM docker.io/library/golang:alpine AS source

RUN apk --no-cache add curl && \
    KOPS_VERSION="v1.21.4" && \
    curl -Lo /usr/local/bin/kops https://github.com/kubernetes/kops/releases/download/${KOPS_VERSION}/kops-linux-amd64 && \
    chmod +x /usr/local/bin/kops

WORKDIR $GOPATH/src/github.com/christiantragesser/dispatch
ADD go.mod .
ADD go.sum .
ADD main.go .
COPY dispatch ./dispatch
COPY status ./status
COPY tuiaction ./tuiaction
COPY tuicreate ./tuicreate
COPY tuidelete ./tuidelete

RUN go get -d -v

FROM source as linux-build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' -a \
    -o /go/bin/dispatch-linux-amd64 . && \
    sha256sum /go/bin/dispatch-linux-amd64 > /opt/dispatch-linux-amd64.sha256

FROM source AS macos-build
RUN CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' -a \
    -o /go/bin/dispatch-darwin-amd64 . && \
    sha256sum /go/bin/dispatch-darwin-amd64 > /opt/dispatch-darwin-amd64.sha256

FROM scratch AS linux-binary
COPY --from=linux-build /go/bin/dispatch-linux-amd64 /dispatch-linux-amd64
COPY --from=linux-build /opt/dispatch-linux-amd64.sha256 /dispatch-linux-amd64.sha256

FROM scratch AS macos-binary
COPY --from=macos-build /go/bin/dispatch-darwin-amd64 /dispatch-darwin-amd64
COPY --from=macos-build /opt/dispatch-darwin-amd64.sha256 /dispatch-darwin-amd64.sha256

FROM gcr.io/distroless/static as publish
COPY --from=source /usr/local/bin/kops /usr/local/bin/kops
COPY --from=linux-build /go/bin/dispatch-linux-amd64 /usr/local/bin/dispatch

CMD [ "dispatch" ]