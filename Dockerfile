FROM golang:1.18-bullseye as builder
ARG CLANG_VERSION=13

WORKDIR /srv
COPY . .
RUN apt-get update && apt-get install -y lsb-release wget software-properties-common &&  \
    wget https://apt.llvm.org/llvm.sh && \
    chmod +x llvm.sh && \
    ./llvm.sh $CLANG_VERSION
RUN apt-get install -y libc++-13-dev libssl-dev gperf cmake libc++abi-${CLANG_VERSION}-dev zlib1g-dev
RUN make

FROM scratch
COPY --from=builder /srv/main /main
ENTRYPOINT ["/main"]
