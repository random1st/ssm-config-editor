# SSM Config Editor

`ssm-config-editor` is a command line tool that simplifies managing AWS Systems Manager (SSM) parameters. Inspired by `kubectl edit`, this tool offers various commands to create, edit, delete, list, and upload SSM parameters with support for optional formats (JSON, YAML, ENV).

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Commands](#commands)
    - [list](#list)
    - [get](#get)
    - [create](#create)
    - [edit](#edit)
    - [delete](#delete)
    - [upload](#upload)
- [License](#license)

## Installation

To install the `ssm-config-editor`, use one of the following methods:

1. Download the latest binary from the [Releases](https://github.com/random1st/ssm-config-editor/releases) page.

2. Build from source:

```bash
git clone https://github.com/random1st/ssm-config-editor.git
cd ssm-config-editor
go build -o ssm-config-editor .
```

3. Install using `go install`:

```bash
go install github.com/random1st/ssm-config-editor/cmd/ssm@latest
```

## Usage

```bash
ssm [command] [flags]
```

## Commands

### list

List SSM parameters with optional prefix filter:

```bash
ssm list --region <AWS_REGION> --prefix <PREFIX>
```

### get

Get the value of an SSM parameter:

```bash
ssm get <SSM_KEY> --region <AWS_REGION>
```

### create

Create a new SSM parameter:

```bash
ssm create <SSM_KEY> --region <AWS_REGION> --format <FORMAT> --from <SOURCE_KEY>
```

### edit

Edit an existing SSM parameter:

```bash
ssm edit <SSM_KEY> --region <AWS_REGION> --format <FORMAT>
```

### delete

Delete an SSM parameter:

```bash
ssm delete <SSM_KEY> --region <AWS_REGION>
```

### upload

Upload an SSM parameter value from a file:

```bash
ssm upload <SSM_KEY> <FILE_PATH> --region <AWS_REGION>
```
## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
