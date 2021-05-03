#!/usr/bin/env bash

SCRIPTS_DIR="$( cd "$( dirname "$0" )" >/dev/null 2>&1 && pwd )"
"$SCRIPTS_DIR/../sync_readme.sh"

git_status="$(git status -s)"
if [ -n "$git_status" ]; then
    echo "$git_status"
    echo "scripts/sync_readme.sh modified the git status. If updating the README.md, update the root README.md and run that script."
    exit 1
fi
