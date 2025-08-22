#!/bin/bash

# GitHub Issues Creation Script for TODO Migration
# This script creates all TODO-related issues using the GitHub CLI

set -e

echo "🚀 Creating GitHub issues for TODO migration..."
echo ""

# Repository
REPO="rios0rios0/terraform-provider-http"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Function to create an issue from template
create_issue() {
    local issue_file="$1"
    
    if [[ ! -f "$issue_file" ]]; then
        echo "❌ Issue template not found: $issue_file"
        return 1
    fi
    
    echo "📄 Processing: $(basename "$issue_file")"
    
    # Extract title and labels from the template
    title=$(grep "^**Title:**" "$issue_file" | sed 's/\*\*Title:\*\* //')
    labels=$(grep "^**Labels:**" "$issue_file" | sed 's/\*\*Labels:\*\* //')
    
    # Extract body (everything after **Body:** line)
    body=$(sed -n '/^\*\*Body:\*\*$/,$p' "$issue_file" | tail -n +2)
    
    echo "   Title: $title"
    echo "   Labels: $labels"
    
    # Create the issue using GitHub CLI
    if gh issue create \
        --title "$title" \
        --body "$body" \
        --label "$labels" \
        --repo "$REPO"; then
        echo "✅ Successfully created issue: $title"
        echo ""
        return 0
    else
        echo "❌ Failed to create issue: $title"
        echo ""
        return 1
    fi
}

# Check if GitHub CLI is available
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is not installed."
    echo "Please install it from: https://cli.github.com/"
    echo ""
    echo "Alternative: Manual creation commands:"
    echo ""
    
    # Output manual commands as fallback
    for i in {01..09}; do
        for file in "$SCRIPT_DIR"/issue-$i-*.md; do
            if [[ -f "$file" ]]; then
                title=$(grep "^**Title:**" "$file" | sed 's/\*\*Title:\*\* //')
                labels=$(grep "^**Labels:**" "$file" | sed 's/\*\*Labels:\*\* //')
                
                echo "# Issue $i: $title"
                echo "gh issue create \\"
                echo "  --title \"$title\" \\"
                echo "  --label \"$labels\" \\"
                echo "  --body-file \"$file\" \\"
                echo "  --repo \"$REPO\""
                echo ""
                break
            fi
        done
    done
    
    echo "# Or use the GitHub web interface:"
    echo "# 1. Go to https://github.com/$REPO/issues/new"
    echo "# 2. Copy title, labels, and body from each file in docs/github-issues/"
    echo "# 3. Create the issue"
    exit 1
fi

# Check if GitHub CLI is authenticated
if ! gh auth status >/dev/null 2>&1; then
    echo "❌ GitHub CLI is not authenticated."
    echo "Please run: gh auth login"
    echo ""
    echo "After authentication, run this script again."
    exit 1
fi

echo "✅ GitHub CLI is authenticated"
echo ""

# Create issues from templates
created_count=0
failed_count=0

for i in {01..09}; do
    for file in "$SCRIPT_DIR"/issue-$i-*.md; do
        if [[ -f "$file" ]]; then
            if create_issue "$file"; then
                ((created_count++))
            else
                ((failed_count++))
            fi
            break
        fi
    done
done

echo "🎉 Summary:"
echo "   Created: $created_count issues"
echo "   Failed: $failed_count issues"
echo ""

if [[ $failed_count -eq 0 ]]; then
    echo "✅ All issues created successfully!"
    echo "View them at: https://github.com/$REPO/issues"
else
    echo "⚠️  Some issues failed to create. Please check the output above."
    exit 1
fi