package proto

import (
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCommandPayload(t *testing.T) {
	makeCommand := func(cmdType PacketType, payload interface{}) *Packet {
		payloadBytes, err := json.Marshal(payload)
		So(err, ShouldBeNil)
		return &Packet{Type: cmdType, Data: payloadBytes}
	}

	Convey("Send", t, func() {
		expected := &SendCommand{Content: "Test"}
		cmd := makeCommand(SendType, expected)
		payload, err := cmd.Payload()
		So(err, ShouldBeNil)
		So(payload, ShouldResemble, expected)
	})

	Convey("Log", t, func() {
		expected := &LogCommand{N: 5}
		cmd := makeCommand(LogType, expected)
		payload, err := cmd.Payload()
		So(err, ShouldBeNil)
		So(payload, ShouldResemble, expected)
	})

	Convey("Error", t, func() {
		cmd := &Packet{Type: PacketType("unknown")}
		_, err := cmd.Payload()
		So(err, ShouldResemble, fmt.Errorf("invalid command type: unknown"))
	})
}
