package main

import (
    "fmt"
    "github.com/go-hive/hive"
)

func main() {
    lobby := hive.StartPlugin("lobby", "plugins/lobby/lobby", nil)
    code, resp, err := lobby.Invoke("hello", []byte("world"))
    if err != nil {
        panic(err)
    }
    fmt.Printf("%d %s\n", code, string(resp))
}
