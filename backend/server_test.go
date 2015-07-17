package backend

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCheckOrigin(t *testing.T) {
	tc := func(host, origin string) *http.Request {
		return &http.Request{
			Header: http.Header{"Origin": []string{origin}},
			Host:   host,
		}
	}

	Convey("CheckOrigin", t, func() {
		Convey("Accept if no origin is given", func() {
			So(checkOrigin(&http.Request{Host: "heim"}), ShouldBeTrue)
		})

		Convey("Accept if origin host matches request host", func() {
			So(checkOrigin(tc("heim", "http://heim/room/test")), ShouldBeTrue)
		})

		Convey("Accept if www. plus origin host matches request host", func() {
			So(checkOrigin(tc("heim", "http://www.heim/room/test")), ShouldBeTrue)
		})

		Convey("Reject if all prefix + origin host combinations fail to match request host", func() {
			So(checkOrigin(tc("heim", "http://ftp.heim/room/test")), ShouldBeFalse)
			So(checkOrigin(tc("heim", "http://heim2/room/test")), ShouldBeFalse)
		})

		Convey("Reject if origin is not a valid URL", func() {
			So(checkOrigin(tc("heim", "http://heim/%")), ShouldBeFalse)
		})
	})
}
