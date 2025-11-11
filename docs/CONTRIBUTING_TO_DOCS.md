# nanogit Documentation

This directory contains the source files for the nanogit documentation site, published at [https://grafana.github.io/nanogit](https://grafana.github.io/nanogit).

## Structure

```
docs/
├── index.md                    # Home page
├── getting-started/
│   ├── installation.md         # Installation instructions
│   └── quick-start.md          # Quick start guide
├── architecture/
│   ├── overview.md             # Architecture overview
│   ├── storage.md              # Storage backend architecture
│   ├── delta-resolution.md     # Delta resolution implementation
│   └── performance.md          # Performance characteristics
└── changelog.md                # Version history (from CHANGELOG.md)
```

**Note**: Only `changelog.md` is copied from the root `CHANGELOG.md` during the build process. All other documentation files live directly in the `docs/` directory.

**API Documentation**: The complete API reference is available on [GoDoc](https://pkg.go.dev/github.com/grafana/nanogit), not duplicated here.

### What's NOT on GitHub Pages

Repository-specific files remain in the GitHub repository root only:
- **README.md** - Repository overview (GitHub homepage)
- **CONTRIBUTING.md** - Available in the repo for contributors
- **CODE_OF_CONDUCT.md** - GitHub displays this in Community tab
- **SECURITY.md** - GitHub displays this in Security tab
- **LICENSE.md** - GitHub displays this automatically
- **RELEASING.md** - Internal maintainer documentation
- **perf/README.md** - For contributors running benchmarks

GitHub Pages focuses on **user-facing documentation** for people using nanogit as a library, not repository maintenance.

## Building Locally

### Prerequisites

- Python 3.x
- pip

### Installation

**Recommended: Use a virtual environment**

```bash
# Create and activate virtual environment
python3 -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

**Or use make target (works outside venv on macOS/Linux)**

```bash
make docs-install
```

This handles Python 3.13+ externally-managed environments automatically.

**Note for macOS users**: If `mkdocs` is not in your PATH after installation, either:
- Add `~/Library/Python/3.x/bin` to your PATH, or
- Use `python3 -m mkdocs` instead of `mkdocs`

### Build and Serve

```bash
# Serve with live reload (recommended for development)
make docs

# Or use individual targets:
make docs-prepare    # Copy files from root
make docs-serve      # Serve at http://localhost:8000
make docs-build      # Build static site

# Or manually (if mkdocs not in PATH):
python3 -m mkdocs serve
```

**Note**: The `docs-serve` and `docs-build` targets automatically run `docs-prepare` to ensure files are up-to-date.

The documentation will be available at `http://localhost:8000`.

## Configuration

Documentation configuration is managed in `mkdocs.yml` at the repository root.

## Deployment

Documentation is automatically built and deployed to GitHub Pages when changes are pushed to the `main` branch. The deployment is handled by the `.github/workflows/docs.yml` workflow.

### Initial GitHub Pages Setup

If GitHub Pages hasn't been enabled yet, follow these one-time setup steps:

1. **Enable GitHub Pages**:
   - Go to repository **Settings** → **Pages**
   - Under "Build and deployment", set **Source** to `GitHub Actions`

2. **Verify Workflow Permissions**:
   - Go to **Settings** → **Actions** → **General**
   - Under "Workflow permissions", ensure "Read and write permissions" is selected

3. **Deploy**:
   - Push changes to `main` branch
   - GitHub Actions will automatically build and deploy
   - Documentation will be available at: https://grafana.github.io/nanogit

### Troubleshooting

**Pages not appearing:**
- Check workflow completed successfully in Actions tab
- Verify GitHub Pages is enabled with "GitHub Actions" as source
- Ensure workflow has proper permissions

**Build failures:**
- Review workflow logs in Actions tab
- Test locally with `make docs-build` before pushing
- Check for broken internal links

## Contributing to Documentation

1. **Edit existing pages**: Modify the Markdown files in the appropriate directory
2. **Add new pages**: Create new Markdown files and update the `nav` section in `mkdocs.yml`
3. **Test locally**: Run `mkdocs serve` to preview your changes
4. **Submit PR**: Follow the standard contribution process

## Style Guide

- Use clear, concise language
- Include code examples where appropriate
- Use proper Markdown formatting
- Add links to related documentation
- Keep navigation structure simple and intuitive

## Links

- **Live documentation**: https://grafana.github.io/nanogit
- **Repository**: https://github.com/grafana/nanogit
- **MkDocs**: https://www.mkdocs.org
- **Material theme**: https://squidfunk.github.io/mkdocs-material/
