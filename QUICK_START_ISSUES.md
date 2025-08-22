# Quick Start: Create GitHub Issues

The GitHub issue templates are ready! To create all 9 issues, run these commands:

## 1. Authenticate with GitHub CLI (if not already done)
```bash
gh auth login
```

## 2. Navigate to the issue templates directory
```bash
cd docs/github-issues
```

## 3. Run the automated creation script
```bash
./create-issues.sh
```

## Expected Result
✅ 9 GitHub issues will be created automatically:

1. **Implement delete functionality for HTTP request resources** (high-priority)
2. **Add UUID validation for resource import operations** (validation)
3. **Enhanced URL validation at provider and resource levels** (validation)  
4. **Implement URL field validation using string validators** (validation)
5. **Evaluate JSON-based configuration parsing optimization** (performance)
6. **Improve import functionality documentation with examples** (documentation)
7. **Implement Read method for drift detection and state comparison** (high-priority)
8. **Review and improve Terraform validator implementation** (validation)
9. **Refactor test builder pattern to fluent API** (testing)

## Troubleshooting

**If GitHub CLI is not installed:**
- macOS: `brew install gh`
- Ubuntu/Debian: `sudo apt install gh`
- Or download from: https://cli.github.com/

**If authentication fails:**
- Run: `gh auth login`
- Choose your preferred authentication method
- Follow the prompts to authenticate

**If the script fails:**
- Check the README.md in this directory for manual creation instructions
- Each issue template can be copied manually to the GitHub web interface

## Alternative: Manual Creation
If automated creation doesn't work, go to:
https://github.com/rios0rios0/terraform-provider-http/issues/new

Then copy title, labels, and body from each `issue-*.md` file in this directory.

---

**Note:** The firewall restrictions mentioned in the comments have been removed, so the script should work properly now.