FROM golang:1.22-alpine

WORKDIR /draft

RUN apk add gcc musl-dev python3-dev libffi-dev openssl-dev cargo make
RUN apk add py3-pip

RUN python3 -m venv az-cli-env
RUN az-cli-env/bin/pip install --upgrade pip
RUN az-cli-env/bin/pip install --upgrade setuptools
RUN az-cli-env/bin/pip install --upgrade azure-cli
RUN az-cli-env/bin/pip install --upgrade setuptools
RUN az-cli-env/bin/az --version

ENV PATH "$PATH:/draft/az-cli-env/bin"

RUN apk add github-cli

COPY . ./
RUN make go-generate

RUN go mod download
ENTRYPOINT ["go"]
