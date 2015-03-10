package mock

import (
	"testing"

	"euphoria.io/heim/proto"
	"euphoria.io/scope"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMemLogLatest(t *testing.T) {
	ctx := scope.New()
	msgs := []proto.Message{
		{ID: 1, Content: "A"},
		{ID: 2, Content: "B"},
		{ID: 15, Content: "C"},
		{ID: 19, Content: "D"},
		{ID: 20, Content: "E"},
	}

	Convey("Partial response", t, func() {
		log := newMemLog()
		slice, err := log.Latest(ctx, 5, 0)
		So(err, ShouldBeNil)
		So(slice, ShouldNotBeNil)
		So(len(slice), ShouldEqual, 0)

		log.post(&msgs[0])
		log.post(&msgs[1])
		log.post(&msgs[2])
		slice, err = log.Latest(ctx, 5, 0)
		So(err, ShouldBeNil)
		So(slice, ShouldResemble, msgs[:3])
	})

	Convey("Full response", t, func() {
		log := newMemLog()
		for _, msg := range msgs {
			posted := msg
			log.post(&posted)
		}

		slice, err := log.Latest(ctx, 3, 0)
		So(err, ShouldBeNil)
		So(slice, ShouldResemble, msgs[2:])
	})

	Convey("Before", t, func() {
		log := newMemLog()
		for _, msg := range msgs {
			posted := msg
			log.post(&posted)
		}

		slice, err := log.Latest(ctx, 3, 20)
		So(err, ShouldBeNil)
		So(slice, ShouldResemble, msgs[1:4])
	})
}
