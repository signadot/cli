#!/bin/sh

GOARCH=$1

wget https://go.dev/dl/go1.21.5.linux-${GOARCH}.tar.gz
rm -rf /usr/local/go 
tar -C /usr/local -xzf go1.21.5.linux-${GOARCH}.tar.gz
