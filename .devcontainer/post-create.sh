#!/bin/sh

set -e

#
# copy signadot config from mount
#
if [ -d $HOME/.signadot ]; then
	true;
else
	mkdir $HOME/.signadot
fi

if [ -f $HOME/.signadot/config.yaml  ]; then
	true;
else
	cp $HOME/.signadot-localhost/config.yaml $HOME/.signadot/config.yaml
fi

#
# rewrite kubeconfig
#
if [ -d $HOME/.kube ]; then
	true;
else
	go run /workspaces/cli/.devcontainer/rewrite_kubeconfig
fi
