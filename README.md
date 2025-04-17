# goinstaller

<p align="center">
  <h3 align="center">goinstaller</h3>
  <p align="center">A streamlined installer for Go binaries with enhanced security features.</p>
  <p align="center">
    <a href="/LICENSE.md"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
  </p>
</p>

---

goinstaller is a fork of [godownloader](https://github.com/goreleaser/godownloader) that generates shell scripts to download the right package and version of Go binaries released with [GoReleaser](https://github.com/goreleaser/goreleaser). The generated scripts make it easy for users to install Go applications with a simple curl | bash command.

## Features

* **GoReleaser Integration**: Reads GoReleaser YAML files to create custom installation scripts
* **Cross-Platform Support**: Works on various operating systems and architectures
* **Checksum Verification**: Ensures the integrity of downloaded binaries
* **GitHub Attestation Verification**: Verifies GitHub attestations to enhance security (new feature)
* **Fast Installation**: Much faster than 'go get' (sometimes up to 100x)

## Usage

### Basic Usage

```bash
# Generate an installation script for a GitHub repository
goinstaller --repo=owner/repo > install.sh

# Generate an installation script from a local GoReleaser config
goinstaller --file=.goreleaser.yml > install.sh

# Generate an installation script with attestation verification
goinstaller --repo=owner/repo --require-attestation > install.sh
```

### Example

Let's say you are using [hugo](https://gohugo.io), the static website generator, with [travis-ci](https://travis-ci.org).

Your old `.travis.yml` file might have:

```yaml
install:
  - go get github.com/gohugoio/hugo
```

This can take up to 30 seconds!

With goinstaller, you can create a script:

```bash
# create an installer script
goinstaller --repo=gohugoio/hugo > ./goinstaller-hugo.sh
```

Add `goinstaller-hugo.sh` to your GitHub repo and edit your `.travis.yml`:

```yaml
install:
  - ./goinstaller-hugo.sh v0.37.1
```

Without a version number, GitHub is queried to get the latest version number:

```yaml
install:
  - ./goinstaller-hugo.sh
```

Typical download time is 0.3 seconds, or 100x improvement.

Your new `hugo` binary is in `./bin`, so change your Makefile or scripts to use `./bin/hugo`.

The default installation directory can be changed with the `-b` flag or the `BINDIR` environment variable.

## Installation

```bash
# Install the latest version
go install github.com/haya14busa/goinstaller@latest

# Or install a specific version
go install github.com/haya14busa/goinstaller@v0.1.0
```

## Differences from Original GoDownloader

goinstaller is a streamlined fork of GoDownloader with the following changes:

* Removed support for Equinox.io
* Removed support for "naked" releases on GitHub
* Removed tree walking functionality
* Added GitHub attestation verification
* Enhanced security features
* Improved code structure and maintainability

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

## Acknowledgments

* Original [GoDownloader](https://github.com/goreleaser/godownloader) project
* [GoReleaser](https://github.com/goreleaser/goreleaser) team
