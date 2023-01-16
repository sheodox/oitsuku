# oitsuku (追いつく, to catch up)

This is a small terminal utility for updating outdated node modules to `@latest` with npm.

https://user-images.githubusercontent.com/3468630/212238194-4320523e-c02b-4629-b670-91f686c95231.mp4

## Setup

This requires [`go`](https://go.dev/)

1. Clone this repo
1. `go build`
1. Put the file in your PATH or `alias oitsuku="~/path/to/clone/oitsuku"`

## Usage

Just go to a repository that uses npm and run `oitsuku`.

Use arrow keys or `j`/`k` to go up and down, `q` or `escape` exits. Press space to select a package to update, and hit enter to install the selected packages. Pressing `a` selects or deselects all.
