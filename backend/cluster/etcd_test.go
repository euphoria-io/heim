package cluster_test

import (
	"heim/backend/cluster"
	"heim/backend/cluster/clustertest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEtcdCluster(t *testing.T) {
	s, err := clustertest.StartEtcd()
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Skipf("etcd not in PATH, skipping tests")
	}
	defer s.Shutdown()

	Convey("Observe peer departure", t, func() {
		a := s.Join("/departure", "a", "0")
		// no defer a.Part() because we'll do that explicitly
		b := s.Join("/departure", "b", "0")
		defer b.Part()

		So(<-a.Watch(), ShouldResemble, &cluster.PeerJoinedEvent{cluster.PeerDesc{ID: "b", Era: "0"}})
		a.Part()
		So(<-b.Watch(), ShouldResemble, &cluster.PeerLostEvent{cluster.PeerDesc{ID: "a"}})
	})

	Convey("Observe initial peers upon joining", t, func() {
		a := s.Join("/initial", "a", "0")
		defer a.Part()
		So(a.Peers(), ShouldResemble,
			[]cluster.PeerDesc{
				{ID: "a", Era: "0"},
			})

		b := s.Join("/initial", "b", "0")
		defer b.Part()
		So(b.Peers(), ShouldResemble,
			[]cluster.PeerDesc{
				{ID: "a", Era: "0"},
				{ID: "b", Era: "0"},
			})
	})

	Convey("Updates are seen", t, func() {
		a := s.Join("/updates", "a", "0")
		defer a.Part()
		b := s.Join("/updates", "b", "0")
		defer b.Part()

		b.Update(&cluster.PeerDesc{ID: "b", Era: "1"})
		b.Update(&cluster.PeerDesc{ID: "b", Era: "2"})
		So(<-a.Watch(), ShouldResemble, &cluster.PeerJoinedEvent{cluster.PeerDesc{ID: "b", Era: "0"}})
		So(<-a.Watch(), ShouldResemble, &cluster.PeerAliveEvent{cluster.PeerDesc{ID: "b", Era: "0"}})
		//So(<-a.Watch(), ShouldResemble, &cluster.PeerAliveEvent{cluster.PeerDesc{ID: "b", Era: "1"}})
		//So(<-a.Watch(), ShouldResemble, &cluster.PeerAliveEvent{cluster.PeerDesc{ID: "b", Era: "2"}})
	})
}
