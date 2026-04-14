#!/usr/bin/env bash
# This script regenerates deepcopy objects and manifests. It exists as a
# contributor convenience wrapper around the Makefile generation targets.
set -euo pipefail

make generate
make manifests
make docs-generate
