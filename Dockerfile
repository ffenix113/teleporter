FROM flayer/tdlib:v1.8.0 as tdlib

FROM golang:1.18 as base
WORKDIR /src

ARG CLANG_VERSION=11

RUN apt-get update
RUN apt-get install -y libc++1 libc++abi1-${CLANG_VERSION}

COPY . /src
COPY --from=tdlib /src/td/tdlib /src/td/tdlib
RUN go mod vendor

RUN make build CLANG_VERSION=${CLANG_VERSION}

FROM debian:bullseye-slim

COPY --from=base /src/td/tdlib/lib /src/td/tdlib/lib
COPY --from=base '/usr/lib/aarch64-linux-gnu/libc++.so.1' '/usr/lib/aarch64-linux-gnu/'
COPY --from=base '/usr/lib/aarch64-linux-gnu/libc++abi.so.1' '/usr/lib/aarch64-linux-gnu/'
COPY --from=base '/usr/lib/aarch64-linux-gnu/libatomic.so.1' '/usr/lib/aarch64-linux-gnu/'
COPY --from=base /src/main /src/server

ENTRYPOINT ["/src/server"]
