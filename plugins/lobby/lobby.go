package main

import (
    "fmt"
    "github.com/go-hive/hive"
)

type Lobby struct {
}

func (l *Lobby) On(c *hive.Context) {
    fmt.Printf("recv: [%s] data:[%v]", c.Name(), c.Data())
    c.Reply(200, []byte(fmt.Sprintf("response[%s]", c.Data())))
}

func main() {
    lobby := &Lobby{}
    err := hive.RegisterPlugin(lobby, "lobby", nil, nil)
    if err != nil {
        return
    }
}
