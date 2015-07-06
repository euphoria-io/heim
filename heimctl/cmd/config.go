package cmd

import (
	"flag"
	"fmt"
	"os"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/psql"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

var Version = "dev"

var config = flag.String("config", os.Getenv("HEIM_CONFIG"),
	"path to local config (default: use config stored in etcd)")

func getCluster(ctx scope.Context) (cluster.Cluster, error) {
	era, err := snowflake.New()
	if err != nil {
		return nil, fmt.Errorf("era error: %s", err)
	}

	backend.Config.Cluster.Era = era.String()
	backend.Config.Cluster.Version = Version

	c, err := backend.Config.Cluster.EtcdCluster(ctx)
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

func getBackend(ctx scope.Context, cs cluster.Cluster) (*psql.Backend, error) {
	return psql.NewBackend(ctx, backend.Config.DB.DSN, cs, backend.Config.Cluster.DescribeSelf())
}
