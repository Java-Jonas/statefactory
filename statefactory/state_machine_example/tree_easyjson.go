// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package state

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson19a9978cDecodeStatefactoryStateMachineExample(in *jlexer.Lexer, out *_zoneItem) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = ZoneItemID(in.Int())
		case "position":
			if in.IsNull() {
				in.Skip()
				out.Position = nil
			} else {
				if out.Position == nil {
					out.Position = new(_position)
				}
				(*out.Position).UnmarshalEasyJSON(in)
			}
		case "item":
			if in.IsNull() {
				in.Skip()
				out.Item = nil
			} else {
				if out.Item == nil {
					out.Item = new(_item)
				}
				(*out.Item).UnmarshalEasyJSON(in)
			}
		case "operationKind":
			out.OperationKind = OperationKind(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson19a9978cEncodeStatefactoryStateMachineExample(out *jwriter.Writer, in _zoneItem) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		out.RawString(prefix[1:])
		out.Int(int(in.ID))
	}
	{
		const prefix string = ",\"position\":"
		out.RawString(prefix)
		if in.Position == nil {
			out.RawString("null")
		} else {
			(*in.Position).MarshalEasyJSON(out)
		}
	}
	{
		const prefix string = ",\"item\":"
		out.RawString(prefix)
		if in.Item == nil {
			out.RawString("null")
		} else {
			(*in.Item).MarshalEasyJSON(out)
		}
	}
	{
		const prefix string = ",\"operationKind\":"
		out.RawString(prefix)
		out.String(string(in.OperationKind))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v _zoneItem) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson19a9978cEncodeStatefactoryStateMachineExample(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v _zoneItem) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson19a9978cEncodeStatefactoryStateMachineExample(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *_zoneItem) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson19a9978cDecodeStatefactoryStateMachineExample(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *_zoneItem) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson19a9978cDecodeStatefactoryStateMachineExample(l, v)
}
func easyjson19a9978cDecodeStatefactoryStateMachineExample1(in *jlexer.Lexer, out *_zone) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = ZoneID(in.Int())
		case "players":
			if in.IsNull() {
				in.Skip()
				out.Players = nil
			} else {
				in.Delim('[')
				if out.Players == nil {
					if !in.IsDelim(']') {
						out.Players = make([]_player, 0, 1)
					} else {
						out.Players = []_player{}
					}
				} else {
					out.Players = (out.Players)[:0]
				}
				for !in.IsDelim(']') {
					var v1 _player
					(v1).UnmarshalEasyJSON(in)
					out.Players = append(out.Players, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "items":
			if in.IsNull() {
				in.Skip()
				out.Items = nil
			} else {
				in.Delim('[')
				if out.Items == nil {
					if !in.IsDelim(']') {
						out.Items = make([]_zoneItem, 0, 1)
					} else {
						out.Items = []_zoneItem{}
					}
				} else {
					out.Items = (out.Items)[:0]
				}
				for !in.IsDelim(']') {
					var v2 _zoneItem
					(v2).UnmarshalEasyJSON(in)
					out.Items = append(out.Items, v2)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "operationKind":
			out.OperationKind = OperationKind(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson19a9978cEncodeStatefactoryStateMachineExample1(out *jwriter.Writer, in _zone) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		out.RawString(prefix[1:])
		out.Int(int(in.ID))
	}
	{
		const prefix string = ",\"players\":"
		out.RawString(prefix)
		if in.Players == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v3, v4 := range in.Players {
				if v3 > 0 {
					out.RawByte(',')
				}
				(v4).MarshalEasyJSON(out)
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"items\":"
		out.RawString(prefix)
		if in.Items == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v5, v6 := range in.Items {
				if v5 > 0 {
					out.RawByte(',')
				}
				(v6).MarshalEasyJSON(out)
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"operationKind\":"
		out.RawString(prefix)
		out.String(string(in.OperationKind))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v _zone) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson19a9978cEncodeStatefactoryStateMachineExample1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v _zone) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson19a9978cEncodeStatefactoryStateMachineExample1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *_zone) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson19a9978cDecodeStatefactoryStateMachineExample1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *_zone) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson19a9978cDecodeStatefactoryStateMachineExample1(l, v)
}
func easyjson19a9978cDecodeStatefactoryStateMachineExample2(in *jlexer.Lexer, out *_position) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = PositionID(in.Int())
		case "x":
			out.X = float64(in.Float64())
		case "y":
			out.Y = float64(in.Float64())
		case "operationKind":
			out.OperationKind = OperationKind(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson19a9978cEncodeStatefactoryStateMachineExample2(out *jwriter.Writer, in _position) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		out.RawString(prefix[1:])
		out.Int(int(in.ID))
	}
	{
		const prefix string = ",\"x\":"
		out.RawString(prefix)
		out.Float64(float64(in.X))
	}
	{
		const prefix string = ",\"y\":"
		out.RawString(prefix)
		out.Float64(float64(in.Y))
	}
	{
		const prefix string = ",\"operationKind\":"
		out.RawString(prefix)
		out.String(string(in.OperationKind))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v _position) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson19a9978cEncodeStatefactoryStateMachineExample2(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v _position) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson19a9978cEncodeStatefactoryStateMachineExample2(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *_position) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson19a9978cDecodeStatefactoryStateMachineExample2(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *_position) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson19a9978cDecodeStatefactoryStateMachineExample2(l, v)
}
func easyjson19a9978cDecodeStatefactoryStateMachineExample3(in *jlexer.Lexer, out *_player) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = PlayerID(in.Int())
		case "items":
			if in.IsNull() {
				in.Skip()
				out.Items = nil
			} else {
				in.Delim('[')
				if out.Items == nil {
					if !in.IsDelim(']') {
						out.Items = make([]_item, 0, 2)
					} else {
						out.Items = []_item{}
					}
				} else {
					out.Items = (out.Items)[:0]
				}
				for !in.IsDelim(']') {
					var v7 _item
					(v7).UnmarshalEasyJSON(in)
					out.Items = append(out.Items, v7)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "gearScore":
			if in.IsNull() {
				in.Skip()
				out.GearScore = nil
			} else {
				if out.GearScore == nil {
					out.GearScore = new(_gearScore)
				}
				(*out.GearScore).UnmarshalEasyJSON(in)
			}
		case "position":
			if in.IsNull() {
				in.Skip()
				out.Position = nil
			} else {
				if out.Position == nil {
					out.Position = new(_position)
				}
				(*out.Position).UnmarshalEasyJSON(in)
			}
		case "operationKind":
			out.OperationKind = OperationKind(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson19a9978cEncodeStatefactoryStateMachineExample3(out *jwriter.Writer, in _player) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		out.RawString(prefix[1:])
		out.Int(int(in.ID))
	}
	{
		const prefix string = ",\"items\":"
		out.RawString(prefix)
		if in.Items == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v8, v9 := range in.Items {
				if v8 > 0 {
					out.RawByte(',')
				}
				(v9).MarshalEasyJSON(out)
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"gearScore\":"
		out.RawString(prefix)
		if in.GearScore == nil {
			out.RawString("null")
		} else {
			(*in.GearScore).MarshalEasyJSON(out)
		}
	}
	{
		const prefix string = ",\"position\":"
		out.RawString(prefix)
		if in.Position == nil {
			out.RawString("null")
		} else {
			(*in.Position).MarshalEasyJSON(out)
		}
	}
	{
		const prefix string = ",\"operationKind\":"
		out.RawString(prefix)
		out.String(string(in.OperationKind))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v _player) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson19a9978cEncodeStatefactoryStateMachineExample3(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v _player) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson19a9978cEncodeStatefactoryStateMachineExample3(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *_player) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson19a9978cDecodeStatefactoryStateMachineExample3(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *_player) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson19a9978cDecodeStatefactoryStateMachineExample3(l, v)
}
func easyjson19a9978cDecodeStatefactoryStateMachineExample4(in *jlexer.Lexer, out *_item) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = ItemID(in.Int())
		case "gearScore":
			if in.IsNull() {
				in.Skip()
				out.GearScore = nil
			} else {
				if out.GearScore == nil {
					out.GearScore = new(_gearScore)
				}
				(*out.GearScore).UnmarshalEasyJSON(in)
			}
		case "operationKind":
			out.OperationKind = OperationKind(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson19a9978cEncodeStatefactoryStateMachineExample4(out *jwriter.Writer, in _item) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		out.RawString(prefix[1:])
		out.Int(int(in.ID))
	}
	{
		const prefix string = ",\"gearScore\":"
		out.RawString(prefix)
		if in.GearScore == nil {
			out.RawString("null")
		} else {
			(*in.GearScore).MarshalEasyJSON(out)
		}
	}
	{
		const prefix string = ",\"operationKind\":"
		out.RawString(prefix)
		out.String(string(in.OperationKind))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v _item) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson19a9978cEncodeStatefactoryStateMachineExample4(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v _item) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson19a9978cEncodeStatefactoryStateMachineExample4(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *_item) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson19a9978cDecodeStatefactoryStateMachineExample4(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *_item) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson19a9978cDecodeStatefactoryStateMachineExample4(l, v)
}
func easyjson19a9978cDecodeStatefactoryStateMachineExample5(in *jlexer.Lexer, out *_gearScore) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = GearScoreID(in.Int())
		case "level":
			out.Level = int(in.Int())
		case "score":
			out.Score = int(in.Int())
		case "operationKind":
			out.OperationKind = OperationKind(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson19a9978cEncodeStatefactoryStateMachineExample5(out *jwriter.Writer, in _gearScore) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		out.RawString(prefix[1:])
		out.Int(int(in.ID))
	}
	{
		const prefix string = ",\"level\":"
		out.RawString(prefix)
		out.Int(int(in.Level))
	}
	{
		const prefix string = ",\"score\":"
		out.RawString(prefix)
		out.Int(int(in.Score))
	}
	{
		const prefix string = ",\"operationKind\":"
		out.RawString(prefix)
		out.String(string(in.OperationKind))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v _gearScore) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson19a9978cEncodeStatefactoryStateMachineExample5(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v _gearScore) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson19a9978cEncodeStatefactoryStateMachineExample5(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *_gearScore) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson19a9978cDecodeStatefactoryStateMachineExample5(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *_gearScore) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson19a9978cDecodeStatefactoryStateMachineExample5(l, v)
}
func easyjson19a9978cDecodeStatefactoryStateMachineExample6(in *jlexer.Lexer, out *Tree) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "player":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.Player = make(map[PlayerID]_player)
				for !in.IsDelim('}') {
					key := PlayerID(in.IntStr())
					in.WantColon()
					var v10 _player
					(v10).UnmarshalEasyJSON(in)
					(out.Player)[key] = v10
					in.WantComma()
				}
				in.Delim('}')
			}
		case "zone":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.Zone = make(map[ZoneID]_zone)
				for !in.IsDelim('}') {
					key := ZoneID(in.IntStr())
					in.WantColon()
					var v11 _zone
					(v11).UnmarshalEasyJSON(in)
					(out.Zone)[key] = v11
					in.WantComma()
				}
				in.Delim('}')
			}
		case "zoneItem":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.ZoneItem = make(map[ZoneItemID]_zoneItem)
				for !in.IsDelim('}') {
					key := ZoneItemID(in.IntStr())
					in.WantColon()
					var v12 _zoneItem
					(v12).UnmarshalEasyJSON(in)
					(out.ZoneItem)[key] = v12
					in.WantComma()
				}
				in.Delim('}')
			}
		case "item":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.Item = make(map[ItemID]_item)
				for !in.IsDelim('}') {
					key := ItemID(in.IntStr())
					in.WantColon()
					var v13 _item
					(v13).UnmarshalEasyJSON(in)
					(out.Item)[key] = v13
					in.WantComma()
				}
				in.Delim('}')
			}
		case "position":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.Position = make(map[PositionID]_position)
				for !in.IsDelim('}') {
					key := PositionID(in.IntStr())
					in.WantColon()
					var v14 _position
					(v14).UnmarshalEasyJSON(in)
					(out.Position)[key] = v14
					in.WantComma()
				}
				in.Delim('}')
			}
		case "gearScore":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.GearScore = make(map[GearScoreID]_gearScore)
				for !in.IsDelim('}') {
					key := GearScoreID(in.IntStr())
					in.WantColon()
					var v15 _gearScore
					(v15).UnmarshalEasyJSON(in)
					(out.GearScore)[key] = v15
					in.WantComma()
				}
				in.Delim('}')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson19a9978cEncodeStatefactoryStateMachineExample6(out *jwriter.Writer, in Tree) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"player\":"
		out.RawString(prefix[1:])
		if in.Player == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v16First := true
			for v16Name, v16Value := range in.Player {
				if v16First {
					v16First = false
				} else {
					out.RawByte(',')
				}
				out.IntStr(int(v16Name))
				out.RawByte(':')
				(v16Value).MarshalEasyJSON(out)
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"zone\":"
		out.RawString(prefix)
		if in.Zone == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v17First := true
			for v17Name, v17Value := range in.Zone {
				if v17First {
					v17First = false
				} else {
					out.RawByte(',')
				}
				out.IntStr(int(v17Name))
				out.RawByte(':')
				(v17Value).MarshalEasyJSON(out)
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"zoneItem\":"
		out.RawString(prefix)
		if in.ZoneItem == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v18First := true
			for v18Name, v18Value := range in.ZoneItem {
				if v18First {
					v18First = false
				} else {
					out.RawByte(',')
				}
				out.IntStr(int(v18Name))
				out.RawByte(':')
				(v18Value).MarshalEasyJSON(out)
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"item\":"
		out.RawString(prefix)
		if in.Item == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v19First := true
			for v19Name, v19Value := range in.Item {
				if v19First {
					v19First = false
				} else {
					out.RawByte(',')
				}
				out.IntStr(int(v19Name))
				out.RawByte(':')
				(v19Value).MarshalEasyJSON(out)
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"position\":"
		out.RawString(prefix)
		if in.Position == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v20First := true
			for v20Name, v20Value := range in.Position {
				if v20First {
					v20First = false
				} else {
					out.RawByte(',')
				}
				out.IntStr(int(v20Name))
				out.RawByte(':')
				(v20Value).MarshalEasyJSON(out)
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"gearScore\":"
		out.RawString(prefix)
		if in.GearScore == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v21First := true
			for v21Name, v21Value := range in.GearScore {
				if v21First {
					v21First = false
				} else {
					out.RawByte(',')
				}
				out.IntStr(int(v21Name))
				out.RawByte(':')
				(v21Value).MarshalEasyJSON(out)
			}
			out.RawByte('}')
		}
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v Tree) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson19a9978cEncodeStatefactoryStateMachineExample6(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v Tree) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson19a9978cEncodeStatefactoryStateMachineExample6(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *Tree) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson19a9978cDecodeStatefactoryStateMachineExample6(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *Tree) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson19a9978cDecodeStatefactoryStateMachineExample6(l, v)
}
