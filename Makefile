.PHONY: tdlib build

CGO_LDFLAGS += -lcrypto
CGO_LDFLAGS += -L/usr/lib/x86_64-linux-gnu
CGO_LDFLAGS += -L$(PWD)/td/build
CGO_LDFLAGS += -L$(PWD)/td/build/tddb
CGO_LDFLAGS += -L$(PWD)/td/build/tdactor
CGO_LDFLAGS += -L$(PWD)/td/build/sqlite
CGO_LDFLAGS += -L$(PWD)/td/build/tdnet
CGO_LDFLAGS += -L$(PWD)/td/build/tdutils
CGO_CFLAGS += -I$(PWD)/td/tdlib/include

# Can be a commit as well.
TDLIB_VERSION = v1.8.0

CLANG_VERSION = 13
CLANG = /usr/bin/clang-$(CLANG_VERSION)
CLANG_PP = /usr/bin/clang++-$(CLANG_VERSION)

build: tdlib
	CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS) -stdlib=libc++" CC=$(CLANG) go build main.go

$(CLANG_PP):
	apt-get update
	apt-get upgrade
	apt-get install make git zlib1g-dev libssl-dev gperf cmake clang-$(CLANG_VERSION) libc++-dev libc++abi-dev llvm-$(CLANG_VERSION)

td: $(CLANG_PP)
	rm -rf td
	git clone https://github.com/tdlib/td.git \
	git checkout master && git pull && \
	git checkout $(TDLIB_VERSION)


tdlib: td
	cd td && \
	rm -rf build && \
	mkdir build && \
	cd build && \
	CXXFLAGS="-stdlib=libc++" CC=$(CLANG) CXX=$(CLANG_PP) cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib .. && \
	cmake --build . --target install -j5 && \
	make install

