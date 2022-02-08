package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type user struct {
	id   string
	name string
	msg  chan string
}

var (
	informPipe = make(chan string)
	allUser    = make(map[string]user)
	lock       = sync.RWMutex{}
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	go listenInform()
	for {
		fmt.Println("监听中~")
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		userAddr := conn.RemoteAddr().String()
		newUser := user{
			id:   userAddr,
			name: userAddr,
			msg:  make(chan string),
		}
		lock.Lock()
		allUser[newUser.id] = newUser
		lock.Unlock()
		informPipe <- fmt.Sprintf("用户[%s]上线了", newUser.name)
		go handler(conn, newUser)
		go listenUser(conn, newUser)
	}
}

func listenUser(conn net.Conn, user user) {
	for info := range user.msg {
		_, err := conn.Write([]byte(info))
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func handler(conn net.Conn, user user) {
	defer fmt.Printf("%s 下线了\n", user.name)
	var logOut, waitOut = make(chan bool), make(chan bool)
	go checkLog(conn, &user, logOut, waitOut)
	for {
		accept := make([]byte, 1024)
		cnt, err := conn.Read(accept)
		if cnt == 0 {
			if strings.Contains(err.Error(), "remote") {
				logOut <- true
			}
			return
		}
		acceptString := string(accept[:cnt])
		fmt.Println("accept: ", acceptString)
		if len(acceptString) == 4 && acceptString == "\\who" {
			var userSend []string
			lock.Lock()
			for _, u := range allUser {
				userSend = append(userSend, u.name)
			}
			lock.Unlock()
			user.msg <- strings.Join(userSend, "\n")
		} else if len(acceptString) > 8 && acceptString[:8] == "\\rename|" {
			rename := strings.Split(acceptString, "|")[1]
			user.name = rename
			lock.Lock()
			allUser[user.id] = user
			lock.Unlock()
			fmt.Printf("用户%s, 修改名称成功: %s\n", user.id, rename)
			user.msg <- fmt.Sprintf("名字修改成功: %s", rename)
		} else if len(acceptString) > 8 && strings.HasPrefix(acceptString, "\\sendTo:") {
			infoMsg := strings.Split(strings.Replace(acceptString, ":", "|", -1), "|")
			lock.Lock()
			for k, v := range allUser {
				if v.name == infoMsg[1] {
					allUser[k].msg <- infoMsg[2]
				}
			}
			lock.Unlock()
		}
		waitOut <- true
	}
}

func listenInform() {
	for info := range informPipe {
		lock.Lock()
		for _, user := range allUser {
			user.msg <- info
		}
		lock.Unlock()
	}
}

func checkLog(conn net.Conn, user *user, flag, wait chan bool) {
	for {
		select {
		case <-flag:
			_ = conn.Close()
			delete(allUser, user.id)
			informPipe <- fmt.Sprintf("%s 主动下线了\n", (*user).name)
			return
		case <-time.After(30 * time.Second):
			_ = conn.Close()
			delete(allUser, user.id)
			informPipe <- fmt.Sprintf("%s 超时下线了\n", (*user).name)
			return
		case <-wait:
			fmt.Printf("%s 时间重置\n", (*user).name)
		}
	}
}
