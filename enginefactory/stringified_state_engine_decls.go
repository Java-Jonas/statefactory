// this file was generated by https://github.com/Java-Jonas/decltostring

package enginefactory

const _AddPlayer_zone_func string = `func (_zone zone) AddPlayer(se *Engine) player {
	zone := se.Zone(_zone.zone.ID)
	if zone.zone.OperationKind_ == OperationKindDelete {
		return player{player: playerCore{OperationKind_: OperationKindDelete}}
	}
	player := se.createPlayer(true)
	zone.zone.Players = append(zone.zone.Players, player.player.ID)
	zone.zone.OperationKind_ = OperationKindUpdate
	se.Patch.Zone[zone.zone.ID] = zone.zone
	return player
}`

const _AddItem_zone_func string = `func (_zone zone) AddItem(se *Engine) zoneItem {
	zone := se.Zone(_zone.zone.ID)
	if zone.zone.OperationKind_ == OperationKindDelete {
		return zoneItem{zoneItem: zoneItemCore{OperationKind_: OperationKindDelete}}
	}
	zoneItem := se.createZoneItem(true)
	zone.zone.Items = append(zone.zone.Items, zoneItem.zoneItem.ID)
	zone.zone.OperationKind_ = OperationKindUpdate
	se.Patch.Zone[zone.zone.ID] = zone.zone
	return zoneItem
}`

const _AddTags_zone_func string = `func (_zone zone) AddTags(se *Engine, tags ...string) {
	zone := se.Zone(_zone.zone.ID)
	if zone.zone.OperationKind_ == OperationKindDelete {
		return
	}
	zone.zone.Tags = append(zone.zone.Tags, tags...)
	zone.zone.OperationKind_ = OperationKindUpdate
	se.Patch.Zone[zone.zone.ID] = zone.zone
}`

const _AddItem_player_func string = `func (_player player) AddItem(se *Engine) item {
	player := se.Player(_player.player.ID)
	if player.player.OperationKind_ == OperationKindDelete {
		return item{item: itemCore{OperationKind_: OperationKindDelete}}
	}
	item := se.createItem(true)
	player.player.Items = append(player.player.Items, item.item.ID)
	player.player.OperationKind_ = OperationKindUpdate
	se.Patch.Player[player.player.ID] = player.player
	return item
}`

const assembleGearScore_Engine_func string = `func (se *Engine) assembleGearScore(gearScoreID GearScoreID) (GearScore, bool) {
	gearScoreData, hasUpdated := se.Patch.GearScore[gearScoreID]
	if !hasUpdated {
		return GearScore{}, false
	}
	var gearScore GearScore
	gearScore.ID = gearScoreData.ID
	gearScore.OperationKind_ = gearScoreData.OperationKind_
	gearScore.Level = gearScoreData.Level
	gearScore.Score = gearScoreData.Score
	return gearScore, true
}`

const assemblePosition_Engine_func string = `func (se *Engine) assemblePosition(positionID PositionID) (Position, bool) {
	positionData, hasUpdated := se.Patch.Position[positionID]
	if !hasUpdated {
		return Position{}, false
	}
	var position Position
	position.ID = positionData.ID
	position.OperationKind_ = positionData.OperationKind_
	position.X = positionData.X
	position.Y = positionData.Y
	return position, true
}`

const assembleItem_Engine_func string = `func (se *Engine) assembleItem(itemID ItemID) (Item, bool) {
	itemData, hasUpdated := se.Patch.Item[itemID]
	if !hasUpdated {
		itemData = se.State.Item[itemID]
	}
	var item Item
	if treeGearScore, gearScoreHasUpdated := se.assembleGearScore(itemData.GearScore); gearScoreHasUpdated {
		hasUpdated = true
		item.GearScore = &treeGearScore
	}
	item.ID = itemData.ID
	item.OperationKind_ = itemData.OperationKind_
	return item, hasUpdated
}`

const assembleZoneItem_Engine_func string = `func (se *Engine) assembleZoneItem(zoneItemID ZoneItemID) (ZoneItem, bool) {
	zoneItemData, hasUpdated := se.Patch.ZoneItem[zoneItemID]
	if !hasUpdated {
		zoneItemData = se.State.ZoneItem[zoneItemID]
	}
	var zoneItem ZoneItem
	if treeItem, itemHasUpdated := se.assembleItem(zoneItemData.Item); itemHasUpdated {
		hasUpdated = true
		zoneItem.Item = &treeItem
	}
	if treePosition, positionHasUpdated := se.assemblePosition(zoneItemData.Position); positionHasUpdated {
		hasUpdated = true
		zoneItem.Position = &treePosition
	}
	zoneItem.ID = zoneItemData.ID
	zoneItem.OperationKind_ = zoneItemData.OperationKind_
	return zoneItem, hasUpdated
}`

const assemblePlayer_Engine_func string = `func (se *Engine) assemblePlayer(playerID PlayerID) (Player, bool) {
	playerData, hasUpdated := se.Patch.Player[playerID]
	if !hasUpdated {
		playerData = se.State.Player[playerID]
	}
	var player Player
	if treeGearScore, gearScoreHasUpdated := se.assembleGearScore(playerData.GearScore); gearScoreHasUpdated {
		hasUpdated = true
		player.GearScore = &treeGearScore
	}
	for _, itemID := range deduplicateItemIDs(se.State.Player[playerData.ID].Items, se.Patch.Player[playerData.ID].Items) {
		if treeItem, itemHasUpdated := se.assembleItem(itemID); itemHasUpdated {
			hasUpdated = true
			player.Items = append(player.Items, treeItem)
		}
	}
	if treePosition, positionHasUpdated := se.assemblePosition(playerData.Position); positionHasUpdated {
		hasUpdated = true
		player.Position = &treePosition
	}
	player.ID = playerData.ID
	player.OperationKind_ = playerData.OperationKind_
	return player, hasUpdated
}`

const assembleZone_Engine_func string = `func (se *Engine) assembleZone(zoneID ZoneID) (Zone, bool) {
	zoneData, hasUpdated := se.Patch.Zone[zoneID]
	if !hasUpdated {
		zoneData = se.State.Zone[zoneID]
	}
	var zone Zone
	for _, zoneItemID := range deduplicateZoneItemIDs(se.State.Zone[zoneData.ID].Items, se.Patch.Zone[zoneData.ID].Items) {
		if treeZoneItem, zoneItemHasUpdated := se.assembleZoneItem(zoneItemID); zoneItemHasUpdated {
			hasUpdated = true
			zone.Items = append(zone.Items, treeZoneItem)
		}
	}
	for _, playerID := range deduplicatePlayerIDs(se.State.Zone[zoneData.ID].Players, se.Patch.Zone[zoneData.ID].Players) {
		if treePlayer, playerHasUpdated := se.assemblePlayer(playerID); playerHasUpdated {
			hasUpdated = true
			zone.Players = append(zone.Players, treePlayer)
		}
	}
	zone.ID = zoneData.ID
	zone.OperationKind_ = zoneData.OperationKind_
	zone.Tags = zoneData.Tags
	return zone, hasUpdated
}`

const assembleTree_Engine_func string = `func (se *Engine) assembleTree() Tree {
	tree := newTree()
	for _, gearScoreData := range se.Patch.GearScore {
		if !gearScoreData.HasParent_ {
			gearScore, hasUpdated := se.assembleGearScore(gearScoreData.ID)
			if hasUpdated {
				tree.GearScore[gearScoreData.ID] = gearScore
			}
		}
	}
	for _, itemData := range se.Patch.Item {
		if !itemData.HasParent_ {
			item, hasUpdated := se.assembleItem(itemData.ID)
			if hasUpdated {
				tree.Item[itemData.ID] = item
			}
		}
	}
	for _, playerData := range se.Patch.Player {
		if !playerData.HasParent_ {
			player, hasUpdated := se.assemblePlayer(playerData.ID)
			if hasUpdated {
				tree.Player[playerData.ID] = player
			}
		}
	}
	for _, positionData := range se.Patch.Position {
		if !positionData.HasParent_ {
			position, hasUpdated := se.assemblePosition(positionData.ID)
			if hasUpdated {
				tree.Position[positionData.ID] = position
			}
		}
	}
	for _, zoneData := range se.Patch.Zone {
		zone, hasUpdated := se.assembleZone(zoneData.ID)
		if hasUpdated {
			tree.Zone[zoneData.ID] = zone
		}
	}
	for _, zoneItemData := range se.Patch.ZoneItem {
		if !zoneItemData.HasParent_ {
			zoneItem, hasUpdated := se.assembleZoneItem(zoneItemData.ID)
			if hasUpdated {
				tree.ZoneItem[zoneItemData.ID] = zoneItem
			}
		}
	}
	for _, gearScoreData := range se.State.GearScore {
		if !gearScoreData.HasParent_ {
			if _, ok := tree.GearScore[gearScoreData.ID]; !ok {
				gearScore, hasUpdated := se.assembleGearScore(gearScoreData.ID)
				if hasUpdated {
					tree.GearScore[gearScoreData.ID] = gearScore
				}
			}
		}
	}
	for _, itemData := range se.State.Item {
		if !itemData.HasParent_ {
			if _, ok := tree.Item[itemData.ID]; !ok {
				item, hasUpdated := se.assembleItem(itemData.ID)
				if hasUpdated {
					tree.Item[itemData.ID] = item
				}
			}
		}
	}
	for _, playerData := range se.State.Player {
		if !playerData.HasParent_ {
			if _, ok := tree.Player[playerData.ID]; !ok {
				player, hasUpdated := se.assemblePlayer(playerData.ID)
				if hasUpdated {
					tree.Player[playerData.ID] = player
				}
			}
		}
	}
	for _, positionData := range se.State.Position {
		if !positionData.HasParent_ {
			if _, ok := tree.Position[positionData.ID]; !ok {
				position, hasUpdated := se.assemblePosition(positionData.ID)
				if hasUpdated {
					tree.Position[positionData.ID] = position
				}
			}
		}
	}
	for _, zoneData := range se.State.Zone {
		if _, ok := tree.Zone[zoneData.ID]; !ok {
			zone, hasUpdated := se.assembleZone(zoneData.ID)
			if hasUpdated {
				tree.Zone[zoneData.ID] = zone
			}
		}
	}
	for _, zoneItemData := range se.State.ZoneItem {
		if !zoneItemData.HasParent_ {
			if _, ok := tree.ZoneItem[zoneItemData.ID]; !ok {
				zoneItem, hasUpdated := se.assembleZoneItem(zoneItemData.ID)
				if hasUpdated {
					tree.ZoneItem[zoneItemData.ID] = zoneItem
				}
			}
		}
	}
	return tree
}`

const _CreateGearScore_Engine_func string = `func (se *Engine) CreateGearScore() gearScore {
	return se.createGearScore(false)
}`

const createGearScore_Engine_func string = `func (se *Engine) createGearScore(hasParent bool) gearScore {
	var element gearScoreCore
	element.ID = GearScoreID(se.GenerateID())
	element.HasParent_ = hasParent
	element.OperationKind_ = OperationKindUpdate
	se.Patch.GearScore[element.ID] = element
	return gearScore{gearScore: element}
}`

const _CreatePosition_Engine_func string = `func (se *Engine) CreatePosition() position {
	return se.createPosition(false)
}`

const createPosition_Engine_func string = `func (se *Engine) createPosition(hasParent bool) position {
	var element positionCore
	element.ID = PositionID(se.GenerateID())
	element.HasParent_ = hasParent
	element.OperationKind_ = OperationKindUpdate
	se.Patch.Position[element.ID] = element
	return position{position: element}
}`

const _CreateItem_Engine_func string = `func (se *Engine) CreateItem() item {
	return se.createItem(false)
}`

const createItem_Engine_func string = `func (se *Engine) createItem(hasParent bool) item {
	var element itemCore
	element.ID = ItemID(se.GenerateID())
	element.HasParent_ = hasParent
	elementGearScore := se.createGearScore(true)
	element.GearScore = elementGearScore.gearScore.ID
	element.OperationKind_ = OperationKindUpdate
	se.Patch.Item[element.ID] = element
	return item{item: element}
}`

const _CreateZoneItem_Engine_func string = `func (se *Engine) CreateZoneItem() zoneItem {
	return se.createZoneItem(false)
}`

const createZoneItem_Engine_func string = `func (se *Engine) createZoneItem(hasParent bool) zoneItem {
	var element zoneItemCore
	element.ID = ZoneItemID(se.GenerateID())
	element.HasParent_ = hasParent
	elementItem := se.createItem(true)
	element.Item = elementItem.item.ID
	elementPosition := se.createPosition(true)
	element.Position = elementPosition.position.ID
	element.OperationKind_ = OperationKindUpdate
	se.Patch.ZoneItem[element.ID] = element
	return zoneItem{zoneItem: element}
}`

const _CreatePlayer_Engine_func string = `func (se *Engine) CreatePlayer() player {
	return se.createPlayer(false)
}`

const createPlayer_Engine_func string = `func (se *Engine) createPlayer(hasParent bool) player {
	var element playerCore
	element.ID = PlayerID(se.GenerateID())
	element.HasParent_ = hasParent
	elementGearScore := se.createGearScore(true)
	element.GearScore = elementGearScore.gearScore.ID
	elementPosition := se.createPosition(true)
	element.Position = elementPosition.position.ID
	element.OperationKind_ = OperationKindUpdate
	se.Patch.Player[element.ID] = element
	return player{player: element}
}`

const _CreateZone_Engine_func string = `func (se *Engine) CreateZone() zone {
	return se.createZone()
}`

const createZone_Engine_func string = `func (se *Engine) createZone() zone {
	var element zoneCore
	element.ID = ZoneID(se.GenerateID())
	element.OperationKind_ = OperationKindUpdate
	se.Patch.Zone[element.ID] = element
	return zone{zone: element}
}`

const _DeletePlayer_Engine_func string = `func (se *Engine) DeletePlayer(playerID PlayerID) {
	player := se.Player(playerID).player
	if player.HasParent_ {
		return
	}
	se.deletePlayer(playerID)
}`

const deletePlayer_Engine_func string = `func (se *Engine) deletePlayer(playerID PlayerID) {
	player := se.Player(playerID).player
	player.OperationKind_ = OperationKindDelete
	se.Patch.Player[player.ID] = player
	se.deleteGearScore(player.GearScore)
	for _, itemID := range player.Items {
		se.deleteItem(itemID)
	}
	se.deletePosition(player.Position)
}`

const _DeleteGearScore_Engine_func string = `func (se *Engine) DeleteGearScore(gearScoreID GearScoreID) {
	gearScore := se.GearScore(gearScoreID).gearScore
	if gearScore.HasParent_ {
		return
	}
	se.deleteGearScore(gearScoreID)
}`

const deleteGearScore_Engine_func string = `func (se *Engine) deleteGearScore(gearScoreID GearScoreID) {
	gearScore := se.GearScore(gearScoreID).gearScore
	gearScore.OperationKind_ = OperationKindDelete
	se.Patch.GearScore[gearScore.ID] = gearScore
}`

const _DeletePosition_Engine_func string = `func (se *Engine) DeletePosition(positionID PositionID) {
	position := se.Position(positionID).position
	if position.HasParent_ {
		return
	}
	se.deletePosition(positionID)
}`

const deletePosition_Engine_func string = `func (se *Engine) deletePosition(positionID PositionID) {
	position := se.Position(positionID).position
	position.OperationKind_ = OperationKindDelete
	se.Patch.Position[position.ID] = position
}`

const _DeleteItem_Engine_func string = `func (se *Engine) DeleteItem(itemID ItemID) {
	item := se.Item(itemID).item
	if item.HasParent_ {
		return
	}
	se.deleteItem(itemID)
}`

const deleteItem_Engine_func string = `func (se *Engine) deleteItem(itemID ItemID) {
	item := se.Item(itemID).item
	item.OperationKind_ = OperationKindDelete
	se.Patch.Item[item.ID] = item
	se.deleteGearScore(item.GearScore)
}`

const _DeleteZoneItem_Engine_func string = `func (se *Engine) DeleteZoneItem(zoneItemID ZoneItemID) {
	zoneItem := se.ZoneItem(zoneItemID).zoneItem
	if zoneItem.HasParent_ {
		return
	}
	se.deleteZoneItem(zoneItemID)
}`

const deleteZoneItem_Engine_func string = `func (se *Engine) deleteZoneItem(zoneItemID ZoneItemID) {
	zoneItem := se.ZoneItem(zoneItemID).zoneItem
	zoneItem.OperationKind_ = OperationKindDelete
	se.Patch.ZoneItem[zoneItem.ID] = zoneItem
	se.deleteItem(zoneItem.Item)
	se.deletePosition(zoneItem.Position)
}`

const _DeleteZone_Engine_func string = `func (se *Engine) DeleteZone(zoneID ZoneID) {
	se.deleteZone(zoneID)
}`

const deleteZone_Engine_func string = `func (se *Engine) deleteZone(zoneID ZoneID) {
	zone := se.Zone(zoneID).zone
	zone.OperationKind_ = OperationKindDelete
	se.Patch.Zone[zone.ID] = zone
	for _, zoneItemID := range zone.Items {
		se.deleteZoneItem(zoneItemID)
	}
	for _, playerID := range zone.Players {
		se.deletePlayer(playerID)
	}
}`

const _Player_Engine_func string = `func (se *Engine) Player(playerID PlayerID) player {
	patchingPlayer, ok := se.Patch.Player[playerID]
	if ok {
		return player{player: patchingPlayer}
	}
	currentPlayer := se.State.Player[playerID]
	return player{player: currentPlayer}
}`

const _ID_player_func string = `func (_player player) ID(se *Engine) PlayerID {
	return _player.player.ID
}`

const _Items_player_func string = `func (_player player) Items(se *Engine) []item {
	player := se.Player(_player.player.ID)
	var items []item
	for _, itemID := range player.player.Items {
		items = append(items, se.Item(itemID))
	}
	return items
}`

const _GearScore_player_func string = `func (_player player) GearScore(se *Engine) gearScore {
	player := se.Player(_player.player.ID)
	return se.GearScore(player.player.GearScore)
}`

const _Position_player_func string = `func (_player player) Position(se *Engine) position {
	player := se.Player(_player.player.ID)
	return se.Position(player.player.Position)
}`

const _GearScore_Engine_func string = `func (se *Engine) GearScore(gearScoreID GearScoreID) gearScore {
	patchingGearScore, ok := se.Patch.GearScore[gearScoreID]
	if ok {
		return gearScore{gearScore: patchingGearScore}
	}
	currentGearScore := se.State.GearScore[gearScoreID]
	return gearScore{gearScore: currentGearScore}
}`

const _ID_gearScore_func string = `func (_gearScore gearScore) ID(se *Engine) GearScoreID {
	return _gearScore.gearScore.ID
}`

const _Level_gearScore_func string = `func (_gearScore gearScore) Level(se *Engine) int {
	gearScore := se.GearScore(_gearScore.gearScore.ID)
	return gearScore.gearScore.Level
}`

const _Score_gearScore_func string = `func (_gearScore gearScore) Score(se *Engine) int {
	gearScore := se.GearScore(_gearScore.gearScore.ID)
	return gearScore.gearScore.Score
}`

const _Item_Engine_func string = `func (se *Engine) Item(itemID ItemID) item {
	patchingItem, ok := se.Patch.Item[itemID]
	if ok {
		return item{item: patchingItem}
	}
	currentItem := se.State.Item[itemID]
	return item{item: currentItem}
}`

const _ID_item_func string = `func (_item item) ID(se *Engine) ItemID {
	return _item.item.ID
}`

const _GearScore_item_func string = `func (_item item) GearScore(se *Engine) gearScore {
	item := se.Item(_item.item.ID)
	return se.GearScore(item.item.GearScore)
}`

const _Position_Engine_func string = `func (se *Engine) Position(positionID PositionID) position {
	patchingPosition, ok := se.Patch.Position[positionID]
	if ok {
		return position{position: patchingPosition}
	}
	currentPosition := se.State.Position[positionID]
	return position{position: currentPosition}
}`

const _ID_position_func string = `func (_position position) ID(se *Engine) PositionID {
	return _position.position.ID
}`

const _X_position_func string = `func (_position position) X(se *Engine) float64 {
	position := se.Position(_position.position.ID)
	return position.position.X
}`

const _Y_position_func string = `func (_position position) Y(se *Engine) float64 {
	position := se.Position(_position.position.ID)
	return position.position.Y
}`

const _ZoneItem_Engine_func string = `func (se *Engine) ZoneItem(zoneItemID ZoneItemID) zoneItem {
	patchingZoneItem, ok := se.Patch.ZoneItem[zoneItemID]
	if ok {
		return zoneItem{zoneItem: patchingZoneItem}
	}
	currentZoneItem := se.State.ZoneItem[zoneItemID]
	return zoneItem{zoneItem: currentZoneItem}
}`

const _ID_zoneItem_func string = `func (_zoneItem zoneItem) ID(se *Engine) ZoneItemID {
	return _zoneItem.zoneItem.ID
}`

const _Position_zoneItem_func string = `func (_zoneItem zoneItem) Position(se *Engine) position {
	zoneItem := se.ZoneItem(_zoneItem.zoneItem.ID)
	return se.Position(zoneItem.zoneItem.Position)
}`

const _Item_zoneItem_func string = `func (_zoneItem zoneItem) Item(se *Engine) item {
	zoneItem := se.ZoneItem(_zoneItem.zoneItem.ID)
	return se.Item(zoneItem.zoneItem.Item)
}`

const _Zone_Engine_func string = `func (se *Engine) Zone(zoneID ZoneID) zone {
	patchingZone, ok := se.Patch.Zone[zoneID]
	if ok {
		return zone{zone: patchingZone}
	}
	currentZone := se.State.Zone[zoneID]
	return zone{zone: currentZone}
}`

const _ID_zone_func string = `func (_zone zone) ID(se *Engine) ZoneID {
	return _zone.zone.ID
}`

const _Players_zone_func string = `func (_zone zone) Players(se *Engine) []player {
	zone := se.Zone(_zone.zone.ID)
	var players []player
	for _, playerID := range zone.zone.Players {
		players = append(players, se.Player(playerID))
	}
	return players
}`

const _Items_zone_func string = `func (_zone zone) Items(se *Engine) []zoneItem {
	zone := se.Zone(_zone.zone.ID)
	var items []zoneItem
	for _, zoneItemID := range zone.zone.Items {
		items = append(items, se.ZoneItem(zoneItemID))
	}
	return items
}`

const _Tags_zone_func string = `func (_zone zone) Tags(se *Engine) []string {
	zone := se.Zone(_zone.zone.ID)
	var tags []string
	for _, element := range zone.zone.Tags {
		tags = append(tags, element)
	}
	return tags
}`

const deduplicateGearScoreIDs_func string = `func deduplicateGearScoreIDs(a []GearScoreID, b []GearScoreID) []GearScoreID {
	check := make(map[GearScoreID]bool)
	deduped := make([]GearScoreID, 0)
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
}`

const deduplicateItemIDs_func string = `func deduplicateItemIDs(a []ItemID, b []ItemID) []ItemID {
	check := make(map[ItemID]bool)
	deduped := make([]ItemID, 0)
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
}`

const deduplicatePlayerIDs_func string = `func deduplicatePlayerIDs(a []PlayerID, b []PlayerID) []PlayerID {
	check := make(map[PlayerID]bool)
	deduped := make([]PlayerID, 0)
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
}`

const deduplicatePositionIDs_func string = `func deduplicatePositionIDs(a []PositionID, b []PositionID) []PositionID {
	check := make(map[PositionID]bool)
	deduped := make([]PositionID, 0)
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
}`

const deduplicateZoneIDs_func string = `func deduplicateZoneIDs(a []ZoneID, b []ZoneID) []ZoneID {
	check := make(map[ZoneID]bool)
	deduped := make([]ZoneID, 0)
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
}`

const deduplicateZoneItemIDs_func string = `func deduplicateZoneItemIDs(a []ZoneItemID, b []ZoneItemID) []ZoneItemID {
	check := make(map[ZoneItemID]bool)
	deduped := make([]ZoneItemID, 0)
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
}`

const _RemovePlayers_zone_func string = `func (_zone zone) RemovePlayers(se *Engine, playersToRemove ...PlayerID) zone {
	zone := se.Zone(_zone.zone.ID)
	if zone.zone.OperationKind_ == OperationKindDelete {
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
				se.deletePlayer(element)
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
	zone.zone.OperationKind_ = OperationKindUpdate
	se.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}`

const _RemoveItems_zone_func string = `func (_zone zone) RemoveItems(se *Engine, itemsToRemove ...ZoneItemID) zone {
	zone := se.Zone(_zone.zone.ID)
	if zone.zone.OperationKind_ == OperationKindDelete {
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
				se.deleteZoneItem(element)
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
	zone.zone.OperationKind_ = OperationKindUpdate
	se.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}`

const _RemoveItems_player_func string = `func (_player player) RemoveItems(se *Engine, itemsToRemove ...ItemID) player {
	player := se.Player(_player.player.ID)
	if player.player.OperationKind_ == OperationKindDelete {
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
				se.deleteItem(element)
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
	player.player.OperationKind_ = OperationKindUpdate
	se.Patch.Player[player.player.ID] = player.player
	return player
}`

const _RemoveTags_zone_func string = `func (_zone zone) RemoveTags(se *Engine, tagsToRemove ...string) zone {
	zone := se.Zone(_zone.zone.ID)
	if zone.zone.OperationKind_ == OperationKindDelete {
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
	zone.zone.OperationKind_ = OperationKindUpdate
	se.Patch.Zone[zone.zone.ID] = zone.zone
	return zone
}`

const _SetLevel_gearScore_func string = `func (_gearScore gearScore) SetLevel(se *Engine, newLevel int) gearScore {
	gearScore := se.GearScore(_gearScore.gearScore.ID)
	if gearScore.gearScore.OperationKind_ == OperationKindDelete {
		return gearScore
	}
	gearScore.gearScore.Level = newLevel
	gearScore.gearScore.OperationKind_ = OperationKindUpdate
	se.Patch.GearScore[gearScore.gearScore.ID] = gearScore.gearScore
	return gearScore
}`

const _SetScore_gearScore_func string = `func (_gearScore gearScore) SetScore(se *Engine, newScore int) gearScore {
	gearScore := se.GearScore(_gearScore.gearScore.ID)
	if gearScore.gearScore.OperationKind_ == OperationKindDelete {
		return gearScore
	}
	gearScore.gearScore.Score = newScore
	gearScore.gearScore.OperationKind_ = OperationKindUpdate
	se.Patch.GearScore[gearScore.gearScore.ID] = gearScore.gearScore
	return gearScore
}`

const _SetX_position_func string = `func (_position position) SetX(se *Engine, newX float64) position {
	position := se.Position(_position.position.ID)
	if position.position.OperationKind_ == OperationKindDelete {
		return position
	}
	position.position.X = newX
	position.position.OperationKind_ = OperationKindUpdate
	se.Patch.Position[position.position.ID] = position.position
	return position
}`

const _SetY_position_func string = `func (_position position) SetY(se *Engine, newY float64) position {
	position := se.Position(_position.position.ID)
	if position.position.OperationKind_ == OperationKindDelete {
		return position
	}
	position.position.Y = newY
	position.position.OperationKind_ = OperationKindUpdate
	se.Patch.Position[position.position.ID] = position.position
	return position
}`

const _GearScoreID_type string = `type GearScoreID int`

const _ItemID_type string = `type ItemID int`

const _PlayerID_type string = `type PlayerID int`

const _PositionID_type string = `type PositionID int`

const _ZoneID_type string = `type ZoneID int`

const _ZoneItemID_type string = `type ZoneItemID int`

const _State_type string = `type State struct {
	GearScore	map[GearScoreID]gearScoreCore	` + "`" + `json:"gearScore"` + "`" + `
	Item		map[ItemID]itemCore		` + "`" + `json:"item"` + "`" + `
	Player		map[PlayerID]playerCore		` + "`" + `json:"player"` + "`" + `
	Position	map[PositionID]positionCore	` + "`" + `json:"position"` + "`" + `
	Zone		map[ZoneID]zoneCore		` + "`" + `json:"zone"` + "`" + `
	ZoneItem	map[ZoneItemID]zoneItemCore	` + "`" + `json:"zoneItem"` + "`" + `
}`

const newState_func string = `func newState() State {
	return State{GearScore: make(map[GearScoreID]gearScoreCore), Item: make(map[ItemID]itemCore), Player: make(map[PlayerID]playerCore), Position: make(map[PositionID]positionCore), Zone: make(map[ZoneID]zoneCore), ZoneItem: make(map[ZoneItemID]zoneItemCore)}
}`

const zoneCore_type string = `type zoneCore struct {
	ID		ZoneID		` + "`" + `json:"id"` + "`" + `
	Items		[]ZoneItemID	` + "`" + `json:"items"` + "`" + `
	Players		[]PlayerID	` + "`" + `json:"players"` + "`" + `
	Tags		[]string	` + "`" + `json:"tags"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
}`

const zone_type string = `type zone struct{ zone zoneCore }`

const zoneItemCore_type string = `type zoneItemCore struct {
	ID		ZoneItemID	` + "`" + `json:"id"` + "`" + `
	Item		ItemID		` + "`" + `json:"item"` + "`" + `
	Position	PositionID	` + "`" + `json:"position"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
	HasParent_	bool		` + "`" + `json:"hasParent_"` + "`" + `
}`

const zoneItem_type string = `type zoneItem struct{ zoneItem zoneItemCore }`

const itemCore_type string = `type itemCore struct {
	ID		ItemID		` + "`" + `json:"id"` + "`" + `
	GearScore	GearScoreID	` + "`" + `json:"gearScore"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
	HasParent_	bool		` + "`" + `json:"hasParent_"` + "`" + `
}`

const item_type string = `type item struct{ item itemCore }`

const playerCore_type string = `type playerCore struct {
	ID		PlayerID	` + "`" + `json:"id"` + "`" + `
	GearScore	GearScoreID	` + "`" + `json:"gearScore"` + "`" + `
	Items		[]ItemID	` + "`" + `json:"items"` + "`" + `
	Position	PositionID	` + "`" + `json:"position"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
	HasParent_	bool		` + "`" + `json:"hasParent_"` + "`" + `
}`

const player_type string = `type player struct{ player playerCore }`

const gearScoreCore_type string = `type gearScoreCore struct {
	ID		GearScoreID	` + "`" + `json:"id"` + "`" + `
	Level		int		` + "`" + `json:"level"` + "`" + `
	Score		int		` + "`" + `json:"score"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
	HasParent_	bool		` + "`" + `json:"hasParent_"` + "`" + `
}`

const gearScore_type string = `type gearScore struct{ gearScore gearScoreCore }`

const positionCore_type string = `type positionCore struct {
	ID		PositionID	` + "`" + `json:"id"` + "`" + `
	X		float64		` + "`" + `json:"x"` + "`" + `
	Y		float64		` + "`" + `json:"y"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
	HasParent_	bool		` + "`" + `json:"hasParent_"` + "`" + `
}`

const position_type string = `type position struct{ position positionCore }`

const _OperationKind_type string = `type OperationKind string`

const _OperationKindDelete_type string = `const (
	OperationKindDelete	OperationKind	= "DELETE"
	OperationKindUpdate			= "UPDATE"
)`

const _Engine_type string = `type Engine struct {
	State	State
	Patch	State
	IDgen	int
}`

const newEngine_func string = `func newEngine() *Engine {
	return &Engine{IDgen: 1, Patch: newState(), State: newState()}
}`

const _GenerateID_Engine_func string = `func (se *Engine) GenerateID() int {
	newID := se.IDgen
	se.IDgen = se.IDgen + 1
	return newID
}`

const _UpdateState_Engine_func string = `func (se *Engine) UpdateState() {
	for _, gearScore := range se.Patch.GearScore {
		if gearScore.OperationKind_ == OperationKindDelete {
			delete(se.State.GearScore, gearScore.ID)
		} else {
			se.State.GearScore[gearScore.ID] = gearScore
		}
	}
	for _, item := range se.Patch.Item {
		if item.OperationKind_ == OperationKindDelete {
			delete(se.State.Item, item.ID)
		} else {
			se.State.Item[item.ID] = item
		}
	}
	for _, player := range se.Patch.Player {
		if player.OperationKind_ == OperationKindDelete {
			delete(se.State.Player, player.ID)
		} else {
			se.State.Player[player.ID] = player
		}
	}
	for _, position := range se.Patch.Position {
		if position.OperationKind_ == OperationKindDelete {
			delete(se.State.Position, position.ID)
		} else {
			se.State.Position[position.ID] = position
		}
	}
	for _, zone := range se.Patch.Zone {
		if zone.OperationKind_ == OperationKindDelete {
			delete(se.State.Zone, zone.ID)
		} else {
			se.State.Zone[zone.ID] = zone
		}
	}
	for _, zoneItem := range se.Patch.ZoneItem {
		if zoneItem.OperationKind_ == OperationKindDelete {
			delete(se.State.ZoneItem, zoneItem.ID)
		} else {
			se.State.ZoneItem[zoneItem.ID] = zoneItem
		}
	}
	for key := range se.Patch.GearScore {
		delete(se.Patch.GearScore, key)
	}
	for key := range se.Patch.Item {
		delete(se.Patch.Item, key)
	}
	for key := range se.Patch.Player {
		delete(se.Patch.Player, key)
	}
	for key := range se.Patch.Position {
		delete(se.Patch.Position, key)
	}
	for key := range se.Patch.Zone {
		delete(se.Patch.Zone, key)
	}
	for key := range se.Patch.ZoneItem {
		delete(se.Patch.ZoneItem, key)
	}
}`

const _Tree_type string = `type Tree struct {
	GearScore	map[GearScoreID]GearScore	` + "`" + `json:"gearScore"` + "`" + `
	Item		map[ItemID]Item			` + "`" + `json:"item"` + "`" + `
	Player		map[PlayerID]Player		` + "`" + `json:"player"` + "`" + `
	Position	map[PositionID]Position		` + "`" + `json:"position"` + "`" + `
	Zone		map[ZoneID]Zone			` + "`" + `json:"zone"` + "`" + `
	ZoneItem	map[ZoneItemID]ZoneItem		` + "`" + `json:"zoneItem"` + "`" + `
}`

const newTree_func string = `func newTree() Tree {
	return Tree{GearScore: make(map[GearScoreID]GearScore), Item: make(map[ItemID]Item), Player: make(map[PlayerID]Player), Position: make(map[PositionID]Position), Zone: make(map[ZoneID]Zone), ZoneItem: make(map[ZoneItemID]ZoneItem)}
}`

const _ZoneItem_type string = `type ZoneItem struct {
	ID		ZoneItemID	` + "`" + `json:"id"` + "`" + `
	Item		*Item		` + "`" + `json:"item"` + "`" + `
	Position	*Position	` + "`" + `json:"position"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
}`

const _Item_type string = `type Item struct {
	ID		ItemID		` + "`" + `json:"id"` + "`" + `
	GearScore	*GearScore	` + "`" + `json:"gearScore"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
}`

const _Position_type string = `type Position struct {
	ID		PositionID	` + "`" + `json:"id"` + "`" + `
	X		float64		` + "`" + `json:"x"` + "`" + `
	Y		float64		` + "`" + `json:"y"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
}`

const _GearScore_type string = `type GearScore struct {
	ID		GearScoreID	` + "`" + `json:"id"` + "`" + `
	Level		int		` + "`" + `json:"level"` + "`" + `
	Score		int		` + "`" + `json:"score"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
}`

const _Player_type string = `type Player struct {
	ID		PlayerID	` + "`" + `json:"id"` + "`" + `
	GearScore	*GearScore	` + "`" + `json:"gearScore"` + "`" + `
	Items		[]Item		` + "`" + `json:"items"` + "`" + `
	Position	*Position	` + "`" + `json:"position"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
}`

const _Zone_type string = `type Zone struct {
	ID		ZoneID		` + "`" + `json:"id"` + "`" + `
	Items		[]ZoneItem	` + "`" + `json:"items"` + "`" + `
	Players		[]Player	` + "`" + `json:"players"` + "`" + `
	Tags		[]string	` + "`" + `json:"tags"` + "`" + `
	OperationKind_	OperationKind	` + "`" + `json:"operationKind_"` + "`" + `
}`
