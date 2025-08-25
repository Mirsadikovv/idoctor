FROM golang:1.24.2-alpine3.20 as builder

ENV GOPATH /go
ENV PATH $PATH:$GOPATH/bin
ENV GOPRIVATE=git.sriss.uz

RUN set -ex && \
  apk add --no-cache gcc musl-dev git


ARG CI_JOB_LOGIN
ARG CI_JOB_TOKEN

WORKDIR /app

COPY go.mod ./

RUN git config --global url."https://git.sriss.uz/".insteadOf "http://git.sriss.uz/"

RUN printf "machine git.sriss.uz\nlogin %s\npassword %s\n" "$CI_JOB_LOGIN"  "$CI_JOB_TOKEN" > ~/.netrc && \
  chmod 600 ~/.netrc

RUN go mod download

COPY . .

ARG HTTP_PORT=80
EXPOSE ${HTTP_PORT}

# Сборка Go-приложения
RUN go build -ldflags "-s -w" -o main .