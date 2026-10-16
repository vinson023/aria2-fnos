#!/bin/sh
# 在 alpine:3.22 容器内构建完全静态链接的 aria2c（musl）。
# 目的：消除 Core 的全部宿主动态依赖（glibc 版本、libz、libexpat、libsqlite3、
# libssl3、libcrypto3、libssh2、libstdc++、libgcc_s），根除"换机后无法启动"的第二类根因。
#
# aria2 的静态开关是环境变量 ARIA2_STATIC=yes（configure.ac:95/187/1019），
# 它同时做两件事：让 pkg-config 走 --static 以带出 Libs.private，以及给最终链接加 -all-static。
# 只给 configure 传 --enable-static-link 或只设 LDFLAGS=-static 都**不会**产出静态二进制。
set -eu

apk add --no-cache build-base autoconf automake libtool pkgconf gettext-tiny \
  linux-headers ca-certificates git \
  zlib-static zlib-dev expat-static expat-dev sqlite-static sqlite-dev \
  libssh2-static libssh2-dev openssl-libs-static openssl-dev patch >/dev/null 2>&1

echo "=== toolchain ==="
gcc --version | head -1
g++ --version | head -1
echo "=== static archives ==="
for a in libz.a libexpat.a libsqlite3.a libssh2.a libssl.a libcrypto.a libstdc++.a libc.a; do
  p=$(find /usr/lib -name "$a" -print -quit 2>/dev/null || true)
  echo "$a -> ${p:-MISSING}"
done

cd /inputs
sha256sum -c SHA256SUMS

rm -rf /src
mkdir -p /src
tar -xzf /inputs/aria2-release-1.36.0.tar.gz -C /src --strip-components=1
cd /src
for p in /inputs/patch/*.patch; do
  echo "=== patch: $(basename "$p")"
  patch --batch --forward --fuzz=3 -p1 < "$p"
done

export ARIA2_STATIC=yes
export CFLAGS="-O2 -pipe"
export CXXFLAGS="-O2 -pipe"

echo "=== autoreconf ==="
autoreconf -fi
echo "=== configure ==="
# 这里不用 tee：管道后的 $? 属于管道末端命令，busybox ash 又没有 PIPESTATUS，
# 用管道会让 configure 的失败被吞掉。改为重定向后原样打印，set -e 能拿到真实退出码。
./configure \
  --prefix=/opt/aria2 \
  --enable-static --disable-shared \
  --with-openssl --without-gnutls \
  --disable-nls \
  --disable-dependency-tracking \
  --without-libuv --without-libgmp > /tmp/configure.log 2>&1
tail -n 45 /tmp/configure.log

echo "=== 静态开关门禁 ==="
# configure 摘要的 "Static build: yes" 与 Makefile 里的 -all-static 缺一不可；
# 任一缺失即中止，绝不静默产出动态二进制。
if ! grep -q "Static build: *yes" /tmp/configure.log; then
  echo "FATAL: configure 未识别 ARIA2_STATIC=yes"
  grep -n "Static build" /tmp/configure.log || echo "(摘要中无 Static build 行)"
  exit 1
fi
if ! grep -q -- "-all-static" src/Makefile; then
  echo "FATAL: src/Makefile 未包含 -all-static"
  exit 1
fi
echo "OK: ARIA2_STATIC=yes 已生效，最终链接使用 -all-static"

echo "=== make ==="
make -j"$(nproc)"
make install

mkdir -p /out
cp /opt/aria2/bin/aria2c /out/aria2c
strip /out/aria2c

echo "=== 链接形态门禁 ==="
file /out/aria2c > /out/aria2c.file
cat /out/aria2c.file
ldd /out/aria2c > /out/aria2c.ldd 2>&1 || true
cat /out/aria2c.ldd
readelf -d /out/aria2c > /out/aria2c.readelf 2>&1 || true

if ! grep -q "statically linked" /out/aria2c.file; then
  echo "FATAL: 产物不是静态链接，交付无意义"
  exit 1
fi
if grep -q "(NEEDED)" /out/aria2c.readelf; then
  echo "FATAL: 产物仍存在动态依赖 NEEDED 条目"
  grep "(NEEDED)" /out/aria2c.readelf
  exit 1
fi
echo "OK: 静态链接且无 NEEDED 动态依赖"

/out/aria2c --version > /out/aria2c.version
cat /out/aria2c.version
sha256sum /out/aria2c | tee /out/aria2c.sha256
ls -l /out/aria2c