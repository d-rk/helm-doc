#!/bin/sh
helm plugin remove doc
rm `helm home`/plugins/helm-doc
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ln -s $DIR `helm home`/plugins/helm-doc
echo "created symlink to `helm home`/plugins/helm-doc"