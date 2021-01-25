#!/bin/bash
set -e

golangci-lint cache clean
golangci-lint run