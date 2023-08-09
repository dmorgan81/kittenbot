FROM tdewolff/minify:latest AS minify-stage

WORKDIR /assets

COPY internal/page/assets ./
RUN minify -b -o bin/latest.html -r .

FROM golang:1.20 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go main.go
COPY internal/ internal/
COPY --from=minify-stage /assets/bin/latest.html internal/page/assets/latest.html
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build --ldflags '-extldflags "-static"' -o /kittenbot

FROM public.ecr.aws/lambda/go:1

COPY --from=build-stage /kittenbot ${LAMBDA_TASK_ROOT}/kittenbot
RUN ln -s ${LAMBDA_TASK_ROOT}/kittenbot ${LAMBDA_TASK_ROOT}/kittenbot-html && \
    ln -s ${LAMBDA_TASK_ROOT}/kittenbot ${LAMBDA_TASK_ROOT}/kittenbot-image
