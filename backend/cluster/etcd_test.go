package cluster_test

import (
	"heim/backend/cluster"
	"heim/backend/cluster/clustertest"
	"heim/proto/security"
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
		So(<-a.Watch(), ShouldResemble, &cluster.PeerAliveEvent{cluster.PeerDesc{ID: "b", Era: "1"}})
		So(<-a.Watch(), ShouldResemble, &cluster.PeerAliveEvent{cluster.PeerDesc{ID: "b", Era: "2"}})
	})

	Convey("Secrets are created if necessary", t, func() {
		kms := security.LocalKMS()
		a := s.Join("/secrets1", "a", "0")
		defer a.Part()

		secret, err := a.GetSecret(kms, "test1", 16)
		So(err, ShouldBeNil)
		So(len(secret), ShouldEqual, 16)

		secretCopy, err := a.GetSecret(kms, "test1", 16)
		So(err, ShouldBeNil)
		So(string(secretCopy), ShouldEqual, string(secret))
	})

	Convey("Race to create secret is conceded gracefully", t, func() {
		kms := &syncKMS{
			KMS: security.LocalKMS(),
			c:   make(chan struct{}),
		}

		a := s.Join("/secrets1", "a", "0")
		defer a.Part()

		sc := make(chan []byte)
		errc := make(chan error)
		go func() {
			s, err := a.GetSecret(kms, "test2", 16)
			errc <- err
			sc <- s
		}()

		// Synchronize with secret generation.
		<-kms.c

		// Set the secret before releasing the goroutine.
		secret, err := a.GetSecret(kms.KMS, "test2", 16)
		So(err, ShouldBeNil)

		// Release the goroutine and verify it gets the secret that was set.
		kms.c <- struct{}{}
		So(<-errc, ShouldBeNil)
		So(string(<-sc), ShouldEqual, string(secret))
	})
}

type syncKMS struct {
	security.KMS
	c chan struct{}
}

func (s *syncKMS) GenerateNonce(bytes int) ([]byte, error) {
	s.c <- struct{}{}
	<-s.c
	return s.KMS.GenerateNonce(bytes)
}
