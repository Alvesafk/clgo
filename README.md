# clgo

clgo is a [cloc](https://github.com/AlDanial/cloc) implementation made in **Go**.

It uses concurrency to quickly scan the entry folder and count the lines very quickly.

On a cold start, without any file on cache, clgo went through all the Linux git repo in **~3 seconds**.

It's still simple and it doesn't have support to all languages and suffixes, it doesn't handle many edge cases yet, so, feel free to open a issue with a language to add or a edge case problem to report.

## Install

You can build it from source:
```sh
git clone https://github.com/Alvesafk/clgo --depth=1
cd clgo
go build -o bin/clgo cmd/main.go
```
After this, you can put the binary on you `$PATH`.

Using `go install`:
```sh
go install github.com/Alvesafk/clgo@latest
```
The binary will be in you `go/bin` directory, so make sure to configure this.

## Roadmap

Some stuff that i want to add to **clgo**.
- [ ] Ability to input multiple dirs or files on program entry.
- [ ] Support to more languages and edge cases.
- [ ] Git integration.
- [ ] Better customization with flags.

There is a lot of other stuff to add, but the focus is on this features now.

Thanks for reading, sincerely Alvesafk.
