#!/bin/bash -eux

exit_code=0
for testcase in ./tests/func/test_*.sh ; do
    ./tests/func/teardown.sh

    if ! $testcase ; then
        exit_code=$((exit_code|$?))
    fi
done

exit $exit_code
