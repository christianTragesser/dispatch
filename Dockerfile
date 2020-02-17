FROM python:3-alpine

#install KOPS
RUN apk add --no-cache curl openssh-keygen && \
curl -Lo /usr/local/bin/kops https://github.com/kubernetes/kops/releases/download/$(curl -s https://api.github.com/repos/kubernetes/kops/releases/latest | grep tag_name | cut -d '"' -f 4)/kops-linux-amd64 && \
chmod +x /usr/local/bin/kops

#install kubectl
RUN curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
chmod +x /usr/local/bin/kubectl

#install Helm
# RUN HELM_VERSION=$(curl -s https://github.com/helm/helm/releases/latest | cut -d '/' -f 8 | sed 's/">redirected<//') && \
RUN HELM_VERSION='v3.1.0' && \
    mkdir /opt/helm && \
    curl https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar xz --directory /opt/helm && \
    ln -s /opt/helm/linux-amd64/helm /usr/local/bin/helm

#install aws utilities
RUN pip install --no-cache-dir awscli boto3 && mkdir -p /opt/dispatch

ADD *.py /opt/dispatch/

CMD ["python", "/opt/dispatch/main.py"]
