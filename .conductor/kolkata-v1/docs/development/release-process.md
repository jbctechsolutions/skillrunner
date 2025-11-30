# Release Guide

Quick reference for creating releases of Skillrunner.

## Quick Release

```bash
# 1. Update version and commit changes
git checkout main
git pull

# 2. Create tag
git tag -a v0.1.0 -m "Release v0.1.0: Initial release with multi-provider routing"

# 3. Push tag
git push origin v0.1.0

# 4. GitHub Actions will automatically:
#    - Build binaries for all platforms
#    - Run tests
#    - Create GitHub release
#    - Upload artifacts
```

## Pre-Release Checklist

- [ ] All tests pass: `make test`
- [ ] Code is formatted: `make fmt`
- [ ] All PRs for this release are merged
- [ ] Version follows semver (MAJOR.MINOR.PATCH)
- [ ] Breaking changes documented
- [ ] Update CHANGELOG.md (if exists)

## Version Numbers

### Standard Release
- `v1.0.0` - Major release (breaking changes)
- `v0.2.0` - Minor release (new features)
- `v0.1.1` - Patch release (bug fixes)

### Pre-Release
- `v1.0.0-alpha.1` - Alpha (internal testing)
- `v1.0.0-beta.1` - Beta (public testing)
- `v1.0.0-rc.1` - Release candidate

## What Happens Automatically

1. **Tests Run** - All test suites execute
2. **Build** - Binaries for all platforms are compiled
3. **Checksums** - SHA256 checksums generated
4. **Release** - GitHub release created with:
   - Auto-generated release notes
   - All binaries attached
   - Checksums file
5. **Verification** - Binaries tested on each platform

## Manual Release (Alternative)

If tag-based release doesn't work:

1. Go to: https://github.com/jbctechsolutions/skillrunner/actions/workflows/release.yml
2. Click "Run workflow"
3. Enter version (e.g., `v0.1.0`)
4. Click "Run workflow"

## After Release

1. Verify release on GitHub: https://github.com/jbctechsolutions/skillrunner/releases
2. Test installation: `curl -sSL https://raw.githubusercontent.com/jbctechsolutions/skillrunner/main/install.sh | bash`
3. Verify binary works: `skillrunner --version`
4. Announce release (Twitter, Discord, etc.)

## Rollback

If a release has issues:

1. Delete the tag: `git tag -d v0.1.0 && git push origin :refs/tags/v0.1.0`
2. Delete the GitHub release
3. Fix issues
4. Create new release with patch version

## Troubleshooting

### Release workflow didn't trigger
- Ensure tag format is `v*` (starts with v)
- Verify tag is pushed: `git push origin v0.1.0`
- Check GitHub Actions tab for errors

### Binaries missing
- Check workflow logs
- Ensure `make build-all` succeeds locally
- Verify disk space in Actions runner

### Installation script fails
- Test install.sh locally
- Verify release binaries are accessible
- Check checksums match

## Resources

- Full CI/CD documentation: [docs/CICD.md](./CICD.md)
- GitHub Releases: https://github.com/jbctechsolutions/skillrunner/releases
- GitHub Actions: https://github.com/jbctechsolutions/skillrunner/actions
