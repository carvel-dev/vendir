#!/bin/bash

# This file verifies that configuration examples work

set -x -u

read -d '' ytt_test << EOF
ytt_lines = File.readlines("./README.md")
ytt_lines = ytt_lines.select { |line| line.start_with?("ytt") and line.include?("-f") }
ytt_lines = ytt_lines.map { |line| line.split("|")[0] }
raise "Expected at least one ytt command" if ytt_lines.empty?
ytt_lines.each { |line| system(line + ">/dev/null && 2>&1") or raise "Failed on '#{line}'" }
puts "YTT CHECKS SUCCESS"
EOF

set -e

ruby -e "$ytt_test"
