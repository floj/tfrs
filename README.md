# tfrs

Thin wrapper around Terraform/OpenTofu that provides interactive fuzzy selection
of resources and modules for the `-target` option.

`tfrs` collects the resources and modules of a Terraform configuration, lets you
pick them interactively with [`fzf`](https://github.com/junegunn/fzf), and either
prints the selected `-target`-ready addresses or runs a command with them
appended.

## Requirements

- [`fzf`](https://github.com/junegunn/fzf) on your `PATH`.
- `terraform` or `tofu` (only required for `--use-state`).

## Install

```sh
go install github.com/floj/tfrs@latest
```

Prebuilt binaries for Linux, macOS and Windows are attached to each
[release](https://github.com/floj/tfrs/releases).

## Usage

```sh
# Print selected targets, prefixed and space separated
tfrs

# Run a command with the selected resources appended (as -target args)
tfrs --prefix '-target=' --exec terraform -- plan

# Descend two levels into submodules
tfrs --depth 2

# Discover resources from the state instead of parsing the configuration
tfrs --use-state
```

Selecting the `<all>` entry clears the selection so the wrapped command runs
against everything. Cancelling `fzf` (Esc / Ctrl-C) exits cleanly without
running anything.

## Flags

| Flag          | Env var           | Default      | Description                                                        |
| ------------- | ----------------- | ------------ | ------------------------------------------------------------------ |
| `--list`      | `TFRS_LIST_ONLY`  | `false`      | Just list the (prefixed) resources, one per line.                  |
| `--chdir`     | `TFRS_CHDIR`      | `.`          | Look up resources from this directory.                             |
| `--tf-bin`    | `TFRS_TF_BIN`     | `terraform`  | Path to the terraform/OpenTofu binary (used by `--use-state`).     |
| `--depth`     | `TFRS_MAX_DEPTH`  | `0`          | How many levels to descend into submodules.                        |
| `--prefix`    | `TFRS_PREFIX`     | *(empty)*    | Add as a prefix before each selected entry (e.g. `-target=`).      |
| `--exec`      | `TFRS_EXEC_CMD`   | *(empty)*    | If set, run this command and pass the selected, prefixed resources.|
| `--use-state` | `TFRS_USE_STATE`  | `false`      | Use `terraform state list` to determine the available resources.   |
| `--version`   |                   |              | Print version information.                                         |

## License

See [LICENSE](LICENSE).
