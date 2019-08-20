FROM golang:1.12-buster

# Rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH \
    RUST_VERSION=1.37.0

RUN curl -sSLf -o rustup-init "https://static.rust-lang.org/rustup/archive/1.18.3/x86_64-unknown-linux-gnu/rustup-init" \
 && echo 'a46fe67199b7bcbbde2dcbc23ae08db6f29883e260e23899a88b9073effc9076  rustup-init' | sha256sum -c \
 && chmod +x rustup-init \
 && ./rustup-init -y --no-modify-path --default-toolchain "$RUST_VERSION" \
 && rm rustup-init \
 && chmod -R a+w $RUSTUP_HOME $CARGO_HOME

# Tools
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
        cmake \
 && apt-get clean

WORKDIR /usr/src

# Current host user
ARG UNAME=quiche
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID -o $UNAME && \
    useradd -m -u $UID -g $GID -o -s /bin/bash $UNAME
USER $UNAME
