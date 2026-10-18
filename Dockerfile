FROM golang:1.27.2-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY webhooksig ./webhooksig
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/labd ./cmd/labd

FROM scratch

ENV ADDR=:8080
COPY --from=build --chown=65532:65532 /out/labd /labd
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/labd"]
