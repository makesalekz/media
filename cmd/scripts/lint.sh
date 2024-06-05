#!/bin/bash

UPDATED_FILES=$(git diff --name-only origin/dev . | grep "internal/.*\.go$")
UPDATED_DIRS=$(git diff --dirstat=files,0 origin/dev . | grep internal | sed -En "s/^[ 0-9.]+\% //p")

for dir in $UPDATED_DIRS
do
  FILES_TO_LINT=$(grep "$dir" <<< "$UPDATED_FILES"  | tr '\n' ' ')
  FILES_IN_DIR=$(ls $dir)
  FILES_TO_EXCLUDE=""
  for file in $FILES_IN_DIR
  do
    if ! [[ $FILES_TO_LINT =~ $file ]]; then
      FILES_TO_EXCLUDE="$FILES_TO_EXCLUDE|((^|/)$file.*)"
    fi
  done
  if [[ ${#FILES_TO_EXCLUDE} -ge 1 ]]; then
    FILES_TO_EXCLUDE="${FILES_TO_EXCLUDE:1}"
    golangci-lint run --fix -c .golangci-lint.yml --exclude-files "$FILES_TO_EXCLUDE" "$dir"
  else
    golangci-lint run --fix -c .golangci-lint.yml "$FILES_TO_LINT"
  fi
done