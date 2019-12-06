#!/bin/bash

for style in ./styles/*.json; do
    echo "Generating screenshot for ${style}"
    filename="`basename -s .json ${style}`.png"

    light=""
    if [[ $style == *"light"* ]]; then
        light="-l"
    fi

    ./termshot ${light} -o ./styles/ -f "$filename" ./cmd/gold/gold -s ${style} ./cmd/gold
    pngcrush -ow "./styles/$filename"
done
