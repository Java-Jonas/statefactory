package main

import (
	"context"
	"net/http"
	"nhooyr.io/websocket"
	"time"
)

type Connector interface {
	Close()
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType []byte) error
}

type Connection struct {
	Conn          *websocket.Conn
	ctx           context.Context
	cancelContext context.CancelFunc
}

func NewConnection(conn *websocket.Conn, r *http.Request) *Connection {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*1000)
	return &Connection{
		Conn:          conn,
		ctx:           ctx,
		cancelContext: cancel,
	}
}

func (c *Connection) Close() {
	c.Conn.Close(websocket.StatusNormalClosure, "")
	c.cancelContext()
}

func (c *Connection) ReadMessage() (int, []byte, error) {
	msgType, msg, err := c.Conn.Read(c.ctx)
	return int(msgType), msg, err
}

func (c *Connection) WriteMessage(message []byte) error {
	err := c.Conn.Write(c.ctx, websocket.MessageText, message)
	if err != nil {
		return err
	}
	return nil
}
