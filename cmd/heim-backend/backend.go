package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"heim/aws/kms"
	"heim/backend"
	"heim/backend/cluster"
	"heim/backend/console"
	"heim/backend/psql"
	_ "heim/cmd" // for -newflags
	"heim/proto"
	"heim/proto/security"
	"heim/proto/snowflake"
)

var (
	addr    = flag.String("http", ":8080", "")
	id      = flag.String("id", "singleton", "")
	psqlDSN = flag.String("psql", "psql", "")
	static  = flag.String("static", "", "")

	ctrlAddr     = flag.String("control", ":2222", "")
	ctrlHostKey  = flag.String("control-hostkey", "", "")
	ctrlAuthKeys = flag.String("control-authkeys", "", "")

	kmsAWSRegion    = flag.String("kms-aws-region", "us-west-2", "")
	kmsAWSKeyID     = flag.String("kms-aws-key-id", "", "")
	kmsLocalKeyFile = flag.String("kms-local-key-file", "", "")
)

var version string

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("hostname error: %s", err)
	}

	era, err := snowflake.New()
	if err != nil {
		return fmt.Errorf("era error: %s", err)
	}

	serverDesc := &cluster.PeerDesc{
		ID:      id,
		Era:     era.String(),
		Version: version,
	}

	c, err := cluster.EtcdClusterFromFlags(serverDesc)
	if err != nil {
		return fmt.Errorf("cluster error: %s", err)
	}

	kms, err := getKMS()
	if err != nil {
		return fmt.Errorf("kms error: %s", err)
	}

	b, err := psql.NewBackend(*psqlDSN, c, serverDesc)
	if err != nil {
		return fmt.Errorf("backend error: %s", err)
	}
	defer b.Close()

	if err := controller(b, kms); err != nil {
		return fmt.Errorf("controller error: %s", err)
	}

	server, err := backend.NewServer(b, c, kms, serverDesc.ID, serverDesc.Era, *static)
	if err != nil {
		return fmt.Errorf("server error: %s", err)
	}

	fmt.Printf("serving era %s on %s\n", serverDesc.Era, *addr)
	http.ListenAndServe(*addr, newVersioningHandler(server))
	return nil
}

func controller(b proto.Backend, kms security.KMS) error {
	if *ctrlAddr != "" {
		ctrl, err := console.NewController(*ctrlAddr, b, kms)
		if err != nil {
			return err
		}

		if *ctrlHostKey != "" {
			if err := ctrl.AddHostKey(*ctrlHostKey); err != nil {
				return err
			}
		}

		if *ctrlAuthKeys != "" {
			if err := ctrl.AddAuthorizedKeys(*ctrlAuthKeys); err != nil {
				return err
			}
		}

		go ctrl.Serve()
	}
	return nil
}

type versioningHandler struct {
	version string
	handler http.Handler
}

func newVersioningHandler(handler http.Handler) http.Handler {
	return &versioningHandler{
		version: version,
		handler: handler,
	}
}

func (vh *versioningHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if vh.version != "" {
		w.Header().Set("X-Heim-Version", vh.version)
	}
	vh.handler.ServeHTTP(w, r)
}

func getKMS() (security.KMS, error) {
	switch {
	case *kmsLocalKeyFile != "":
		kms, err := localKMS(*kmsLocalKeyFile)
		if err != nil {
			return nil, fmt.Errorf("kms-local-key-file: %s", err)
		}
		return kms, nil
	case *kmsAWSKeyID != "":
		if *kmsAWSRegion == "" {
			return nil, fmt.Errorf("--kms-aws-region required if --kms-aws-key-id is specified")
		}
		kms, err := kms.New(*kmsAWSRegion, *kmsAWSKeyID)
		if err != nil {
			return nil, fmt.Errorf("kms-aws: %s", err)
		}
		return kms, nil
	default:
		return nil, fmt.Errorf("--kms-aws-key-id or --kms-local-key-file required")
	}
}

func localKMS(keyPath string) (security.KMS, error) {
	f, err := os.Open(keyPath)
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
