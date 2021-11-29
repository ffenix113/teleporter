.PHONY: tdlib build

CGO_LDFLAGS += -lcrypto
CGO_LDFLAGS += -L/usr/lib/x86_64-linux-gnu
CGO_LDFLAGS += -L/home/eugene/GoProjects/teleporter/td/build
CGO_LDFLAGS += -L/home/eugene/GoProjects/teleporter/td/build/tddb
CGO_LDFLAGS += -L/home/eugene/GoProjects/teleporter/td/build/tdactor
CGO_LDFLAGS += -L/home/eugene/GoProjects/teleporter/td/build/sqlite
CGO_LDFLAGS += -L/home/eugene/GoProjects/teleporter/td/build/tdnet
CGO_LDFLAGS += -L/home/eugene/GoProjects/teleporter/td/build/tdutils
CGO_CFLAGS += -I/home/eugene/GoProjects/teleporter/td/tdlib/include

/usr/bin/clang++-13:
	apt-get update
	apt-get upgrade
	aptitude install make git zlib1g-dev libssl-dev gperf cmake clang-13 libc++-dev libc++abi-dev llvm-13

td: /usr/bin/clang++-13
	rm -rf td
	git clone https://github.com/tdlib/td.git

tdlib: td
	cd td && \
	rm -rf build && \
	mkdir build && \
	cd build && \
	CXXFLAGS="-stdlib=libc++" CC=/usr/bin/clang-13 CXX=/usr/bin/clang++-13 cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib .. && \
	cmake --build . --target install -j5 && \
	make install

build:
	CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS) -stdlib=libc++" CC=clang-13 go build main.go
