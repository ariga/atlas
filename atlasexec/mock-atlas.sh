#!/bin/bash

# TEST_BATCH provide the directory contains all
# outputs for multiple runs. The path should be absolute
# or related to current working directory.
if [[ "$TEST_BATCH" != "" ]]; then
  COUNTER_FILE=$TEST_BATCH/counter
  COUNTER=$(cat $COUNTER_FILE 2>/dev/null)
  COUNTER=$((COUNTER+1))
  DIR_CUR="$TEST_BATCH/$COUNTER"
  if [ ! -d "$DIR_CUR" ]; then
    >&2 echo -n "$DIR_CUR does not exist, quitting..."
    exit 1
  fi
  # Save counter for the next runs
  echo -n $COUNTER > $COUNTER_FILE
  if [ -f "$DIR_CUR/args" ]; then
    TEST_ARGS=$(cat $DIR_CUR/args)
  fi
  if [ -f "$DIR_CUR/stderr" ]; then
    TEST_STDERR=$(cat $DIR_CUR/stderr)
  fi
  if [ -f "$DIR_CUR/stdout" ]; then
    TEST_STDOUT=$(cat $DIR_CUR/stdout)
  fi
fi

if [[ "$TEST_ARGS" != "$@" ]]; then
  >&2 echo "Receive unexpected args: $@"
  exit 1
fi

if [[ "$TEST_STDOUT" != "" ]]; then
  printf "%s" "$TEST_STDOUT"
  if [[ "$TEST_STDERR" == "" ]]; then
    # `migrate down` and `migrate lint` commands print result to stdout
    # but the error code is set to 1.
    exit ${TEST_EXIT_CODE:-0} # No stderr
  fi
  # In some cases, Atlas will write the error in stderr
  # when if the command is partially successful.
  # eg. Run the apply commands with multiple environments.
  >&2 echo -n $TEST_STDERR
  exit 1
fi

TEST_STDERR="${TEST_STDERR:-Missing stderr either stdout input for the test}"
>&2 echo -n $TEST_STDERR
exit 1
