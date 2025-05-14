# GitHub Actions Digest Pinner

GitHub Actions Digest Pinner is a tool to help you pin GitHub Actions to specific digests for better security and reliability.

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

### Alternative Installation Using `go install`

You can also install the tool using `go install`, but note that the `version` command will not work because the
`ldflags` are not set during the `go install` process:

```bash
go install github.com/zisuu/github-actions-digest-pinner/cmd/github-actions-digest-pinner@latest
```

## Usage

### Basic Usage

Run the tool in your repository:

```bash
github-actions-digest-pinner update
```

### Commands

- **`scan`**: Scans the repository for GitHub Actions workflows and lists the actions it would update.

  ```bash
  github-actions-digest-pinner scan --dir <directory> --verbose
  ```

- **`update`**: Updates GitHub Actions workflows to use pinned digests.

  ```bash
  github-actions-digest-pinner update --dir <directory> --timeout 30 --verbose
  ```

## Configuration

The tool does not require configuration files but supports the following flags:

- `--dir`: Specify the directory containing GitHub workflows (default: current directory).
- `--verbose`: Enable verbose output.
- `--timeout`: Set the API timeout in seconds (default: 30).

## Output

The tool provides detailed logs when run with the `--verbose` flag, including:

- Workflow files found.
- Actions parsed from each workflow file.
- Actions updated with their resolved digests.

## Issues

If you encounter any issues, please report them on the [GitHub Issues page](https://github.com/zisuu/github-actions-digest-pinner/issues).

## Contributing

Contributions are welcome! To contribute:

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Submit a pull request with a clear description of your changes.

See the `CONTRIBUTING.md` for more details.

## License

This project is licensed under the [MIT License](LICENSE).
