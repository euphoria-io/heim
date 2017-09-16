/*
github.com/twitter/snowflake in golang
*/

package main

import (
	"github.com/sdming/gosnow"
	"fmt"
)

func main() {

	v := gosnow.Default()
	//v := gosnow.NewSnowFlake(100)
	for i := 0; i < 10; i++ {
		id := v.Next()
		fmt.Println(id)
	}
}
