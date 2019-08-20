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
