#!/usr/bin/env bash

if command -v tofu &> /dev/null
then
    tofu fmt -recursive ./examples/
else
    terraform fmt -recursive ./examples/
fi
