package backend

import (
	"testing"

	"golang.org/x/net/context"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMemLogLatest(t *testing.T) {
	ctx := context.Background()
	msgs := []Message{
		{Content: "A"},
		{Content: "B"},
		{Content: "C"},
		{Content: "D"},
		{Content: "E"},
	}

	Convey("Partial response", t, func() {
		log := newMemLog()
		slice, err := log.Latest(ctx, 5)
		So(err, ShouldBeNil)
		So(slice, ShouldNotBeNil)
		So(len(slice), ShouldEqual, 0)

		log.post(&msgs[0])
		log.post(&msgs[1])
		log.post(&msgs[2])
		slice, err = log.Latest(ctx, 5)
		So(err, ShouldBeNil)
		So(slice, ShouldResemble, msgs[:3])
	})

	Convey("Full response", t, func() {
		log := newMemLog()
		for _, msg := range msgs {
			posted := msg
			log.post(&posted)
		}

		slice, err := log.Latest(ctx, 3)
		So(err, ShouldBeNil)
		So(slice, ShouldResemble, msgs[2:])
	})
}
