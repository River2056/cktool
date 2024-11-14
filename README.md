# Fetch logs from git tags
this tool will fetch log directly from your git repository using tags
will include logs from starting tag to ending tag (exclusive)

when issuing a `git log --oneline`, your log will look like this
```
xxx log1 tag: XXX-XXX <- starting tag
...
...
yyy log2 tag: YYY-YYY <- ending tag
```
this tool will fetch every commit message in between starting tag and ending tag (excluding the commit in ending tag)


# How to use this tool?
1. install Go on your machine
2. modify the `Makefile` accordingly
    - usually just the build, build_param, build_count rule
3. either run `go install` or run the make rule: `make build_target_linux` or `make build_target_darwin` according to your platform
4. use the `-h` flag to check cli tool usage

Or you could just use prebuit binaries in this repository, just remember to add `cktool` to your `PATH`
