# GitHub Actions Digest Pinner

GitHub Actions Digest Pinner is a tool to help you pin GitHub Actions to specific digests for better security and
reliability.

## Features

- Finds and updates GitHub Actions references in your repository.
- Ensures all actions are pinned to specific digests.

## Installation

**Recommended:** Just download the binary from the [releases page](https://github.com/zisuu/github-actions-digest-pinner/releases)

If you want you can also install it using `go`, but be aware the the `version` command will not work because the ldflags are not set during the `go install` process.

```bash
go install github.com/zisuu/github-actions-digest-pinner/cmd/github-actions-digest-pinner@latest
```

## Usage

```bash
# Run the tool in your repository
github-actions-digest-pinner
```

## Issues

If you encounter any issues, please report them on the [GitHub Issues page](https://github.com/zisuu/github-actions-digest-pinner/issues)

## Contributing

Contributions are welcome! Please see the `CONTRIBUTING.md` for guidelines.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
