#!/usr/bin/env python3
"""
GitHub Issues Creator - Direct API approach
Creates all 9 TODO migration issues using GitHub REST API
"""

import os
import re
import json
import requests
import sys
from pathlib import Path

def parse_issue_template(file_path):
    """Parse an issue template file and extract title, labels, and body"""
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Extract title
    title_match = re.search(r'\*\*Title:\*\* (.+)', content)
    title = title_match.group(1) if title_match else "Unknown Title"
    
    # Extract labels
    labels_match = re.search(r'\*\*Labels:\*\* (.+)', content)
    labels_str = labels_match.group(1) if labels_match else ""
    labels = [label.strip() for label in labels_str.split(',') if label.strip()]
    
    # Extract body (everything after **Body:** line)
    body_match = re.search(r'\*\*Body:\*\*\n(.*)', content, re.DOTALL)
    body = body_match.group(1).strip() if body_match else content
    
    return {
        'title': title,
        'body': body,
        'labels': labels
    }

def create_github_issue(repo, issue_data, token=None):
    """Create a GitHub issue using requests"""
    api_url = f"https://api.github.com/repos/{repo}/issues"
    
    headers = {
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
        'User-Agent': 'TODO-Migration-Script/1.0'
    }
    
    if token:
        headers['Authorization'] = f'Bearer {token}'
    
    payload = {
        'title': issue_data['title'],
        'body': issue_data['body'],
        'labels': issue_data['labels']
    }
    
    try:
        response = requests.post(api_url, headers=headers, json=payload, timeout=30)
        
        if response.status_code == 201:
            issue_data_response = response.json()
            print(f"✅ Created issue #{issue_data_response['number']}: {issue_data['title']}")
            print(f"   URL: {issue_data_response['html_url']}")
            return True
        else:
            print(f"❌ Failed to create issue: {issue_data['title']}")
            print(f"   Status: {response.status_code}")
            print(f"   Response: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Network error creating issue: {issue_data['title']}")
        print(f"   Error: {e}")
        return False
    except Exception as e:
        print(f"❌ Error creating issue: {issue_data['title']}")
        print(f"   Error: {e}")
        return False

def main():
    repo = "rios0rios0/terraform-provider-http"
    script_dir = Path(__file__).parent
    issues_dir = script_dir
    
    print("🚀 GitHub Issues Creator - Direct API")
    print("=====================================")
    print(f"Repository: {repo}")
    print(f"Issues directory: {issues_dir}")
    print("")
    
    # Check for GitHub token
    token = os.environ.get('GITHUB_TOKEN')
    if not token:
        print("🔐 AUTHENTICATION SETUP REQUIRED")
        print("=================================")
        print("")
        print("This script requires a GitHub Personal Access Token to create issues.")
        print("")
        print("📝 SETUP STEPS:")
        print("1. Go to: https://github.com/settings/tokens/new")
        print("2. Create a token with 'repo' or 'public_repo' scope")
        print("3. Copy the token")
        print("4. Run: export GITHUB_TOKEN=your_token_here")
        print("5. Re-run this script")
        print("")
        print("🔒 SECURITY NOTE:")
        print("   - Keep your token secure and never commit it to version control")
        print("   - The token will only be used to create issues in your repository")
        print("   - You can revoke the token anytime at: https://github.com/settings/tokens")
        print("")
        print("🆘 ALTERNATIVE METHODS:")
        print("   - Use GitHub CLI: gh auth login && ./create-issues.sh")
        print("   - Manual creation: https://github.com/rios0rios0/terraform-provider-http/issues/new")
        return 1
    
    print("✅ GitHub token found")
    
    # Find all issue template files
    issue_files = []
    for i in range(1, 10):  # issue-01 to issue-09
        pattern = f"issue-{i:02d}-*.md"
        matching_files = list(issues_dir.glob(pattern))
        if matching_files:
            issue_files.append(matching_files[0])
    
    if not issue_files:
        print(f"❌ No issue template files found in {issues_dir}")
        print("   Expected files: issue-01-*.md through issue-09-*.md")
        return 1
    
    print(f"📋 Found {len(issue_files)} issue templates:")
    for i, file in enumerate(issue_files, 1):
        print(f"   {i}. {file.name}")
    print("")
    
    print("🚀 Creating GitHub issues...")
    print("")
    
    created_count = 0
    failed_count = 0
    
    for i, file_path in enumerate(issue_files, 1):
        print(f"📄 [{i}/{len(issue_files)}] Processing: {file_path.name}")
        
        try:
            issue_data = parse_issue_template(file_path)
            print(f"   Title: {issue_data['title']}")
            print(f"   Labels: {', '.join(issue_data['labels'])}")
            
            if create_github_issue(repo, issue_data, token):
                created_count += 1
            else:
                failed_count += 1
                
        except Exception as e:
            print(f"❌ Error processing {file_path}: {e}")
            failed_count += 1
        
        print("")
    
    print("🎉 CREATION SUMMARY")
    print("==================")
    print(f"✅ Created: {created_count} issues")
    print(f"❌ Failed: {failed_count} issues")
    print(f"📊 Total: {len(issue_files)} issues")
    print("")
    
    if failed_count == 0:
        print("🎊 SUCCESS! All issues created successfully!")
        print(f"🔗 View them at: https://github.com/{repo}/issues")
        print("")
        print("🧹 CLEANUP OPTIONS:")
        print("   # Remove TODO migration files:")
        print("   rm -rf docs/github-issues/")
        print("   git add -A && git commit -m 'Remove TODO migration templates after issue creation'")
        print("")
        print("✨ TODO migration completed successfully!")
        return 0
    else:
        print("⚠️  Some issues failed to create.")
        print("")
        print("💡 TROUBLESHOOTING:")
        print("   - Check your internet connection")
        print("   - Verify your GitHub token has 'repo' or 'public_repo' scope")
        print("   - Ensure the repository exists and you have write access")
        print("   - Try creating one issue manually to test permissions")
        return 1

if __name__ == "__main__":
    sys.exit(main())