# syntax=docker/dockerfile:1.7
FROM golang:1.26-bookworm AS highs-build

RUN apt-get update && apt-get install -y --no-install-recommends \
    cmake ninja-build git g++ pkg-config \
    && rm -rf /var/lib/apt/lists/*

ARG HIGHS_VERSION=v1.15.1
ARG HIGHS_COMMIT=04024d701f79feb8e2f18bc3df0dffc04ef05088
RUN git clone --depth 1 --branch "${HIGHS_VERSION}" https://github.com/ERGO-Code/HiGHS.git /src/highs \
    && test "$(git -C /src/highs rev-parse HEAD)" = "${HIGHS_COMMIT}" \
    && cmake -S /src/highs -B /src/highs-build -G Ninja \
       -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/opt/highs \
       -DFAST_BUILD=ON -DBUILD_SHARED_LIBS=ON -DBUILD_SHARED_EXTRAS_LIB=OFF \
       -DBUILD_CXX_EXE=OFF -DBUILD_EXAMPLES=OFF -DBUILD_TESTING=OFF -DZLIB=OFF -DHIPO=OFF \
    && cmake --build /src/highs-build --parallel "$(nproc)" \
    && cmake --install /src/highs-build

FROM highs-build AS app-build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1 \
    PKG_CONFIG_PATH=/opt/highs/lib/pkgconfig \
    LD_LIBRARY_PATH=/opt/highs/lib
RUN --mount=type=cache,target=/root/.cache/go-build HIGHS_EXPECT_VERSION=1.15.1 go test -race -count=1 ./...
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /out/highs-example ./cmd/example

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends libstdc++6 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=app-build /opt/highs/lib/libhighs.so.1.15.1 /opt/highs/lib/
COPY --from=app-build /opt/highs/share/doc/HIGHS/LICENSE.txt /usr/share/doc/highs/LICENSE.txt
COPY --from=highs-build /src/highs/THIRD_PARTY_NOTICES.md /usr/share/doc/highs/THIRD_PARTY_NOTICES.md
COPY --from=app-build /out/highs-example /usr/local/bin/highs-example
ENV LD_LIBRARY_PATH=/opt/highs/lib
RUN ln -s libhighs.so.1.15.1 /opt/highs/lib/libhighs.so.1 \
    && ldd /usr/local/bin/highs-example \
    && test -z "$(ldd /usr/local/bin/highs-example | grep 'not found')"
USER 65532:65532
CMD ["/usr/local/bin/highs-example"]
