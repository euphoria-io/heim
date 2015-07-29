package etcd_test

import (
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/mock"
	"euphoria.io/heim/cluster/etcd/clustertest"
	"euphoria.io/heim/proto"
)

func TestIntegration(t *testing.T) {
	etcd, err := clustertest.StartEtcd()
	if err != nil {
		t.Fatal(err)
	}
	if etcd == nil {
		t.Fatal("can't test euphoria.io/heim/cluster/etcd: etcd not available in PATH")
	}
	defer etcd.Shutdown()

	backend.IntegrationTest(
		t, func(heim *proto.Heim) (proto.Backend, error) {
			heim.Cluster = etcd.Join("/test", "testcase", "era")
			return &mock.TestBackend{}, nil
		})
}
