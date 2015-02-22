package cmd

import (
	"flag"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func newFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	NewFlagsFlag(fs, "newflags")
	return fs
}

func TestNewFlags(t *testing.T) {
	args := []string{"-x", "x", "-newflags", "y,z", "-y", "y", "-z=true"}

	Convey("New flags that aren't explicitly defined are parsed", t, func() {
		fs := newFlagSet()
		x := fs.String("x", "", "")
		So(fs.Parse(args), ShouldBeNil)
		So(*x, ShouldEqual, "x")
		So(fs.Args(), ShouldResemble, []string{})
	})

	Convey("Given new flags shouldn't clobber flags that are defined", t, func() {
		fs := newFlagSet()
		x := fs.String("x", "", "")
		z := fs.Bool("z", false, "")
		So(fs.Parse(args), ShouldBeNil)
		So(*x, ShouldEqual, "x")
		So(*z, ShouldBeTrue)
		So(fs.Args(), ShouldResemble, []string{})
	})
}
