# Contributing to NanoGit

Thank you for your interest in contributing to NanoGit! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to keep our community approachable and respectable.

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check the issue list as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* Use a clear and descriptive title
* Describe the exact steps to reproduce the problem
* Provide specific examples to demonstrate the steps
* Describe the behavior you observed after following the steps
* Explain which behavior you expected to see instead and why
* Include screenshots if possible
* Include the output of any error messages

### Suggesting Enhancements

If you have a suggestion for a new feature or enhancement, please include as much detail as possible:

* Use a clear and descriptive title
* Provide a step-by-step description of the suggested enhancement
* Provide specific examples to demonstrate the steps
* Describe the current behavior and explain which behavior you expected to see instead
* Explain why this enhancement would be useful to most users

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code lints
6. Issue that pull request!

### Development Process

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/nanogit.git
   cd nanogit
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Create a new branch for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. Make your changes and commit them:
   ```bash
   git add .
   git commit -m "Description of your changes"
   ```

5. Push your changes:
   ```bash
   git push origin feature/your-feature-name
   ```

6. Create a Pull Request from your branch to `main`

### Testing

We use Go's built-in testing framework. To run the tests:

```bash
make test # run all tests
make test-unit # run only unit tests
make test-integration # run only integration tests
```

### Code Style

* Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
* Use `make fmt` to format your code.
* Run `make lint` to check for style issues.

### Documentation

* Update documentation for any new features or changes.
* Follow the existing documentation style.
* Include examples where appropriate.

## Getting Help

If you need help, you can:

* Open an issue
* Check the existing documentation

## License

By contributing to NanoGit, you agree that your contributions will be licensed under the project's [Apache License 2.0](LICENSE.md). 