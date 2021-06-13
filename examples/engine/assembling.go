package state

type assembleConfig struct {
	forceInclude bool // include everything, regardless of update status
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

func (engine *Engine) assemblePlayerTargetRef(playerID PlayerID, check *recursionCheck, config assembleConfig) (*AnyOfPlayer_ZoneItemReference, bool, bool) {
	statePlayer := engine.State.Player[playerID]
	patchPlayer, playerIsInPatch := engine.Patch.Player[playerID]

	// ref not set at all
	if statePlayer.Target == 0 && (!playerIsInPatch || patchPlayer.Target == 0) {
		return nil, false, false
	}

	// force include
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

	// ref was definitely created
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

	// ref was definitely removed
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

	// immediate replacement of refs
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

	// element got updated - OperationKindUpdate
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

func (engine *Engine) assembleItemBoundToRef(itemID ItemID, check *recursionCheck, config assembleConfig) (*PlayerReference, bool, bool) {
	stateItem := engine.State.Item[itemID]
	patchItem, itemIsInPatch := engine.Patch.Item[itemID]

	// ref not set at all
	if stateItem.BoundTo == 0 && (!itemIsInPatch || patchItem.BoundTo == 0) {
		return nil, false, false
	}

	// force include
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

	// ref was definitely created
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

	// ref was definitely removed
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

	// immediate replacement of refs
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

	// OperationKindUpdate element got updated
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

func (engine *Engine) assembleTree() Tree {

	config := assembleConfig{
		forceInclude: false,
	}

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