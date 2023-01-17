# go-dedup

go-dedup is a portable golang (Windows/Linux) file de-duplication tool.

## Description

Go-dedup find files in path arguments, make fast hash with xxhash , and display identical files names.

Options includes deletion, link(linux), interactive deletion, ignore pattern and deletion pattern.

## Getting Started

```shell
go mod tidy
go build
go install
```

### Executing program

* How to run the program
* Step-by-step bullets

```shell
$ ./godedup.exe -h
Usage of C:\dev\src\projects\godedup\godedup.exe:
  -S    Silent (no output)
  -f    force relink (even with already linked files)
  -fromFile string
        compare items list from this file
  -ignore string
        ignore file path regexp
  -it
        interactive deletion
  -k    use kryptographic hash ( blake2 instead of xxhash )
  -link
        rm and link
  -maxsize int
        maximal file size(Mo) (default 650)
  -minsize int
        minimal file size (default 4096)
  -path string
        path to dedup (default "/tmp,/dev/null")
  -rm string
        rm regexp (default "%d")
  -toFile string
        output duplicates files in this file

```
