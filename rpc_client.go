package main

import (
	"fmt"
	"log"
	"net/rpc"
)

func main() {
	c, err := rpc.Dial("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	var num int
	rpcVal := 1234
	if err := c.Call("TestRPCRcvr.Test", rpcVal, &num); err != nil {
		log.Fatal(err)
	}

	fmt.Println(num)
}
