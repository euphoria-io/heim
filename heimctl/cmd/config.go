package cmd

import (
	"flag"
	"fmt"
	"os"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/backend/psql"
	"euphoria.io/heim/proto/snowflake"
)

var Version = "dev"

var config = flag.String("config", "", "path to local config (default: use config stored in etcd)")

func getCluster() (cluster.Cluster, error) {
	if backend.Config.Cluster.ServerID == "" {
		id, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("hostname error: %s", err)
		}
		backend.Config.Cluster.ServerID = id
	}

	era, err := snowflake.New()
	if err != nil {
		return nil, fmt.Errorf("era error: %s", err)
	}

	backend.Config.Cluster.Era = era.String()
	backend.Config.Cluster.Version = Version

	c, err := backend.Config.Cluster.EtcdCluster()
	if err != nil {
		return nil, fmt.Errorf("cluster error: %s", err)
	}

	if *config == "" {
		if err := backend.Config.LoadFromEtcd(c); err != nil {
			return nil, fmt.Errorf("config: %s", err)
		}
	} else {
		if err := backend.Config.LoadFromFile(*config); err != nil {
			return nil, fmt.Errorf("config: %s", err)
		}
	}

	return c, nil
}

func getBackend(cs cluster.Cluster) (*psql.Backend, error) {
	return psql.NewBackend(backend.Config.DB.DSN, cs, backend.Config.Cluster.DescribeSelf())
}
