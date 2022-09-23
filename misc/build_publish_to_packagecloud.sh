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
__soft_errors=0
__soft_errors_str=""

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

    set +e
    echo "Building .deb ${PACKAGE_FILE_DEB}..."
    if ! _build_package "deb" "${PACKAGE_NAME}" "${PACKAGE_VERSION}" "${INPUT_DIR}" "${ARCH}" "dist/${PACKAGE_FILE_DEB}" ; then
      _=$(( __soft_errors++ ))
      __soft_errors_str="${__soft_errors_str}\nFailed to build ${PACKAGE_FILE_DEB}."
    fi
    echo "Building .rpm ${PACKAGE_FILE_RPM}..."
    if ! _build_package "rpm" "${PACKAGE_NAME}" "${PACKAGE_VERSION}" "${INPUT_DIR}" "${ARCH}" "dist/${PACKAGE_FILE_RPM}" ; then
      _=$(( __soft_errors++ ))
      __soft_errors_str="${__soft_errors_str}\nFailed to build ${PACKAGE_FILE_DEB}."
    fi
    set -e
  done
}

_publish_all_packages(){
  DEB_DISTROS="
any/any
debian/jessie
debian/stretch
debian/buster
debian/bullseye

linuxmint/sarah
linuxmint/serena
linuxmint/sonya
linuxmint/sylvia
linuxmint/tara
linuxmint/tessa
linuxmint/tina
linuxmint/tricia
linuxmint/ulyana

ubuntu/bionic
ubuntu/focal
ubuntu/jammy
ubuntu/trusty
ubuntu/xenial
ubuntu/zesty
"
  while read -r PACKAGE_FILE; do
    for DEB_DISTRO in $DEB_DISTROS; do
      [ "${DEB_DISTRO}" = "" ] && continue

      # XXX: Getting ready for supporting any/any publishing. Publish non-amd64 for any/any only
      [[ ! "${PACKAGE_FILE}" =~ "amd64" ]] && [ "${DEB_DISTRO}" != "any/any" ] && continue

      echo "Pushing ${PACKAGE_FILE} to packagecloud (${DEB_DISTRO})..."
      set +e
      if ! package_cloud push "${PACKAGECLOUD_REPO}/${DEB_DISTRO}" "${PACKAGE_FILE}" ; then
        _=$(( __soft_errors++ ))
        __soft_errors_str="${__soft_errors_str}\nFailed to publish ${PACKAGE_FILE} (${DEB_DISTRO})."
      fi
      set -e
    done
  done < <(find dist -type f -name "*.deb")

  RPM_DISTROS="
rpm_any/rpm_any
el/6
el/7

fedora/31
fedora/32
fedora/33
"
  while read -r PACKAGE_FILE; do
    for RPM_DISTRO in $RPM_DISTROS; do
      [ "${RPM_DISTRO}" = "" ] && continue
      echo "Pushing ${PACKAGE_FILE} to packagecloud..."
      set +e
      if ! package_cloud push "${PACKAGECLOUD_REPO}/${RPM_DISTRO}" "${PACKAGE_FILE}" ; then
        _=$(( __soft_errors++ ))
        __soft_errors_str="${__soft_errors_str}\nFailed to publish ${PACKAGE_FILE} (${RPM_DISTRO})."
      fi
      set -e
    done
  done < <(find dist -type f -name "*.rpm")
}

## Main

_install_dependencies
_build_all_packages
_publish_all_packages

if [ "${__soft_errors}" != "0" ] ; then
  echo "We got ${__soft_errors} (soft) errors."
  echo -e "${__soft_errors_str}"
  exit 1
fi
