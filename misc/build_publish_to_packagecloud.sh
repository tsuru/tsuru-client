#!/bin/bash
set -eo pipefail

PACKAGE_NAME="tsuru-client"
PACKAGE_VERSION="${GITHUB_REF#"refs/tags/"}"
[ "${GITHUB_REF}" == "" ] && echo "No GITHUB_REF found, exiting" && exit 1
[ "${PACKAGE_VERSION}" == "" ] && echo "PACKAGE_VERSION is empty, exiting" && exit 1
[ "${PACKAGECLOUD_TOKEN}" == "" ] && echo "No packagecloud token found, exiting" && exit 1

PACKAGECLOUD_REPO="tsuru/rc"
if [[ ${PACKAGE_VERSION} =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  PACKAGECLOUD_REPO="tsuru/stable"
fi


_install_dependencies() {
  if ! command -v rpm &>/dev/null ; then
    sudo apt-get update
    sudo apt-get install -y rpm
  fi
  if ! gem list -i fpm -v ">= 1.14.2" &>/dev/null ; then
    gem install fpm -v '>= 1.14.2'
  fi
  if ! gem list -i package_cloud -v ">= 0.3.10" &>/dev/null ; then
    gem install package_cloud -v '>= 0.3.10'
  fi
}

_build_package() {
  type="$1" ; shift
  package_name="$1" ; shift
  version="$1" ; shift
  input_dir="$1" ; shift
  arch=$1 ; shift
  output_file="$1" ; shift

  description="tsuru is the command line interface for the tsuru server

Tsuru is an open source platform as a service software. This package installs
the client used by application developers to communicate with tsuru server."

  [ "${arch}" = "386" ] && arch="i386"

  fpm \
    -s dir -t "${type}" \
    -p "${output_file}" \
    --chdir "${input_dir}" \
    --name "${package_name}" \
    --version "${version}" \
    --architecture "${arch}" \
    --maintainer "tsuru@g.globo" \
    --vendor "Tsuru team <tsuru@g.globo>" \
    --url "https://tsuru.io" \
    --description "${description}" \
    --no-depends \
    --license bsd3 \
    --log INFO \
    tsuru=/usr/bin/tsuru
}

_build_all_packages(){
  INPUT_DIRS=$(find dist -type d -name "tsuru_linux_*")
  for INPUT_DIR in $INPUT_DIRS; do
    ARCH=$(echo "${INPUT_DIR}" | sed -e 's/.*tsuru_linux_//' -e 's/_v1//' )
    PACKAGE_FILE_DEB="${PACKAGE_NAME}_${PACKAGE_VERSION}_${ARCH}.deb"
    PACKAGE_FILE_RPM="${PACKAGE_NAME}_${PACKAGE_VERSION}_${ARCH}.rpm"

    echo "Building .deb ${PACKAGE_FILE_DEB}..."
    _build_package "deb" "${PACKAGE_NAME}" "${PACKAGE_VERSION}" "${INPUT_DIR}" "${ARCH}" "dist/${PACKAGE_FILE_DEB}"
    echo "Building .rpm ${PACKAGE_FILE_RPM}..."
    _build_package "rpm" "${PACKAGE_NAME}" "${PACKAGE_VERSION}" "${INPUT_DIR}" "${ARCH}" "dist/${PACKAGE_FILE_RPM}"
  done
}

_publish_all_packages(){
  # package cloud accepts numbers on some distros, althouth not documented
  # DEB_DISTROS="any/any" # this breaks old repos :(
  DEB_DISTROS="
debian/stretch
debian/buster
debian/bullseye

ubuntu/14.04
ubuntu/16.04
ubuntu/18.04
ubuntu/20.04
ubuntu/20.10
ubuntu/21.04
ubuntu/21.10
ubuntu/22.04

linuxmint/5
linuxmint/19
linuxmint/19.1
linuxmint/19.2
linuxmint/19.3
linuxmint/20
linuxmint/20.1
linuxmint/20.2
linuxmint/20.3
"
  while read -r PACKAGE_FILE; do
    for DEB_DISTRO in $DEB_DISTROS; do
      [ "${DEB_DISTRO}" = "" ] && continue
      echo "Pushing ${PACKAGE_FILE} to packagecloud (${DEB_DISTRO})..."
      package_cloud push "${PACKAGECLOUD_REPO}/${DEB_DISTRO}" "${PACKAGE_FILE}"
    done
  done < <(find dist -type f -name "*.deb")

  # RPM_DISTROS="rpm_any/rpm_any" # this breaks old repos :(
  RPM_DISTROS="
el/6
el/7
el/8
el/9

fedora/31
fedora/32
fedora/33
fedora/34
fedora/35
fedora/36
"
  while read -r PACKAGE_FILE; do
    for RPM_DISTRO in $RPM_DISTROS; do
      [ "${RPM_DISTRO}" = "" ] && continue
      echo "Pushing ${PACKAGE_FILE} to packagecloud..."
      package_cloud push "${PACKAGECLOUD_REPO}/${RPM_DISTRO}" "${PACKAGE_FILE}"
    done
  done < <(find dist -type f -name "*.rpm")
}

## Main

_install_dependencies
_build_all_packages
_publish_all_packages
