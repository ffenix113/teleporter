.PHONY: tdlib build
# C defines concurrency level for tdlib build
C = 5

# For building service for other architectures
GOOS = linux
GOARCH = amd64

CGO_LDFLAGS += -lcrypto
CGO_LDFLAGS += -L/usr/lib/aarch64-linux-gnu
CGO_LDFLAGS += -L$(PWD)/td/build
#CGO_LDFLAGS += -L$(PWD)/td/build/tddb
#CGO_LDFLAGS += -L$(PWD)/td/build/tdactor
#CGO_LDFLAGS += -L$(PWD)/td/build/sqlite
#CGO_LDFLAGS += -L$(PWD)/td/build/tdnet
#CGO_LDFLAGS += -L$(PWD)/td/build/tdutils
CGO_LDFLAGS += -L$(PWD)/td/tdlib/lib
CGO_CFLAGS += -I$(PWD)/td/tdlib/include

# Can be a commit as well.
TDLIB_VERSION = v1.8.0

CLANG_VERSION = 11
CLANG = /usr/bin/clang-$(CLANG_VERSION)
CLANG_PP = /usr/bin/clang++-$(CLANG_VERSION)

build: $(CLANG_PP)
	CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS) -stdlib=libc++" CC=$(CLANG) go build main.go

build-docker:
	test ! "X$(VERSION)" = "X"  || ( echo "VERSION is not defined" && exit 1 )
	docker buildx build --platform linux/arm64 -t flayer/teleporter:$(VERSION) -f Dockerfile --push .

$(CLANG_PP):
	apt-get update
	apt-get install -y make git zlib1g-dev libssl-dev gperf cmake clang-$(CLANG_VERSION) libc++-dev libc++abi-dev llvm-$(CLANG_VERSION)

td:
	rm -rf td
	git clone https://github.com/tdlib/td.git && \
	cd td && git checkout master && \
	git checkout $(TDLIB_VERSION)


tdlib: td $(CLANG_PP)
	cd td && \
	patch -i ../td_CMakeLists.txt.patch CMakeLists.txt && \
	rm -rf build && \
	mkdir build && \
	cd build && \
	CXXFLAGS="-stdlib=libc++" CC=$(CLANG) CXX=$(CLANG_PP) $(CMAKE) -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib .. && \
	cmake --build . --target install -j$(C)

tdlib-split: td $(CLANG_PP)
	cd td && \
	patch -i ../td_CMakeLists.txt.patch CMakeLists.txt && \
	rm -rf build && \
	mkdir build && \
	cd build && \
	CXXFLAGS="-stdlib=libc++" CC=$(CLANG) CXX=$(CLANG_PP) $(CMAKE) -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib .. && \
	cmake --build . --target prepare_cross_compiling -j$(C) && \
	cd .. && \
	php SplitSource.php && \
	cd build && \
	cmake --build . --target install -j$(C) && \
	cd .. && \
	php SplitSource.php --undo

gen-certs:
	openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes
