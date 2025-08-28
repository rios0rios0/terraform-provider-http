# Create GitHub Issues from TODO Migration

This directory contains scripts to create 9 GitHub issues based on the TODOs found in the codebase during the migration process.

## Quick Start

### Option 1: GitHub CLI (Recommended)
```bash
# First, authenticate with GitHub CLI if not already done
gh auth login

# Then run the script
./create-github-issues.sh
```

### Option 2: GitHub API with Personal Access Token
```bash
# Get a personal access token from: https://github.com/settings/tokens/new
# Permissions needed: repo (to create issues)

export GITHUB_TOKEN="your_token_here"
./create-issues-api.sh
```

## Issues to be Created

The scripts will create **9 GitHub issues** representing the TODOs found in the codebase:

### High Priority Issues (2)
1. **Implement delete functionality for HTTP request resources**
   - Location: Resource lifecycle management
   - Priority: High

2. **Implement Read method for drift detection and state comparison**  
   - Location: `internal/provider/resource_http_request.go:335`
   - Priority: High

### Medium Priority Issues (2)
3. **Add UUID validation for resource import operations**
   - Component: Resource import and validation
   - Priority: Medium

4. **Enhanced URL validation at provider and resource levels**
   - Component: Provider configuration validation  
   - Priority: Medium

### Low Priority Issues (5)
5. **Implement URL field validation using string validators**
   - Location: `internal/provider/provider.go:59`
   - Priority: Low

6. **Evaluate JSON-based configuration parsing optimization**
   - Location: `internal/provider/provider.go:205`
   - Priority: Low

7. **Improve import functionality documentation with examples**
   - Location: `internal/provider/resource_http_request.go:154`
   - Priority: Low

8. **Review and improve Terraform validator implementation**
   - Location: `internal/infrastructure/validators/string_not_empty.go:37`
   - Priority: Low

9. **Refactor test builder pattern to fluent API**
   - Location: `test/infrastructure/builders/provider_value_builder.go:73` and `90`
   - Priority: Low

## Issue Details

Each issue includes:
- ✅ Detailed description and context
- ✅ Clear acceptance criteria
- ✅ Technical notes with code references
- ✅ Appropriate labels (enhancement, high-priority, validation, etc.)
- ✅ Priority level and component information

## Troubleshooting

**GitHub CLI not authenticated:**
```bash
gh auth login
```

**Personal access token issues:**
- Ensure token has `repo` permissions
- Get token from: https://github.com/settings/tokens/new

**Issues already exist:**
The scripts will show errors if issues with the same titles already exist. This is normal if you've run the script before.

---

All issues are based on the TODO comments that were found and documented during the migration process in PR #19.