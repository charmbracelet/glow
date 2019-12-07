#!/bin/bash

set -e

for element in ./styles/examples/*.md; do
    echo "Generating screenshot for element ${element}"
    basename="`basename -s .md ${element}`"
    stylename="${basename}.style"
    filename="${basename}.png"

    ./termshot -o ./styles/examples/ -f "$filename" ./cmd/gold/gold -s ./styles/examples/${stylename} ${element}
    pngcrush -ow "./styles/examples/$filename"
done
