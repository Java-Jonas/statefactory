package integrationtest

import (
	"context"
	"fmt"
	"log"

	"github.com/jobergner/backent-cli/integrationtest/state"

	// "os"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func dialServer(serverResponseChannel chan state.Message) (*websocket.Conn, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	c, _, err := websocket.Dial(ctx, "http://localhost:3496/ws", nil)
	if err != nil {
		panic(err)
	}

	go runReadMessages(serverResponseChannel, c, ctx)

	return c, ctx, cancel
}

func runReadMessages(serverResponseChannel chan state.Message, conn *websocket.Conn, ctx context.Context) {

	defer fmt.Println("client discontinued")

	for {
		_, serverResponseBytes, err := conn.Read(ctx)
		if err != nil {
			panic(err)
		}

		var serverResponse state.Message
		err = serverResponse.UnmarshalJSON(serverResponseBytes)
		if err != nil {
			panic(err)
		}

		select {
		case serverResponseChannel <- serverResponse:
		default:
			log.Println("serverResponseChannel full")
		}
	}
}

func runSendMessage(ctx context.Context, con *websocket.Conn) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			break
		}
	}
}

func sendActionMovePlayer(ctx context.Context, c *websocket.Conn) {
	params := state.MovePlayerParams{
		ChangeX: 1,
	}
	b, err := params.MarshalJSON()
	if err != nil {
		panic(err)
	}
	msg := state.Message{
		Kind:    state.MessageKindAction_movePlayer,
		Content: b,
	}
	err = wsjson.Write(ctx, c, msg)
	if err != nil {
		panic(err)
	}
}

func sendActionAddItemToPlayer(ctx context.Context, c *websocket.Conn) {
	params := state.AddItemToPlayerParams{
		Item:    state.ItemID(0),
		NewName: "myItem",
	}
	b, err := params.MarshalJSON()
	if err != nil {
		panic(err)
	}
	msg := state.Message{
		Kind:    state.MessageKindAction_addItemToPlayer,
		Content: b,
	}
	err = wsjson.Write(ctx, c, msg)
	if err != nil {
		panic(err)
	}
}

func sendActionUnknownKind(ctx context.Context, c *websocket.Conn) {
	msg := state.Message{
		Kind: "whoami",
	}
	err := wsjson.Write(ctx, c, msg)
	if err != nil {
		panic(err)
	}
}

func sendActionBadContent(ctx context.Context, c *websocket.Conn) {
	msg := state.Message{
		Kind:    state.MessageKindAction_movePlayer,
		Content: []byte(`{ badcontent123# "playerID": 0, "changeX": 1, "changeY": 0}`),
	}
	err := wsjson.Write(ctx, c, msg)
	if err != nil {
		panic(err)
	}
}

func sendBadAction(ctx context.Context, c *websocket.Conn) {
	err := wsjson.Write(ctx, c, "foo bar")
	if err != nil {
		panic(err)
	}
}
