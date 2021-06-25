// this file was generated by https://github.com/Java-Jonas/decltostring

package serverfactory

const gets_generated_go_import string = `import (
	"fmt"
	"net/http"
)`

const messageKindAction_addItemToPlayer_type string = `const (
	messageKindAction_addItemToPlayer	messageKind	= 1
	messageKindAction_movePlayer		messageKind	= 2
	messageKindAction_spawnZoneItems	messageKind	= 3
)`

const _MovePlayerParams_type string = `type MovePlayerParams struct {
	ChangeX	float64		` + "`" + `json:"changeX"` + "`" + `
	ChangeY	float64		` + "`" + `json:"changeY"` + "`" + `
	Player	PlayerID	` + "`" + `json:"player"` + "`" + `
}`

const _AddItemToPlayerParams_type string = `type AddItemToPlayerParams struct {
	Item	ItemID	` + "`" + `json:"item"` + "`" + `
	NewName	string	` + "`" + `json:"newName"` + "`" + `
}`

const _SpawnZoneItemsParams_type string = `type SpawnZoneItemsParams struct {
	Items []ItemID ` + "`" + `json:"items"` + "`" + `
}`

const _AddItemToPlayerResponse_type string = `type AddItemToPlayerResponse struct {
	PlayerPath string ` + "`" + `json:"playerPath"` + "`" + `
}`

const _SpawnZoneItemsResponse_type string = `type SpawnZoneItemsResponse struct {
	NewZoneItemPaths []string ` + "`" + `json:"newZoneItemPaths"` + "`" + `
}`

const actions_type string = `type actions struct {
	addItemToPlayer	func(AddItemToPlayerParams, *Engine) AddItemToPlayerResponse
	movePlayer	func(MovePlayerParams, *Engine)
	spawnZoneItems	func(SpawnZoneItemsParams, *Engine) SpawnZoneItemsResponse
}`

const processClientMessage_Room_func string = `func (r *Room) processClientMessage(msg message) (message, error) {
	switch messageKind(msg.Kind) {
	case messageKindAction_addItemToPlayer:
		var params AddItemToPlayerParams
		err := params.UnmarshalJSON(msg.Content)
		if err != nil {
			return message{}, err
		}
		res := r.actions.addItemToPlayer(params, r.state)
		resContent, err := res.MarshalJSON()
		if err != nil {
			return message{}, err
		}
		return message{msg.Kind, resContent, msg.client}, nil
	case messageKindAction_movePlayer:
		var params MovePlayerParams
		err := params.UnmarshalJSON(msg.Content)
		if err != nil {
			return message{}, err
		}
		r.actions.movePlayer(params, r.state)
		return message{}, nil
	case messageKindAction_spawnZoneItems:
		var params SpawnZoneItemsParams
		err := params.UnmarshalJSON(msg.Content)
		if err != nil {
			return message{}, err
		}
		res := r.actions.spawnZoneItems(params, r.state)
		resContent, err := res.MarshalJSON()
		if err != nil {
			return message{}, err
		}
		return message{msg.Kind, resContent, msg.client}, nil
	default:
		return message{}, fmt.Errorf("unknown message kind in: %s", printMessage(msg))
	}
}`

const _Start_func string = `func Start(addItemToPlayer func(AddItemToPlayerParams, *Engine) AddItemToPlayerResponse, movePlayer func(MovePlayerParams, *Engine), spawnZoneItems func(SpawnZoneItemsParams, *Engine) SpawnZoneItemsResponse, onDeploy func(*Engine), onFrameTick func(*Engine)) error {
	a := actions{addItemToPlayer, movePlayer, spawnZoneItems}
	setupRoutes(a, onDeploy, onFrameTick)
	err := http.ListenAndServe(":8080", nil)
	return err
}`
