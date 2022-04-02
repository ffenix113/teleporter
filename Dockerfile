FROM tdlib:1.8.0 as base
WORKDIR /src

COPY . /src

RUN make build

ENTRYPOINT ["/src/main"]
