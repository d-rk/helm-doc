# helm-doc
a helm plugin for documentation generation

[![Build Status](https://travis-ci.com/random-dwi/helm-doc.svg?branch=master)](https://travis-ci.com/random-dwi/helm-doc)

## installation

choose the variant needed and install with the following:

```bash
version=0.2.0
#variant=Linux_x86_64
#variant=Linux_i386
#variant=Darwin_x86_64
#variant=Darwin_i386
helm plugin install https://github.com/random-dwi/helm-doc/releases/download/${version}/helm-doc_${version}_${variant}.tar.gz

# workaround to fix file permissions
find `helm home` -name helm-doc | xargs -L1 chmod +x
```

## usage

```sh
# show help
helm doc -h

# generate doc for a chart
helm doc [chart]
```