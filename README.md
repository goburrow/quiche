Go binding for [Cloudflare Quiche](https://github.com/cloudflare/quiche)

```
git clone --recursive https://github.com/goburrow/quiche
```

## Build

Build environment:

```
docker build -t quiche:builder -f docker/Dockerfile docker/
docker run -i -t -v "$PWD:/usr/src/quiche" -w "/usr/src/quiche" quiche:builder
```

Build dependencies:

```
cd deps/quiche
cargo build --release
```


Build application using this Go library:

```
GO_LDFLAGS="-L/absolute/path/to/libquiche" go build
```

To create a static binary, `CGO_LDFLAGS` may need to include `-ldl` (Linux) or `-framework Security` (MacOS)
