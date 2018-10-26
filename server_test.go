package server

import (
	"log"
	"net"
	"testing"
	"time"
)

type Handler struct{}

var s *Server

func init() {
	s = NewServer()
	go s.Listen(":7777", new(Handler))
}

func (h *Handler) OnMsg(input []byte, conn net.Conn, connHandler *ConnHandler) bool {
	log.Print(string(input))
	s.BadConn(conn)
	return true
}

func (h *Handler) OnBad(conn net.Conn) {
	log.Print("bad conn")
}

func (h *Handler) OnConn(conn net.Conn) {
	log.Print("conn coming")
}

func TestServer(t *testing.T) {
	time.Sleep(1 * time.Second)
	go conn()
	time.Sleep(1 * time.Second)
	s.Close()
}

func conn() {
	if conn, err := net.Dial("tcp", "127.0.0.1:7777"); err != nil {
		panic("连接出错")
	} else {
		handler := NewConnHandler(conn)
		handler.Write([]byte("hello world"))
	}
}
