#!/usr/bin/env bash

set -e

workdir="$(realpath "$(dirname "${0}")")"
cd "${workdir}"

version="$(< debian-build/control.tpl grep 'Version: ' | sed 's/Version: //g')"

build-binary() {
  os="${1:?}"
  arch="${2:?}"
  rm    -rf "${workdir:?}/bin/${os}/${arch}/"
  mkdir -p  "${workdir:?}/bin/${os}/${arch}/"
  GOOS=${os} GOARCH=${arch} go build -o "${workdir}/bin/${os}/${arch}/zsh-history-tidy" -trimpath -ldflags '-w -s' .
  echo "OK: bin/${os}/${arch}/zsh-history-tidy"
}

build-deb() {
  arch="${1:?}"
  rm    -f  "${workdir}/debian-build/zsh-history-tidy/DEBIAN/control"
  rm    -rf "${workdir}/debian-build/zsh-history-tidy/usr/bin/"
  mkdir -p  "${workdir}/debian-build/zsh-history-tidy/usr/bin/"
  ARCH=${arch} envsubst < "${workdir}/debian-build/control.tpl" > "${workdir}/debian-build/zsh-history-tidy/DEBIAN/control"
  rsync -a "${workdir}/bin/linux/${arch}/" "${workdir}/debian-build/zsh-history-tidy/usr/bin/"
  dpkg -b "${workdir}/debian-build/zsh-history-tidy/" "${workdir}/output/zsh-history-tidy_${version}_${arch}.deb"
  echo "OK: output/zsh-history-tidy_${version}_${arch}.deb"
}

#### MAIN ####
rm    -rf "${workdir}/output/"
mkdir -p  "${workdir}/output/"
build-binary linux amd64
build-binary linux arm64
build-deb amd64
build-deb arm64
