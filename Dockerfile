FROM python:2.7-alpine

#install KOPS
RUN apk add --no-cache curl openssh-keygen && \
curl -Lo /usr/local/bin/kops https://github.com/kubernetes/kops/releases/download/$(curl -s https://api.github.com/repos/kubernetes/kops/releases/latest | grep tag_name | cut -d '"' -f 4)/kops-linux-amd64 && \
chmod +x /usr/local/bin/kops

#install kubectl
RUN curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
chmod +x /usr/local/bin/kubectl

#install aws utilities
RUN pip install --no-cache-dir awscli boto3 && mkdir -p /opt/dispatch

ADD *.py /opt/dispatch/

CMD ["python", "/opt/dispatch/main.py"]
