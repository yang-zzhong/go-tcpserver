package server

// package:
// \r\n[]byte\r\n[]byte

import (
	"bufio"
	"net"
	"strconv"
)

const (
	STAT_START = iota
	STAT_HEADER_BEGIN
	STAT_HEADER_END
	STAT_HEADER
	STAT_BODY
	STAT_END
	STAT_QUIT
)

type MsgHandler func([]byte, net.Conn) bool

type Package struct {
	conn    net.Conn
	stat    int
	Reading bool
}

func NewPackage(conn net.Conn) *Package {
	return &Package{conn, STAT_START, false}
}

func (p *Package) Write(pack []byte) error {
	l := []byte(strconv.Itoa(len(pack)))
	msg := append([]byte("\r\n"), l...)
	msg = append(msg, []byte("\r\n")...)
	msg = append(msg, pack...)
	_, err := p.conn.Write(msg)
	return err
}

func (p *Package) EndPoping() {
	p.Reading = false
}

func (p *Package) Poping(handler MsgHandler) (err error) {
	nb := make(chan byte)
	done := make(chan bool)
	lens := []byte{}
	bs := []byte{}
	var l int
	p.Reading = true
	go func() {
		defer func() {
			if e := recover(); e != nil {
				err = e.(error)
				done <- true
			}
		}()
		reader := bufio.NewReader(p.conn)
		for {
			b, err := reader.ReadByte()
			if err != nil {
				panic(err)
			}
			if !p.Reading {
				done <- true
				break
			}
			nb <- b
		}
	}()
	for {
		select {
		case <-done:
			return
		case b := <-nb:
			switch p.stat {
			case STAT_START:
				if b != '\r' {
					continue
				}
				p.stat = STAT_HEADER_BEGIN
			case STAT_HEADER_BEGIN:
				if b != '\n' {
					p.stat = STAT_START
					continue
				}
				p.stat = STAT_HEADER
			case STAT_HEADER:
				if b == '\r' {
					p.stat = STAT_HEADER_END
					continue
				}
				lens = append(lens, b)
			case STAT_HEADER_END:
				if b != '\n' {
					p.stat = STAT_START
					continue
				}
				l, _ = strconv.Atoi(string(lens))
				lens = lens[:0]
				p.stat = STAT_BODY
			case STAT_BODY:
				bs = append(bs, b)
				if len(bs) == l-1 {
					p.stat = STAT_END
				}
			case STAT_END:
				bs = append(bs, b)
				if handler(bs, p.conn) {
					p.stat = STAT_QUIT
					break
				}
				bs = bs[:0]
				p.stat = STAT_START
			}
			if p.stat == STAT_QUIT {
				break
			}
		}
	}
	close(nb)
	close(done)

	return
}
