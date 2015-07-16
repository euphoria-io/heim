package cmd

import (
	"flag"
	"fmt"
	"os"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/mock"
	"euphoria.io/heim/backend/psql"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

var (
	Version = "dev"

	config = flag.String("config", os.Getenv("HEIM_CONFIG"),
		"path to local config (default: use config stored in etcd)")

	initializedConfig  *backend.ServerConfig
	initializedCluster cluster.Cluster
)

func getConfig(ctx scope.Context) (*backend.ServerConfig, error) {
	if initializedConfig != nil {
		return initializedConfig, nil
	}

	if *config == "" {
		if initializedCluster == nil {
			var err error
			initializedCluster, err = backend.Config.Cluster.EtcdCluster(ctx)
			if err != nil {
				return nil, err
			}
		}
		if err := backend.Config.LoadFromEtcd(initializedCluster); err != nil {
			return nil, fmt.Errorf("config: %s", err)
		}
	} else {
		if err := backend.Config.LoadFromFile(*config); err != nil {
			return nil, fmt.Errorf("config: %s", err)
		}
	}

	backend.RegisterBackend("mock", func(*proto.Heim) (proto.Backend, error) {
		return &mock.TestBackend{}, nil
	})
	backend.RegisterBackend("psql", func(heim *proto.Heim) (proto.Backend, error) {
		return psql.NewBackend(heim, backend.Config.DB.DSN)
	})

	return &backend.Config, nil
}

func getCluster(ctx scope.Context) (cluster.Cluster, error) {
	if initializedCluster != nil {
		return initializedCluster, nil
	}

	cfg, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}

	initializedCluster, err = cfg.Cluster.EtcdCluster(ctx)
	return initializedCluster, err
}

func getHeim(ctx scope.Context) (*proto.Heim, error) {
	cfg, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}

	era, err := snowflake.New()
	if err != nil {
		return nil, fmt.Errorf("era error: %s", err)
	}

	cfg.Cluster.Era = era.String()
	cfg.Cluster.Version = Version
	return cfg.Heim(ctx)
}

func getHeimWithPsqlBackend(ctx scope.Context) (*proto.Heim, *psql.Backend, error) {
	heim, err := getHeim(ctx)
	if err != nil {
		return nil, nil, err
	}

	b, ok := heim.Backend.(*psql.Backend)
	if !ok {
		return nil, nil, fmt.Errorf(
			"only psql backend is supported; -psql, HEIM_DSN, or db.dsn must be specified")
	}

	return heim, b, nil
}
