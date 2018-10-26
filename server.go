package server

import (
	"log"
	"net"
)

type Server struct {
	newConn chan net.Conn
	badConn chan net.Conn
	conns   map[net.Conn]*ConnHandler
	done    chan bool
}

type MessageHandler interface {
	OnConn(conn net.Conn)
	OnMsg(msg []byte, conn net.Conn, h *ConnHandler) bool
	OnBad(conn net.Conn)
}

type ConnHandler struct {
	conn net.Conn
	pack *Package
}

func NewConnHandler(conn net.Conn) *ConnHandler {
	return &ConnHandler{conn, NewPackage(conn)}
}

func (handler *ConnHandler) Reading(h MsgHandler) error {
	return handler.pack.Poping(h)
}

func (handler *ConnHandler) Terminate() {
	if handler.pack.Reading {
		handler.pack.EndPoping()
	}
}

func (handler *ConnHandler) Write(bs []byte) error {
	return handler.pack.Write(bs)
}

func NewServer() *Server {
	return &Server{
		make(chan net.Conn),
		make(chan net.Conn),
		make(map[net.Conn]*ConnHandler),
		make(chan bool)}
}

func (s *Server) Close() {
	log.Print("退出咯")
	go func() {
		s.done <- true
	}()
}

func (s *Server) Handler(conn net.Conn) *ConnHandler {
	return s.conns[conn]
}

func (s *Server) BadConn(conn net.Conn) {
	s.badConn <- conn
}

func (s *Server) Listen(port string, handler MessageHandler) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	go s.accept(listener)
	s.selectConn(handler)
}

func (s *Server) accept(listener net.Listener) {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
		}
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		s.newConn <- conn
	}
}

func (s *Server) selectConn(handler MessageHandler) {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
		}
	}()
	for {
		select {
		case conn := <-s.newConn:
			s.conns[conn] = NewConnHandler(conn)
			handler.OnConn(conn)
			go s.read(conn, handler)
		case conn := <-s.badConn:
			s.conns[conn].Terminate()
			_ = conn.Close()
			handler.OnBad(conn)
			delete(s.conns, conn)
		case <-s.done:
			return
		}
	}
}

func (s *Server) read(conn net.Conn, handler MessageHandler) {
	err := s.conns[conn].Reading(func(input []byte, conn net.Conn) bool {
		return handler.OnMsg(input, conn, s.conns[conn])
	})
	if err != nil {
		log.Print(err)
		s.badConn <- conn
	}
}
