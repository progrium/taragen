---
title: "Install"
kind: manual
num: 1
order: 1
part: "getting started"
---

With Homebrew:

```sh
brew tap progrium/homebrew-taps
brew install taragen
```

Or download a release binary for your platform (macOS, Linux, and Windows, on
amd64 and arm64) from the GitHub releases page and put it on your `PATH`.

To build from source you need only Go:

```sh
make build   # writes .local/bin/taragen
make link    # symlinks it into /usr/local/bin
```
