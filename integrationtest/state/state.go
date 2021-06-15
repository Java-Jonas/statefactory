package state



import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"nhooyr.io/websocket"
	"strconv"
	"time"
)

type Client struct {
	room		*Room
	conn		Connector
	messageChannel	chan []byte
	id		uuid.UUID
}

func newClient(websocketConnector Connector) (*Client, error) {
	clientID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	c := Client{conn: websocketConnector, messageChannel: make(chan []byte, 32), id: clientID}
	return &c, nil
}
func (c *Client) discontinue() {
	c.room.unregisterChannel <- c
	c.conn.Close()
}
func (c *Client) assignToRoom(room *Room) {
	c.room = room
}
func (c *Client) forwardToRoom(msg message) {
	select {
	case c.room.clientMessageChannel <- msg:
	default:
		log.Println("message dropped")
	}
}
func (c *Client) runReadMessages() {
	defer c.discontinue()
	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		var msg message
		err = msg.UnmarshalJSON(msgBytes)
		if err != nil {
			log.Println(err)
		}
		c.forwardToRoom(msg)
	}
}
func (c *Client) runWriteMessages() {
	defer c.discontinue()
	for {
		msg, ok := <-c.messageChannel
		if !ok {
			return
		}
		c.conn.WriteMessage(msg)
	}
}

type Connector interface {
	Close()
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType []byte) error
}
type Connection struct {
	Conn		*websocket.Conn
	ctx		context.Context
	cancelContext	context.CancelFunc
}

func NewConnection(conn *websocket.Conn, r *http.Request) *Connection {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	return &Connection{Conn: conn, ctx: ctx, cancelContext: cancel}
}
func (c *Connection) Close() {
	c.Conn.Close(websocket.StatusNormalClosure, "")
	c.cancelContext()
}
func (c *Connection) ReadMessage() (int, []byte, error) {
	msgType, msg, err := c.Conn.Read(c.ctx)
	return int(msgType), msg, err
}
func (c *Connection) WriteMessage(msg []byte) error {
	err := c.Conn.Write(c.ctx, websocket.MessageText, msg)
	if err != nil {
		return err
	}
	return nil
}
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}
func wsEndpoint(w http.ResponseWriter, r *http.Request, room *Room) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	websocketConnection, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		log.Println(err)
		return
	}
	c, err := newClient(NewConnection(websocketConnection, r))
	if err != nil {
		log.Println(err)
		return
	}
	c.assignToRoom(room)
	room.registerChannel <- c
	go c.runReadMessages()
	go c.runWriteMessages()
	<-r.Context().Done()
}
func setupRoutes(a actions, onDeploy func(*Engine), onFrameTick func(*Engine)) {
	room := newRoom(a, onDeploy, onFrameTick)
	room.Deploy()
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsEndpoint(w, r, room)
	})
}

type messageKind int
type message struct {
	Kind	messageKind	`json:"kind"`
	Content	[]byte		`json:"content"`
}
type Room struct {
	clients			map[*Client]bool
	clientMessageChannel	chan message
	registerChannel		chan *Client
	unregisterChannel	chan *Client
	incomingClients		map[*Client]bool
	state			*Engine
	actions			actions
	onDeploy		func(*Engine)
	onFrameTick		func(*Engine)
}

func newRoom(a actions, onDeploy func(*Engine), onFrameTick func(*Engine)) *Room {
	return &Room{clients: make(map[*Client]bool), clientMessageChannel: make(chan message, 264), registerChannel: make(chan *Client), unregisterChannel: make(chan *Client), incomingClients: make(map[*Client]bool), state: newEngine(), onDeploy: onDeploy, onFrameTick: onFrameTick, actions: a}
}
func (r *Room) registerClient(client *Client) {
	r.incomingClients[client] = true
}
func (r *Room) unregisterClient(client *Client) {
	if _, ok := r.clients[client]; ok {
		close(client.messageChannel)
		delete(r.clients, client)
		delete(r.incomingClients, client)
	}
}
func (r *Room) broadcastPatchToClients(patchBytes []byte) error {
	for client := range r.clients {
		select {
		case client.messageChannel <- patchBytes:
		default:
			r.unregisterClient(client)
			log.Println("client dropped")
		}
	}
	return nil
}
func (r *Room) runHandleConnections() {
	for {
		select {
		case client := <-r.registerChannel:
			r.registerClient(client)
		case client := <-r.unregisterChannel:
			r.unregisterClient(client)
		}
	}
}
func (r *Room) answerInitRequests() error {
	tree := r.state.assembleTree(true)
	stateBytes, err := tree.MarshalJSON()
	if err != nil {
		return err
	}
	for client := range r.incomingClients {
		select {
		case client.messageChannel <- stateBytes:
		default:
			r.unregisterClient(client)
			log.Println("client dropped")
		}
	}
	return nil
}
func (r *Room) promoteIncomingClients() {
	for client := range r.incomingClients {
		r.clients[client] = true
		delete(r.incomingClients, client)
	}
}
func (r *Room) processFrame() error {
Exit:
	for {
		select {
		case msg := <-r.clientMessageChannel:
			err := r.processClientMessage(msg)
			if err != nil {
				return err
			}
		default:
			break Exit
		}
	}
	r.onFrameTick(r.state)
	return nil
}
func (r *Room) publishPatch() error {
	r.state.walkTree()
	tree := r.state.assembleTree(false)
	patchBytes, err := tree.MarshalJSON()
	if err != nil {
		return err
	}
	err = r.broadcastPatchToClients(patchBytes)
	if err != nil {
		return err
	}
	return nil
}
func (r *Room) handleIncomingClients() error {
	err := r.answerInitRequests()
	if err != nil {
		return err
	}
	r.promoteIncomingClients()
	return nil
}
func (r *Room) runProcessingFrames() {
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
		err := r.processFrame()
		if err != nil {
			log.Println(err)
		}
		err = r.publishPatch()
		if err != nil {
			log.Println(err)
		}
		r.state.UpdateState()
		err = r.handleIncomingClients()
		if err != nil {
			log.Println(err)
		}
	}
}
func (r *Room) Deploy() {
	r.onDeploy(r.state)
	go r.runHandleConnections()
	go r.runProcessingFrames()
}

func (_equipmentSet equipmentSet) AddEquipment(itemID ItemID) {
	equipmentSet := _equipmentSet.equipmentSet.engine.EquipmentSet(_equipmentSet.equipmentSet.ID)
	if equipmentSet.equipmentSet.OperationKind == OperationKindDelete {
		return
	}
	if equipmentSet.equipmentSet.engine.Item(itemID).item.OperationKind == OperationKindDelete {
		return
	}
	ref := equipmentSet.equipmentSet.engine.createEquipmentSetEquipmentRef(itemID, equipmentSet.equipmentSet.ID)
	equipmentSet.equipmentSet.Equipment = append(equipmentSet.equipmentSet.Equipment, ref.ID)
	equipmentSet.equipmentSet.OperationKind = OperationKindUpdate
	equipmentSet.equipmentSet.engine.Patch.EquipmentSet[equipmentSet.equipmentSet.ID] = equipmentSet.equipmentSet
}
func (_player player) AddEquipmentSet(equipmentSetID EquipmentSetID) {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return
	}
	if player.player.engine.EquipmentSet(equipmentSetID).equipmentSet.OperationKind == OperationKindDelete {
		return
	}
	ref := player.player.engine.createPlayerEquipmentSetRef(equipmentSetID, player.player.ID)
	player.player.EquipmentSets = append(player.player.EquipmentSets, ref.ID)
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
}
func (_player player) AddGuildMember(playerID PlayerID) {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return
	}
	if player.player.engine.Player(playerID).player.OperationKind == OperationKindDelete {
		return
	}
	ref := player.player.engine.createPlayerGuildMemberRef(playerID, player.player.ID)
	player.player.GuildMembers = append(player.player.GuildMembers, ref.ID)
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
}
func (_player player) AddItem() item {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return item{item: itemCore{OperationKind: OperationKindDelete}}
	}
	item := player.player.engine.createItem(true)
	player.player.Items = append(player.player.Items, item.item.ID)
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return item
}
func (_player player) AddTargetedByPlayer(playerID PlayerID) {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return
	}
	if player.player.engine.Player(playerID).player.OperationKind == OperationKindDelete {
		return
	}
	anyContainer := player.player.engine.createAnyOfPlayer_ZoneItem(false).anyOfPlayer_ZoneItem
	anyContainer.setPlayer(playerID)
	ref := player.player.engine.createPlayerTargetedByRef(anyContainer.ID, player.player.ID)
	player.player.TargetedBy = append(player.player.TargetedBy, ref.ID)
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
}
func (_player player) AddTargetedByZoneItem(zoneItemID ZoneItemID) {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return
	}
	if player.player.engine.ZoneItem(zoneItemID).zoneItem.OperationKind == OperationKindDelete {
		return
	}
	anyContainer := player.player.engine.createAnyOfPlayer_ZoneItem(false).anyOfPlayer_ZoneItem
	anyContainer.setZoneItem(zoneItemID)
	ref := player.player.engine.createPlayerTargetedByRef(anyContainer.ID, player.player.ID)
	player.player.TargetedBy = append(player.player.TargetedBy, ref.ID)
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
}
func (_zone zone) AddInteractableItem() item {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return item{item: itemCore{OperationKind: OperationKindDelete}}
	}
	item := zone.zone.engine.createItem(true)
	anyContainer := zone.zone.engine.createAnyOfItem_Player_ZoneItem(false).anyOfItem_Player_ZoneItem
	anyContainer.setItem(item.item.ID)
	zone.zone.Interactables = append(zone.zone.Interactables, anyContainer.ID)
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return item
}
func (_zone zone) AddInteractablePlayer() player {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return player{player: playerCore{OperationKind: OperationKindDelete}}
	}
	player := zone.zone.engine.createPlayer(true)
	anyContainer := zone.zone.engine.createAnyOfItem_Player_ZoneItem(false).anyOfItem_Player_ZoneItem
	anyContainer.setPlayer(player.player.ID)
	zone.zone.Interactables = append(zone.zone.Interactables, anyContainer.ID)
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return player
}
func (_zone zone) AddInteractableZoneItem() zoneItem {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zoneItem{zoneItem: zoneItemCore{OperationKind: OperationKindDelete}}
	}
	zoneItem := zone.zone.engine.createZoneItem(true)
	anyContainer := zone.zone.engine.createAnyOfItem_Player_ZoneItem(false).anyOfItem_Player_ZoneItem
	anyContainer.setZoneItem(zoneItem.zoneItem.ID)
	zone.zone.Interactables = append(zone.zone.Interactables, anyContainer.ID)
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zoneItem
}
func (_zone zone) AddItem() zoneItem {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zoneItem{zoneItem: zoneItemCore{OperationKind: OperationKindDelete}}
	}
	zoneItem := zone.zone.engine.createZoneItem(true)
	zone.zone.Items = append(zone.zone.Items, zoneItem.zoneItem.ID)
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zoneItem
}
func (_zone zone) AddPlayer() player {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return player{player: playerCore{OperationKind: OperationKindDelete}}
	}
	player := zone.zone.engine.createPlayer(true)
	zone.zone.Players = append(zone.zone.Players, player.player.ID)
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return player
}
func (_zone zone) AddTags(tags ...string) {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return
	}
	zone.zone.Tags = append(zone.zone.Tags, tags...)
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
}
func (_any anyOfPlayer_Position) Kind() ElementKind {
	any := _any.anyOfPlayer_Position.engine.anyOfPlayer_Position(_any.anyOfPlayer_Position.ID)
	return any.anyOfPlayer_Position.ElementKind
}
func (_any anyOfPlayer_Position) SetPlayer() player {
	player := _any.anyOfPlayer_Position.engine.createPlayer(true)
	_any.anyOfPlayer_Position.setPlayer(player.ID())
	return player
}
func (_any anyOfPlayer_PositionCore) setPlayer(playerID PlayerID) {
	any := _any.engine.anyOfPlayer_Position(_any.ID).anyOfPlayer_Position
	if any.Position != 0 {
		any.engine.deletePosition(any.Position)
		any.Position = 0
	}
	any.ElementKind = ElementKindPlayer
	any.Player = playerID
	any.engine.Patch.AnyOfPlayer_Position[any.ID] = any
}
func (_any anyOfPlayer_Position) SetPosition() position {
	position := _any.anyOfPlayer_Position.engine.createPosition(true)
	_any.anyOfPlayer_Position.setPosition(position.ID())
	return position
}
func (_any anyOfPlayer_PositionCore) setPosition(positionID PositionID) {
	any := _any.engine.anyOfPlayer_Position(_any.ID).anyOfPlayer_Position
	if any.Player != 0 {
		any.engine.deletePlayer(any.Player)
		any.Player = 0
	}
	any.ElementKind = ElementKindPosition
	any.Position = positionID
	any.engine.Patch.AnyOfPlayer_Position[any.ID] = any
}
func (_any anyOfPlayer_PositionCore) deleteChild() {
	any := _any.engine.anyOfPlayer_Position(_any.ID).anyOfPlayer_Position
	switch any.ElementKind {
	case ElementKindPlayer:
		any.engine.deletePlayer(any.Player)
	case ElementKindPosition:
		any.engine.deletePosition(any.Position)
	}
}
func (_any anyOfPlayer_ZoneItem) Kind() ElementKind {
	any := _any.anyOfPlayer_ZoneItem.engine.anyOfPlayer_ZoneItem(_any.anyOfPlayer_ZoneItem.ID)
	return any.anyOfPlayer_ZoneItem.ElementKind
}
func (_any anyOfPlayer_ZoneItem) SetPlayer() player {
	player := _any.anyOfPlayer_ZoneItem.engine.createPlayer(true)
	_any.anyOfPlayer_ZoneItem.setPlayer(player.ID())
	return player
}
func (_any anyOfPlayer_ZoneItemCore) setPlayer(playerID PlayerID) {
	any := _any.engine.anyOfPlayer_ZoneItem(_any.ID).anyOfPlayer_ZoneItem
	if any.ZoneItem != 0 {
		any.engine.deleteZoneItem(any.ZoneItem)
		any.ZoneItem = 0
	}
	any.ElementKind = ElementKindPlayer
	any.Player = playerID
	any.engine.Patch.AnyOfPlayer_ZoneItem[any.ID] = any
}
func (_any anyOfPlayer_ZoneItem) SetZoneItem() zoneItem {
	zoneItem := _any.anyOfPlayer_ZoneItem.engine.createZoneItem(true)
	_any.anyOfPlayer_ZoneItem.setZoneItem(zoneItem.ID())
	return zoneItem
}
func (_any anyOfPlayer_ZoneItemCore) setZoneItem(zoneItemID ZoneItemID) {
	any := _any.engine.anyOfPlayer_ZoneItem(_any.ID).anyOfPlayer_ZoneItem
	if any.Player != 0 {
		any.engine.deletePlayer(any.Player)
		any.Player = 0
	}
	any.ElementKind = ElementKindZoneItem
	any.ZoneItem = zoneItemID
	any.engine.Patch.AnyOfPlayer_ZoneItem[any.ID] = any
}
func (_any anyOfPlayer_ZoneItemCore) deleteChild() {
	any := _any.engine.anyOfPlayer_ZoneItem(_any.ID).anyOfPlayer_ZoneItem
	switch any.ElementKind {
	case ElementKindPlayer:
		any.engine.deletePlayer(any.Player)
	case ElementKindZoneItem:
		any.engine.deleteZoneItem(any.ZoneItem)
	}
}
func (_any anyOfItem_Player_ZoneItem) Kind() ElementKind {
	any := _any.anyOfItem_Player_ZoneItem.engine.anyOfItem_Player_ZoneItem(_any.anyOfItem_Player_ZoneItem.ID)
	return any.anyOfItem_Player_ZoneItem.ElementKind
}
func (_any anyOfItem_Player_ZoneItem) SetItem() item {
	item := _any.anyOfItem_Player_ZoneItem.engine.createItem(true)
	_any.anyOfItem_Player_ZoneItem.setItem(item.ID())
	return item
}
func (_any anyOfItem_Player_ZoneItemCore) setItem(itemID ItemID) {
	any := _any.engine.anyOfItem_Player_ZoneItem(_any.ID).anyOfItem_Player_ZoneItem
	if any.Player != 0 {
		any.engine.deletePlayer(any.Player)
		any.Player = 0
	}
	if any.ZoneItem != 0 {
		any.engine.deleteZoneItem(any.ZoneItem)
		any.ZoneItem = 0
	}
	any.ElementKind = ElementKindItem
	any.Item = itemID
	any.engine.Patch.AnyOfItem_Player_ZoneItem[any.ID] = any
}
func (_any anyOfItem_Player_ZoneItem) SetPlayer() player {
	player := _any.anyOfItem_Player_ZoneItem.engine.createPlayer(true)
	_any.anyOfItem_Player_ZoneItem.setPlayer(player.ID())
	return player
}
func (_any anyOfItem_Player_ZoneItemCore) setPlayer(playerID PlayerID) {
	any := _any.engine.anyOfItem_Player_ZoneItem(_any.ID).anyOfItem_Player_ZoneItem
	if any.Item != 0 {
		any.engine.deleteItem(any.Item)
		any.Item = 0
	}
	if any.ZoneItem != 0 {
		any.engine.deleteZoneItem(any.ZoneItem)
		any.ZoneItem = 0
	}
	any.ElementKind = ElementKindPlayer
	any.Player = playerID
	any.engine.Patch.AnyOfItem_Player_ZoneItem[any.ID] = any
}
func (_any anyOfItem_Player_ZoneItem) SetZoneItem() zoneItem {
	zoneItem := _any.anyOfItem_Player_ZoneItem.engine.createZoneItem(true)
	_any.anyOfItem_Player_ZoneItem.setZoneItem(zoneItem.ID())
	return zoneItem
}
func (_any anyOfItem_Player_ZoneItemCore) setZoneItem(zoneItemID ZoneItemID) {
	any := _any.engine.anyOfItem_Player_ZoneItem(_any.ID).anyOfItem_Player_ZoneItem
	if any.Item != 0 {
		any.engine.deleteItem(any.Item)
		any.Item = 0
	}
	if any.Player != 0 {
		any.engine.deletePlayer(any.Player)
		any.Player = 0
	}
	any.ElementKind = ElementKindZoneItem
	any.ZoneItem = zoneItemID
	any.engine.Patch.AnyOfItem_Player_ZoneItem[any.ID] = any
}
func (_any anyOfItem_Player_ZoneItemCore) deleteChild() {
	any := _any.engine.anyOfItem_Player_ZoneItem(_any.ID).anyOfItem_Player_ZoneItem
	switch any.ElementKind {
	case ElementKindItem:
		any.engine.deleteItem(any.Item)
	case ElementKindPlayer:
		any.engine.deletePlayer(any.Player)
	case ElementKindZoneItem:
		any.engine.deleteZoneItem(any.ZoneItem)
	}
}

type assembleConfig struct{ forceInclude bool }

func (engine *Engine) assembleTree(assembleEntireTree bool) Tree {
	for key := range engine.Tree.EquipmentSet {
		delete(engine.Tree.EquipmentSet, key)
	}
	for key := range engine.Tree.GearScore {
		delete(engine.Tree.GearScore, key)
	}
	for key := range engine.Tree.Item {
		delete(engine.Tree.Item, key)
	}
	for key := range engine.Tree.Player {
		delete(engine.Tree.Player, key)
	}
	for key := range engine.Tree.Position {
		delete(engine.Tree.Position, key)
	}
	for key := range engine.Tree.Zone {
		delete(engine.Tree.Zone, key)
	}
	for key := range engine.Tree.ZoneItem {
		delete(engine.Tree.ZoneItem, key)
	}
	config := assembleConfig{forceInclude: assembleEntireTree}
	for _, equipmentSetData := range engine.Patch.EquipmentSet {
		equipmentSet, include, _ := engine.assembleEquipmentSet(equipmentSetData.ID, nil, config)
		if include {
			engine.Tree.EquipmentSet[equipmentSetData.ID] = equipmentSet
		}
	}
	for _, gearScoreData := range engine.Patch.GearScore {
		if !gearScoreData.HasParent {
			gearScore, include, _ := engine.assembleGearScore(gearScoreData.ID, nil, config)
			if include {
				engine.Tree.GearScore[gearScoreData.ID] = gearScore
			}
		}
	}
	for _, itemData := range engine.Patch.Item {
		if !itemData.HasParent {
			item, include, _ := engine.assembleItem(itemData.ID, nil, config)
			if include {
				engine.Tree.Item[itemData.ID] = item
			}
		}
	}
	for _, playerData := range engine.Patch.Player {
		if !playerData.HasParent {
			player, include, _ := engine.assemblePlayer(playerData.ID, nil, config)
			if include {
				engine.Tree.Player[playerData.ID] = player
			}
		}
	}
	for _, positionData := range engine.Patch.Position {
		if !positionData.HasParent {
			position, include, _ := engine.assemblePosition(positionData.ID, nil, config)
			if include {
				engine.Tree.Position[positionData.ID] = position
			}
		}
	}
	for _, zoneData := range engine.Patch.Zone {
		zone, include, _ := engine.assembleZone(zoneData.ID, nil, config)
		if include {
			engine.Tree.Zone[zoneData.ID] = zone
		}
	}
	for _, zoneItemData := range engine.Patch.ZoneItem {
		if !zoneItemData.HasParent {
			zoneItem, include, _ := engine.assembleZoneItem(zoneItemData.ID, nil, config)
			if include {
				engine.Tree.ZoneItem[zoneItemData.ID] = zoneItem
			}
		}
	}
	for _, equipmentSetData := range engine.State.EquipmentSet {
		if _, ok := engine.Tree.EquipmentSet[equipmentSetData.ID]; !ok {
			equipmentSet, include, _ := engine.assembleEquipmentSet(equipmentSetData.ID, nil, config)
			if include {
				engine.Tree.EquipmentSet[equipmentSetData.ID] = equipmentSet
			}
		}
	}
	for _, gearScoreData := range engine.State.GearScore {
		if !gearScoreData.HasParent {
			if _, ok := engine.Tree.GearScore[gearScoreData.ID]; !ok {
				gearScore, include, _ := engine.assembleGearScore(gearScoreData.ID, nil, config)
				if include {
					engine.Tree.GearScore[gearScoreData.ID] = gearScore
				}
			}
		}
	}
	for _, itemData := range engine.State.Item {
		if !itemData.HasParent {
			if _, ok := engine.Tree.Item[itemData.ID]; !ok {
				item, include, _ := engine.assembleItem(itemData.ID, nil, config)
				if include {
					engine.Tree.Item[itemData.ID] = item
				}
			}
		}
	}
	for _, playerData := range engine.State.Player {
		if !playerData.HasParent {
			if _, ok := engine.Tree.Player[playerData.ID]; !ok {
				player, include, _ := engine.assemblePlayer(playerData.ID, nil, config)
				if include {
					engine.Tree.Player[playerData.ID] = player
				}
			}
		}
	}
	for _, positionData := range engine.State.Position {
		if !positionData.HasParent {
			if _, ok := engine.Tree.Position[positionData.ID]; !ok {
				position, include, _ := engine.assemblePosition(positionData.ID, nil, config)
				if include {
					engine.Tree.Position[positionData.ID] = position
				}
			}
		}
	}
	for _, zoneData := range engine.State.Zone {
		if _, ok := engine.Tree.Zone[zoneData.ID]; !ok {
			zone, include, _ := engine.assembleZone(zoneData.ID, nil, config)
			if include {
				engine.Tree.Zone[zoneData.ID] = zone
			}
		}
	}
	for _, zoneItemData := range engine.State.ZoneItem {
		if !zoneItemData.HasParent {
			if _, ok := engine.Tree.ZoneItem[zoneItemData.ID]; !ok {
				zoneItem, include, _ := engine.assembleZoneItem(zoneItemData.ID, nil, config)
				if include {
					engine.Tree.ZoneItem[zoneItemData.ID] = zoneItem
				}
			}
		}
	}
	return engine.Tree
}
func (engine *Engine) assembleEquipmentSet(equipmentSetID EquipmentSetID, check *recursionCheck, config assembleConfig) (EquipmentSet, bool, bool) {
	if check != nil {
		if alreadyExists := check.equipmentSet[equipmentSetID]; alreadyExists {
			return EquipmentSet{}, false, false
		} else {
			check.equipmentSet[equipmentSetID] = true
		}
	}
	equipmentSetData, hasUpdated := engine.Patch.EquipmentSet[equipmentSetID]
	if !hasUpdated {
		equipmentSetData = engine.State.EquipmentSet[equipmentSetID]
	}
	var equipmentSet EquipmentSet
	for _, equipmentSetEquipmentRefID := range mergeEquipmentSetEquipmentRefIDs(engine.State.EquipmentSet[equipmentSetData.ID].Equipment, engine.Patch.EquipmentSet[equipmentSetData.ID].Equipment) {
		if treeEquipmentSetEquipmentRef, include, childHasUpdated := engine.assembleEquipmentSetEquipmentRef(equipmentSetEquipmentRefID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			equipmentSet.Equipment = append(equipmentSet.Equipment, treeEquipmentSetEquipmentRef)
		}
	}
	equipmentSet.ID = equipmentSetData.ID
	equipmentSet.OperationKind = equipmentSetData.OperationKind
	equipmentSet.Name = equipmentSetData.Name
	return equipmentSet, hasUpdated || config.forceInclude, hasUpdated
}
func (engine *Engine) assembleGearScore(gearScoreID GearScoreID, check *recursionCheck, config assembleConfig) (GearScore, bool, bool) {
	if check != nil {
		if alreadyExists := check.gearScore[gearScoreID]; alreadyExists {
			return GearScore{}, false, false
		} else {
			check.gearScore[gearScoreID] = true
		}
	}
	gearScoreData, hasUpdated := engine.Patch.GearScore[gearScoreID]
	if !hasUpdated {
		gearScoreData = engine.State.GearScore[gearScoreID]
	}
	var gearScore GearScore
	gearScore.ID = gearScoreData.ID
	gearScore.OperationKind = gearScoreData.OperationKind
	gearScore.Level = gearScoreData.Level
	gearScore.Score = gearScoreData.Score
	return gearScore, hasUpdated || config.forceInclude, hasUpdated
}
func (engine *Engine) assembleItem(itemID ItemID, check *recursionCheck, config assembleConfig) (Item, bool, bool) {
	if check != nil {
		if alreadyExists := check.item[itemID]; alreadyExists {
			return Item{}, false, false
		} else {
			check.item[itemID] = true
		}
	}
	itemData, hasUpdated := engine.Patch.Item[itemID]
	if !hasUpdated {
		itemData = engine.State.Item[itemID]
	}
	var item Item
	if treeItemBoundToRef, include, childHasUpdated := engine.assembleItemBoundToRef(itemID, check, config); include {
		if childHasUpdated {
			hasUpdated = true
		}
		item.BoundTo = treeItemBoundToRef
	}
	if treeGearScore, include, childHasUpdated := engine.assembleGearScore(itemData.GearScore, check, config); include {
		if childHasUpdated {
			hasUpdated = true
		}
		item.GearScore = &treeGearScore
	}
	anyOfPlayer_PositionContainer := engine.anyOfPlayer_Position(itemData.Origin).anyOfPlayer_Position
	if anyOfPlayer_PositionContainer.ElementKind == ElementKindPlayer {
		playerID := anyOfPlayer_PositionContainer.Player
		if treePlayer, include, childHasUpdated := engine.assemblePlayer(playerID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			item.Origin = &treePlayer
		}
	} else if anyOfPlayer_PositionContainer.ElementKind == ElementKindPosition {
		positionID := anyOfPlayer_PositionContainer.Position
		if treePosition, include, childHasUpdated := engine.assemblePosition(positionID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			item.Origin = &treePosition
		}
	}
	item.ID = itemData.ID
	item.OperationKind = itemData.OperationKind
	item.Name = itemData.Name
	return item, hasUpdated || config.forceInclude, hasUpdated
}
func (engine *Engine) assemblePlayer(playerID PlayerID, check *recursionCheck, config assembleConfig) (Player, bool, bool) {
	if check != nil {
		if alreadyExists := check.player[playerID]; alreadyExists {
			return Player{}, false, false
		} else {
			check.player[playerID] = true
		}
	}
	playerData, hasUpdated := engine.Patch.Player[playerID]
	if !hasUpdated {
		playerData = engine.State.Player[playerID]
	}
	var player Player
	for _, playerEquipmentSetRefID := range mergePlayerEquipmentSetRefIDs(engine.State.Player[playerData.ID].EquipmentSets, engine.Patch.Player[playerData.ID].EquipmentSets) {
		if treePlayerEquipmentSetRef, include, childHasUpdated := engine.assemblePlayerEquipmentSetRef(playerEquipmentSetRefID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			player.EquipmentSets = append(player.EquipmentSets, treePlayerEquipmentSetRef)
		}
	}
	if treeGearScore, include, childHasUpdated := engine.assembleGearScore(playerData.GearScore, check, config); include {
		if childHasUpdated {
			hasUpdated = true
		}
		player.GearScore = &treeGearScore
	}
	for _, playerGuildMemberRefID := range mergePlayerGuildMemberRefIDs(engine.State.Player[playerData.ID].GuildMembers, engine.Patch.Player[playerData.ID].GuildMembers) {
		if treePlayerGuildMemberRef, include, childHasUpdated := engine.assemblePlayerGuildMemberRef(playerGuildMemberRefID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			player.GuildMembers = append(player.GuildMembers, treePlayerGuildMemberRef)
		}
	}
	for _, itemID := range mergeItemIDs(engine.State.Player[playerData.ID].Items, engine.Patch.Player[playerData.ID].Items) {
		if treeItem, include, childHasUpdated := engine.assembleItem(itemID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			player.Items = append(player.Items, treeItem)
		}
	}
	if treePosition, include, childHasUpdated := engine.assemblePosition(playerData.Position, check, config); include {
		if childHasUpdated {
			hasUpdated = true
		}
		player.Position = &treePosition
	}
	if treePlayerTargetRef, include, childHasUpdated := engine.assemblePlayerTargetRef(playerID, check, config); include {
		if childHasUpdated {
			hasUpdated = true
		}
		player.Target = treePlayerTargetRef
	}
	for _, playerTargetedByRefID := range mergePlayerTargetedByRefIDs(engine.State.Player[playerData.ID].TargetedBy, engine.Patch.Player[playerData.ID].TargetedBy) {
		if treePlayerTargetedByRef, include, childHasUpdated := engine.assemblePlayerTargetedByRef(playerTargetedByRefID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			player.TargetedBy = append(player.TargetedBy, treePlayerTargetedByRef)
		}
	}
	player.ID = playerData.ID
	player.OperationKind = playerData.OperationKind
	return player, hasUpdated || config.forceInclude, hasUpdated
}
func (engine *Engine) assemblePosition(positionID PositionID, check *recursionCheck, config assembleConfig) (Position, bool, bool) {
	if check != nil {
		if alreadyExists := check.position[positionID]; alreadyExists {
			return Position{}, false, false
		} else {
			check.position[positionID] = true
		}
	}
	positionData, hasUpdated := engine.Patch.Position[positionID]
	if !hasUpdated {
		positionData = engine.State.Position[positionID]
	}
	var position Position
	position.ID = positionData.ID
	position.OperationKind = positionData.OperationKind
	position.X = positionData.X
	position.Y = positionData.Y
	return position, hasUpdated || config.forceInclude, hasUpdated
}
func (engine *Engine) assembleZone(zoneID ZoneID, check *recursionCheck, config assembleConfig) (Zone, bool, bool) {
	if check != nil {
		if alreadyExists := check.zone[zoneID]; alreadyExists {
			return Zone{}, false, false
		} else {
			check.zone[zoneID] = true
		}
	}
	zoneData, hasUpdated := engine.Patch.Zone[zoneID]
	if !hasUpdated {
		zoneData = engine.State.Zone[zoneID]
	}
	var zone Zone
	for _, anyOfItem_Player_ZoneItemID := range mergeAnyOfItem_Player_ZoneItemIDs(engine.State.Zone[zoneData.ID].Interactables, engine.Patch.Zone[zoneData.ID].Interactables) {
		anyOfItem_Player_ZoneItemContainer := engine.anyOfItem_Player_ZoneItem(anyOfItem_Player_ZoneItemID).anyOfItem_Player_ZoneItem
		if anyOfItem_Player_ZoneItemContainer.ElementKind == ElementKindItem {
			itemID := anyOfItem_Player_ZoneItemContainer.Item
			if treeItem, include, childHasUpdated := engine.assembleItem(itemID, check, config); include {
				if childHasUpdated {
					hasUpdated = true
				}
				zone.Interactables = append(zone.Interactables, treeItem)
			}
		} else if anyOfItem_Player_ZoneItemContainer.ElementKind == ElementKindPlayer {
			playerID := anyOfItem_Player_ZoneItemContainer.Player
			if treePlayer, include, childHasUpdated := engine.assemblePlayer(playerID, check, config); include {
				if childHasUpdated {
					hasUpdated = true
				}
				zone.Interactables = append(zone.Interactables, treePlayer)
			}
		} else if anyOfItem_Player_ZoneItemContainer.ElementKind == ElementKindZoneItem {
			zoneItemID := anyOfItem_Player_ZoneItemContainer.ZoneItem
			if treeZoneItem, include, childHasUpdated := engine.assembleZoneItem(zoneItemID, check, config); include {
				if childHasUpdated {
					hasUpdated = true
				}
				zone.Interactables = append(zone.Interactables, treeZoneItem)
			}
		}
	}
	for _, zoneItemID := range mergeZoneItemIDs(engine.State.Zone[zoneData.ID].Items, engine.Patch.Zone[zoneData.ID].Items) {
		if treeZoneItem, include, childHasUpdated := engine.assembleZoneItem(zoneItemID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			zone.Items = append(zone.Items, treeZoneItem)
		}
	}
	for _, playerID := range mergePlayerIDs(engine.State.Zone[zoneData.ID].Players, engine.Patch.Zone[zoneData.ID].Players) {
		if treePlayer, include, childHasUpdated := engine.assemblePlayer(playerID, check, config); include {
			if childHasUpdated {
				hasUpdated = true
			}
			zone.Players = append(zone.Players, treePlayer)
		}
	}
	zone.ID = zoneData.ID
	zone.OperationKind = zoneData.OperationKind
	zone.Tags = zoneData.Tags
	return zone, hasUpdated || config.forceInclude, hasUpdated
}
func (engine *Engine) assembleZoneItem(zoneItemID ZoneItemID, check *recursionCheck, config assembleConfig) (ZoneItem, bool, bool) {
	if check != nil {
		if alreadyExists := check.zoneItem[zoneItemID]; alreadyExists {
			return ZoneItem{}, false, false
		} else {
			check.zoneItem[zoneItemID] = true
		}
	}
	zoneItemData, hasUpdated := engine.Patch.ZoneItem[zoneItemID]
	if !hasUpdated {
		zoneItemData = engine.State.ZoneItem[zoneItemID]
	}
	var zoneItem ZoneItem
	if treeItem, include, childHasUpdated := engine.assembleItem(zoneItemData.Item, check, config); include {
		if childHasUpdated {
			hasUpdated = true
		}
		zoneItem.Item = &treeItem
	}
	if treePosition, include, childHasUpdated := engine.assemblePosition(zoneItemData.Position, check, config); include {
		if childHasUpdated {
			hasUpdated = true
		}
		zoneItem.Position = &treePosition
	}
	zoneItem.ID = zoneItemData.ID
	zoneItem.OperationKind = zoneItemData.OperationKind
	return zoneItem, hasUpdated || config.forceInclude, hasUpdated
}
func (engine *Engine) assembleEquipmentSetEquipmentRef(refID EquipmentSetEquipmentRefID, check *recursionCheck, config assembleConfig) (ItemReference, bool, bool) {
	if config.forceInclude {
		ref := engine.equipmentSetEquipmentRef(refID).equipmentSetEquipmentRef
		referencedElement := engine.Item(ref.ReferencedElementID).item
		if check == nil {
			check = newRecursionCheck()
		}
		referencedDataStatus := ReferencedDataUnchanged
		if _, _, hasUpdatedDownstream := engine.assembleItem(referencedElement.ID, check, config); hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		path, _ := engine.PathTrack.item[referencedElement.ID]
		return ItemReference{ref.OperationKind, ref.ReferencedElementID, ElementKindItem, referencedDataStatus, path.toJSONPath(), nil}, true, ref.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
	}
	if patchRef, hasUpdated := engine.Patch.EquipmentSetEquipmentRef[refID]; hasUpdated {
		if patchRef.OperationKind == OperationKindUpdate {
			config.forceInclude = true
		}
		referencedElement := engine.Item(patchRef.ReferencedElementID).item
		if check == nil {
			check = newRecursionCheck()
		}
		element, _, hasUpdatedDownstream := engine.assembleItem(referencedElement.ID, check, config)
		referencedDataStatus := ReferencedDataUnchanged
		if hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		var el *Item
		if patchRef.OperationKind == OperationKindUpdate {
			el = &element
		}
		path, _ := engine.PathTrack.item[referencedElement.ID]
		return ItemReference{patchRef.OperationKind, patchRef.ReferencedElementID, ElementKindItem, referencedDataStatus, path.toJSONPath(), el}, true, patchRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
	}
	ref := engine.equipmentSetEquipmentRef(refID).equipmentSetEquipmentRef
	if check == nil {
		check = newRecursionCheck()
	}
	if _, _, hasUpdatedDownstream := engine.assembleItem(ref.ReferencedElementID, check, config); hasUpdatedDownstream {
		path, _ := engine.PathTrack.item[ref.ReferencedElementID]
		return ItemReference{OperationKindUnchanged, ref.ReferencedElementID, ElementKindItem, ReferencedDataModified, path.toJSONPath(), nil}, true, true
	}
	return ItemReference{}, false, false
}
func (engine *Engine) assembleItemBoundToRef(itemID ItemID, check *recursionCheck, config assembleConfig) (*PlayerReference, bool, bool) {
	stateItem := engine.State.Item[itemID]
	patchItem, itemIsInPatch := engine.Patch.Item[itemID]
	if stateItem.BoundTo == 0 && (!itemIsInPatch || patchItem.BoundTo == 0) {
		return nil, false, false
	}
	if config.forceInclude {
		ref := engine.itemBoundToRef(patchItem.BoundTo)
		referencedElement := engine.Player(ref.itemBoundToRef.ReferencedElementID).player
		if check == nil {
			check = newRecursionCheck()
		}
		referencedDataStatus := ReferencedDataUnchanged
		if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		path, _ := engine.PathTrack.player[referencedElement.ID]
		return &PlayerReference{ref.itemBoundToRef.OperationKind, referencedElement.ID, ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, ref.itemBoundToRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
	}
	if stateItem.BoundTo == 0 && (itemIsInPatch && patchItem.BoundTo != 0) {
		config.forceInclude = true
		ref := engine.itemBoundToRef(patchItem.BoundTo)
		referencedElement := engine.Player(ref.itemBoundToRef.ReferencedElementID).player
		if check == nil {
			check = newRecursionCheck()
		}
		referencedDataStatus := ReferencedDataUnchanged
		element, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config)
		if hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		path, _ := engine.PathTrack.player[referencedElement.ID]
		return &PlayerReference{OperationKindUpdate, referencedElement.ID, ElementKindPlayer, referencedDataStatus, path.toJSONPath(), &element}, true, referencedDataStatus == ReferencedDataModified
	}
	if stateItem.BoundTo != 0 && (itemIsInPatch && patchItem.BoundTo == 0) {
		ref := engine.itemBoundToRef(stateItem.BoundTo)
		referencedElement := engine.Player(ref.itemBoundToRef.ReferencedElementID).player
		if check == nil {
			check = newRecursionCheck()
		}
		referencedDataStatus := ReferencedDataUnchanged
		if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		path, _ := engine.PathTrack.player[referencedElement.ID]
		return &PlayerReference{OperationKindDelete, referencedElement.ID, ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, referencedDataStatus == ReferencedDataModified
	}
	if stateItem.BoundTo != 0 && (itemIsInPatch && patchItem.BoundTo != 0) {
		if stateItem.BoundTo != patchItem.BoundTo {
			ref := engine.itemBoundToRef(patchItem.BoundTo)
			referencedElement := engine.Player(ref.itemBoundToRef.ReferencedElementID).player
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.player[referencedElement.ID]
			return &PlayerReference{OperationKindUpdate, referencedElement.ID, ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, referencedDataStatus == ReferencedDataModified
		}
	}
	if stateItem.BoundTo != 0 {
		ref := engine.itemBoundToRef(stateItem.BoundTo)
		if check == nil {
			check = newRecursionCheck()
		}
		if _, _, hasUpdatedDownstream := engine.assemblePlayer(ref.ID(), check, config); hasUpdatedDownstream {
			path, _ := engine.PathTrack.player[ref.ID()]
			return &PlayerReference{OperationKindUnchanged, ref.ID(), ElementKindPlayer, ReferencedDataModified, path.toJSONPath(), nil}, true, true
		}
	}
	return nil, false, false
}
func (engine *Engine) assemblePlayerEquipmentSetRef(refID PlayerEquipmentSetRefID, check *recursionCheck, config assembleConfig) (EquipmentSetReference, bool, bool) {
	if config.forceInclude {
		ref := engine.playerEquipmentSetRef(refID).playerEquipmentSetRef
		referencedElement := engine.EquipmentSet(ref.ReferencedElementID).equipmentSet
		if check == nil {
			check = newRecursionCheck()
		}
		referencedDataStatus := ReferencedDataUnchanged
		if _, _, hasUpdatedDownstream := engine.assembleEquipmentSet(referencedElement.ID, check, config); hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		path, _ := engine.PathTrack.equipmentSet[referencedElement.ID]
		return EquipmentSetReference{ref.OperationKind, ref.ReferencedElementID, ElementKindEquipmentSet, referencedDataStatus, path.toJSONPath(), nil}, true, ref.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
	}
	if patchRef, hasUpdated := engine.Patch.PlayerEquipmentSetRef[refID]; hasUpdated {
		if patchRef.OperationKind == OperationKindUpdate {
			config.forceInclude = true
		}
		referencedElement := engine.EquipmentSet(patchRef.ReferencedElementID).equipmentSet
		if check == nil {
			check = newRecursionCheck()
		}
		element, _, hasUpdatedDownstream := engine.assembleEquipmentSet(referencedElement.ID, check, config)
		referencedDataStatus := ReferencedDataUnchanged
		if hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		var el *EquipmentSet
		if patchRef.OperationKind == OperationKindUpdate {
			el = &element
		}
		path, _ := engine.PathTrack.equipmentSet[referencedElement.ID]
		return EquipmentSetReference{patchRef.OperationKind, patchRef.ReferencedElementID, ElementKindEquipmentSet, referencedDataStatus, path.toJSONPath(), el}, true, patchRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
	}
	ref := engine.playerEquipmentSetRef(refID).playerEquipmentSetRef
	if check == nil {
		check = newRecursionCheck()
	}
	if _, _, hasUpdatedDownstream := engine.assembleEquipmentSet(ref.ReferencedElementID, check, config); hasUpdatedDownstream {
		path, _ := engine.PathTrack.equipmentSet[ref.ReferencedElementID]
		return EquipmentSetReference{OperationKindUnchanged, ref.ReferencedElementID, ElementKindEquipmentSet, ReferencedDataModified, path.toJSONPath(), nil}, true, true
	}
	return EquipmentSetReference{}, false, false
}
func (engine *Engine) assemblePlayerGuildMemberRef(refID PlayerGuildMemberRefID, check *recursionCheck, config assembleConfig) (PlayerReference, bool, bool) {
	if config.forceInclude {
		ref := engine.playerGuildMemberRef(refID).playerGuildMemberRef
		referencedElement := engine.Player(ref.ReferencedElementID).player
		if check == nil {
			check = newRecursionCheck()
		}
		referencedDataStatus := ReferencedDataUnchanged
		if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		path, _ := engine.PathTrack.player[referencedElement.ID]
		return PlayerReference{ref.OperationKind, ref.ReferencedElementID, ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, ref.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
	}
	if patchRef, hasUpdated := engine.Patch.PlayerGuildMemberRef[refID]; hasUpdated {
		if patchRef.OperationKind == OperationKindUpdate {
			config.forceInclude = true
		}
		referencedElement := engine.Player(patchRef.ReferencedElementID).player
		if check == nil {
			check = newRecursionCheck()
		}
		element, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config)
		referencedDataStatus := ReferencedDataUnchanged
		if hasUpdatedDownstream {
			referencedDataStatus = ReferencedDataModified
		}
		var el *Player
		if patchRef.OperationKind == OperationKindUpdate {
			el = &element
		}
		path, _ := engine.PathTrack.player[referencedElement.ID]
		return PlayerReference{patchRef.OperationKind, patchRef.ReferencedElementID, ElementKindPlayer, referencedDataStatus, path.toJSONPath(), el}, true, patchRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
	}
	ref := engine.playerGuildMemberRef(refID).playerGuildMemberRef
	if check == nil {
		check = newRecursionCheck()
	}
	if _, _, hasUpdatedDownstream := engine.assemblePlayer(ref.ReferencedElementID, check, config); hasUpdatedDownstream {
		path, _ := engine.PathTrack.player[ref.ReferencedElementID]
		return PlayerReference{OperationKindUnchanged, ref.ReferencedElementID, ElementKindPlayer, ReferencedDataModified, path.toJSONPath(), nil}, true, true
	}
	return PlayerReference{}, false, false
}
func (engine *Engine) assemblePlayerTargetRef(playerID PlayerID, check *recursionCheck, config assembleConfig) (*AnyOfPlayer_ZoneItemReference, bool, bool) {
	statePlayer := engine.State.Player[playerID]
	patchPlayer, playerIsInPatch := engine.Patch.Player[playerID]
	if statePlayer.Target == 0 && (!playerIsInPatch || patchPlayer.Target == 0) {
		return nil, false, false
	}
	if config.forceInclude {
		ref := engine.playerTargetRef(patchPlayer.Target)
		anyContainer := engine.anyOfPlayer_ZoneItem(ref.playerTargetRef.ReferencedElementID)
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
			referencedElement := engine.Player(anyContainer.anyOfPlayer_ZoneItem.Player).player
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.player[referencedElement.ID]
			return &AnyOfPlayer_ZoneItemReference{ref.playerTargetRef.OperationKind, int(referencedElement.ID), ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, ref.playerTargetRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
		} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
			referencedElement := engine.ZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem).zoneItem
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			if _, _, hasUpdatedDownstream := engine.assembleZoneItem(referencedElement.ID, check, config); hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.zoneItem[referencedElement.ID]
			return &AnyOfPlayer_ZoneItemReference{ref.playerTargetRef.OperationKind, int(referencedElement.ID), ElementKindZoneItem, referencedDataStatus, path.toJSONPath(), nil}, true, ref.playerTargetRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
		}
	}
	if statePlayer.Target == 0 && (playerIsInPatch && patchPlayer.Target != 0) {
		config.forceInclude = true
		ref := engine.playerTargetRef(patchPlayer.Target)
		anyContainer := engine.anyOfPlayer_ZoneItem(ref.playerTargetRef.ReferencedElementID)
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
			referencedElement := engine.Player(anyContainer.anyOfPlayer_ZoneItem.Player).player
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			element, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config)
			if hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.player[referencedElement.ID]
			return &AnyOfPlayer_ZoneItemReference{OperationKindUpdate, int(referencedElement.ID), ElementKindPlayer, referencedDataStatus, path.toJSONPath(), &element}, true, referencedDataStatus == ReferencedDataModified
		} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
			referencedElement := engine.ZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem).zoneItem
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			element, _, hasUpdatedDownstream := engine.assembleZoneItem(referencedElement.ID, check, config)
			if hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.zoneItem[referencedElement.ID]
			return &AnyOfPlayer_ZoneItemReference{OperationKindUpdate, int(referencedElement.ID), ElementKindZoneItem, referencedDataStatus, path.toJSONPath(), &element}, true, referencedDataStatus == ReferencedDataModified
		}
	}
	if statePlayer.Target != 0 && (playerIsInPatch && patchPlayer.Target == 0) {
		ref := engine.playerTargetRef(statePlayer.Target)
		anyContainer := engine.anyOfPlayer_ZoneItem(ref.playerTargetRef.ReferencedElementID)
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
			referencedElement := engine.Player(anyContainer.anyOfPlayer_ZoneItem.Player).player
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.player[referencedElement.ID]
			return &AnyOfPlayer_ZoneItemReference{OperationKindDelete, int(referencedElement.ID), ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, referencedDataStatus == ReferencedDataModified
		} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
			referencedElement := engine.ZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem).zoneItem
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			if _, _, hasUpdatedDownstream := engine.assembleZoneItem(referencedElement.ID, check, config); hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.zoneItem[referencedElement.ID]
			return &AnyOfPlayer_ZoneItemReference{OperationKindDelete, int(referencedElement.ID), ElementKindZoneItem, referencedDataStatus, path.toJSONPath(), nil}, true, referencedDataStatus == ReferencedDataModified
		}
	}
	if statePlayer.Target != 0 && (playerIsInPatch && patchPlayer.Target != 0) {
		if statePlayer.Target != patchPlayer.Target {
			ref := engine.playerTargetRef(patchPlayer.Target)
			anyContainer := engine.anyOfPlayer_ZoneItem(ref.playerTargetRef.ReferencedElementID)
			if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
				referencedElement := engine.Player(anyContainer.anyOfPlayer_ZoneItem.Player).player
				if check == nil {
					check = newRecursionCheck()
				}
				referencedDataStatus := ReferencedDataUnchanged
				if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
					referencedDataStatus = ReferencedDataModified
				}
				path, _ := engine.PathTrack.player[referencedElement.ID]
				return &AnyOfPlayer_ZoneItemReference{OperationKindUpdate, int(referencedElement.ID), ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, referencedDataStatus == ReferencedDataModified
			} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
				referencedElement := engine.ZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem).zoneItem
				if check == nil {
					check = newRecursionCheck()
				}
				referencedDataStatus := ReferencedDataUnchanged
				if _, _, hasUpdatedDownstream := engine.assembleZoneItem(referencedElement.ID, check, config); hasUpdatedDownstream {
					referencedDataStatus = ReferencedDataModified
				}
				path, _ := engine.PathTrack.zoneItem[referencedElement.ID]
				return &AnyOfPlayer_ZoneItemReference{OperationKindUpdate, int(referencedElement.ID), ElementKindZoneItem, referencedDataStatus, path.toJSONPath(), nil}, true, referencedDataStatus == ReferencedDataModified
			}
		}
	}
	if statePlayer.Target != 0 {
		ref := engine.playerTargetRef(statePlayer.Target)
		anyContainer := engine.anyOfPlayer_ZoneItem(ref.playerTargetRef.ReferencedElementID)
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
			if check == nil {
				check = newRecursionCheck()
			}
			if _, _, hasUpdatedDownstream := engine.assemblePlayer(anyContainer.anyOfPlayer_ZoneItem.Player, check, config); hasUpdatedDownstream {
				path, _ := engine.PathTrack.player[anyContainer.anyOfPlayer_ZoneItem.Player]
				return &AnyOfPlayer_ZoneItemReference{OperationKindUnchanged, int(anyContainer.anyOfPlayer_ZoneItem.Player), ElementKindPlayer, ReferencedDataModified, path.toJSONPath(), nil}, true, true
			}
		} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
			if check == nil {
				check = newRecursionCheck()
			}
			if _, _, hasUpdatedDownstream := engine.assembleZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem, check, config); hasUpdatedDownstream {
				path, _ := engine.PathTrack.zoneItem[anyContainer.anyOfPlayer_ZoneItem.ZoneItem]
				return &AnyOfPlayer_ZoneItemReference{OperationKindUnchanged, int(anyContainer.anyOfPlayer_ZoneItem.ZoneItem), ElementKindZoneItem, ReferencedDataModified, path.toJSONPath(), nil}, true, true
			}
		}
	}
	return nil, false, false
}
func (engine *Engine) assemblePlayerTargetedByRef(refID PlayerTargetedByRefID, check *recursionCheck, config assembleConfig) (AnyOfPlayer_ZoneItemReference, bool, bool) {
	if config.forceInclude {
		ref := engine.playerTargetedByRef(refID).playerTargetedByRef
		anyContainer := engine.anyOfPlayer_ZoneItem(ref.ReferencedElementID)
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
			referencedElement := engine.Player(anyContainer.anyOfPlayer_ZoneItem.Player).player
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			if _, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config); hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.player[referencedElement.ID]
			return AnyOfPlayer_ZoneItemReference{ref.OperationKind, int(referencedElement.ID), ElementKindPlayer, referencedDataStatus, path.toJSONPath(), nil}, true, ref.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
		} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
			referencedElement := engine.ZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem).zoneItem
			if check == nil {
				check = newRecursionCheck()
			}
			referencedDataStatus := ReferencedDataUnchanged
			if _, _, hasUpdatedDownstream := engine.assembleZoneItem(referencedElement.ID, check, config); hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			path, _ := engine.PathTrack.zoneItem[referencedElement.ID]
			return AnyOfPlayer_ZoneItemReference{ref.OperationKind, int(referencedElement.ID), ElementKindZoneItem, referencedDataStatus, path.toJSONPath(), nil}, true, ref.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
		}
	}
	if patchRef, hasUpdated := engine.Patch.PlayerTargetedByRef[refID]; hasUpdated {
		if patchRef.OperationKind == OperationKindUpdate {
			config.forceInclude = true
		}
		anyContainer := engine.anyOfPlayer_ZoneItem(patchRef.ReferencedElementID)
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
			referencedElement := engine.Player(anyContainer.anyOfPlayer_ZoneItem.Player).player
			if check == nil {
				check = newRecursionCheck()
			}
			element, _, hasUpdatedDownstream := engine.assemblePlayer(referencedElement.ID, check, config)
			referencedDataStatus := ReferencedDataUnchanged
			if hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			var el *Player
			if patchRef.OperationKind == OperationKindUpdate {
				el = &element
			}
			path, _ := engine.PathTrack.player[referencedElement.ID]
			return AnyOfPlayer_ZoneItemReference{patchRef.OperationKind, int(referencedElement.ID), ElementKindPlayer, referencedDataStatus, path.toJSONPath(), el}, true, patchRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
		} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
			referencedElement := engine.ZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem).zoneItem
			if check == nil {
				check = newRecursionCheck()
			}
			element, _, hasUpdatedDownstream := engine.assembleZoneItem(referencedElement.ID, check, config)
			referencedDataStatus := ReferencedDataUnchanged
			if hasUpdatedDownstream {
				referencedDataStatus = ReferencedDataModified
			}
			var el *ZoneItem
			if patchRef.OperationKind == OperationKindUpdate {
				el = &element
			}
			path, _ := engine.PathTrack.zoneItem[referencedElement.ID]
			return AnyOfPlayer_ZoneItemReference{patchRef.OperationKind, int(referencedElement.ID), ElementKindZoneItem, referencedDataStatus, path.toJSONPath(), el}, true, patchRef.OperationKind == OperationKindUpdate || referencedDataStatus == ReferencedDataModified
		}
	}
	ref := engine.playerTargetedByRef(refID).playerTargetedByRef
	if check == nil {
		check = newRecursionCheck()
	}
	anyContainer := engine.anyOfPlayer_ZoneItem(ref.ReferencedElementID)
	if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindPlayer {
		if _, _, hasUpdatedDownstream := engine.assemblePlayer(anyContainer.anyOfPlayer_ZoneItem.Player, check, config); hasUpdatedDownstream {
			path, _ := engine.PathTrack.player[anyContainer.anyOfPlayer_ZoneItem.Player]
			return AnyOfPlayer_ZoneItemReference{OperationKindUnchanged, int(anyContainer.anyOfPlayer_ZoneItem.Player), ElementKindPlayer, ReferencedDataModified, path.toJSONPath(), nil}, true, true
		}
	} else if anyContainer.anyOfPlayer_ZoneItem.ElementKind == ElementKindZoneItem {
		if _, _, hasUpdatedDownstream := engine.assembleZoneItem(anyContainer.anyOfPlayer_ZoneItem.ZoneItem, check, config); hasUpdatedDownstream {
			path, _ := engine.PathTrack.zoneItem[anyContainer.anyOfPlayer_ZoneItem.ZoneItem]
			return AnyOfPlayer_ZoneItemReference{OperationKindUnchanged, int(anyContainer.anyOfPlayer_ZoneItem.ZoneItem), ElementKindZoneItem, ReferencedDataModified, path.toJSONPath(), nil}, true, true
		}
	}
	return AnyOfPlayer_ZoneItemReference{}, false, false
}
func (engine *Engine) CreateEquipmentSet() equipmentSet {
	return engine.createEquipmentSet()
}
func (engine *Engine) createEquipmentSet() equipmentSet {
	var element equipmentSetCore
	element.engine = engine
	element.ID = EquipmentSetID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.EquipmentSet[element.ID] = element
	return equipmentSet{equipmentSet: element}
}
func (engine *Engine) CreateGearScore() gearScore {
	return engine.createGearScore(false)
}
func (engine *Engine) createGearScore(hasParent bool) gearScore {
	var element gearScoreCore
	element.engine = engine
	element.ID = GearScoreID(engine.GenerateID())
	element.HasParent = hasParent
	element.OperationKind = OperationKindUpdate
	engine.Patch.GearScore[element.ID] = element
	return gearScore{gearScore: element}
}
func (engine *Engine) CreateItem() item {
	return engine.createItem(false)
}
func (engine *Engine) createItem(hasParent bool) item {
	var element itemCore
	element.engine = engine
	element.ID = ItemID(engine.GenerateID())
	element.HasParent = hasParent
	elementGearScore := engine.createGearScore(true)
	element.GearScore = elementGearScore.gearScore.ID
	elementOrigin := engine.createAnyOfPlayer_Position(true)
	element.Origin = elementOrigin.anyOfPlayer_Position.ID
	element.OperationKind = OperationKindUpdate
	engine.Patch.Item[element.ID] = element
	return item{item: element}
}
func (engine *Engine) CreatePlayer() player {
	return engine.createPlayer(false)
}
func (engine *Engine) createPlayer(hasParent bool) player {
	var element playerCore
	element.engine = engine
	element.ID = PlayerID(engine.GenerateID())
	element.HasParent = hasParent
	elementGearScore := engine.createGearScore(true)
	element.GearScore = elementGearScore.gearScore.ID
	elementPosition := engine.createPosition(true)
	element.Position = elementPosition.position.ID
	element.OperationKind = OperationKindUpdate
	engine.Patch.Player[element.ID] = element
	return player{player: element}
}
func (engine *Engine) CreatePosition() position {
	return engine.createPosition(false)
}
func (engine *Engine) createPosition(hasParent bool) position {
	var element positionCore
	element.engine = engine
	element.ID = PositionID(engine.GenerateID())
	element.HasParent = hasParent
	element.OperationKind = OperationKindUpdate
	engine.Patch.Position[element.ID] = element
	return position{position: element}
}
func (engine *Engine) CreateZone() zone {
	return engine.createZone()
}
func (engine *Engine) createZone() zone {
	var element zoneCore
	element.engine = engine
	element.ID = ZoneID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.Zone[element.ID] = element
	return zone{zone: element}
}
func (engine *Engine) CreateZoneItem() zoneItem {
	return engine.createZoneItem(false)
}
func (engine *Engine) createZoneItem(hasParent bool) zoneItem {
	var element zoneItemCore
	element.engine = engine
	element.ID = ZoneItemID(engine.GenerateID())
	element.HasParent = hasParent
	elementItem := engine.createItem(true)
	element.Item = elementItem.item.ID
	elementPosition := engine.createPosition(true)
	element.Position = elementPosition.position.ID
	element.OperationKind = OperationKindUpdate
	engine.Patch.ZoneItem[element.ID] = element
	return zoneItem{zoneItem: element}
}
func (engine *Engine) createEquipmentSetEquipmentRef(referencedElementID ItemID, parentID EquipmentSetID) equipmentSetEquipmentRefCore {
	var element equipmentSetEquipmentRefCore
	element.engine = engine
	element.ReferencedElementID = referencedElementID
	element.ParentID = parentID
	element.ID = EquipmentSetEquipmentRefID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.EquipmentSetEquipmentRef[element.ID] = element
	return element
}
func (engine *Engine) createItemBoundToRef(referencedElementID PlayerID, parentID ItemID) itemBoundToRefCore {
	var element itemBoundToRefCore
	element.engine = engine
	element.ReferencedElementID = referencedElementID
	element.ParentID = parentID
	element.ID = ItemBoundToRefID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.ItemBoundToRef[element.ID] = element
	return element
}
func (engine *Engine) createPlayerEquipmentSetRef(referencedElementID EquipmentSetID, parentID PlayerID) playerEquipmentSetRefCore {
	var element playerEquipmentSetRefCore
	element.engine = engine
	element.ReferencedElementID = referencedElementID
	element.ParentID = parentID
	element.ID = PlayerEquipmentSetRefID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.PlayerEquipmentSetRef[element.ID] = element
	return element
}
func (engine *Engine) createPlayerGuildMemberRef(referencedElementID PlayerID, parentID PlayerID) playerGuildMemberRefCore {
	var element playerGuildMemberRefCore
	element.engine = engine
	element.ReferencedElementID = referencedElementID
	element.ParentID = parentID
	element.ID = PlayerGuildMemberRefID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.PlayerGuildMemberRef[element.ID] = element
	return element
}
func (engine *Engine) createPlayerTargetRef(referencedElementID AnyOfPlayer_ZoneItemID, parentID PlayerID) playerTargetRefCore {
	var element playerTargetRefCore
	element.engine = engine
	element.ReferencedElementID = referencedElementID
	element.ParentID = parentID
	element.ID = PlayerTargetRefID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.PlayerTargetRef[element.ID] = element
	return element
}
func (engine *Engine) createPlayerTargetedByRef(referencedElementID AnyOfPlayer_ZoneItemID, parentID PlayerID) playerTargetedByRefCore {
	var element playerTargetedByRefCore
	element.engine = engine
	element.ReferencedElementID = referencedElementID
	element.ParentID = parentID
	element.ID = PlayerTargetedByRefID(engine.GenerateID())
	element.OperationKind = OperationKindUpdate
	engine.Patch.PlayerTargetedByRef[element.ID] = element
	return element
}
func (engine *Engine) createAnyOfPlayer_Position(setDefaultValue bool) anyOfPlayer_Position {
	var element anyOfPlayer_PositionCore
	element.engine = engine
	element.ID = AnyOfPlayer_PositionID(engine.GenerateID())
	if setDefaultValue {
		elementPlayer := engine.createPlayer(true)
		element.Player = elementPlayer.player.ID
		element.ElementKind = ElementKindPlayer
	}
	element.OperationKind = OperationKindUpdate
	engine.Patch.AnyOfPlayer_Position[element.ID] = element
	return anyOfPlayer_Position{anyOfPlayer_Position: element}
}
func (engine *Engine) createAnyOfPlayer_ZoneItem(setDefaultValue bool) anyOfPlayer_ZoneItem {
	var element anyOfPlayer_ZoneItemCore
	element.engine = engine
	element.ID = AnyOfPlayer_ZoneItemID(engine.GenerateID())
	if setDefaultValue {
		elementPlayer := engine.createPlayer(true)
		element.Player = elementPlayer.player.ID
		element.ElementKind = ElementKindPlayer
	}
	element.OperationKind = OperationKindUpdate
	engine.Patch.AnyOfPlayer_ZoneItem[element.ID] = element
	return anyOfPlayer_ZoneItem{anyOfPlayer_ZoneItem: element}
}
func (engine *Engine) createAnyOfItem_Player_ZoneItem(setDefaultValue bool) anyOfItem_Player_ZoneItem {
	var element anyOfItem_Player_ZoneItemCore
	element.engine = engine
	element.ID = AnyOfItem_Player_ZoneItemID(engine.GenerateID())
	if setDefaultValue {
		elementItem := engine.createItem(true)
		element.Item = elementItem.item.ID
		element.ElementKind = ElementKindItem
	}
	element.OperationKind = OperationKindUpdate
	engine.Patch.AnyOfItem_Player_ZoneItem[element.ID] = element
	return anyOfItem_Player_ZoneItem{anyOfItem_Player_ZoneItem: element}
}
func (engine *Engine) DeleteEquipmentSet(equipmentSetID EquipmentSetID) {
	engine.deleteEquipmentSet(equipmentSetID)
}
func (engine *Engine) deleteEquipmentSet(equipmentSetID EquipmentSetID) {
	equipmentSet := engine.EquipmentSet(equipmentSetID).equipmentSet
	engine.dereferencePlayerEquipmentSetRefs(equipmentSetID)
	for _, equipmentID := range equipmentSet.Equipment {
		engine.deleteEquipmentSetEquipmentRef(equipmentID)
	}
	if _, ok := engine.State.EquipmentSet[equipmentSetID]; ok {
		equipmentSet.OperationKind = OperationKindDelete
		engine.Patch.EquipmentSet[equipmentSet.ID] = equipmentSet
	} else {
		delete(engine.Patch.EquipmentSet, equipmentSetID)
	}
}
func (engine *Engine) DeleteGearScore(gearScoreID GearScoreID) {
	gearScore := engine.GearScore(gearScoreID).gearScore
	if gearScore.HasParent {
		return
	}
	engine.deleteGearScore(gearScoreID)
}
func (engine *Engine) deleteGearScore(gearScoreID GearScoreID) {
	gearScore := engine.GearScore(gearScoreID).gearScore
	if _, ok := engine.State.GearScore[gearScoreID]; ok {
		gearScore.OperationKind = OperationKindDelete
		engine.Patch.GearScore[gearScore.ID] = gearScore
	} else {
		delete(engine.Patch.GearScore, gearScoreID)
	}
}
func (engine *Engine) DeleteItem(itemID ItemID) {
	item := engine.Item(itemID).item
	if item.HasParent {
		return
	}
	engine.deleteItem(itemID)
}
func (engine *Engine) deleteItem(itemID ItemID) {
	item := engine.Item(itemID).item
	engine.dereferenceEquipmentSetEquipmentRefs(itemID)
	engine.deleteItemBoundToRef(item.BoundTo)
	engine.deleteGearScore(item.GearScore)
	engine.deleteAnyOfPlayer_Position(item.Origin, true)
	if _, ok := engine.State.Item[itemID]; ok {
		item.OperationKind = OperationKindDelete
		engine.Patch.Item[item.ID] = item
	} else {
		delete(engine.Patch.Item, itemID)
	}
}
func (engine *Engine) DeletePlayer(playerID PlayerID) {
	player := engine.Player(playerID).player
	if player.HasParent {
		return
	}
	engine.deletePlayer(playerID)
}
func (engine *Engine) deletePlayer(playerID PlayerID) {
	player := engine.Player(playerID).player
	engine.dereferenceItemBoundToRefs(playerID)
	engine.dereferencePlayerGuildMemberRefs(playerID)
	engine.dereferencePlayerTargetRefsPlayer(playerID)
	engine.dereferencePlayerTargetedByRefsPlayer(playerID)
	for _, equipmentSetID := range player.EquipmentSets {
		engine.deletePlayerEquipmentSetRef(equipmentSetID)
	}
	engine.deleteGearScore(player.GearScore)
	for _, guildMemberID := range player.GuildMembers {
		engine.deletePlayerGuildMemberRef(guildMemberID)
	}
	for _, itemID := range player.Items {
		engine.deleteItem(itemID)
	}
	engine.deletePosition(player.Position)
	engine.deletePlayerTargetRef(player.Target)
	for _, targetedByID := range player.TargetedBy {
		engine.deletePlayerTargetedByRef(targetedByID)
	}
	if _, ok := engine.State.Player[playerID]; ok {
		player.OperationKind = OperationKindDelete
		engine.Patch.Player[player.ID] = player
	} else {
		delete(engine.Patch.Player, playerID)
	}
}
func (engine *Engine) DeletePosition(positionID PositionID) {
	position := engine.Position(positionID).position
	if position.HasParent {
		return
	}
	engine.deletePosition(positionID)
}
func (engine *Engine) deletePosition(positionID PositionID) {
	position := engine.Position(positionID).position
	if _, ok := engine.State.Position[positionID]; ok {
		position.OperationKind = OperationKindDelete
		engine.Patch.Position[position.ID] = position
	} else {
		delete(engine.Patch.Position, positionID)
	}
}
func (engine *Engine) DeleteZone(zoneID ZoneID) {
	engine.deleteZone(zoneID)
}
func (engine *Engine) deleteZone(zoneID ZoneID) {
	zone := engine.Zone(zoneID).zone
	for _, interactableID := range zone.Interactables {
		engine.deleteAnyOfItem_Player_ZoneItem(interactableID, true)
	}
	for _, itemID := range zone.Items {
		engine.deleteZoneItem(itemID)
	}
	for _, playerID := range zone.Players {
		engine.deletePlayer(playerID)
	}
	if _, ok := engine.State.Zone[zoneID]; ok {
		zone.OperationKind = OperationKindDelete
		engine.Patch.Zone[zone.ID] = zone
	} else {
		delete(engine.Patch.Zone, zoneID)
	}
}
func (engine *Engine) DeleteZoneItem(zoneItemID ZoneItemID) {
	zoneItem := engine.ZoneItem(zoneItemID).zoneItem
	if zoneItem.HasParent {
		return
	}
	engine.deleteZoneItem(zoneItemID)
}
func (engine *Engine) deleteZoneItem(zoneItemID ZoneItemID) {
	zoneItem := engine.ZoneItem(zoneItemID).zoneItem
	engine.dereferencePlayerTargetRefsZoneItem(zoneItemID)
	engine.dereferencePlayerTargetedByRefsZoneItem(zoneItemID)
	engine.deleteItem(zoneItem.Item)
	engine.deletePosition(zoneItem.Position)
	if _, ok := engine.State.ZoneItem[zoneItemID]; ok {
		zoneItem.OperationKind = OperationKindDelete
		engine.Patch.ZoneItem[zoneItem.ID] = zoneItem
	} else {
		delete(engine.Patch.ZoneItem, zoneItemID)
	}
}
func (engine *Engine) deleteEquipmentSetEquipmentRef(equipmentSetEquipmentRefID EquipmentSetEquipmentRefID) {
	equipmentSetEquipmentRef := engine.equipmentSetEquipmentRef(equipmentSetEquipmentRefID).equipmentSetEquipmentRef
	if _, ok := engine.State.EquipmentSetEquipmentRef[equipmentSetEquipmentRefID]; ok {
		equipmentSetEquipmentRef.OperationKind = OperationKindDelete
		engine.Patch.EquipmentSetEquipmentRef[equipmentSetEquipmentRef.ID] = equipmentSetEquipmentRef
	} else {
		delete(engine.Patch.EquipmentSetEquipmentRef, equipmentSetEquipmentRefID)
	}
}
func (engine *Engine) deleteItemBoundToRef(itemBoundToRefID ItemBoundToRefID) {
	itemBoundToRef := engine.itemBoundToRef(itemBoundToRefID).itemBoundToRef
	if _, ok := engine.State.ItemBoundToRef[itemBoundToRefID]; ok {
		itemBoundToRef.OperationKind = OperationKindDelete
		engine.Patch.ItemBoundToRef[itemBoundToRef.ID] = itemBoundToRef
	} else {
		delete(engine.Patch.ItemBoundToRef, itemBoundToRefID)
	}
}
func (engine *Engine) deletePlayerEquipmentSetRef(playerEquipmentSetRefID PlayerEquipmentSetRefID) {
	playerEquipmentSetRef := engine.playerEquipmentSetRef(playerEquipmentSetRefID).playerEquipmentSetRef
	if _, ok := engine.State.PlayerEquipmentSetRef[playerEquipmentSetRefID]; ok {
		playerEquipmentSetRef.OperationKind = OperationKindDelete
		engine.Patch.PlayerEquipmentSetRef[playerEquipmentSetRef.ID] = playerEquipmentSetRef
	} else {
		delete(engine.Patch.PlayerEquipmentSetRef, playerEquipmentSetRefID)
	}
}
func (engine *Engine) deletePlayerGuildMemberRef(playerGuildMemberRefID PlayerGuildMemberRefID) {
	playerGuildMemberRef := engine.playerGuildMemberRef(playerGuildMemberRefID).playerGuildMemberRef
	if _, ok := engine.State.PlayerGuildMemberRef[playerGuildMemberRefID]; ok {
		playerGuildMemberRef.OperationKind = OperationKindDelete
		engine.Patch.PlayerGuildMemberRef[playerGuildMemberRef.ID] = playerGuildMemberRef
	} else {
		delete(engine.Patch.PlayerGuildMemberRef, playerGuildMemberRefID)
	}
}
func (engine *Engine) deletePlayerTargetRef(playerTargetRefID PlayerTargetRefID) {
	playerTargetRef := engine.playerTargetRef(playerTargetRefID).playerTargetRef
	engine.deleteAnyOfPlayer_ZoneItem(playerTargetRef.ReferencedElementID, false)
	if _, ok := engine.State.PlayerTargetRef[playerTargetRefID]; ok {
		playerTargetRef.OperationKind = OperationKindDelete
		engine.Patch.PlayerTargetRef[playerTargetRef.ID] = playerTargetRef
	} else {
		delete(engine.Patch.PlayerTargetRef, playerTargetRefID)
	}
}
func (engine *Engine) deletePlayerTargetedByRef(playerTargetedByRefID PlayerTargetedByRefID) {
	playerTargetedByRef := engine.playerTargetedByRef(playerTargetedByRefID).playerTargetedByRef
	engine.deleteAnyOfPlayer_ZoneItem(playerTargetedByRef.ReferencedElementID, false)
	if _, ok := engine.State.PlayerTargetedByRef[playerTargetedByRefID]; ok {
		playerTargetedByRef.OperationKind = OperationKindDelete
		engine.Patch.PlayerTargetedByRef[playerTargetedByRef.ID] = playerTargetedByRef
	} else {
		delete(engine.Patch.PlayerTargetedByRef, playerTargetedByRefID)
	}
}
func (engine *Engine) deleteAnyOfPlayer_Position(anyOfPlayer_PositionID AnyOfPlayer_PositionID, deleteChild bool) {
	anyOfPlayer_Position := engine.anyOfPlayer_Position(anyOfPlayer_PositionID).anyOfPlayer_Position
	if deleteChild {
		anyOfPlayer_Position.deleteChild()
	}
	if _, ok := engine.State.AnyOfPlayer_Position[anyOfPlayer_PositionID]; ok {
		anyOfPlayer_Position.OperationKind = OperationKindDelete
		engine.Patch.AnyOfPlayer_Position[anyOfPlayer_Position.ID] = anyOfPlayer_Position
	} else {
		delete(engine.Patch.AnyOfPlayer_Position, anyOfPlayer_PositionID)
	}
}
func (engine *Engine) deleteAnyOfPlayer_ZoneItem(anyOfPlayer_ZoneItemID AnyOfPlayer_ZoneItemID, deleteChild bool) {
	anyOfPlayer_ZoneItem := engine.anyOfPlayer_ZoneItem(anyOfPlayer_ZoneItemID).anyOfPlayer_ZoneItem
	if deleteChild {
		anyOfPlayer_ZoneItem.deleteChild()
	}
	if _, ok := engine.State.AnyOfPlayer_ZoneItem[anyOfPlayer_ZoneItemID]; ok {
		anyOfPlayer_ZoneItem.OperationKind = OperationKindDelete
		engine.Patch.AnyOfPlayer_ZoneItem[anyOfPlayer_ZoneItem.ID] = anyOfPlayer_ZoneItem
	} else {
		delete(engine.Patch.AnyOfPlayer_ZoneItem, anyOfPlayer_ZoneItemID)
	}
}
func (engine *Engine) deleteAnyOfItem_Player_ZoneItem(anyOfItem_Player_ZoneItemID AnyOfItem_Player_ZoneItemID, deleteChild bool) {
	anyOfItem_Player_ZoneItem := engine.anyOfItem_Player_ZoneItem(anyOfItem_Player_ZoneItemID).anyOfItem_Player_ZoneItem
	if deleteChild {
		anyOfItem_Player_ZoneItem.deleteChild()
	}
	if _, ok := engine.State.AnyOfItem_Player_ZoneItem[anyOfItem_Player_ZoneItemID]; ok {
		anyOfItem_Player_ZoneItem.OperationKind = OperationKindDelete
		engine.Patch.AnyOfItem_Player_ZoneItem[anyOfItem_Player_ZoneItem.ID] = anyOfItem_Player_ZoneItem
	} else {
		delete(engine.Patch.AnyOfItem_Player_ZoneItem, anyOfItem_Player_ZoneItemID)
	}
}
func (engine *Engine) EquipmentSet(equipmentSetID EquipmentSetID) equipmentSet {
	patchingEquipmentSet, ok := engine.Patch.EquipmentSet[equipmentSetID]
	if ok {
		return equipmentSet{equipmentSet: patchingEquipmentSet}
	}
	currentEquipmentSet, ok := engine.State.EquipmentSet[equipmentSetID]
	if ok {
		return equipmentSet{equipmentSet: currentEquipmentSet}
	}
	return equipmentSet{equipmentSet: equipmentSetCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_equipmentSet equipmentSet) ID() EquipmentSetID {
	return _equipmentSet.equipmentSet.ID
}
func (_equipmentSet equipmentSet) Equipment() []equipmentSetEquipmentRef {
	equipmentSet := _equipmentSet.equipmentSet.engine.EquipmentSet(_equipmentSet.equipmentSet.ID)
	var equipment []equipmentSetEquipmentRef
	for _, refID := range equipmentSet.equipmentSet.Equipment {
		equipment = append(equipment, equipmentSet.equipmentSet.engine.equipmentSetEquipmentRef(refID))
	}
	return equipment
}
func (_equipmentSet equipmentSet) Name() string {
	equipmentSet := _equipmentSet.equipmentSet.engine.EquipmentSet(_equipmentSet.equipmentSet.ID)
	return equipmentSet.equipmentSet.Name
}
func (engine *Engine) GearScore(gearScoreID GearScoreID) gearScore {
	patchingGearScore, ok := engine.Patch.GearScore[gearScoreID]
	if ok {
		return gearScore{gearScore: patchingGearScore}
	}
	currentGearScore, ok := engine.State.GearScore[gearScoreID]
	if ok {
		return gearScore{gearScore: currentGearScore}
	}
	return gearScore{gearScore: gearScoreCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_gearScore gearScore) ID() GearScoreID {
	return _gearScore.gearScore.ID
}
func (_gearScore gearScore) Level() int {
	gearScore := _gearScore.gearScore.engine.GearScore(_gearScore.gearScore.ID)
	return gearScore.gearScore.Level
}
func (_gearScore gearScore) Score() int {
	gearScore := _gearScore.gearScore.engine.GearScore(_gearScore.gearScore.ID)
	return gearScore.gearScore.Score
}
func (engine *Engine) Item(itemID ItemID) item {
	patchingItem, ok := engine.Patch.Item[itemID]
	if ok {
		return item{item: patchingItem}
	}
	currentItem, ok := engine.State.Item[itemID]
	if ok {
		return item{item: currentItem}
	}
	return item{item: itemCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_item item) ID() ItemID {
	return _item.item.ID
}
func (_item item) BoundTo() (itemBoundToRef, bool) {
	item := _item.item.engine.Item(_item.item.ID)
	return item.item.engine.itemBoundToRef(item.item.BoundTo), item.item.BoundTo != 0
}
func (_item item) GearScore() gearScore {
	item := _item.item.engine.Item(_item.item.ID)
	return item.item.engine.GearScore(item.item.GearScore)
}
func (_item item) Name() string {
	item := _item.item.engine.Item(_item.item.ID)
	return item.item.Name
}
func (_item item) Origin() anyOfPlayer_Position {
	item := _item.item.engine.Item(_item.item.ID)
	return item.item.engine.anyOfPlayer_Position(item.item.Origin)
}
func (engine *Engine) Player(playerID PlayerID) player {
	patchingPlayer, ok := engine.Patch.Player[playerID]
	if ok {
		return player{player: patchingPlayer}
	}
	currentPlayer, ok := engine.State.Player[playerID]
	if ok {
		return player{player: currentPlayer}
	}
	return player{player: playerCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_player player) ID() PlayerID {
	return _player.player.ID
}
func (_player player) EquipmentSets() []playerEquipmentSetRef {
	player := _player.player.engine.Player(_player.player.ID)
	var equipmentSets []playerEquipmentSetRef
	for _, refID := range player.player.EquipmentSets {
		equipmentSets = append(equipmentSets, player.player.engine.playerEquipmentSetRef(refID))
	}
	return equipmentSets
}
func (_player player) GearScore() gearScore {
	player := _player.player.engine.Player(_player.player.ID)
	return player.player.engine.GearScore(player.player.GearScore)
}
func (_player player) GuildMembers() []playerGuildMemberRef {
	player := _player.player.engine.Player(_player.player.ID)
	var guildMembers []playerGuildMemberRef
	for _, refID := range player.player.GuildMembers {
		guildMembers = append(guildMembers, player.player.engine.playerGuildMemberRef(refID))
	}
	return guildMembers
}
func (_player player) Items() []item {
	player := _player.player.engine.Player(_player.player.ID)
	var items []item
	for _, itemID := range player.player.Items {
		items = append(items, player.player.engine.Item(itemID))
	}
	return items
}
func (_player player) Position() position {
	player := _player.player.engine.Player(_player.player.ID)
	return player.player.engine.Position(player.player.Position)
}
func (_player player) Target() (playerTargetRef, bool) {
	player := _player.player.engine.Player(_player.player.ID)
	return player.player.engine.playerTargetRef(player.player.Target), player.player.Target != 0
}
func (_player player) TargetedBy() []playerTargetedByRef {
	player := _player.player.engine.Player(_player.player.ID)
	var targetedBy []playerTargetedByRef
	for _, refID := range player.player.TargetedBy {
		targetedBy = append(targetedBy, player.player.engine.playerTargetedByRef(refID))
	}
	return targetedBy
}
func (engine *Engine) Position(positionID PositionID) position {
	patchingPosition, ok := engine.Patch.Position[positionID]
	if ok {
		return position{position: patchingPosition}
	}
	currentPosition, ok := engine.State.Position[positionID]
	if ok {
		return position{position: currentPosition}
	}
	return position{position: positionCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_position position) ID() PositionID {
	return _position.position.ID
}
func (_position position) X() float64 {
	position := _position.position.engine.Position(_position.position.ID)
	return position.position.X
}
func (_position position) Y() float64 {
	position := _position.position.engine.Position(_position.position.ID)
	return position.position.Y
}
func (engine *Engine) Zone(zoneID ZoneID) zone {
	patchingZone, ok := engine.Patch.Zone[zoneID]
	if ok {
		return zone{zone: patchingZone}
	}
	currentZone, ok := engine.State.Zone[zoneID]
	if ok {
		return zone{zone: currentZone}
	}
	return zone{zone: zoneCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_zone zone) ID() ZoneID {
	return _zone.zone.ID
}
func (_zone zone) Interactables() []anyOfItem_Player_ZoneItem {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	var interactables []anyOfItem_Player_ZoneItem
	for _, anyOfItem_Player_ZoneItemID := range zone.zone.Interactables {
		interactables = append(interactables, zone.zone.engine.anyOfItem_Player_ZoneItem(anyOfItem_Player_ZoneItemID))
	}
	return interactables
}
func (_zone zone) Items() []zoneItem {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	var items []zoneItem
	for _, zoneItemID := range zone.zone.Items {
		items = append(items, zone.zone.engine.ZoneItem(zoneItemID))
	}
	return items
}
func (_zone zone) Players() []player {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	var players []player
	for _, playerID := range zone.zone.Players {
		players = append(players, zone.zone.engine.Player(playerID))
	}
	return players
}
func (_zone zone) Tags() []string {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	var tags []string
	for _, element := range zone.zone.Tags {
		tags = append(tags, element)
	}
	return tags
}
func (engine *Engine) ZoneItem(zoneItemID ZoneItemID) zoneItem {
	patchingZoneItem, ok := engine.Patch.ZoneItem[zoneItemID]
	if ok {
		return zoneItem{zoneItem: patchingZoneItem}
	}
	currentZoneItem, ok := engine.State.ZoneItem[zoneItemID]
	if ok {
		return zoneItem{zoneItem: currentZoneItem}
	}
	return zoneItem{zoneItem: zoneItemCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_zoneItem zoneItem) ID() ZoneItemID {
	return _zoneItem.zoneItem.ID
}
func (_zoneItem zoneItem) Item() item {
	zoneItem := _zoneItem.zoneItem.engine.ZoneItem(_zoneItem.zoneItem.ID)
	return zoneItem.zoneItem.engine.Item(zoneItem.zoneItem.Item)
}
func (_zoneItem zoneItem) Position() position {
	zoneItem := _zoneItem.zoneItem.engine.ZoneItem(_zoneItem.zoneItem.ID)
	return zoneItem.zoneItem.engine.Position(zoneItem.zoneItem.Position)
}
func (engine *Engine) equipmentSetEquipmentRef(equipmentSetEquipmentRefID EquipmentSetEquipmentRefID) equipmentSetEquipmentRef {
	patchingEquipmentSetEquipmentRef, ok := engine.Patch.EquipmentSetEquipmentRef[equipmentSetEquipmentRefID]
	if ok {
		return equipmentSetEquipmentRef{equipmentSetEquipmentRef: patchingEquipmentSetEquipmentRef}
	}
	currentEquipmentSetEquipmentRef, ok := engine.State.EquipmentSetEquipmentRef[equipmentSetEquipmentRefID]
	if ok {
		return equipmentSetEquipmentRef{equipmentSetEquipmentRef: currentEquipmentSetEquipmentRef}
	}
	return equipmentSetEquipmentRef{equipmentSetEquipmentRef: equipmentSetEquipmentRefCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_equipmentSetEquipmentRef equipmentSetEquipmentRef) ID() ItemID {
	return _equipmentSetEquipmentRef.equipmentSetEquipmentRef.ReferencedElementID
}
func (engine *Engine) itemBoundToRef(itemBoundToRefID ItemBoundToRefID) itemBoundToRef {
	patchingItemBoundToRef, ok := engine.Patch.ItemBoundToRef[itemBoundToRefID]
	if ok {
		return itemBoundToRef{itemBoundToRef: patchingItemBoundToRef}
	}
	currentItemBoundToRef, ok := engine.State.ItemBoundToRef[itemBoundToRefID]
	if ok {
		return itemBoundToRef{itemBoundToRef: currentItemBoundToRef}
	}
	return itemBoundToRef{itemBoundToRef: itemBoundToRefCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_itemBoundToRef itemBoundToRef) ID() PlayerID {
	return _itemBoundToRef.itemBoundToRef.ReferencedElementID
}
func (engine *Engine) playerEquipmentSetRef(playerEquipmentSetRefID PlayerEquipmentSetRefID) playerEquipmentSetRef {
	patchingPlayerEquipmentSetRef, ok := engine.Patch.PlayerEquipmentSetRef[playerEquipmentSetRefID]
	if ok {
		return playerEquipmentSetRef{playerEquipmentSetRef: patchingPlayerEquipmentSetRef}
	}
	currentPlayerEquipmentSetRef, ok := engine.State.PlayerEquipmentSetRef[playerEquipmentSetRefID]
	if ok {
		return playerEquipmentSetRef{playerEquipmentSetRef: currentPlayerEquipmentSetRef}
	}
	return playerEquipmentSetRef{playerEquipmentSetRef: playerEquipmentSetRefCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_playerEquipmentSetRef playerEquipmentSetRef) ID() EquipmentSetID {
	return _playerEquipmentSetRef.playerEquipmentSetRef.ReferencedElementID
}
func (engine *Engine) playerGuildMemberRef(playerGuildMemberRefID PlayerGuildMemberRefID) playerGuildMemberRef {
	patchingPlayerGuildMemberRef, ok := engine.Patch.PlayerGuildMemberRef[playerGuildMemberRefID]
	if ok {
		return playerGuildMemberRef{playerGuildMemberRef: patchingPlayerGuildMemberRef}
	}
	currentPlayerGuildMemberRef, ok := engine.State.PlayerGuildMemberRef[playerGuildMemberRefID]
	if ok {
		return playerGuildMemberRef{playerGuildMemberRef: currentPlayerGuildMemberRef}
	}
	return playerGuildMemberRef{playerGuildMemberRef: playerGuildMemberRefCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_playerGuildMemberRef playerGuildMemberRef) ID() PlayerID {
	return _playerGuildMemberRef.playerGuildMemberRef.ReferencedElementID
}
func (engine *Engine) playerTargetRef(playerTargetRefID PlayerTargetRefID) playerTargetRef {
	patchingPlayerTargetRef, ok := engine.Patch.PlayerTargetRef[playerTargetRefID]
	if ok {
		return playerTargetRef{playerTargetRef: patchingPlayerTargetRef}
	}
	currentPlayerTargetRef, ok := engine.State.PlayerTargetRef[playerTargetRefID]
	if ok {
		return playerTargetRef{playerTargetRef: currentPlayerTargetRef}
	}
	return playerTargetRef{playerTargetRef: playerTargetRefCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_playerTargetRef playerTargetRef) ID() AnyOfPlayer_ZoneItemID {
	return _playerTargetRef.playerTargetRef.ReferencedElementID
}
func (engine *Engine) playerTargetedByRef(playerTargetedByRefID PlayerTargetedByRefID) playerTargetedByRef {
	patchingPlayerTargetedByRef, ok := engine.Patch.PlayerTargetedByRef[playerTargetedByRefID]
	if ok {
		return playerTargetedByRef{playerTargetedByRef: patchingPlayerTargetedByRef}
	}
	currentPlayerTargetedByRef, ok := engine.State.PlayerTargetedByRef[playerTargetedByRefID]
	if ok {
		return playerTargetedByRef{playerTargetedByRef: currentPlayerTargetedByRef}
	}
	return playerTargetedByRef{playerTargetedByRef: playerTargetedByRefCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_playerTargetedByRef playerTargetedByRef) ID() AnyOfPlayer_ZoneItemID {
	return _playerTargetedByRef.playerTargetedByRef.ReferencedElementID
}
func (engine *Engine) anyOfPlayer_Position(anyOfPlayer_PositionID AnyOfPlayer_PositionID) anyOfPlayer_Position {
	patchingAnyOfPlayer_Position, ok := engine.Patch.AnyOfPlayer_Position[anyOfPlayer_PositionID]
	if ok {
		return anyOfPlayer_Position{anyOfPlayer_Position: patchingAnyOfPlayer_Position}
	}
	currentAnyOfPlayer_Position, ok := engine.State.AnyOfPlayer_Position[anyOfPlayer_PositionID]
	if ok {
		return anyOfPlayer_Position{anyOfPlayer_Position: currentAnyOfPlayer_Position}
	}
	return anyOfPlayer_Position{anyOfPlayer_Position: anyOfPlayer_PositionCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_anyOfPlayer_Position anyOfPlayer_Position) ID() AnyOfPlayer_PositionID {
	return _anyOfPlayer_Position.anyOfPlayer_Position.ID
}
func (_anyOfPlayer_Position anyOfPlayer_Position) Player() player {
	anyOfPlayer_Position := _anyOfPlayer_Position.anyOfPlayer_Position.engine.anyOfPlayer_Position(_anyOfPlayer_Position.anyOfPlayer_Position.ID)
	return anyOfPlayer_Position.anyOfPlayer_Position.engine.Player(anyOfPlayer_Position.anyOfPlayer_Position.Player)
}
func (_anyOfPlayer_Position anyOfPlayer_Position) Position() position {
	anyOfPlayer_Position := _anyOfPlayer_Position.anyOfPlayer_Position.engine.anyOfPlayer_Position(_anyOfPlayer_Position.anyOfPlayer_Position.ID)
	return anyOfPlayer_Position.anyOfPlayer_Position.engine.Position(anyOfPlayer_Position.anyOfPlayer_Position.Position)
}
func (engine *Engine) anyOfPlayer_ZoneItem(anyOfPlayer_ZoneItemID AnyOfPlayer_ZoneItemID) anyOfPlayer_ZoneItem {
	patchingAnyOfPlayer_ZoneItem, ok := engine.Patch.AnyOfPlayer_ZoneItem[anyOfPlayer_ZoneItemID]
	if ok {
		return anyOfPlayer_ZoneItem{anyOfPlayer_ZoneItem: patchingAnyOfPlayer_ZoneItem}
	}
	currentAnyOfPlayer_ZoneItem, ok := engine.State.AnyOfPlayer_ZoneItem[anyOfPlayer_ZoneItemID]
	if ok {
		return anyOfPlayer_ZoneItem{anyOfPlayer_ZoneItem: currentAnyOfPlayer_ZoneItem}
	}
	return anyOfPlayer_ZoneItem{anyOfPlayer_ZoneItem: anyOfPlayer_ZoneItemCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_anyOfPlayer_ZoneItem anyOfPlayer_ZoneItem) ID() AnyOfPlayer_ZoneItemID {
	return _anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.ID
}
func (_anyOfPlayer_ZoneItem anyOfPlayer_ZoneItem) Player() player {
	anyOfPlayer_ZoneItem := _anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.engine.anyOfPlayer_ZoneItem(_anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.ID)
	return anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.engine.Player(anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.Player)
}
func (_anyOfPlayer_ZoneItem anyOfPlayer_ZoneItem) ZoneItem() zoneItem {
	anyOfPlayer_ZoneItem := _anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.engine.anyOfPlayer_ZoneItem(_anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.ID)
	return anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.engine.ZoneItem(anyOfPlayer_ZoneItem.anyOfPlayer_ZoneItem.ZoneItem)
}
func (engine *Engine) anyOfItem_Player_ZoneItem(anyOfItem_Player_ZoneItemID AnyOfItem_Player_ZoneItemID) anyOfItem_Player_ZoneItem {
	patchingAnyOfItem_Player_ZoneItem, ok := engine.Patch.AnyOfItem_Player_ZoneItem[anyOfItem_Player_ZoneItemID]
	if ok {
		return anyOfItem_Player_ZoneItem{anyOfItem_Player_ZoneItem: patchingAnyOfItem_Player_ZoneItem}
	}
	currentAnyOfItem_Player_ZoneItem, ok := engine.State.AnyOfItem_Player_ZoneItem[anyOfItem_Player_ZoneItemID]
	if ok {
		return anyOfItem_Player_ZoneItem{anyOfItem_Player_ZoneItem: currentAnyOfItem_Player_ZoneItem}
	}
	return anyOfItem_Player_ZoneItem{anyOfItem_Player_ZoneItem: anyOfItem_Player_ZoneItemCore{OperationKind: OperationKindDelete, engine: engine}}
}
func (_anyOfItem_Player_ZoneItem anyOfItem_Player_ZoneItem) ID() AnyOfItem_Player_ZoneItemID {
	return _anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.ID
}
func (_anyOfItem_Player_ZoneItem anyOfItem_Player_ZoneItem) Item() item {
	anyOfItem_Player_ZoneItem := _anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.engine.anyOfItem_Player_ZoneItem(_anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.ID)
	return anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.engine.Item(anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.Item)
}
func (_anyOfItem_Player_ZoneItem anyOfItem_Player_ZoneItem) Player() player {
	anyOfItem_Player_ZoneItem := _anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.engine.anyOfItem_Player_ZoneItem(_anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.ID)
	return anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.engine.Player(anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.Player)
}
func (_anyOfItem_Player_ZoneItem anyOfItem_Player_ZoneItem) ZoneItem() zoneItem {
	anyOfItem_Player_ZoneItem := _anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.engine.anyOfItem_Player_ZoneItem(_anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.ID)
	return anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.engine.ZoneItem(anyOfItem_Player_ZoneItem.anyOfItem_Player_ZoneItem.ZoneItem)
}
func deduplicateEquipmentSetEquipmentRefIDs(a []EquipmentSetEquipmentRefID, b []EquipmentSetEquipmentRefID) []EquipmentSetEquipmentRefID {
	check := make(map[EquipmentSetEquipmentRefID]bool)
	deduped := make([]EquipmentSetEquipmentRefID, 0)
	for _, val := range a {
		check[val] = true
	}
	for _, val := range b {
		check[val] = true
	}
	for val := range check {
		deduped = append(deduped, val)
	}
	return deduped
}
func deduplicateItemBoundToRefIDs(a []ItemBoundToRefID, b []ItemBoundToRefID) []ItemBoundToRefID {
	check := make(map[ItemBoundToRefID]bool)
	deduped := make([]ItemBoundToRefID, 0)
	for _, val := range a {
		check[val] = true
	}
	for _, val := range b {
		check[val] = true
	}
	for val := range check {
		deduped = append(deduped, val)
	}
	return deduped
}
func deduplicatePlayerEquipmentSetRefIDs(a []PlayerEquipmentSetRefID, b []PlayerEquipmentSetRefID) []PlayerEquipmentSetRefID {
	check := make(map[PlayerEquipmentSetRefID]bool)
	deduped := make([]PlayerEquipmentSetRefID, 0)
	for _, val := range a {
		check[val] = true
	}
	for _, val := range b {
		check[val] = true
	}
	for val := range check {
		deduped = append(deduped, val)
	}
	return deduped
}
func deduplicatePlayerGuildMemberRefIDs(a []PlayerGuildMemberRefID, b []PlayerGuildMemberRefID) []PlayerGuildMemberRefID {
	check := make(map[PlayerGuildMemberRefID]bool)
	deduped := make([]PlayerGuildMemberRefID, 0)
	for _, val := range a {
		check[val] = true
	}
	for _, val := range b {
		check[val] = true
	}
	for val := range check {
		deduped = append(deduped, val)
	}
	return deduped
}
func deduplicatePlayerTargetRefIDs(a []PlayerTargetRefID, b []PlayerTargetRefID) []PlayerTargetRefID {
	check := make(map[PlayerTargetRefID]bool)
	deduped := make([]PlayerTargetRefID, 0)
	for _, val := range a {
		check[val] = true
	}
	for _, val := range b {
		check[val] = true
	}
	for val := range check {
		deduped = append(deduped, val)
	}
	return deduped
}
func deduplicatePlayerTargetedByRefIDs(a []PlayerTargetedByRefID, b []PlayerTargetedByRefID) []PlayerTargetedByRefID {
	check := make(map[PlayerTargetedByRefID]bool)
	deduped := make([]PlayerTargetedByRefID, 0)
	for _, val := range a {
		check[val] = true
	}
	for _, val := range b {
		check[val] = true
	}
	for val := range check {
		deduped = append(deduped, val)
	}
	return deduped
}
func (engine Engine) allEquipmentSetEquipmentRefIDs() []EquipmentSetEquipmentRefID {
	var stateEquipmentSetEquipmentRefIDs []EquipmentSetEquipmentRefID
	for equipmentSetEquipmentRefID := range engine.State.EquipmentSetEquipmentRef {
		stateEquipmentSetEquipmentRefIDs = append(stateEquipmentSetEquipmentRefIDs, equipmentSetEquipmentRefID)
	}
	var patchEquipmentSetEquipmentRefIDs []EquipmentSetEquipmentRefID
	for equipmentSetEquipmentRefID := range engine.Patch.EquipmentSetEquipmentRef {
		patchEquipmentSetEquipmentRefIDs = append(patchEquipmentSetEquipmentRefIDs, equipmentSetEquipmentRefID)
	}
	return deduplicateEquipmentSetEquipmentRefIDs(stateEquipmentSetEquipmentRefIDs, patchEquipmentSetEquipmentRefIDs)
}
func (engine Engine) allItemBoundToRefIDs() []ItemBoundToRefID {
	var stateItemBoundToRefIDs []ItemBoundToRefID
	for itemBoundToRefID := range engine.State.ItemBoundToRef {
		stateItemBoundToRefIDs = append(stateItemBoundToRefIDs, itemBoundToRefID)
	}
	var patchItemBoundToRefIDs []ItemBoundToRefID
	for itemBoundToRefID := range engine.Patch.ItemBoundToRef {
		patchItemBoundToRefIDs = append(patchItemBoundToRefIDs, itemBoundToRefID)
	}
	return deduplicateItemBoundToRefIDs(stateItemBoundToRefIDs, patchItemBoundToRefIDs)
}
func (engine Engine) allPlayerEquipmentSetRefIDs() []PlayerEquipmentSetRefID {
	var statePlayerEquipmentSetRefIDs []PlayerEquipmentSetRefID
	for playerEquipmentSetRefID := range engine.State.PlayerEquipmentSetRef {
		statePlayerEquipmentSetRefIDs = append(statePlayerEquipmentSetRefIDs, playerEquipmentSetRefID)
	}
	var patchPlayerEquipmentSetRefIDs []PlayerEquipmentSetRefID
	for playerEquipmentSetRefID := range engine.Patch.PlayerEquipmentSetRef {
		patchPlayerEquipmentSetRefIDs = append(patchPlayerEquipmentSetRefIDs, playerEquipmentSetRefID)
	}
	return deduplicatePlayerEquipmentSetRefIDs(statePlayerEquipmentSetRefIDs, patchPlayerEquipmentSetRefIDs)
}
func (engine Engine) allPlayerGuildMemberRefIDs() []PlayerGuildMemberRefID {
	var statePlayerGuildMemberRefIDs []PlayerGuildMemberRefID
	for playerGuildMemberRefID := range engine.State.PlayerGuildMemberRef {
		statePlayerGuildMemberRefIDs = append(statePlayerGuildMemberRefIDs, playerGuildMemberRefID)
	}
	var patchPlayerGuildMemberRefIDs []PlayerGuildMemberRefID
	for playerGuildMemberRefID := range engine.Patch.PlayerGuildMemberRef {
		patchPlayerGuildMemberRefIDs = append(patchPlayerGuildMemberRefIDs, playerGuildMemberRefID)
	}
	return deduplicatePlayerGuildMemberRefIDs(statePlayerGuildMemberRefIDs, patchPlayerGuildMemberRefIDs)
}
func (engine Engine) allPlayerTargetRefIDs() []PlayerTargetRefID {
	var statePlayerTargetRefIDs []PlayerTargetRefID
	for playerTargetRefID := range engine.State.PlayerTargetRef {
		statePlayerTargetRefIDs = append(statePlayerTargetRefIDs, playerTargetRefID)
	}
	var patchPlayerTargetRefIDs []PlayerTargetRefID
	for playerTargetRefID := range engine.Patch.PlayerTargetRef {
		patchPlayerTargetRefIDs = append(patchPlayerTargetRefIDs, playerTargetRefID)
	}
	return deduplicatePlayerTargetRefIDs(statePlayerTargetRefIDs, patchPlayerTargetRefIDs)
}
func (engine Engine) allPlayerTargetedByRefIDs() []PlayerTargetedByRefID {
	var statePlayerTargetedByRefIDs []PlayerTargetedByRefID
	for playerTargetedByRefID := range engine.State.PlayerTargetedByRef {
		statePlayerTargetedByRefIDs = append(statePlayerTargetedByRefIDs, playerTargetedByRefID)
	}
	var patchPlayerTargetedByRefIDs []PlayerTargetedByRefID
	for playerTargetedByRefID := range engine.Patch.PlayerTargetedByRef {
		patchPlayerTargetedByRefIDs = append(patchPlayerTargetedByRefIDs, playerTargetedByRefID)
	}
	return deduplicatePlayerTargetedByRefIDs(statePlayerTargetedByRefIDs, patchPlayerTargetedByRefIDs)
}
func mergeEquipmentSetIDs(currentIDs, nextIDs []EquipmentSetID) []EquipmentSetID {
	ids := make([]EquipmentSetID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeGearScoreIDs(currentIDs, nextIDs []GearScoreID) []GearScoreID {
	ids := make([]GearScoreID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeItemIDs(currentIDs, nextIDs []ItemID) []ItemID {
	ids := make([]ItemID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergePlayerIDs(currentIDs, nextIDs []PlayerID) []PlayerID {
	ids := make([]PlayerID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergePositionIDs(currentIDs, nextIDs []PositionID) []PositionID {
	ids := make([]PositionID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeZoneIDs(currentIDs, nextIDs []ZoneID) []ZoneID {
	ids := make([]ZoneID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeZoneItemIDs(currentIDs, nextIDs []ZoneItemID) []ZoneItemID {
	ids := make([]ZoneItemID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeEquipmentSetEquipmentRefIDs(currentIDs, nextIDs []EquipmentSetEquipmentRefID) []EquipmentSetEquipmentRefID {
	ids := make([]EquipmentSetEquipmentRefID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeItemBoundToRefIDs(currentIDs, nextIDs []ItemBoundToRefID) []ItemBoundToRefID {
	ids := make([]ItemBoundToRefID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergePlayerEquipmentSetRefIDs(currentIDs, nextIDs []PlayerEquipmentSetRefID) []PlayerEquipmentSetRefID {
	ids := make([]PlayerEquipmentSetRefID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergePlayerGuildMemberRefIDs(currentIDs, nextIDs []PlayerGuildMemberRefID) []PlayerGuildMemberRefID {
	ids := make([]PlayerGuildMemberRefID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergePlayerTargetRefIDs(currentIDs, nextIDs []PlayerTargetRefID) []PlayerTargetRefID {
	ids := make([]PlayerTargetRefID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergePlayerTargetedByRefIDs(currentIDs, nextIDs []PlayerTargetedByRefID) []PlayerTargetedByRefID {
	ids := make([]PlayerTargetedByRefID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeAnyOfPlayer_PositionIDs(currentIDs, nextIDs []AnyOfPlayer_PositionID) []AnyOfPlayer_PositionID {
	ids := make([]AnyOfPlayer_PositionID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeAnyOfPlayer_ZoneItemIDs(currentIDs, nextIDs []AnyOfPlayer_ZoneItemID) []AnyOfPlayer_ZoneItemID {
	ids := make([]AnyOfPlayer_ZoneItemID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}
func mergeAnyOfItem_Player_ZoneItemIDs(currentIDs, nextIDs []AnyOfItem_Player_ZoneItemID) []AnyOfItem_Player_ZoneItemID {
	ids := make([]AnyOfItem_Player_ZoneItemID, len(currentIDs))
	copy(ids, currentIDs)
	var j int
	for _, currentID := range currentIDs {
		if len(nextIDs) <= j || currentID != nextIDs[j] {
			continue
		}
		j += 1
	}
	for _, nextID := range nextIDs[j:] {
		ids = append(ids, nextID)
	}
	return ids
}

type pathTrack struct {
	_iterations  int
	equipmentSet map[EquipmentSetID]path
	gearScore    map[GearScoreID]path
	item         map[ItemID]path
	player       map[PlayerID]path
	position     map[PositionID]path
	zone         map[ZoneID]path
	zoneItem     map[ZoneItemID]path
}

func newPathTrack() pathTrack {
	return pathTrack{equipmentSet: make(map[EquipmentSetID]path), gearScore: make(map[GearScoreID]path), item: make(map[ItemID]path), player: make(map[PlayerID]path), position: make(map[PositionID]path), zone: make(map[ZoneID]path), zoneItem: make(map[ZoneItemID]path)}
}

const (
	equipmentSetIdentifier  int = -1
	gearScoreIdentifier     int = -2
	itemIdentifier          int = -3
	originIdentifier        int = -4
	playerIdentifier        int = -5
	itemsIdentifier         int = -6
	positionIdentifier      int = -7
	zoneIdentifier          int = -8
	interactablesIdentifier int = -9
	playersIdentifier       int = -10
	zoneItemIdentifier      int = -11
)

func (p path) equipmentSet() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, equipmentSetIdentifier)
	return newPath
}
func (p path) gearScore() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, gearScoreIdentifier)
	return newPath
}
func (p path) item() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, itemIdentifier)
	return newPath
}
func (p path) origin() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, originIdentifier)
	return newPath
}
func (p path) player() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, playerIdentifier)
	return newPath
}
func (p path) items() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, itemsIdentifier)
	return newPath
}
func (p path) position() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, positionIdentifier)
	return newPath
}
func (p path) zone() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, zoneIdentifier)
	return newPath
}
func (p path) interactables() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, interactablesIdentifier)
	return newPath
}
func (p path) players() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, playersIdentifier)
	return newPath
}
func (p path) zoneItem() path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, zoneItemIdentifier)
	return newPath
}

type path []int

func newPath(elementIdentifier, id int) path {
	return []int{elementIdentifier, id}
}
func (p path) index(i int) path {
	newPath := make([]int, len(p), len(p)+1)
	copy(newPath, p)
	newPath = append(newPath, i)
	return newPath
}
func (p path) equals(parentPath path) bool {
	if len(p) != len(parentPath) {
		return false
	}
	for i, segment := range parentPath {
		if segment != p[i] {
			return false
		}
	}
	return true
}
func (p path) toJSONPath() string {
	jsonPath := "$"
	for i, seg := range p {
		if seg < 0 {
			jsonPath += "." + pathIdentifierToString(seg)
		} else if i == 1 {
			jsonPath += "." + strconv.Itoa(seg)
		} else {
			jsonPath += "[" + strconv.Itoa(seg) + "]"
		}
	}
	return jsonPath
}
func pathIdentifierToString(identifier int) string {
	switch identifier {
	case equipmentSetIdentifier:
		return "equipmentSet"
	case gearScoreIdentifier:
		return "gearScore"
	case itemIdentifier:
		return "item"
	case originIdentifier:
		return "origin"
	case playerIdentifier:
		return "player"
	case itemsIdentifier:
		return "items"
	case positionIdentifier:
		return "position"
	case zoneIdentifier:
		return "zone"
	case interactablesIdentifier:
		return "interactables"
	case playersIdentifier:
		return "players"
	case zoneItemIdentifier:
		return "zoneItem"
	}
	return ""
}
func (_ref equipmentSetEquipmentRef) Get() item {
	ref := _ref.equipmentSetEquipmentRef.engine.equipmentSetEquipmentRef(_ref.equipmentSetEquipmentRef.ID)
	return ref.equipmentSetEquipmentRef.engine.Item(ref.equipmentSetEquipmentRef.ReferencedElementID)
}
func (_ref itemBoundToRef) IsSet() bool {
	ref := _ref.itemBoundToRef.engine.itemBoundToRef(_ref.itemBoundToRef.ID)
	return ref.itemBoundToRef.ID != 0
}
func (_ref itemBoundToRef) Unset() {
	ref := _ref.itemBoundToRef.engine.itemBoundToRef(_ref.itemBoundToRef.ID)
	ref.itemBoundToRef.engine.deleteItemBoundToRef(ref.itemBoundToRef.ID)
	parent := ref.itemBoundToRef.engine.Item(ref.itemBoundToRef.ParentID).item
	if parent.OperationKind == OperationKindDelete {
		return
	}
	parent.BoundTo = 0
	parent.OperationKind = OperationKindUpdate
	ref.itemBoundToRef.engine.Patch.Item[parent.ID] = parent
}
func (_ref itemBoundToRef) Get() player {
	ref := _ref.itemBoundToRef.engine.itemBoundToRef(_ref.itemBoundToRef.ID)
	return ref.itemBoundToRef.engine.Player(ref.itemBoundToRef.ReferencedElementID)
}
func (_ref playerEquipmentSetRef) Get() equipmentSet {
	ref := _ref.playerEquipmentSetRef.engine.playerEquipmentSetRef(_ref.playerEquipmentSetRef.ID)
	return ref.playerEquipmentSetRef.engine.EquipmentSet(ref.playerEquipmentSetRef.ReferencedElementID)
}
func (_ref playerGuildMemberRef) Get() player {
	ref := _ref.playerGuildMemberRef.engine.playerGuildMemberRef(_ref.playerGuildMemberRef.ID)
	return ref.playerGuildMemberRef.engine.Player(ref.playerGuildMemberRef.ReferencedElementID)
}
func (_ref playerTargetRef) IsSet() bool {
	ref := _ref.playerTargetRef.engine.playerTargetRef(_ref.playerTargetRef.ID)
	return ref.playerTargetRef.ID != 0
}
func (_ref playerTargetRef) Unset() {
	ref := _ref.playerTargetRef.engine.playerTargetRef(_ref.playerTargetRef.ID)
	ref.playerTargetRef.engine.deletePlayerTargetRef(ref.playerTargetRef.ID)
	parent := ref.playerTargetRef.engine.Player(ref.playerTargetRef.ParentID).player
	if parent.OperationKind == OperationKindDelete {
		return
	}
	parent.Target = 0
	parent.OperationKind = OperationKindUpdate
	ref.playerTargetRef.engine.Patch.Player[parent.ID] = parent
}
func (_ref playerTargetRef) Get() anyOfPlayer_ZoneItem {
	ref := _ref.playerTargetRef.engine.playerTargetRef(_ref.playerTargetRef.ID)
	return ref.playerTargetRef.engine.anyOfPlayer_ZoneItem(ref.playerTargetRef.ReferencedElementID)
}
func (_ref playerTargetedByRef) Get() anyOfPlayer_ZoneItem {
	ref := _ref.playerTargetedByRef.engine.playerTargetedByRef(_ref.playerTargetedByRef.ID)
	return ref.playerTargetedByRef.engine.anyOfPlayer_ZoneItem(ref.playerTargetedByRef.ReferencedElementID)
}
func (engine *Engine) dereferenceEquipmentSetEquipmentRefs(itemID ItemID) {
	for _, refID := range engine.allEquipmentSetEquipmentRefIDs() {
		ref := engine.equipmentSetEquipmentRef(refID)
		if ref.equipmentSetEquipmentRef.ReferencedElementID == itemID {
			parent := engine.EquipmentSet(ref.equipmentSetEquipmentRef.ParentID)
			parent.RemoveEquipment(itemID)
		}
	}
}
func (engine *Engine) dereferenceItemBoundToRefs(playerID PlayerID) {
	for _, refID := range engine.allItemBoundToRefIDs() {
		ref := engine.itemBoundToRef(refID)
		if ref.itemBoundToRef.ReferencedElementID == playerID {
			ref.Unset()
		}
	}
}
func (engine *Engine) dereferencePlayerEquipmentSetRefs(equipmentSetID EquipmentSetID) {
	for _, refID := range engine.allPlayerEquipmentSetRefIDs() {
		ref := engine.playerEquipmentSetRef(refID)
		if ref.playerEquipmentSetRef.ReferencedElementID == equipmentSetID {
			parent := engine.Player(ref.playerEquipmentSetRef.ParentID)
			parent.RemoveEquipmentSets(equipmentSetID)
		}
	}
}
func (engine *Engine) dereferencePlayerGuildMemberRefs(playerID PlayerID) {
	for _, refID := range engine.allPlayerGuildMemberRefIDs() {
		ref := engine.playerGuildMemberRef(refID)
		if ref.playerGuildMemberRef.ReferencedElementID == playerID {
			parent := engine.Player(ref.playerGuildMemberRef.ParentID)
			parent.RemoveGuildMembers(playerID)
		}
	}
}
func (engine *Engine) dereferencePlayerTargetRefsPlayer(playerID PlayerID) {
	for _, refID := range engine.allPlayerTargetRefIDs() {
		ref := engine.playerTargetRef(refID)
		anyContainer := ref.Get()
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind != ElementKindPlayer {
			continue
		}
		if anyContainer.anyOfPlayer_ZoneItem.Player == playerID {
			ref.Unset()
		}
	}
}
func (engine *Engine) dereferencePlayerTargetRefsZoneItem(zoneItemID ZoneItemID) {
	for _, refID := range engine.allPlayerTargetRefIDs() {
		ref := engine.playerTargetRef(refID)
		anyContainer := ref.Get()
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind != ElementKindZoneItem {
			continue
		}
		if anyContainer.anyOfPlayer_ZoneItem.ZoneItem == zoneItemID {
			ref.Unset()
		}
	}
}
func (engine *Engine) dereferencePlayerTargetedByRefsPlayer(playerID PlayerID) {
	for _, refID := range engine.allPlayerTargetedByRefIDs() {
		ref := engine.playerTargetedByRef(refID)
		anyContainer := ref.Get()
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind != ElementKindPlayer {
			continue
		}
		if anyContainer.anyOfPlayer_ZoneItem.Player == playerID {
			parent := engine.Player(ref.playerTargetedByRef.ParentID)
			parent.RemoveTargetedByPlayer(playerID)
		}
	}
}
func (engine *Engine) dereferencePlayerTargetedByRefsZoneItem(zoneItemID ZoneItemID) {
	for _, refID := range engine.allPlayerTargetedByRefIDs() {
		ref := engine.playerTargetedByRef(refID)
		anyContainer := ref.Get()
		if anyContainer.anyOfPlayer_ZoneItem.ElementKind != ElementKindZoneItem {
			continue
		}
		if anyContainer.anyOfPlayer_ZoneItem.ZoneItem == zoneItemID {
			parent := engine.Player(ref.playerTargetedByRef.ParentID)
			parent.RemoveTargetedByZoneItem(zoneItemID)
		}
	}
}
func (_equipmentSet equipmentSet) RemoveEquipment(equipmentToRemove ...ItemID) equipmentSet {
	equipmentSet := _equipmentSet.equipmentSet.engine.EquipmentSet(_equipmentSet.equipmentSet.ID)
	if equipmentSet.equipmentSet.OperationKind == OperationKindDelete {
		return equipmentSet
	}
	var wereElementsAltered bool
	var newElements []EquipmentSetEquipmentRefID
	for _, refElement := range equipmentSet.equipmentSet.Equipment {
		element := equipmentSet.equipmentSet.engine.equipmentSetEquipmentRef(refElement).equipmentSetEquipmentRef.ReferencedElementID
		var toBeRemoved bool
		for _, elementToRemove := range equipmentToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				equipmentSet.equipmentSet.engine.deleteEquipmentSetEquipmentRef(refElement)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, refElement)
		}
	}
	if !wereElementsAltered {
		return equipmentSet
	}
	equipmentSet.equipmentSet.Equipment = newElements
	equipmentSet.equipmentSet.OperationKind = OperationKindUpdate
	equipmentSet.equipmentSet.engine.Patch.EquipmentSet[equipmentSet.equipmentSet.ID] = equipmentSet.equipmentSet
	return equipmentSet
}
func (_player player) RemoveEquipmentSets(equipmentSetsToRemove ...EquipmentSetID) player {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return player
	}
	var wereElementsAltered bool
	var newElements []PlayerEquipmentSetRefID
	for _, refElement := range player.player.EquipmentSets {
		element := player.player.engine.playerEquipmentSetRef(refElement).playerEquipmentSetRef.ReferencedElementID
		var toBeRemoved bool
		for _, elementToRemove := range equipmentSetsToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				player.player.engine.deletePlayerEquipmentSetRef(refElement)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, refElement)
		}
	}
	if !wereElementsAltered {
		return player
	}
	player.player.EquipmentSets = newElements
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return player
}
func (_player player) RemoveGuildMembers(guildMembersToRemove ...PlayerID) player {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return player
	}
	var wereElementsAltered bool
	var newElements []PlayerGuildMemberRefID
	for _, refElement := range player.player.GuildMembers {
		element := player.player.engine.playerGuildMemberRef(refElement).playerGuildMemberRef.ReferencedElementID
		var toBeRemoved bool
		for _, elementToRemove := range guildMembersToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				player.player.engine.deletePlayerGuildMemberRef(refElement)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, refElement)
		}
	}
	if !wereElementsAltered {
		return player
	}
	player.player.GuildMembers = newElements
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return player
}
func (_player player) RemoveItems(itemsToRemove ...ItemID) player {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return player
	}
	var wereElementsAltered bool
	var newElements []ItemID
	for _, element := range player.player.Items {
		var toBeRemoved bool
		for _, elementToRemove := range itemsToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				player.player.engine.deleteItem(element)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, element)
		}
	}
	if !wereElementsAltered {
		return player
	}
	player.player.Items = newElements
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return player
}
func (_player player) RemoveTargetedByPlayer(playersToRemove ...PlayerID) player {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return player
	}
	var wereElementsAltered bool
	var newElements []PlayerTargetedByRefID
	for _, refElement := range player.player.TargetedBy {
		anyContainer := player.player.engine.playerTargetedByRef(refElement).Get()
		element := anyContainer.Player().ID()
		if element == 0 {
			continue
		}
		var toBeRemoved bool
		for _, elementToRemove := range playersToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				player.player.engine.deletePlayerTargetedByRef(refElement)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, refElement)
		}
	}
	if !wereElementsAltered {
		return player
	}
	player.player.TargetedBy = newElements
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return player
}
func (_player player) RemoveTargetedByZoneItem(zoneItemsToRemove ...ZoneItemID) player {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return player
	}
	var wereElementsAltered bool
	var newElements []PlayerTargetedByRefID
	for _, refElement := range player.player.TargetedBy {
		anyContainer := player.player.engine.playerTargetedByRef(refElement).Get()
		element := anyContainer.ZoneItem().ID()
		if element == 0 {
			continue
		}
		var toBeRemoved bool
		for _, elementToRemove := range zoneItemsToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				player.player.engine.deletePlayerTargetedByRef(refElement)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, refElement)
		}
	}
	if !wereElementsAltered {
		return player
	}
	player.player.TargetedBy = newElements
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return player
}
func (_zone zone) RemoveInteractablesItem(itemsToRemove ...ItemID) zone {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zone
	}
	var wereElementsAltered bool
	var newElements []AnyOfItem_Player_ZoneItemID
	for _, anyContainerID := range zone.zone.Interactables {
		anyContainer := zone.zone.engine.anyOfItem_Player_ZoneItem(anyContainerID)
		element := anyContainer.Item().ID()
		if element == 0 {
			continue
		}
		var toBeRemoved bool
		for _, elementToRemove := range itemsToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				zone.zone.engine.deleteItem(element)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, anyContainer.anyOfItem_Player_ZoneItem.ID)
		}
	}
	if !wereElementsAltered {
		return zone
	}
	zone.zone.Interactables = newElements
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}
func (_zone zone) RemoveInteractablesPlayer(playersToRemove ...PlayerID) zone {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zone
	}
	var wereElementsAltered bool
	var newElements []AnyOfItem_Player_ZoneItemID
	for _, anyContainerID := range zone.zone.Interactables {
		anyContainer := zone.zone.engine.anyOfItem_Player_ZoneItem(anyContainerID)
		element := anyContainer.Player().ID()
		if element == 0 {
			continue
		}
		var toBeRemoved bool
		for _, elementToRemove := range playersToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				zone.zone.engine.deletePlayer(element)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, anyContainer.anyOfItem_Player_ZoneItem.ID)
		}
	}
	if !wereElementsAltered {
		return zone
	}
	zone.zone.Interactables = newElements
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}
func (_zone zone) RemoveInteractablesZoneItem(zoneItemsToRemove ...ZoneItemID) zone {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zone
	}
	var wereElementsAltered bool
	var newElements []AnyOfItem_Player_ZoneItemID
	for _, anyContainerID := range zone.zone.Interactables {
		anyContainer := zone.zone.engine.anyOfItem_Player_ZoneItem(anyContainerID)
		element := anyContainer.ZoneItem().ID()
		if element == 0 {
			continue
		}
		var toBeRemoved bool
		for _, elementToRemove := range zoneItemsToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				zone.zone.engine.deleteZoneItem(element)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, anyContainer.anyOfItem_Player_ZoneItem.ID)
		}
	}
	if !wereElementsAltered {
		return zone
	}
	zone.zone.Interactables = newElements
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}
func (_zone zone) RemoveItems(itemsToRemove ...ZoneItemID) zone {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zone
	}
	var wereElementsAltered bool
	var newElements []ZoneItemID
	for _, element := range zone.zone.Items {
		var toBeRemoved bool
		for _, elementToRemove := range itemsToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				zone.zone.engine.deleteZoneItem(element)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, element)
		}
	}
	if !wereElementsAltered {
		return zone
	}
	zone.zone.Items = newElements
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}
func (_zone zone) RemovePlayers(playersToRemove ...PlayerID) zone {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zone
	}
	var wereElementsAltered bool
	var newElements []PlayerID
	for _, element := range zone.zone.Players {
		var toBeRemoved bool
		for _, elementToRemove := range playersToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				zone.zone.engine.deletePlayer(element)
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, element)
		}
	}
	if !wereElementsAltered {
		return zone
	}
	zone.zone.Players = newElements
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}
func (_zone zone) RemoveTags(tagsToRemove ...string) zone {
	zone := _zone.zone.engine.Zone(_zone.zone.ID)
	if zone.zone.OperationKind == OperationKindDelete {
		return zone
	}
	var wereElementsAltered bool
	var newElements []string
	for _, element := range zone.zone.Tags {
		var toBeRemoved bool
		for _, elementToRemove := range tagsToRemove {
			if element == elementToRemove {
				toBeRemoved = true
				wereElementsAltered = true
				break
			}
		}
		if !toBeRemoved {
			newElements = append(newElements, element)
		}
	}
	if !wereElementsAltered {
		return zone
	}
	zone.zone.Tags = newElements
	zone.zone.OperationKind = OperationKindUpdate
	zone.zone.engine.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}
func (_equipmentSet equipmentSet) SetName(newName string) equipmentSet {
	equipmentSet := _equipmentSet.equipmentSet.engine.EquipmentSet(_equipmentSet.equipmentSet.ID)
	if equipmentSet.equipmentSet.OperationKind == OperationKindDelete {
		return equipmentSet
	}
	equipmentSet.equipmentSet.Name = newName
	equipmentSet.equipmentSet.OperationKind = OperationKindUpdate
	equipmentSet.equipmentSet.engine.Patch.EquipmentSet[equipmentSet.equipmentSet.ID] = equipmentSet.equipmentSet
	return equipmentSet
}
func (_gearScore gearScore) SetLevel(newLevel int) gearScore {
	gearScore := _gearScore.gearScore.engine.GearScore(_gearScore.gearScore.ID)
	if gearScore.gearScore.OperationKind == OperationKindDelete {
		return gearScore
	}
	gearScore.gearScore.Level = newLevel
	gearScore.gearScore.OperationKind = OperationKindUpdate
	gearScore.gearScore.engine.Patch.GearScore[gearScore.gearScore.ID] = gearScore.gearScore
	return gearScore
}
func (_gearScore gearScore) SetScore(newScore int) gearScore {
	gearScore := _gearScore.gearScore.engine.GearScore(_gearScore.gearScore.ID)
	if gearScore.gearScore.OperationKind == OperationKindDelete {
		return gearScore
	}
	gearScore.gearScore.Score = newScore
	gearScore.gearScore.OperationKind = OperationKindUpdate
	gearScore.gearScore.engine.Patch.GearScore[gearScore.gearScore.ID] = gearScore.gearScore
	return gearScore
}
func (_item item) SetName(newName string) item {
	item := _item.item.engine.Item(_item.item.ID)
	if item.item.OperationKind == OperationKindDelete {
		return item
	}
	item.item.Name = newName
	item.item.OperationKind = OperationKindUpdate
	item.item.engine.Patch.Item[item.item.ID] = item.item
	return item
}
func (_position position) SetX(newX float64) position {
	position := _position.position.engine.Position(_position.position.ID)
	if position.position.OperationKind == OperationKindDelete {
		return position
	}
	position.position.X = newX
	position.position.OperationKind = OperationKindUpdate
	position.position.engine.Patch.Position[position.position.ID] = position.position
	return position
}
func (_position position) SetY(newY float64) position {
	position := _position.position.engine.Position(_position.position.ID)
	if position.position.OperationKind == OperationKindDelete {
		return position
	}
	position.position.Y = newY
	position.position.OperationKind = OperationKindUpdate
	position.position.engine.Patch.Position[position.position.ID] = position.position
	return position
}
func (_item item) SetBoundTo(playerID PlayerID) item {
	item := _item.item.engine.Item(_item.item.ID)
	if item.item.OperationKind == OperationKindDelete {
		return item
	}
	if item.item.engine.Player(playerID).player.OperationKind == OperationKindDelete {
		return item
	}
	if item.item.BoundTo != 0 {
		item.item.engine.deleteItemBoundToRef(item.item.BoundTo)
	}
	ref := item.item.engine.createItemBoundToRef(playerID, item.item.ID)
	item.item.BoundTo = ref.ID
	item.item.OperationKind = OperationKindUpdate
	item.item.engine.Patch.Item[item.item.ID] = item.item
	return item
}
func (_player player) SetTargetPlayer(playerID PlayerID) player {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return player
	}
	if player.player.engine.Player(playerID).player.OperationKind == OperationKindDelete {
		return player
	}
	if player.player.Target != 0 {
		player.player.engine.deletePlayerTargetRef(player.player.Target)
	}
	anyContainer := player.player.engine.createAnyOfPlayer_ZoneItem(false)
	anyContainer.anyOfPlayer_ZoneItem.setPlayer(playerID)
	ref := player.player.engine.createPlayerTargetRef(anyContainer.anyOfPlayer_ZoneItem.ID, player.player.ID)
	player.player.Target = ref.ID
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return player
}
func (_player player) SetTargetZoneItem(zoneItemID ZoneItemID) player {
	player := _player.player.engine.Player(_player.player.ID)
	if player.player.OperationKind == OperationKindDelete {
		return player
	}
	if player.player.engine.ZoneItem(zoneItemID).zoneItem.OperationKind == OperationKindDelete {
		return player
	}
	if player.player.Target != 0 {
		player.player.engine.deletePlayerTargetRef(player.player.Target)
	}
	anyContainer := player.player.engine.createAnyOfPlayer_ZoneItem(false)
	anyContainer.anyOfPlayer_ZoneItem.setZoneItem(zoneItemID)
	ref := player.player.engine.createPlayerTargetRef(anyContainer.anyOfPlayer_ZoneItem.ID, player.player.ID)
	player.player.Target = ref.ID
	player.player.OperationKind = OperationKindUpdate
	player.player.engine.Patch.Player[player.player.ID] = player.player
	return player
}

type EquipmentSetID int
type GearScoreID int
type ItemID int
type PlayerID int
type PositionID int
type ZoneID int
type ZoneItemID int
type EquipmentSetEquipmentRefID int
type ItemBoundToRefID int
type PlayerEquipmentSetRefID int
type PlayerGuildMemberRefID int
type PlayerTargetRefID int
type PlayerTargetedByRefID int
type AnyOfPlayer_PositionID int
type AnyOfPlayer_ZoneItemID int
type AnyOfItem_Player_ZoneItemID int
type State struct {
	EquipmentSet              map[EquipmentSetID]equipmentSetCore                           `json:"equipmentSet"`
	GearScore                 map[GearScoreID]gearScoreCore                                 `json:"gearScore"`
	Item                      map[ItemID]itemCore                                           `json:"item"`
	Player                    map[PlayerID]playerCore                                       `json:"player"`
	Position                  map[PositionID]positionCore                                   `json:"position"`
	Zone                      map[ZoneID]zoneCore                                           `json:"zone"`
	ZoneItem                  map[ZoneItemID]zoneItemCore                                   `json:"zoneItem"`
	EquipmentSetEquipmentRef  map[EquipmentSetEquipmentRefID]equipmentSetEquipmentRefCore   `json:"equipmentSetEquipmentRef"`
	ItemBoundToRef            map[ItemBoundToRefID]itemBoundToRefCore                       `json:"itemBoundToRef"`
	PlayerEquipmentSetRef     map[PlayerEquipmentSetRefID]playerEquipmentSetRefCore         `json:"playerEquipmentSetRef"`
	PlayerGuildMemberRef      map[PlayerGuildMemberRefID]playerGuildMemberRefCore           `json:"playerGuildMemberRef"`
	PlayerTargetRef           map[PlayerTargetRefID]playerTargetRefCore                     `json:"playerTargetRef"`
	PlayerTargetedByRef       map[PlayerTargetedByRefID]playerTargetedByRefCore             `json:"playerTargetedByRef"`
	AnyOfPlayer_Position      map[AnyOfPlayer_PositionID]anyOfPlayer_PositionCore           `json:"anyOfPlayer_Position"`
	AnyOfPlayer_ZoneItem      map[AnyOfPlayer_ZoneItemID]anyOfPlayer_ZoneItemCore           `json:"anyOfPlayer_ZoneItem"`
	AnyOfItem_Player_ZoneItem map[AnyOfItem_Player_ZoneItemID]anyOfItem_Player_ZoneItemCore `json:"anyOfItem_Player_ZoneItem"`
}

func newState() State {
	return State{EquipmentSet: make(map[EquipmentSetID]equipmentSetCore), GearScore: make(map[GearScoreID]gearScoreCore), Item: make(map[ItemID]itemCore), Player: make(map[PlayerID]playerCore), Position: make(map[PositionID]positionCore), Zone: make(map[ZoneID]zoneCore), ZoneItem: make(map[ZoneItemID]zoneItemCore), EquipmentSetEquipmentRef: make(map[EquipmentSetEquipmentRefID]equipmentSetEquipmentRefCore), ItemBoundToRef: make(map[ItemBoundToRefID]itemBoundToRefCore), PlayerEquipmentSetRef: make(map[PlayerEquipmentSetRefID]playerEquipmentSetRefCore), PlayerGuildMemberRef: make(map[PlayerGuildMemberRefID]playerGuildMemberRefCore), PlayerTargetRef: make(map[PlayerTargetRefID]playerTargetRefCore), PlayerTargetedByRef: make(map[PlayerTargetedByRefID]playerTargetedByRefCore), AnyOfPlayer_Position: make(map[AnyOfPlayer_PositionID]anyOfPlayer_PositionCore), AnyOfPlayer_ZoneItem: make(map[AnyOfPlayer_ZoneItemID]anyOfPlayer_ZoneItemCore), AnyOfItem_Player_ZoneItem: make(map[AnyOfItem_Player_ZoneItemID]anyOfItem_Player_ZoneItemCore)}
}

type equipmentSetCore struct {
	ID            EquipmentSetID               `json:"id"`
	Equipment     []EquipmentSetEquipmentRefID `json:"equipment"`
	Name          string                       `json:"name"`
	OperationKind OperationKind                `json:"operationKind"`
	engine        *Engine
}
type equipmentSet struct{ equipmentSet equipmentSetCore }
type gearScoreCore struct {
	ID            GearScoreID   `json:"id"`
	Level         int           `json:"level"`
	Score         int           `json:"score"`
	OperationKind OperationKind `json:"operationKind"`
	HasParent     bool          `json:"hasParent"`
	engine        *Engine
}
type gearScore struct{ gearScore gearScoreCore }
type itemCore struct {
	ID            ItemID                 `json:"id"`
	BoundTo       ItemBoundToRefID       `json:"boundTo"`
	GearScore     GearScoreID            `json:"gearScore"`
	Name          string                 `json:"name"`
	Origin        AnyOfPlayer_PositionID `json:"origin"`
	OperationKind OperationKind          `json:"operationKind"`
	HasParent     bool                   `json:"hasParent"`
	engine        *Engine
}
type item struct{ item itemCore }
type playerCore struct {
	ID            PlayerID                  `json:"id"`
	EquipmentSets []PlayerEquipmentSetRefID `json:"equipmentSets"`
	GearScore     GearScoreID               `json:"gearScore"`
	GuildMembers  []PlayerGuildMemberRefID  `json:"guildMembers"`
	Items         []ItemID                  `json:"items"`
	Position      PositionID                `json:"position"`
	Target        PlayerTargetRefID         `json:"target"`
	TargetedBy    []PlayerTargetedByRefID   `json:"targetedBy"`
	OperationKind OperationKind             `json:"operationKind"`
	HasParent     bool                      `json:"hasParent"`
	engine        *Engine
}
type player struct{ player playerCore }
type positionCore struct {
	ID            PositionID    `json:"id"`
	X             float64       `json:"x"`
	Y             float64       `json:"y"`
	OperationKind OperationKind `json:"operationKind"`
	HasParent     bool          `json:"hasParent"`
	engine        *Engine
}
type position struct{ position positionCore }
type zoneCore struct {
	ID            ZoneID                        `json:"id"`
	Interactables []AnyOfItem_Player_ZoneItemID `json:"interactables"`
	Items         []ZoneItemID                  `json:"items"`
	Players       []PlayerID                    `json:"players"`
	Tags          []string                      `json:"tags"`
	OperationKind OperationKind                 `json:"operationKind"`
	engine        *Engine
}
type zone struct{ zone zoneCore }
type zoneItemCore struct {
	ID            ZoneItemID    `json:"id"`
	Item          ItemID        `json:"item"`
	Position      PositionID    `json:"position"`
	OperationKind OperationKind `json:"operationKind"`
	HasParent     bool          `json:"hasParent"`
	engine        *Engine
}
type zoneItem struct{ zoneItem zoneItemCore }
type equipmentSetEquipmentRefCore struct {
	ID                  EquipmentSetEquipmentRefID `json:"id"`
	ParentID            EquipmentSetID             `json:"parentID"`
	ReferencedElementID ItemID                     `json:"referencedElementID"`
	OperationKind       OperationKind              `json:"operationKind"`
	engine              *Engine
}
type equipmentSetEquipmentRef struct{ equipmentSetEquipmentRef equipmentSetEquipmentRefCore }
type itemBoundToRefCore struct {
	ID                  ItemBoundToRefID `json:"id"`
	ParentID            ItemID           `json:"parentID"`
	ReferencedElementID PlayerID         `json:"referencedElementID"`
	OperationKind       OperationKind    `json:"operationKind"`
	engine              *Engine
}
type itemBoundToRef struct{ itemBoundToRef itemBoundToRefCore }
type playerEquipmentSetRefCore struct {
	ID                  PlayerEquipmentSetRefID `json:"id"`
	ParentID            PlayerID                `json:"parentID"`
	ReferencedElementID EquipmentSetID          `json:"referencedElementID"`
	OperationKind       OperationKind           `json:"operationKind"`
	engine              *Engine
}
type playerEquipmentSetRef struct{ playerEquipmentSetRef playerEquipmentSetRefCore }
type playerGuildMemberRefCore struct {
	ID                  PlayerGuildMemberRefID `json:"id"`
	ParentID            PlayerID               `json:"parentID"`
	ReferencedElementID PlayerID               `json:"referencedElementID"`
	OperationKind       OperationKind          `json:"operationKind"`
	engine              *Engine
}
type playerGuildMemberRef struct{ playerGuildMemberRef playerGuildMemberRefCore }
type playerTargetRefCore struct {
	ID                  PlayerTargetRefID      `json:"id"`
	ParentID            PlayerID               `json:"parentID"`
	ReferencedElementID AnyOfPlayer_ZoneItemID `json:"referencedElementID"`
	OperationKind       OperationKind          `json:"operationKind"`
	engine              *Engine
}
type playerTargetRef struct{ playerTargetRef playerTargetRefCore }
type playerTargetedByRefCore struct {
	ID                  PlayerTargetedByRefID  `json:"id"`
	ParentID            PlayerID               `json:"parentID"`
	ReferencedElementID AnyOfPlayer_ZoneItemID `json:"referencedElementID"`
	OperationKind       OperationKind          `json:"operationKind"`
	engine              *Engine
}
type playerTargetedByRef struct{ playerTargetedByRef playerTargetedByRefCore }
type anyOfPlayer_PositionCore struct {
	ID            AnyOfPlayer_PositionID `json:"id"`
	ElementKind   ElementKind            `json:"elementKind"`
	Player        PlayerID               `json:"player"`
	Position      PositionID             `json:"position"`
	OperationKind OperationKind          `json:"operationKind"`
	engine        *Engine
}
type anyOfPlayer_Position struct{ anyOfPlayer_Position anyOfPlayer_PositionCore }
type anyOfPlayer_ZoneItemCore struct {
	ID            AnyOfPlayer_ZoneItemID `json:"id"`
	ElementKind   ElementKind            `json:"elementKind"`
	Player        PlayerID               `json:"player"`
	ZoneItem      ZoneItemID             `json:"zoneItem"`
	OperationKind OperationKind          `json:"operationKind"`
	engine        *Engine
}
type anyOfPlayer_ZoneItem struct{ anyOfPlayer_ZoneItem anyOfPlayer_ZoneItemCore }
type anyOfItem_Player_ZoneItemCore struct {
	ID            AnyOfItem_Player_ZoneItemID `json:"id"`
	ElementKind   ElementKind                 `json:"elementKind"`
	Item          ItemID                      `json:"item"`
	Player        PlayerID                    `json:"player"`
	ZoneItem      ZoneItemID                  `json:"zoneItem"`
	OperationKind OperationKind               `json:"operationKind"`
	engine        *Engine
}
type anyOfItem_Player_ZoneItem struct{ anyOfItem_Player_ZoneItem anyOfItem_Player_ZoneItemCore }
type OperationKind string

const (
	OperationKindDelete    OperationKind = "DELETE"
	OperationKindUpdate    OperationKind = "UPDATE"
	OperationKindUnchanged OperationKind = "UNCHANGED"
)

type Engine struct {
	State     State
	Patch     State
	Tree      Tree
	PathTrack pathTrack
	IDgen     int
}

func newEngine() *Engine {
	return &Engine{IDgen: 1, Patch: newState(), PathTrack: newPathTrack(), State: newState(), Tree: newTree()}
}
func (engine *Engine) GenerateID() int {
	newID := engine.IDgen
	engine.IDgen = engine.IDgen + 1
	return newID
}
func (engine *Engine) UpdateState() {
	for _, equipmentSet := range engine.Patch.EquipmentSet {
		if equipmentSet.OperationKind == OperationKindDelete {
			delete(engine.State.EquipmentSet, equipmentSet.ID)
		} else {
			equipmentSet.OperationKind = OperationKindUnchanged
			engine.State.EquipmentSet[equipmentSet.ID] = equipmentSet
		}
	}
	for _, gearScore := range engine.Patch.GearScore {
		if gearScore.OperationKind == OperationKindDelete {
			delete(engine.State.GearScore, gearScore.ID)
		} else {
			gearScore.OperationKind = OperationKindUnchanged
			engine.State.GearScore[gearScore.ID] = gearScore
		}
	}
	for _, item := range engine.Patch.Item {
		if item.OperationKind == OperationKindDelete {
			delete(engine.State.Item, item.ID)
		} else {
			item.OperationKind = OperationKindUnchanged
			engine.State.Item[item.ID] = item
		}
	}
	for _, player := range engine.Patch.Player {
		if player.OperationKind == OperationKindDelete {
			delete(engine.State.Player, player.ID)
		} else {
			player.OperationKind = OperationKindUnchanged
			engine.State.Player[player.ID] = player
		}
	}
	for _, position := range engine.Patch.Position {
		if position.OperationKind == OperationKindDelete {
			delete(engine.State.Position, position.ID)
		} else {
			position.OperationKind = OperationKindUnchanged
			engine.State.Position[position.ID] = position
		}
	}
	for _, zone := range engine.Patch.Zone {
		if zone.OperationKind == OperationKindDelete {
			delete(engine.State.Zone, zone.ID)
		} else {
			zone.OperationKind = OperationKindUnchanged
			engine.State.Zone[zone.ID] = zone
		}
	}
	for _, zoneItem := range engine.Patch.ZoneItem {
		if zoneItem.OperationKind == OperationKindDelete {
			delete(engine.State.ZoneItem, zoneItem.ID)
		} else {
			zoneItem.OperationKind = OperationKindUnchanged
			engine.State.ZoneItem[zoneItem.ID] = zoneItem
		}
	}
	for _, equipmentSetEquipmentRef := range engine.Patch.EquipmentSetEquipmentRef {
		if equipmentSetEquipmentRef.OperationKind == OperationKindDelete {
			delete(engine.State.EquipmentSetEquipmentRef, equipmentSetEquipmentRef.ID)
		} else {
			equipmentSetEquipmentRef.OperationKind = OperationKindUnchanged
			engine.State.EquipmentSetEquipmentRef[equipmentSetEquipmentRef.ID] = equipmentSetEquipmentRef
		}
	}
	for _, itemBoundToRef := range engine.Patch.ItemBoundToRef {
		if itemBoundToRef.OperationKind == OperationKindDelete {
			delete(engine.State.ItemBoundToRef, itemBoundToRef.ID)
		} else {
			itemBoundToRef.OperationKind = OperationKindUnchanged
			engine.State.ItemBoundToRef[itemBoundToRef.ID] = itemBoundToRef
		}
	}
	for _, playerEquipmentSetRef := range engine.Patch.PlayerEquipmentSetRef {
		if playerEquipmentSetRef.OperationKind == OperationKindDelete {
			delete(engine.State.PlayerEquipmentSetRef, playerEquipmentSetRef.ID)
		} else {
			playerEquipmentSetRef.OperationKind = OperationKindUnchanged
			engine.State.PlayerEquipmentSetRef[playerEquipmentSetRef.ID] = playerEquipmentSetRef
		}
	}
	for _, playerGuildMemberRef := range engine.Patch.PlayerGuildMemberRef {
		if playerGuildMemberRef.OperationKind == OperationKindDelete {
			delete(engine.State.PlayerGuildMemberRef, playerGuildMemberRef.ID)
		} else {
			playerGuildMemberRef.OperationKind = OperationKindUnchanged
			engine.State.PlayerGuildMemberRef[playerGuildMemberRef.ID] = playerGuildMemberRef
		}
	}
	for _, playerTargetRef := range engine.Patch.PlayerTargetRef {
		if playerTargetRef.OperationKind == OperationKindDelete {
			delete(engine.State.PlayerTargetRef, playerTargetRef.ID)
		} else {
			playerTargetRef.OperationKind = OperationKindUnchanged
			engine.State.PlayerTargetRef[playerTargetRef.ID] = playerTargetRef
		}
	}
	for _, playerTargetedByRef := range engine.Patch.PlayerTargetedByRef {
		if playerTargetedByRef.OperationKind == OperationKindDelete {
			delete(engine.State.PlayerTargetedByRef, playerTargetedByRef.ID)
		} else {
			playerTargetedByRef.OperationKind = OperationKindUnchanged
			engine.State.PlayerTargetedByRef[playerTargetedByRef.ID] = playerTargetedByRef
		}
	}
	for _, anyOfPlayer_Position := range engine.Patch.AnyOfPlayer_Position {
		if anyOfPlayer_Position.OperationKind == OperationKindDelete {
			delete(engine.State.AnyOfPlayer_Position, anyOfPlayer_Position.ID)
		} else {
			anyOfPlayer_Position.OperationKind = OperationKindUnchanged
			engine.State.AnyOfPlayer_Position[anyOfPlayer_Position.ID] = anyOfPlayer_Position
		}
	}
	for _, anyOfPlayer_ZoneItem := range engine.Patch.AnyOfPlayer_ZoneItem {
		if anyOfPlayer_ZoneItem.OperationKind == OperationKindDelete {
			delete(engine.State.AnyOfPlayer_ZoneItem, anyOfPlayer_ZoneItem.ID)
		} else {
			anyOfPlayer_ZoneItem.OperationKind = OperationKindUnchanged
			engine.State.AnyOfPlayer_ZoneItem[anyOfPlayer_ZoneItem.ID] = anyOfPlayer_ZoneItem
		}
	}
	for _, anyOfItem_Player_ZoneItem := range engine.Patch.AnyOfItem_Player_ZoneItem {
		if anyOfItem_Player_ZoneItem.OperationKind == OperationKindDelete {
			delete(engine.State.AnyOfItem_Player_ZoneItem, anyOfItem_Player_ZoneItem.ID)
		} else {
			anyOfItem_Player_ZoneItem.OperationKind = OperationKindUnchanged
			engine.State.AnyOfItem_Player_ZoneItem[anyOfItem_Player_ZoneItem.ID] = anyOfItem_Player_ZoneItem
		}
	}
	for key := range engine.Patch.EquipmentSet {
		delete(engine.Patch.EquipmentSet, key)
	}
	for key := range engine.Patch.GearScore {
		delete(engine.Patch.GearScore, key)
	}
	for key := range engine.Patch.Item {
		delete(engine.Patch.Item, key)
	}
	for key := range engine.Patch.Player {
		delete(engine.Patch.Player, key)
	}
	for key := range engine.Patch.Position {
		delete(engine.Patch.Position, key)
	}
	for key := range engine.Patch.Zone {
		delete(engine.Patch.Zone, key)
	}
	for key := range engine.Patch.ZoneItem {
		delete(engine.Patch.ZoneItem, key)
	}
	for key := range engine.Patch.EquipmentSetEquipmentRef {
		delete(engine.Patch.EquipmentSetEquipmentRef, key)
	}
	for key := range engine.Patch.ItemBoundToRef {
		delete(engine.Patch.ItemBoundToRef, key)
	}
	for key := range engine.Patch.PlayerEquipmentSetRef {
		delete(engine.Patch.PlayerEquipmentSetRef, key)
	}
	for key := range engine.Patch.PlayerGuildMemberRef {
		delete(engine.Patch.PlayerGuildMemberRef, key)
	}
	for key := range engine.Patch.PlayerTargetRef {
		delete(engine.Patch.PlayerTargetRef, key)
	}
	for key := range engine.Patch.PlayerTargetedByRef {
		delete(engine.Patch.PlayerTargetedByRef, key)
	}
	for key := range engine.Patch.AnyOfPlayer_Position {
		delete(engine.Patch.AnyOfPlayer_Position, key)
	}
	for key := range engine.Patch.AnyOfPlayer_ZoneItem {
		delete(engine.Patch.AnyOfPlayer_ZoneItem, key)
	}
	for key := range engine.Patch.AnyOfItem_Player_ZoneItem {
		delete(engine.Patch.AnyOfItem_Player_ZoneItem, key)
	}
}

type ReferencedDataStatus string

const (
	ReferencedDataModified  ReferencedDataStatus = "MODIFIED"
	ReferencedDataUnchanged ReferencedDataStatus = "UNCHANGED"
)

type ElementKind string

const (
	ElementKindEquipmentSet ElementKind = "EquipmentSet"
	ElementKindGearScore    ElementKind = "GearScore"
	ElementKindItem         ElementKind = "Item"
	ElementKindPlayer       ElementKind = "Player"
	ElementKindPosition     ElementKind = "Position"
	ElementKindZone         ElementKind = "Zone"
	ElementKindZoneItem     ElementKind = "ZoneItem"
)

type Tree struct {
	EquipmentSet map[EquipmentSetID]EquipmentSet `json:"equipmentSet"`
	GearScore    map[GearScoreID]GearScore       `json:"gearScore"`
	Item         map[ItemID]Item                 `json:"item"`
	Player       map[PlayerID]Player             `json:"player"`
	Position     map[PositionID]Position         `json:"position"`
	Zone         map[ZoneID]Zone                 `json:"zone"`
	ZoneItem     map[ZoneItemID]ZoneItem         `json:"zoneItem"`
}

func newTree() Tree {
	return Tree{EquipmentSet: make(map[EquipmentSetID]EquipmentSet), GearScore: make(map[GearScoreID]GearScore), Item: make(map[ItemID]Item), Player: make(map[PlayerID]Player), Position: make(map[PositionID]Position), Zone: make(map[ZoneID]Zone), ZoneItem: make(map[ZoneItemID]ZoneItem)}
}

type EquipmentSet struct {
	ID            EquipmentSetID  `json:"id"`
	Equipment     []ItemReference `json:"equipment"`
	Name          string          `json:"name"`
	OperationKind OperationKind   `json:"operationKind"`
}
type EquipmentSetReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            EquipmentSetID       `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	EquipmentSet         *EquipmentSet        `json:"equipmentSet"`
}
type GearScore struct {
	ID            GearScoreID   `json:"id"`
	Level         int           `json:"level"`
	Score         int           `json:"score"`
	OperationKind OperationKind `json:"operationKind"`
}
type GearScoreReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            GearScoreID          `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	GearScore            *GearScore           `json:"gearScore"`
}
type Item struct {
	ID            ItemID           `json:"id"`
	BoundTo       *PlayerReference `json:"boundTo"`
	GearScore     *GearScore       `json:"gearScore"`
	Name          string           `json:"name"`
	Origin        interface{}      `json:"origin"`
	OperationKind OperationKind    `json:"operationKind"`
}
type ItemReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            ItemID               `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	Item                 *Item                `json:"item"`
}
type Player struct {
	ID            PlayerID                        `json:"id"`
	EquipmentSets []EquipmentSetReference         `json:"equipmentSets"`
	GearScore     *GearScore                      `json:"gearScore"`
	GuildMembers  []PlayerReference               `json:"guildMembers"`
	Items         []Item                          `json:"items"`
	Position      *Position                       `json:"position"`
	Target        *AnyOfPlayer_ZoneItemReference  `json:"target"`
	TargetedBy    []AnyOfPlayer_ZoneItemReference `json:"targetedBy"`
	OperationKind OperationKind                   `json:"operationKind"`
}
type PlayerReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            PlayerID             `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	Player               *Player              `json:"player"`
}
type Position struct {
	ID            PositionID    `json:"id"`
	X             float64       `json:"x"`
	Y             float64       `json:"y"`
	OperationKind OperationKind `json:"operationKind"`
}
type PositionReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            PositionID           `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	Position             *Position            `json:"position"`
}
type Zone struct {
	ID            ZoneID        `json:"id"`
	Interactables []interface{} `json:"interactables"`
	Items         []ZoneItem    `json:"items"`
	Players       []Player      `json:"players"`
	Tags          []string      `json:"tags"`
	OperationKind OperationKind `json:"operationKind"`
}
type ZoneReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            ZoneID               `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	Zone                 *Zone                `json:"zone"`
}
type ZoneItem struct {
	ID            ZoneItemID    `json:"id"`
	Item          *Item         `json:"item"`
	Position      *Position     `json:"position"`
	OperationKind OperationKind `json:"operationKind"`
}
type ZoneItemReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            ZoneItemID           `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	ZoneItem             *ZoneItem            `json:"zoneItem"`
}
type AnyOfPlayer_ZoneItemReference struct {
	OperationKind        OperationKind        `json:"operationKind"`
	ElementID            int                  `json:"id"`
	ElementKind          ElementKind          `json:"elementKind"`
	ReferencedDataStatus ReferencedDataStatus `json:"referencedDataStatus"`
	ElementPath          string               `json:"elementPath"`
	Element              interface{}          `json:"element"`
}
type recursionCheck struct {
	equipmentSet map[EquipmentSetID]bool
	gearScore    map[GearScoreID]bool
	item         map[ItemID]bool
	player       map[PlayerID]bool
	position     map[PositionID]bool
	zone         map[ZoneID]bool
	zoneItem     map[ZoneItemID]bool
}

func newRecursionCheck() *recursionCheck {
	return &recursionCheck{equipmentSet: make(map[EquipmentSetID]bool), gearScore: make(map[GearScoreID]bool), item: make(map[ItemID]bool), player: make(map[PlayerID]bool), position: make(map[PositionID]bool), zone: make(map[ZoneID]bool), zoneItem: make(map[ZoneItemID]bool)}
}
func (engine *Engine) walkEquipmentSet(equipmentSetID EquipmentSetID, p path) {
	engine.PathTrack.equipmentSet[equipmentSetID] = p
}
func (engine *Engine) walkGearScore(gearScoreID GearScoreID, p path) {
	engine.PathTrack.gearScore[gearScoreID] = p
}
func (engine *Engine) walkItem(itemID ItemID, p path) {
	itemData, hasUpdated := engine.Patch.Item[itemID]
	if !hasUpdated {
		itemData = engine.State.Item[itemID]
	}
	var gearScorePath path
	if existingPath, pathExists := engine.PathTrack.gearScore[itemData.GearScore]; !pathExists {
		gearScorePath = p.gearScore()
	} else {
		gearScorePath = existingPath
	}
	engine.walkGearScore(itemData.GearScore, gearScorePath)
	originContainer := engine.anyOfPlayer_Position(itemData.Origin).anyOfPlayer_Position
	if originContainer.ElementKind == ElementKindPlayer {
		var originPath path
		if existingPath, pathExists := engine.PathTrack.player[originContainer.Player]; !pathExists {
			originPath = p.origin()
		} else {
			originPath = existingPath
		}
		engine.walkPlayer(originContainer.Player, originPath)
	} else if originContainer.ElementKind == ElementKindPosition {
		var originPath path
		if existingPath, pathExists := engine.PathTrack.position[originContainer.Position]; !pathExists {
			originPath = p.origin()
		} else {
			originPath = existingPath
		}
		engine.walkPosition(originContainer.Position, originPath)
	}
	engine.PathTrack.item[itemID] = p
}
func (engine *Engine) walkPlayer(playerID PlayerID, p path) {
	playerData, hasUpdated := engine.Patch.Player[playerID]
	if !hasUpdated {
		playerData = engine.State.Player[playerID]
	}
	var gearScorePath path
	if existingPath, pathExists := engine.PathTrack.gearScore[playerData.GearScore]; !pathExists {
		gearScorePath = p.gearScore()
	} else {
		gearScorePath = existingPath
	}
	engine.walkGearScore(playerData.GearScore, gearScorePath)
	for i, itemID := range mergeItemIDs(engine.State.Player[playerData.ID].Items, engine.Patch.Player[playerData.ID].Items) {
		var itemsPath path
		if existingPath, pathExists := engine.PathTrack.item[itemID]; !pathExists || !existingPath.equals(p) {
			itemsPath = p.items().index(i)
		} else {
			itemsPath = existingPath
		}
		engine.walkItem(itemID, itemsPath)
	}
	var positionPath path
	if existingPath, pathExists := engine.PathTrack.position[playerData.Position]; !pathExists {
		positionPath = p.position()
	} else {
		positionPath = existingPath
	}
	engine.walkPosition(playerData.Position, positionPath)
	engine.PathTrack.player[playerID] = p
}
func (engine *Engine) walkPosition(positionID PositionID, p path) {
	engine.PathTrack.position[positionID] = p
}
func (engine *Engine) walkZone(zoneID ZoneID, p path) {
	zoneData, hasUpdated := engine.Patch.Zone[zoneID]
	if !hasUpdated {
		zoneData = engine.State.Zone[zoneID]
	}
	for i, anyID := range zoneData.Interactables {
		interactablesContainer := engine.anyOfItem_Player_ZoneItem(anyID).anyOfItem_Player_ZoneItem
		if interactablesContainer.ElementKind == ElementKindItem {
			var interactablesPath path
			if existingPath, pathExists := engine.PathTrack.item[interactablesContainer.Item]; !pathExists || !existingPath.equals(p) {
				interactablesPath = p.interactables().index(i)
			} else {
				interactablesPath = existingPath
			}
			engine.walkItem(interactablesContainer.Item, interactablesPath)
		} else if interactablesContainer.ElementKind == ElementKindPlayer {
			var interactablesPath path
			if existingPath, pathExists := engine.PathTrack.player[interactablesContainer.Player]; !pathExists || !existingPath.equals(p) {
				interactablesPath = p.interactables().index(i)
			} else {
				interactablesPath = existingPath
			}
			engine.walkPlayer(interactablesContainer.Player, interactablesPath)
		} else if interactablesContainer.ElementKind == ElementKindZoneItem {
			var interactablesPath path
			if existingPath, pathExists := engine.PathTrack.zoneItem[interactablesContainer.ZoneItem]; !pathExists || !existingPath.equals(p) {
				interactablesPath = p.interactables().index(i)
			} else {
				interactablesPath = existingPath
			}
			engine.walkZoneItem(interactablesContainer.ZoneItem, interactablesPath)
		}
	}
	for i, zoneItemID := range mergeZoneItemIDs(engine.State.Zone[zoneData.ID].Items, engine.Patch.Zone[zoneData.ID].Items) {
		var itemsPath path
		if existingPath, pathExists := engine.PathTrack.zoneItem[zoneItemID]; !pathExists || !existingPath.equals(p) {
			itemsPath = p.items().index(i)
		} else {
			itemsPath = existingPath
		}
		engine.walkZoneItem(zoneItemID, itemsPath)
	}
	for i, playerID := range mergePlayerIDs(engine.State.Zone[zoneData.ID].Players, engine.Patch.Zone[zoneData.ID].Players) {
		var playersPath path
		if existingPath, pathExists := engine.PathTrack.player[playerID]; !pathExists || !existingPath.equals(p) {
			playersPath = p.players().index(i)
		} else {
			playersPath = existingPath
		}
		engine.walkPlayer(playerID, playersPath)
	}
	engine.PathTrack.zone[zoneID] = p
}
func (engine *Engine) walkZoneItem(zoneItemID ZoneItemID, p path) {
	zoneItemData, hasUpdated := engine.Patch.ZoneItem[zoneItemID]
	if !hasUpdated {
		zoneItemData = engine.State.ZoneItem[zoneItemID]
	}
	var itemPath path
	if existingPath, pathExists := engine.PathTrack.item[zoneItemData.Item]; !pathExists {
		itemPath = p.item()
	} else {
		itemPath = existingPath
	}
	engine.walkItem(zoneItemData.Item, itemPath)
	var positionPath path
	if existingPath, pathExists := engine.PathTrack.position[zoneItemData.Position]; !pathExists {
		positionPath = p.position()
	} else {
		positionPath = existingPath
	}
	engine.walkPosition(zoneItemData.Position, positionPath)
	engine.PathTrack.zoneItem[zoneItemID] = p
}
func (engine *Engine) walkTree() {
	walkedCheck := newRecursionCheck()
	for id, equipmentSetData := range engine.Patch.EquipmentSet {
		engine.walkEquipmentSet(equipmentSetData.ID, newPath(equipmentSetIdentifier, int(id)))
		walkedCheck.equipmentSet[equipmentSetData.ID] = true
	}
	for id, gearScoreData := range engine.Patch.GearScore {
		if !gearScoreData.HasParent {
			engine.walkGearScore(gearScoreData.ID, newPath(gearScoreIdentifier, int(id)))
			walkedCheck.gearScore[gearScoreData.ID] = true
		}
	}
	for id, itemData := range engine.Patch.Item {
		if !itemData.HasParent {
			engine.walkItem(itemData.ID, newPath(itemIdentifier, int(id)))
			walkedCheck.item[itemData.ID] = true
		}
	}
	for id, playerData := range engine.Patch.Player {
		if !playerData.HasParent {
			engine.walkPlayer(playerData.ID, newPath(playerIdentifier, int(id)))
			walkedCheck.player[playerData.ID] = true
		}
	}
	for id, positionData := range engine.Patch.Position {
		if !positionData.HasParent {
			engine.walkPosition(positionData.ID, newPath(positionIdentifier, int(id)))
			walkedCheck.position[positionData.ID] = true
		}
	}
	for id, zoneData := range engine.Patch.Zone {
		engine.walkZone(zoneData.ID, newPath(zoneIdentifier, int(id)))
		walkedCheck.zone[zoneData.ID] = true
	}
	for id, zoneItemData := range engine.Patch.ZoneItem {
		if !zoneItemData.HasParent {
			engine.walkZoneItem(zoneItemData.ID, newPath(zoneItemIdentifier, int(id)))
			walkedCheck.zoneItem[zoneItemData.ID] = true
		}
	}
	for id, equipmentSetData := range engine.State.EquipmentSet {
		if _, ok := walkedCheck.equipmentSet[equipmentSetData.ID]; !ok {
			engine.walkEquipmentSet(equipmentSetData.ID, newPath(equipmentSetIdentifier, int(id)))
		}
	}
	for id, gearScoreData := range engine.State.GearScore {
		if !gearScoreData.HasParent {
			if _, ok := walkedCheck.gearScore[gearScoreData.ID]; !ok {
				engine.walkGearScore(gearScoreData.ID, newPath(gearScoreIdentifier, int(id)))
			}
		}
	}
	for id, itemData := range engine.State.Item {
		if !itemData.HasParent {
			if _, ok := walkedCheck.item[itemData.ID]; !ok {
				engine.walkItem(itemData.ID, newPath(itemIdentifier, int(id)))
			}
		}
	}
	for id, playerData := range engine.State.Player {
		if !playerData.HasParent {
			if _, ok := walkedCheck.player[playerData.ID]; !ok {
				engine.walkPlayer(playerData.ID, newPath(playerIdentifier, int(id)))
			}
		}
	}
	for id, positionData := range engine.State.Position {
		if !positionData.HasParent {
			if _, ok := walkedCheck.position[positionData.ID]; !ok {
				engine.walkPosition(positionData.ID, newPath(positionIdentifier, int(id)))
			}
		}
	}
	for id, zoneData := range engine.State.Zone {
		if _, ok := walkedCheck.zone[zoneData.ID]; !ok {
			engine.walkZone(zoneData.ID, newPath(zoneIdentifier, int(id)))
		}
	}
	for id, zoneItemData := range engine.State.ZoneItem {
		if !zoneItemData.HasParent {
			if _, ok := walkedCheck.zoneItem[zoneItemData.ID]; !ok {
				engine.walkZoneItem(zoneItemData.ID, newPath(zoneItemIdentifier, int(id)))
			}
		}
	}
	engine.PathTrack._iterations += 1
	if engine.PathTrack._iterations == 100 {
		for key := range engine.PathTrack.equipmentSet {
			delete(engine.PathTrack.equipmentSet, key)
		}
		for key := range engine.PathTrack.gearScore {
			delete(engine.PathTrack.gearScore, key)
		}
		for key := range engine.PathTrack.item {
			delete(engine.PathTrack.item, key)
		}
		for key := range engine.PathTrack.player {
			delete(engine.PathTrack.player, key)
		}
		for key := range engine.PathTrack.position {
			delete(engine.PathTrack.position, key)
		}
		for key := range engine.PathTrack.zone {
			delete(engine.PathTrack.zone, key)
		}
		for key := range engine.PathTrack.zoneItem {
			delete(engine.PathTrack.zoneItem, key)
		}
	}
}


const (
	messageKindAction_addItemToPlayer messageKind = 1
	messageKindAction_movePlayer      messageKind = 2
	messageKindAction_spawnZoneItems  messageKind = 3
)

type actions struct {
	addItemToPlayer func(AddItemToPlayerParams, *Engine)
	movePlayer      func(MovePlayerParams, *Engine)
	spawnZoneItems  func(SpawnZoneItemsParams, *Engine)
}
type AddItemToPlayerParams struct {
	Item    ItemID `json:"item"`
	NewName string `json:"newName"`
}
type MovePlayerParams struct {
	ChangeX float64  `json:"changeX"`
	ChangeY float64  `json:"changeY"`
	Player  PlayerID `json:"player"`
}
type SpawnZoneItemsParams struct {
	Items []ItemID `json:"items"`
}

func (r *Room) processClientMessage(msg message) error {
	switch messageKind(msg.Kind) {
	case messageKindAction_addItemToPlayer:
		var params AddItemToPlayerParams
		err := params.UnmarshalJSON(msg.Content)
		if err != nil {
			return err
		}
		r.actions.addItemToPlayer(params, r.state)
	case messageKindAction_movePlayer:
		var params MovePlayerParams
		err := params.UnmarshalJSON(msg.Content)
		if err != nil {
			return err
		}
		r.actions.movePlayer(params, r.state)
	case messageKindAction_spawnZoneItems:
		var params SpawnZoneItemsParams
		err := params.UnmarshalJSON(msg.Content)
		if err != nil {
			return err
		}
		r.actions.spawnZoneItems(params, r.state)
	default:
		return errors.New("unknown message kind")
	}
	return nil
}
func Start(addItemToPlayer func(AddItemToPlayerParams, *Engine), movePlayer func(MovePlayerParams, *Engine), spawnZoneItems func(SpawnZoneItemsParams, *Engine), onDeploy func(*Engine), onFrameTick func(*Engine)) error {
	a := actions{addItemToPlayer, movePlayer, spawnZoneItems}
	setupRoutes(a, onDeploy, onFrameTick)
	err := http.ListenAndServe(":8080", nil)
	return err
}
