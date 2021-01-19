#!/bin/sh

# To generate result file run
#   . example/env.vars && gotiller -d example -o example
# in this dir first

DIR=`dirname $0`

$DIR/multimarkdown -t mmd $DIR/README-main.mmd > $DIR/../README.md
