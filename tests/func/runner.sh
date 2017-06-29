#!/bin/bash -eux

exit_code=0
for testcase in ./tests/func/test_*.sh ; do
    ./tests/func/teardown.sh
    if $testcase ; then
        : $testcase OK >&2
    else
        exit_code=$((exit_code + 1))
        : $testcase FAIL >&2
    fi
done

: $exit_code tests failed.

exit $exit_code
