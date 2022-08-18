#!/bin/bash

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )"
cd ${SCRIPT_DIR}/../cmd/sonobuoy/app/e2e/v2/testLists

prevVer=""
for minor in $(seq 15 99)
do
  IFS=$'\n' read -r -d '' -a versions < <(ls | grep "1.${minor}\.[0-9]*$" | sort -t "." -k1,1n -k2,2n -k3,3n)

  if [[ ${#versions[@]} -eq 0 ]]
  then
    continue
  fi
  echo "Minor version ${minor} has "${#versions[@]}" point releases to consider..."

  for ver in "${versions[@]}"
  do
    if [ "${prevVer}" == "" ]
    then
      echo "Using ${ver} as base version"
      #minVer="${ver}.txt"
      mv "${ver}" "${ver}.txt"
      #first="false"
      prevVer="${ver}"
      continue
    fi
    echo "Processing version ${ver}"

    # Header to reference the minVer list of tests.
    #echo "#${minVer}" > "${ver}.txt"
    echo "#${prevVer}" > "${ver}.txt"

    IFS=$'\n' read -r -d '' -a addTests < <(comm -13 "$prevVer" "$ver")
    echo "  Adding ${#addTests[@]} tests to base version"
    for t in "${addTests[@]}"
    do
      echo "+${t}" >> "${ver}.txt"
    done

    IFS=$'\n' read -r -d '' -a removeTests < <(comm -23 "$prevVer" "$ver")
    echo "  Removing ${#removeTests[@]} tests from base version"
    for t in "${removeTests[@]}"
    do
      echo "-${t}" >> "${ver}.txt"
    done

    prevVer="${ver}"
  done

  echo done with minor version ${minor}
  echo

done