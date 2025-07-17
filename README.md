# gocovmerge

`gocovmerge` takes the results from multiple `go test -coverprofile` runs and merges them into one profile.

## Installation

To install the latest version, run:

```
go install github.com/ja7ad/gocovmerge/cmd/gocovmerge@latest
```

This will install the `gocovmerge` binary into your `$GOPATH/bin` or `$GOBIN` directory.

## Usage

```
gocovmerge [coverprofiles...]
```

- `coverprofiles...`: One or more Go coverage profile files (output from `go test -coverprofile=coverage.out`).

`gocovmerge` takes the source coverprofiles as arguments and outputs a merged version of the files to standard out.

**Note:** You can only merge profiles that were generated from the same source code. If there are source lines that overlap or do not merge, the process will exit with an error code.

### Example

1. Run tests with coverage in different packages or configurations:

    ```sh
    go test -coverprofile=unit_coverage.txt ./...
    go test -coverprofile=integration_coverage.txt ./...
    ```

2. Merge the coverage profiles:

    ```sh
    gocovmerge unit_coverage.txt integration_coverage.txt > merged_coverage.txt
    ```

3. Use the merged coverage profile as needed (e.g., for coverage reporting tools).

## License

See [LICENSE](LICENSE).
