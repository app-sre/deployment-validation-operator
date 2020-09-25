#!/usr/bin/env bash

export USER="${QUAY_USER}"
make docker-login-and-push
