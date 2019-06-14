#!/bin/sh
path=`find $(pwd)/ -type d`
ln -s $path ~/.helm/plugins/helm-doc
