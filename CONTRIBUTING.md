# Contributing to liatrio-otel-collector

We love your input! We want to make contributing to this project as easy and
transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features

## We Develop with Github

We use GitHub to host code, to track issues and feature requests, as well as
accept pull requests.

## We Use [Coding Conventions, e.g., PEP8], So Pull Requests Need To Pass This

1. Read [effective go](https://go.dev/doc/effective_go) as it's a great starting
point.
2. Then read OpenTelemetry's
[coding guidelines](https://github.com/open-telemetry/opentelemetry-collector/blob/main/CONTRIBUTING.md#coding-guidelines)
as we generally follow this.

## All Code Changes Happen Through Pull Requests

Pull requests are the best way to propose changes to the codebase. We actively
welcome your pull requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Any contributions you make will be under the [Software License]

Explain that when someone submits a contribution, they are agreeing that the
project owner can use their contribution under the project's license.

## Report bugs using Github's [issues](https://github.com/liatrio/liatrio-otel-collector/issues)

We use GitHub issues to track public bugs. Report a bug by opening a new issue;
it's that easy!

## Write bug reports with detail, background, and sample code

Great Bug Reports tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can.
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you
tried that didn't work)

## Use a Consistent Coding Style

TODO: Detail the style guide and coding conventions further, if needed.

## Running Checks

After making a code change, run the `make checks` command to validate that all
code is formatted, tested, etc. successfully:

## Adding New Components

New components can be created by running compgen. See compgen's
[README](./cmd/compgen/README.md) for instructions and guidance.

## License

By contributing, you agree that your contributions will be licensed under its
[License Name].

## References

Include any references or resources that might be helpful for the contributor.

Thank you for considering contributing to liatrio-otel-collector!
