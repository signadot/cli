#!/bin/sh

set -e
arch="$(uname -m)"
case $arch in 
	x86_64) arch="amd64";;
	aarch64 | armv8*) arch="arm64";;
	aarch32 | armv7* | armvhf*) arch="arm";;
	i?86) arch="386";;
	*) echo "(!) Architecture $arch unsupported"; exit 1 ;;
esac

version="$(curl -sSL https://dl.k8s.io/release/stable.txt)"
curl -LO https://dl.k8s.io/release/${version}/bin/linux/${arch}/kubectl
install -o root -g root kubectl /usr/local/bin/kubectl
