#!/bin/bash
set -e

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin_nt*) os="windows" ;;
    mingw*) os="windows" ;;
    msys_nt*) os="windows" ;;
  esac
  echo "$os"
}
OS="$(uname_os)"


if [ "${OS}" = "windows" ]; then
  echo "For installing tsuru on windows, download the binary from:"
  echo "  https://github.com/tsuru/tsuru-client/releases/latest"
  exit 2
fi
if command -v "brew" >/dev/null ; then
  echo "installing tsuru with brew..."
  brew tap tsuru/homebrew-tsuru
  brew install tsuru
  exit $?
fi

package_type="null"
if command -v "apt" >/dev/null ; then
  package_type="deb"
  package_installer="apt"
elif command -v "apt-get" >/dev/null ; then
  package_type="deb"
  package_installer="apt-get"
elif command -v "yum" >/dev/null ; then
  package_type="rpm"
  package_installer="yum"
elif command -v "zypper" >/dev/null ; then
  package_type="rpm"
  package_installer="zypper"
fi

if [ "${package_type}" != "null" ]; then
  if command -v "curl" >/dev/null ; then
    echo "installing tsuru with packagecloud.io deb script (will ask for sudo password)"
    curl -s "https://packagecloud.io/install/repositories/tsuru/stable/script.${package_type}.sh" | sudo bash
  elif command -v "wget" >/dev/null ; then
    echo "installing tsuru with packagecloud.io rpm script (will ask for sudo password)"
    wget -q -O - "https://packagecloud.io/install/repositories/tsuru/stable/script.${package_type}.sh" | sudo bash
  else
    echo "curl or wget is required to install tsuru"
    exit 1
  fi

  # update repo config for using any/any instead of os/dist
  sedpattern='s@^(deb(-src)?.* https://packagecloud\.io/tsuru/stable/)\w+/?\s+\w+(.*)@\1any/ any\3@g'
  cfile="/etc/apt/sources.list.d/tsuru_stable.list"
  [ -f "${cfile}" ] && { sed -E "${sedpattern}" "${cfile}" | sudo tee "${cfile}" ; } &>/dev/null
  sedpattern='s@^(baseurl=https://packagecloud\.io/tsuru/stable)/\w+/\w+/(.*)@\1/any/any/\2@g'
  cfile="/etc/zypp/repos.d/tsuru_stable.repo"
  [ -f "${cfile}" ] && { sed -E "${sedpattern}" "${cfile}" | sudo tee "${cfile}" ; } &>/dev/null
  cfile="/etc/yum.repos.d/tsuru_stable.repo"
  [ -f "${cfile}" ] && { sed -E "${sedpattern}" "${cfile}" | sudo tee "${cfile}" ; } &>/dev/null

  # update cache
  if command -v "apt" >/dev/null ; then
    apt update &> /dev/null
  elif command -v "apt-get" >/dev/null ; then
    apt-get update &> /dev/null
  elif command -v "yum" >/dev/null ; then
    yum -q makecache -y --disablerepo='*' --enablerepo='tsuru_stable'
    yum -q makecache -y --disablerepo='*' --enablerepo='tsuru_stable-source'
  elif command -v "zypper" >/dev/null ; then
    zypper --gpg-auto-import-keys refresh tsuru_stable
    zypper --gpg-auto-import-keys refresh tsuru_stable-source
  fi

  sudo "${package_installer}" install tsuru-client
else
  echo "Could not install tsuru on your OS, please download the binary from:"
  echo "  https://github.com/tsuru/tsuru-client/releases/latest"
  exit 2
fi
