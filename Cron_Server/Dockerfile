FROM golang:1.22.2 as base

WORKDIR /Cron_Server

USER root
# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify
EXPOSE 9000

FROM base as production
COPY . .
RUN go build -o cronServer
CMD [ "./cronServer" ]