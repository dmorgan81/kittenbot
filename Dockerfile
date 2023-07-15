FROM golang:1.20 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go index.html ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build --ldflags '-extldflags "-static"' -o /kittenbot

FROM public.ecr.aws/lambda/go:1

COPY --from=build-stage /kittenbot ${LAMBDA_TASK_ROOT}

CMD ["kittenbot"]
