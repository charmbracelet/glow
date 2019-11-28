#!/bin/bash

for style in ./styles/*.json; do
    echo "Generating screenshot for ${style}"
    filename="gold_`basename -s .json ${style}`.png"

    light=""
    if [[ $style == *"light"* ]]; then
        light="-l"
    fi

    ./termshot ${light} -o ./styles/ -f "$filename" ./gold -s ${style}
    pngcrush -ow "./styles/$filename"
done
