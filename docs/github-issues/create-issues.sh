#!/bin/bash

# Script to help create GitHub issues for TODO migration
# This script will output the gh CLI commands needed to create all issues

echo "# GitHub Issues Creation Commands"
echo "# Run these commands to create all TODO-related issues:"
echo ""

# Read each issue file and create gh CLI command
for i in {01..09}; do
    file="/home/runner/work/terraform-provider-http/terraform-provider-http/docs/github-issues/issue-$i-*.md"
    if [ -f $file ]; then
        title=$(grep "^**Title:**" "$file" | sed 's/\*\*Title:\*\* //')
        labels=$(grep "^**Labels:**" "$file" | sed 's/\*\*Labels:\*\* //')
        
        echo "# Issue $i: $title"
        echo "gh issue create \\"
        echo "  --title \"$title\" \\"
        echo "  --label \"$labels\" \\"
        echo "  --body-file \"$file\""
        echo ""
    fi
done

echo "# Alternative: Use the GitHub web interface"
echo "# 1. Go to https://github.com/rios0rios0/terraform-provider-http/issues/new"
echo "# 2. Copy title, labels, and body from each file in docs/github-issues/"
echo "# 3. Create the issue"