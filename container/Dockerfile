FROM golang:1.22.0-alpine3.19 AS builder

ENV CGO_ENABLED=0
ARG VERSION
ARG COMMIT
ARG DATE
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN go build -o /workspace/platformer -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" ./

FROM alpine:3.19.1 AS production

RUN apk add --no-cache git ca-certificates

WORKDIR /
COPY --from=builder /workspace/platformer .
USER 65532:65532

CMD ["/git-releaser", "update"]