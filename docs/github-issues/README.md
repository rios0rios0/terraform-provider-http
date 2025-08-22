# GitHub Issues for TODO Migration

This directory contains issue templates for migrating all TODOs from the codebase to proper GitHub issues.

## Quick Start

### Option 1: Automated Creation (Recommended)

```bash
cd docs/github-issues
./create-issues.sh
```

This script will:
1. Check if GitHub CLI is installed and authenticated
2. Create all 9 issues automatically using the templates
3. Apply appropriate labels to each issue
4. Provide a summary of created issues

### Option 2: Manual Creation via GitHub CLI

If the automated script doesn't work, you can create issues individually:

```bash
# Authenticate first (if not already done)
gh auth login

# Create each issue manually
gh issue create --title "Implement delete functionality for HTTP request resources" --label "enhancement,high-priority" --body-file "issue-01-delete-functionality.md" --repo "rios0rios0/terraform-provider-http"

gh issue create --title "Add UUID validation for resource import operations" --label "enhancement,validation" --body-file "issue-02-uuid-validation.md" --repo "rios0rios0/terraform-provider-http"

# ... (repeat for all 9 issues)
```

### Option 3: Manual Creation via Web Interface

1. Go to [https://github.com/rios0rios0/terraform-provider-http/issues/new](https://github.com/rios0rios0/terraform-provider-http/issues/new)
2. For each issue template file:
   - Copy the **Title** line as the issue title
   - Copy the **Labels** line and add those labels
   - Copy everything after **Body:** as the issue description
3. Click "Submit new issue"

## Issue Templates Overview

| Priority | Issue | Description | Labels |
|----------|-------|-------------|--------|
| High | [Delete functionality](issue-01-delete-functionality.md) | Implement proper delete method for resource lifecycle | `enhancement`, `high-priority` |
| High | [Read method implementation](issue-07-read-method-implementation.md) | Enable drift detection and state comparison | `enhancement`, `high-priority` |
| Medium | [UUID validation](issue-02-uuid-validation.md) | Add UUID validation for import operations | `enhancement`, `validation` |
| Medium | [URL validation (provider)](issue-03-url-validation-provider.md) | Enhanced URL validation at provider level | `enhancement`, `validation` |
| Medium | [URL field validators](issue-04-url-field-validators.md) | Implement URL field validation using string validators | `enhancement`, `validation` |
| Medium | [Validator implementation](issue-08-validator-implementation.md) | Review and improve Terraform validator implementation | `enhancement`, `validation` |
| Low | [JSON config parsing](issue-05-json-config-parsing.md) | Evaluate JSON-based configuration parsing optimization | `enhancement`, `performance` |
| Low | [Import documentation](issue-06-import-documentation.md) | Improve import functionality documentation | `documentation`, `enhancement` |
| Low | [Test builder refactoring](issue-09-test-builder-refactoring.md) | Refactor test builder pattern to fluent API | `enhancement`, `testing` |

## Prerequisites

### For Automated Creation
- [GitHub CLI](https://cli.github.com/) installed
- Authenticated with GitHub (`gh auth login`)
- Write access to the repository

### For Manual Creation
- GitHub account with write access to the repository
- Web browser or GitHub CLI access

## Troubleshooting

### GitHub CLI Not Installed
```bash
# On macOS with Homebrew
brew install gh

# On Ubuntu/Debian
sudo apt install gh

# Or download from https://cli.github.com/
```

### Authentication Issues
```bash
# Re-authenticate with GitHub CLI
gh auth login

# Check authentication status
gh auth status
```

### Permission Issues
- Ensure you have write access to the repository
- Contact repository maintainers if you need access

## What Happens After Creation

Once the issues are created:
1. They will appear in the [repository issues list](https://github.com/rios0rios0/terraform-provider-http/issues)
2. They can be assigned to team members
3. Development work can be tracked against each issue
4. The TODO comments have already been removed from the codebase
5. Each issue includes detailed acceptance criteria and technical notes

## Next Steps

After creating the issues:
1. Review and prioritize the issues based on project needs
2. Assign issues to appropriate team members
3. Create milestones or projects to track progress
4. Begin development work on high-priority items
5. Close this migration issue (#18) once all TODO issues are created