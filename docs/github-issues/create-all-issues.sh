#!/bin/bash

# Comprehensive GitHub Issues Creation Script
# Creates all 9 TODO migration issues with multiple methods

set -e

REPO="rios0rios0/terraform-provider-http"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 GitHub Issues Creation for TODO Migration"
echo "============================================"
echo "Repository: $REPO"
echo "Script location: $SCRIPT_DIR"
echo ""

# Function to create issue using GitHub CLI
create_issue_with_gh() {
    local issue_file="$1"
    local issue_num="$2"
    
    if [[ ! -f "$issue_file" ]]; then
        echo "❌ Issue template not found: $issue_file"
        return 1
    fi
    
    echo "📄 [$issue_num/9] Processing: $(basename "$issue_file")"
    
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
        --repo "$REPO" 2>/dev/null; then
        echo "✅ Successfully created issue: $title"
        echo ""
        return 0
    else
        echo "❌ Failed to create issue: $title"
        echo ""
        return 1
    fi
}

# Check if we're in the right directory
if [[ ! -d "docs/github-issues" ]]; then
    echo "❌ Error: This script must be run from the repository root directory"
    echo "   Current directory: $(pwd)"
    echo "   Expected to find: docs/github-issues/"
    echo ""
    echo "Please run:"
    echo "   cd /path/to/terraform-provider-http"
    echo "   ./docs/github-issues/create-issues.sh"
    exit 1
fi

ISSUES_DIR="docs/github-issues"

# Check if GitHub CLI is available
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is not installed."
    echo ""
    echo "📥 INSTALL GITHUB CLI:"
    echo "   macOS: brew install gh"
    echo "   Ubuntu/Debian: sudo apt install gh"
    echo "   Windows: choco install gh"
    echo "   Or download from: https://cli.github.com/"
    echo ""
    echo "📋 MANUAL CREATION COMMANDS:"
    echo ""
    
    # Output manual commands as fallback
    for i in {01..09}; do
        for file in "$ISSUES_DIR"/issue-$i-*.md; do
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
    
    echo "🌐 WEB INTERFACE METHOD:"
    echo "   1. Go to: https://github.com/$REPO/issues/new"
    echo "   2. Copy title, labels, and body from each file in $ISSUES_DIR/"
    echo "   3. Create each issue manually"
    exit 1
fi

# Check if GitHub CLI is authenticated
if ! gh auth status >/dev/null 2>&1; then
    echo "❌ GitHub CLI is not authenticated."
    echo ""
    echo "🔐 AUTHENTICATION REQUIRED:"
    echo "   gh auth login"
    echo ""
    echo "Then run this script again:"
    echo "   $0"
    exit 1
fi

echo "✅ GitHub CLI is authenticated"
echo "✅ Found issues directory: $ISSUES_DIR"
echo ""

# List available issues
echo "📋 Available issue templates:"
issue_files=()
for i in {01..09}; do
    for file in "$ISSUES_DIR"/issue-$i-*.md; do
        if [[ -f "$file" ]]; then
            echo "   $i. $(basename "$file")"
            issue_files+=("$file")
            break
        fi
    done
done
echo ""

if [[ ${#issue_files[@]} -eq 0 ]]; then
    echo "❌ No issue templates found in $ISSUES_DIR"
    exit 1
fi

echo "🚀 Creating ${#issue_files[@]} GitHub issues..."
echo ""

# Create issues from templates
created_count=0
failed_count=0

for i in "${!issue_files[@]}"; do
    issue_num=$((i + 1))
    if create_issue_with_gh "${issue_files[$i]}" "$issue_num"; then
        ((created_count++))
    else
        ((failed_count++))
    fi
    
    # Small delay to avoid rate limiting
    sleep 1
done

echo "🎉 CREATION SUMMARY:"
echo "   ✅ Created: $created_count issues"
echo "   ❌ Failed: $failed_count issues"
echo "   📊 Total: ${#issue_files[@]} issues"
echo ""

if [[ $failed_count -eq 0 ]]; then
    echo "🎊 SUCCESS! All issues created successfully!"
    echo "🔗 View them at: https://github.com/$REPO/issues"
    echo ""
    echo "🧹 CLEANUP: You can now remove the TODO migration files:"
    echo "   rm -rf docs/github-issues/"
    echo "   git add -A && git commit -m \"Remove TODO migration templates after issue creation\""
else
    echo "⚠️  Some issues failed to create."
    echo ""
    echo "💡 TROUBLESHOOTING:"
    echo "   - Check your internet connection"
    echo "   - Verify repository permissions: https://github.com/$REPO/settings"
    echo "   - Try running individual commands manually"
    echo "   - Check GitHub CLI authentication: gh auth status"
    exit 1
fi

echo "✨ TODO migration completed successfully!"