#!/bin/sh -x -e
gcc -Iquiche/include -Lquiche/target/release -lquiche quiche_test.c
