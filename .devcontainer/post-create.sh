#!/bin/sh

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

