# GitHub Actions Digest Pinner

GitHub Actions Digest Pinner is a tool to help you pin GitHub Actions to specific digests for better security and
reliability.

## Features

- Finds and updates GitHub Actions references in your repository.
- Ensures all actions are pinned to specific digests.

## Installation

**Recommended:** Just download the binary from the [releases page](https://github.com/zisuu/github-actions-digest-pinner/releases).

### Steps to Manually Download and Install the Binary

1. Visit the [releases page](https://github.com/zisuu/github-actions-digest-pinner/releases).
2. Download the appropriate binary for your operating system and architecture (e.g.,
`github-actions-digest-pinner_linux_amd64.tar.gz` for Linux 64-bit).

3. Extract the downloaded `.tar.gz` file:

   ```bash
   tar -xvzf github-actions-digest-pinner_<os>_<arch>.tar.gz
   ```

   Replace `<os>` and `<arch>` with your operating system and architecture.

4. Move the extracted binary to a directory in your `PATH` (e.g., `/usr/local/bin`):

   ```bash
   sudo mv github-actions-digest-pinner /usr/local/bin/
   ```

5. Verify the installation:

   ```bash
   github-actions-digest-pinner --version
   ```

If you want, you can also install it using `go`, but be aware that the `version` command will not
work because the ldflags are not set during the `go install` process.

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
