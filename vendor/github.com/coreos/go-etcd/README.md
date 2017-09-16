# go-etcd

[![GoDoc](https://godoc.org/github.com/coreos/go-etcd/etcd?status.png)](https://godoc.org/github.com/coreos/go-etcd/etcd)

## Usage

The current version of go-etcd supports etcd v2.0+, if you need support for etcd v0.4 please use go-etcd from the [release-0.4](https://github.com/coreos/go-etcd/tree/release-0.4) branch.

```
package main

import (
    "log"

    "github.com/coreos/go-etcd/etcd"
)

func main() {
    machines := []string{"http://127.0.0.1:2379"}
    client := etcd.NewClient(machines)

    if _, err := client.Set("/foo", "bar", 0); err != nil {
        log.Fatal(err)
    }
}
```

## Install

```bash
go get github.com/coreos/go-etcd/etcd
```

## License

See LICENSE file.
