#!/bin/sh

set -eu

pg_dumpall --host=127.0.0.1 -U harrybrwn > backup.sql

