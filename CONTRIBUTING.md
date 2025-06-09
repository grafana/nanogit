# Contributing to NanoGit

Thank you for your interest in contributing to NanoGit! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to keep our community approachable and respectable.

## Prerequisites

Before you begin contributing, ensure you have the following installed:

* [Docker](https://docs.docker.com/get-docker/) - Required for running integration tests
* Go 1.24 or later
* Git

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

We use Go's built-in testing framework with [testify](https://github.com/stretchr/testify) for assertions. To run the tests:

```bash
make test # run all tests
make test-unit # run only unit tests
make test-integration # run only integration tests
```

#### Unit Tests

Unit tests are located alongside the code they test (e.g., `client_test.go` in the same directory as `client.go`). We use testify's `assert` and `require` packages for assertions:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
    // Use require for setup/teardown that must succeed
    require.NoError(t, err)
    
    // Use assert for test conditions
    assert.Equal(t, expected, actual)
}
```

#### Integration Tests

Integration tests are located in the `test/` directory and use [testcontainers-go](https://golang.testcontainers.org/) to manage test dependencies. We use Gitea as our test Git server, running in a Docker container. 

Key features:
- Tests use a shared Git server container for better performance
- Automatic container lifecycle management with testify/suite
- Real Git server for testing using [Gitea](https://gitea.io/) in a Docker container
- Parallel test execution support where appropriate
- Uses `_integration_test.go` naming convention (no build tags required)

Integration tests use the Git CLI to set up test repositories and perform Git operations. This ensures we test against real Git behavior and verify our protocol implementation matches the official specification.

Example structure:
```bash
test/
├── helpers/
│   └── suite.go              # shared test suite infrastructure
├── integration_suite_test.go # main test suite runner
├── auth_test.go             # authentication integration tests
├── refs_test.go             # reference operation tests
├── writer_test.go           # writer operation tests
└── ...                      # other integration test files
```

To run integration tests:
```bash
make test-integration
```

To run a specific integration test:
```bash
go test ./test/... -run TestIntegrationTestSuite/TestSpecificTest
```

Note: Integration tests require Docker to be running on your machine.

#### Writing Tests

1. Unit tests should be fast and not require external dependencies
2. Integration tests should be in the `integration` directory
3. Use testify's `assert` and `require` packages. We prefer the use of `require`.
4. Follow Go's testing best practices
5. Add appropriate test coverage

For more information:
- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Testify Documentation](https://pkg.go.dev/github.com/stretchr/testify)
- [Testcontainers-go Documentation](https://golang.testcontainers.org/)
- [Gitea Documentation](https://docs.gitea.io/)

### Testing with Mocks

nanogit includes generated mocks for easy unit testing using [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter). 

To regenerate mocks after interface changes:

```bash
make generate
```

The generated mocks are located in the `mocks/` directory and provide test doubles for both `Client` and `StagedWriter` interfaces. See [mocks/example_test.go](mocks/example_test.go) for complete usage examples.

### Code Style

* Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
* Use `make fmt` to format your code.
* Run `make lint` to check for style issues.

### Editor Settings

#### Cursor

We provide rules for cursor in `.cursor` directory that defines our coding standards and best practices. The rules cover:

* Code style and formatting
* Testing requirements
* Error handling patterns
* Documentation standards
* Security considerations
* Performance guidelines
* Git protocol compliance
* Code organization
* Versioning practices
* CI/CD requirements
* Code review guidelines
* Maintenance standards
* Accessibility requirements
* Extensibility guidelines

To use these rules in Cursor:
1. Open the project in Cursor
2. The rules will be automatically loaded
3. Cursor will provide inline suggestions based on these rules


### Contributing to Cursor Rules

We welcome contributions to improve our Cursor rules! The rules are designed to help maintain code quality and consistency, but they're not set in stone. If you have suggestions for improvements or find areas that could be enhanced, please feel free to contribute.

#### How to Contribute to Rules

1. **Identify Areas for Improvement**
   - Look for patterns that could be better enforced
   - Identify missing best practices
   - Suggest clearer guidelines for existing rules

2. **Propose Changes**
   - Open an issue to discuss proposed changes
   - Explain the rationale behind your suggestions
   - Provide examples of how the changes would improve the codebase

3. **Submit Pull Requests**
   - Update the relevant rule files in the `.cursor` directory
   - Include clear documentation for any new rules
   - Add examples where appropriate

4. **Review Process**
   - All rule changes will be reviewed by maintainers
   - Changes should align with the project's goals
   - Consider the impact on existing code

#### Rule Categories

Feel free to contribute to any of these categories:

* **Code Style**: Suggest improvements to formatting and style guidelines
* **Testing**: Propose new testing requirements or best practices
* **Error Handling**: Enhance error handling patterns
* **Documentation**: Improve documentation standards
* **Security**: Add new security considerations
* **Performance**: Suggest performance optimizations
* **Git Protocol**: Enhance Git protocol compliance rules
* **Code Organization**: Propose better code structure guidelines
* **Versioning**: Improve versioning practices
* **CI/CD**: Add new CI/CD requirements
* **Code Review**: Enhance code review guidelines
* **Maintenance**: Suggest maintenance standards
* **Accessibility**: Add accessibility requirements
* **Extensibility**: Propose extensibility guidelines

#### Best Practices for Rule Contributions

1. **Keep Rules Clear and Concise**
   - Rules should be easy to understand
   - Avoid overly complex requirements
   - Provide clear examples

2. **Consider Impact**
   - Evaluate the impact on existing code
   - Consider the learning curve for new contributors
   - Balance strictness with practicality

3. **Documentation**
   - Include clear explanations for new rules
   - Provide examples of correct and incorrect usage
   - Link to relevant documentation or resources

4. **Testing**
   - Test rules against existing code
   - Ensure rules don't conflict with each other
   - Verify rules work as expected in Cursor

Remember, the goal is to make the development experience better for everyone. Your contributions can help shape the future of this project's development standards.


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