FROM golang:1.17-alpine

ENV DEBUG "false"
ENV WHALEIO_TOKEN ""
ENV WHALEIO_SCOPE_CURRENCIES ""

WORKDIR /app
COPY go.* /app/
COPY main.go /app/
RUN go get github.com/rs/zerolog/log
RUN go build main.go
CMD ["/app/main"]
