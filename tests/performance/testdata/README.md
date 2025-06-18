# Test Repository Archives

This directory contains compressed archives of pre-generated Git repositories used for performance testing.

## Generating Test Repositories

To create the test repository archives, run:

```bash
cd /path/to/nanogit/tests/performance
go run ./cmd/generate_repo
```

This will create four archives:
- `small-repo.tar.gz` - Small repository (100 files, 50 commits)
- `medium-repo.tar.gz` - Medium repository (750 files, 200 commits)  
- `large-repo.tar.gz` - Large repository (3000 files, 800 commits)
- `xlarge-repo.tar.gz` - Extra-large repository (15000 files, 3000 commits)

## Archive Contents

Each archive contains a complete Git repository with:
- Full commit history
- Realistic file structure (src/, docs/, tests/, etc.)
- Various file types (.go, .js, .py, .md, etc.)
- Binary files simulation
- Standard files (README.md, .gitignore, LICENSE)

## Usage

The performance tests will automatically extract and mount these repositories in the Gitea container. If the archives are missing, the tests will fail with instructions to generate them.

## Benefits

Using pre-created archives instead of generating repositories at test time provides:
- **Faster test startup** - No time spent generating files and commits
- **Consistent test data** - Same repository state every test run
- **Better performance isolation** - Repository generation doesn't affect measurements
- **Reproducible results** - Same test data across different environments