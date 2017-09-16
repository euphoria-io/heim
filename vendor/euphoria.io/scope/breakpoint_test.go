package scope

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBpmap(t *testing.T) {
	Convey("Lookups and storage", t, func() {
		bpm := bpmap{}

		So(bpm.get(false, "a"), ShouldBeNil)
		ch := bpm.get(true, "a")
		So(bpm.get(false, "a"), ShouldEqual, ch)
		So(bpm.get(true, "a"), ShouldEqual, ch)
		So(bpm.get(true, "a", "b", "c"), ShouldBeNil)

		delete(bpm, "a")
		nested := bpm.get(true, "a", "b", "c")
		So(bpm.get(false, "a", "b", "c"), ShouldEqual, nested)
		So(bpm.get(true, "a", "b", "c"), ShouldEqual, nested)
		So(bpm.get(true, "a", "b"), ShouldBeNil)
		So(bpm.get(false, "a", "b"), ShouldBeNil)

		So(bpm.get(true), ShouldBeNil)
		So(bpm.get(false), ShouldBeNil)
	})
}
