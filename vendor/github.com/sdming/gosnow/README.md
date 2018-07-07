gosnow
======

gosnow is a snowflake implementation in golang.

This is a fork of the upstream that replaces the the usage of panic() by returning errors instead.

~~~golang
package main

import (
    "github.com/sdming/gosnow"
    "fmt"
)

func main() {

    v, err := gosnow.Default()
    
    // Alternatively you can set the worker id if you are running multiple snowflakes
    // v, err := gosnow.NewSnowFlake(100)
    
    for i := 0; i < 10; i++ {
        id, err := v.Next()
        fmt.Println(id)
    }
}

~~~