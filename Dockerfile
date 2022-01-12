FROM python:3-alpine as base

FROM base as python
RUN pip install --no-cache-dir awscli boto3 requests && \
    mkdir -p /opt/dispatch
ADD src/*.py /opt/dispatch/

FROM python as lint
RUN pip install pylint && \
    cd /opt && \
    pylint ./dispatch

FROM python as publish
#install KOPS
#RUN KOPS_VERSION=$(curl -s https://api.github.com/repos/kubernetes/kops/releases/latest | grep tag_name | cut -d '"' -f 4) && \
RUN KOPS_VERSION="v1.21.4" && \
    apk add --no-cache curl openssh-keygen && \
    curl -Lo /usr/local/bin/kops https://github.com/kubernetes/kops/releases/download/${KOPS_VERSION}/kops-linux-amd64 && \
    chmod +x /usr/local/bin/kops

#install kubectl
RUN KUBE_VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt) && \
    curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBE_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/local/bin/kubectl

#install Helm
RUN HELM_VERSION=$(curl -s https://github.com/helm/helm/releases/latest | cut -d '/' -f 8 | sed 's/">redirected<//') && \
    mkdir /opt/helm && \
    curl https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar xz --directory /opt/helm && \
    ln -s /opt/helm/linux-amd64/helm /usr/local/bin/helm

CMD ["python", "/opt/dispatch/main.py"]
