#!/bin/bash

# Get all tags from the repository
tags=$(git tag)

# Sort tags using version sorting (ignores "v" prefix)
echo "$tags" | sort -V
