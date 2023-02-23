#!/usr/bin/env sh

go generate
if [ -z "$(git status --untracked-files=no --porcelain)" ]; then
    echo "documentation is up to date"
else
    echo "documentation has not been updated; offending files:"
    git status --untracked-files=no --porcelain
    exit 1
fi
