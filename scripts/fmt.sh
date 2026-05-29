#!/usr/bin/env sh

set -euo pipefail

if command -v tofu &> /dev/null
then
    tofu fmt -recursive ./examples/
else
    terraform fmt -recursive ./examples/
fi
