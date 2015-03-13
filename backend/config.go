package backend

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"euphoria.io/heim/aws/kms"
	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto/security"
)

var Config ServerConfig

func init() {
	flag.StringVar(&Config.HTTP.Listen, "http", ":8080", "")
	flag.StringVar(&Config.HTTP.Static, "static", "", "")

	flag.StringVar(&Config.Cluster.ServerID, "id", "singleton", "")
	flag.StringVar(&Config.Cluster.EtcdHome, "etcd", "", "etcd path for cluster coordination")
	flag.StringVar(&Config.Cluster.EtcdHost, "etcd-host", "", "address of a peer in etcd cluster")
	// TODO: -etcd-peers is deprecated
	flag.Var(&Config.Cluster.EtcdPeers, "etcd-peers", "comma-separated addresses of etcd peers")

	flag.StringVar(&Config.DB.DSN, "psql", "", "")

	flag.StringVar(&Config.Console.Listen, "control", "", "")
	flag.StringVar(&Config.Console.HostKey, "control-hostkey", "", "")
	flag.Var(&Config.Console.AuthKeys, "control-authkeys", "")

	flag.StringVar(&Config.KMS.Amazon.Region, "kms-aws-region", "", "")
	flag.StringVar(&Config.KMS.Amazon.KeyID, "kms-aws-key-id", "", "")
	flag.StringVar(&Config.KMS.AES256.KeyFile, "kms-local-key-file", "", "")
}

type CSV []string

func (k *CSV) String() string { return strings.Join(*k, ",") }

func (k *CSV) Set(flags string) error {
	*k = strings.Split(flags, ",")
	return nil
}

type ServerConfig struct {
	HTTP    HTTPConfig     `yaml:"http"`
	Cluster ClusterConfig  `yaml:"cluster,omitempty"`
	Console ConsoleConfig  `yaml:"console,omitempty"`
	DB      DatabaseConfig `yaml:"database"`
	KMS     KMSConfig      `yaml:"kms"`
}

type ClusterConfig struct {
	ServerID  string `yaml:"server-id"`
	Era       string `yaml:"-"`
	Version   string `yaml:"-"`
	EtcdPeers CSV    `yaml:"-"`
	EtcdHost  string `yaml:"etcd-host,omitempty"`
	EtcdHome  string `yaml:"etcd,omitempty"`
}

func (c *ClusterConfig) EtcdCluster() (cluster.Cluster, error) {
	if c.EtcdHost == "" {
		if len(c.EtcdPeers) > 0 {
			c.EtcdHost = c.EtcdPeers[0]
		} else {
			return nil, fmt.Errorf("cluster: etcd-host must be specified")
		}
	}
	return cluster.EtcdCluster(c.EtcdHome, c.EtcdHost, c.DescribeSelf())
}

func (c *ClusterConfig) DescribeSelf() *cluster.PeerDesc {
	return &cluster.PeerDesc{
		ID:      c.ServerID,
		Era:     c.Era,
		Version: c.Version,
	}
}

type ConsoleConfig struct {
	Listen   string `yaml:"listen"`
	HostKey  string `yaml:"host-key-file"`
	AuthKeys CSV    `yaml:"auth-key-files,flow"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type HTTPConfig struct {
	Listen string `yaml:"listen"`
	Static string `yaml:"static"`
}

type KMSConfig struct {
	AES256 struct {
		KeyFile string `yaml:"key-file"`
	} `yaml:"aes256,omitempty"`

	Amazon struct {
		Region string `yaml:"region"`
		KeyID  string `yaml:"key-id"`
	} `yaml:"amazon,omitempty"`
}

func (kc *KMSConfig) Get() (security.KMS, error) {
	switch {
	case kc.AES256.KeyFile != "":
		kms, err := kc.local()
		if err != nil {
			return nil, fmt.Errorf("kms: aes256: %s", err)
		}
		return kms, nil
	case kc.Amazon.Region != "" || kc.Amazon.KeyID != "":
		kms, err := kc.amazon()
		if err != nil {
			return nil, fmt.Errorf("kms: amazon: %s", err)
		}
		return kms, nil
	default:
		return nil, fmt.Errorf("kms: not configured")
	}
}

func (kc *KMSConfig) local() (security.KMS, error) {
	f, err := os.Open(kc.AES256.KeyFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	keySize := security.AES256.KeySize()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() != int64(keySize) {
		return nil, fmt.Errorf("key must be exactly %d bytes in size", keySize)
	}

	masterKey, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	kms := security.LocalKMS()
	kms.SetMasterKey(masterKey)
	return kms, nil
}

func (kc *KMSConfig) amazon() (security.KMS, error) {
	switch {
	case kc.Amazon.Region == "":
		return nil, fmt.Errorf("region must be specified")
	case kc.Amazon.KeyID == "":
		return nil, fmt.Errorf("key-id must be specified")
	}
	return kms.New(kc.Amazon.Region, kc.Amazon.KeyID)
}
