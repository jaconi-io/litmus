#!/usr/bin/env sh

DIR="charts/gcp/icons"
inkscape -w "512" -h "512" "${DIR}/template.svg" -o "${DIR}/output.png"
