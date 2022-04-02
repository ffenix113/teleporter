FROM tdlib:1.8.0 as base
WORKDIR /src

COPY . /src

ARG GOOS=linux
ARG GOARCH=amd64

RUN make build GOOS=$GOOS GOARCH=$GOARCH

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y libssl1.1 libc++-13

COPY --from=base /src/main /src/server
COPY --from=base /src/td/tdlib /src/td/tdlib

ENTRYPOINT ["/src/server"]
