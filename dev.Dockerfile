FROM golang:1.24.3

WORKDIR /app

ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
COPY . .
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build go build -o nacp ./cmd/nacp

FROM ubuntu
COPY --from=0 /app/nacp /nacp
COPY --from=0 /etc/passwd /etc/passwd
USER nobody
ENTRYPOINT ["/nacp"]
