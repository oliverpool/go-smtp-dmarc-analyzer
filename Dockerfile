FROM golang as build-env

RUN mkdir /build
WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN make test
RUN make build

# Runner
FROM gcr.io/distroless/base

COPY --from=build-env /build/app /

CMD ["/app"]
