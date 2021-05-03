#!/bin/bash

go mod tidy

git_status="$(git status -s)"
if [ -n "$git_status" ]; then
    echo "$git_status"
    echo "go mod tidy modified the git status; did you need add/remove a dependency?"
    exit 1
fi
