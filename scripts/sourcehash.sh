#!/bin/sh

set -e

DEBUG=false
LIST=false
EXCLUDE=''
INCLUDE=''
LANG='go'

usage() {
  echo "Usage"
  echo "  sourcehash.sh [[-l|--list]|-e <pattern>]"
  echo
  echo "Flags"
  echo "  -l, --list               list all the files that would be hashed"
  echo "  -e, --exclude <pattern>  exclude file names as a pattern"
  echo "  -i, --include <pattern>  include spesific file patterns"
  echo "      --lang <language>    find files for a spesific language"
  echo "  -h, --help               print this help message"
  echo
}

while [ $# -gt 0 ]; do
  case $1 in
    -h|--help)
      usage
      exit
      ;;
    -l|--list)
      LIST=true
      shift
      ;;

    -e|--exclude|-e=*|--exclude=*)
      arg="${1#*=}"
      if [ "$arg" = "-e" ] || [ "$arg" = "--exclude" ]; then
        arg="$2"
        shift 2
      else
        shift 1
      fi
      if [ -z "$EXCLUDE" ];then
        EXCLUDE="$arg"
      else
        EXCLUDE="$EXCLUDE $arg"
      fi
      ;;

    -i|--include|-i=*|--include=*)
      arg="${1#*=}"
      if [ "$arg" = "-i" ] || [ "$arg" = "--include" ]; then
        arg="$2"
        shift 2
      else
        shift
      fi
      if [ -z "$INCLUDE" ]; then
        INCLUDE="$arg"
      else
        INCLUDE="$INCLUDE $arg"
      fi
      ;;

    --lang|--lang=*)
      LANG="${1#*=}"
      if [ -z "$LANG" ] || [ "$LANG" = "--lang" ]; then
        LANG="$2"
        shift 2
      else
        shift 1
      fi
      ;;
    -d)
      DEBUG=true
      shift
      ;;
    *)
      echo "Unknown flag \"$1\""
      exit 1
      ;;
  esac
done

list() {
  e=""
  if [ -n "$EXCLUDE" ]; then
    for f in $EXCLUDE; do
      e="-not -path $f $e"
    done
  fi
  i=""
  if [ -n "$INCLUDE" ]; then
    for f in $INCLUDE; do
      i="-o -name $f $i"
    done
  fi

  case "$LANG" in
    go)
      # shellcheck disable=SC2086
      find .                   \
        -type f                \
        \(                     \
          -name '*.go'         \
          -o -name 'go.mod'    \
          -o -name 'go.sum' $i \
        \)                     \
        -not -path './test/*'  \
        -not -path './vendor/*' $e
      ;;
    ts|typescript)
      # shellcheck disable=SC2086
      find .                              \
        -type f                           \
        \(                                \
          -name '*.ts'                    \
          -o -name 'yarn.lock'            \
          -o -name 'package-lock.json' $i \
        \)                                \
        -not -path './node_modules/*' $e
      ;;
    py|python)
      # shellcheck disable=SC2086
      find .                             \
        -type f                          \
        \(                               \
          -name '*.py'                   \
          -o -name 'poetry.lock'         \
          -o -name 'requirements.txt' $i \
        \)                               \
        -not -path './node_modules/*' $e
      ;;
    css)
      # shellcheck disable=SC2086
      find .               \
        -type f            \
        \(                 \
          -name '*.css' $i \
        \)                 \
        -not -path './node_modules' $e
      ;;
    rust)
      # shellcheck disable=SC2086
      find .              \
        -type f           \
        \(                \
          -name '*.rs' $i \
        \) $e
      ;;
    *)
      echo "unknown language \"$LANG\", see (--lang)"
      exit 1
      ;;
  esac
}

if $DEBUG; then
  usage
  echo
  list
  exit
fi

if $LIST; then
  list | sort -d -s
  exit
fi


list         |
  sort -d -s |
  xargs cat  |
  sha256sum  |
  sed -Ee 's/\s|-//g'
