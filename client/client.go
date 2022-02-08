package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	send, recv := make(chan bool), make(chan bool)
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	go func(s chan bool) {
		for {
			reader := bufio.NewReader(os.Stdin)
			send, _, err := reader.ReadLine()
			if err != nil {
				fmt.Println(err)
				return
			}
			_, err = conn.Write(send)
			if err != nil {
				fmt.Println(err)
				return
			}
			s <- true
		}
	}(send)
	go func(r chan bool) {
		for {
			recv := make([]byte, 1024)
			cnt, err := conn.Read(recv)
			if err != nil {
				if err.Error() == "EOF" {
					_ = conn.Close()
					fmt.Println("超时下线")
					os.Exit(0)
				}
				fmt.Println(err)
				return
			}
			fmt.Println(string(recv[:cnt]))
			r <- true
		}
	}(recv)
	for {
		select {
		case <-send:
		case <-recv:
		}
	}
}
