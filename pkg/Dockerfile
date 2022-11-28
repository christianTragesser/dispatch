FROM docker.io/library/golang:alpine AS source

WORKDIR $GOPATH/src/github.com/christiantragesser/dispatch
ADD go.mod .
ADD go.sum .
ADD main.go .
ADD .goreleaser.yaml .
COPY dispatch ./dispatch
COPY tuiaction ./tuiaction
COPY tuicreate ./tuicreate
COPY tuidelete ./tuidelete

FROM goreleaser/goreleaser as build
COPY --from=source $GOPATH/src/github.com/christiantragesser/dispatch $GOPATH/src/github.com/christiantragesser/dispatch
WORKDIR $GOPATH/src/github.com/christiantragesser/dispatch
RUN goreleaser build --snapshot

FROM amazon/aws-cli AS publish
COPY --from=build /go/src/github.com/***/dispatch/dist/dispatch_linux_amd64_v1/dispatch-linux-amd64 /usr/local/bin/dispatch

#install kubectl
RUN KUBE_VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt) && \
    curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBE_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/local/bin/kubectl

ENTRYPOINT [ "" ]
CMD [ "/bin/bash" ]