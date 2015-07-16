package backend

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"euphoria.io/heim/aws/kms"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

var (
	Config ServerConfig

	backendFactories = map[string]proto.BackendFactory{}
)

func init() {
	env := func(key, defaultValue string) string {
		val := os.Getenv(key)
		if val == "" {
			val = defaultValue
		}
		return val
	}

	flag.StringVar(&Config.Cluster.ServerID, "id", env("HEIM_ID", ""), "")
	flag.StringVar(&Config.Cluster.EtcdHome, "etcd", env("HEIM_ETCD_HOME", ""),
		"etcd path for cluster coordination")
	flag.StringVar(&Config.Cluster.EtcdHost, "etcd-host", env("HEIM_ETCD", ""),
		"address of a peer in etcd cluster")

	flag.StringVar(&Config.DB.DSN, "psql", env("HEIM_DSN", ""), "dsn url of heim postgres database")

	flag.StringVar(&Config.Console.HostKey, "console-hostkey", env("HEIM_CONSOLE_HOST_KEY", ""),
		"path to file containing host key for ssh console")
	flag.Var(&Config.Console.AuthKeys, "console-authkeys",
		"comma-separated paths to files containing authorized keys for console clients")
	Config.Console.AuthKeys.Set(env("HEIM_CONSOLE_AUTH_KEYS", ""))

	flag.StringVar(&Config.KMS.Amazon.Region, "kms-aws-region", env("HEIM_KMS_AWS_REGION", ""),
		"name of the AWS region to use for crypto")
	flag.StringVar(&Config.KMS.Amazon.KeyID, "kms-aws-key-id", env("HEIM_KMS_AWS_KEY_ID", ""),
		"id of the AWS key to use for crypto")
	flag.StringVar(&Config.KMS.AES256.KeyFile, "kms-local-key-file", env("HEIM_KMS_LOCAL_KEY", ""),
		"path to file containing a 256-bit key for using local key-management instead of AWS")

	flag.BoolVar(&Config.AllowRoomCreation, "allow-room-creation", true, "allow rooms to be created")

	flag.StringVar(&Config.Email.Server, "smtp-server", "", "address of SMTP server to send mail through")
	flag.StringVar(&Config.Email.AuthMethod, "smtp-auth-method", "",
		`authenticate when using the SMTP server (must be either "CRAM-MD5" or "PLAIN")`)
	flag.StringVar(&Config.Email.Username, "smtp-username", "",
		"authenticate with SMTP server using this username (CRAM-MD5 or PLAIN auth)")
	flag.StringVar(&Config.Email.Password, "smtp-password", "",
		"authenticate with SMTP server using this password (CRAM-MD5 or PLAIN auth)")
	flag.StringVar(&Config.Email.Identity, "smtp-identity", "",
		"authenticate with SMTP server using this identity (PLAIN auth only)")
	flag.BoolVar(&Config.Email.UseTLS, "smtp-use-tls", true, "require TLS with SMTP server")
	flag.StringVar(&Config.Email.Templates, "email-templates", "", "path to email templates")
}

func RegisterBackend(name string, factory proto.BackendFactory) { backendFactories[name] = factory }

type CSV []string

func (k *CSV) String() string { return strings.Join(*k, ",") }

func (k *CSV) Set(flags string) error {
	*k = strings.Split(flags, ",")
	return nil
}

type ServerConfig struct {
	AllowRoomCreation     bool          `yaml:"allow_room_creation"`
	NewAccountMinAgentAge time.Duration `yaml:"new_account_min_agent_age"`
	RoomEntryMinAgentAge  time.Duration `yaml:"room_entry_min_agent_age"`

	Cluster ClusterConfig  `yaml:"cluster,omitempty"`
	Console ConsoleConfig  `yaml:"console,omitempty"`
	DB      DatabaseConfig `yaml:"database"`
	KMS     KMSConfig      `yaml:"kms"`
	Email   EmailConfig    `yaml:"email"`
}

func (cfg *ServerConfig) String() string {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Sprintf("marshal error: %s", err)
	}
	return string(data)
}

func (cfg *ServerConfig) LoadFromEtcd(c cluster.Cluster) error {
	cfgString, err := c.GetValue("config")
	if err != nil {
		return fmt.Errorf("load from etcd: %s", err)
	}

	if err := yaml.Unmarshal([]byte(cfgString), cfg); err != nil {
		return fmt.Errorf("load from etcd: %s", err)
	}

	return nil
}

func (cfg *ServerConfig) LoadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("load from file: %s: %s", path, err)
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("load from file: %s: %s", path, err)
	}

	fmt.Printf("parsing config:\n%s\n", string(data))
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("load from file: %s: %s", path, err)
	}

	return nil
}

func (cfg *ServerConfig) Heim(ctx scope.Context) (*proto.Heim, error) {
	c, err := cfg.Cluster.EtcdCluster(ctx)
	if err != nil {
		return nil, err
	}

	kms, err := cfg.KMS.Get()
	if err != nil {
		return nil, err
	}

	emailer, err := cfg.Email.Get()
	if err != nil {
		return nil, err
	}

	heim := &proto.Heim{
		Context:  ctx,
		Cluster:  c,
		PeerDesc: cfg.Cluster.DescribeSelf(),
		KMS:      kms,
		Emailer:  emailer,
	}

	backend, err := cfg.GetBackend(heim)
	if err != nil {
		if err != nil {
			return nil, err
		}
	}

	heim.Backend = backend
	return heim, nil
}

func (cfg *ServerConfig) backendFactory() string {
	if cfg.DB.DSN == "" {
		return "mock"
	} else {
		return "psql"
	}
}

func (cfg *ServerConfig) GetBackend(heim *proto.Heim) (proto.Backend, error) {
	name := cfg.backendFactory()
	factory, ok := backendFactories[name]
	if !ok {
		return nil, fmt.Errorf("no backend factory registered: %s", name)
	}
	return factory(heim)
}

type ClusterConfig struct {
	ServerID string `yaml:"server-id"`
	Era      string `yaml:"-"`
	Version  string `yaml:"-"`
	EtcdHost string `yaml:"etcd-host,omitempty"`
	EtcdHome string `yaml:"etcd,omitempty"`
}

func (c *ClusterConfig) EtcdCluster(ctx scope.Context) (cluster.Cluster, error) {
	if c.EtcdHost == "" {
		return nil, fmt.Errorf("cluster: etcd-host must be specified")
	}
	return cluster.EtcdCluster(ctx, c.EtcdHome, c.EtcdHost, c.DescribeSelf())
}

func (c *ClusterConfig) DescribeSelf() *cluster.PeerDesc {
	if c.ServerID == "" {
		return nil
	}
	return &cluster.PeerDesc{
		ID:      c.ServerID,
		Era:     c.Era,
		Version: c.Version,
	}
}

type ConsoleConfig struct {
	HostKey  string `yaml:"host-key-file"`
	AuthKeys CSV    `yaml:"auth-key-files,flow"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
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

type EmailConfig struct {
	Server     string `yaml:"server"`
	LocalName  string `yaml:"local_name"`
	AuthMethod string `yaml:"auth_method"` // must be "", "CRAM-MD5", or "PLAIN"
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Identity   string `yaml:"identity"`
	UseTLS     bool   `yaml:"use_tls"`
	Templates  string `yaml:"templates"`
}

func (ec *EmailConfig) Get() (emails.Emailer, error) {
	if ec.Server == "" {
		return &emails.TestEmailer{}, nil
	}

	var sslHost string
	if ec.UseTLS {
		var err error
		sslHost, _, err = net.SplitHostPort(ec.Server)
		if err != nil {
			return nil, err
		}
	}

	var auth smtp.Auth
	switch ec.AuthMethod {
	case "":
	case "CRAM-MD5":
		auth = smtp.CRAMMD5Auth(ec.Username, ec.Password)
	case "PLAIN":
		if !ec.UseTLS {
			return nil, fmt.Errorf("PLAIN authentication requires TLS")
		}
		auth = smtp.PlainAuth(ec.Identity, ec.Username, ec.Password, sslHost)
	}

	// Load templates and configure email sender.
	emailer, err := emails.NewSMTPEmailer(ec.Templates, ec.LocalName, ec.Server, sslHost, auth)
	if err != nil {
		return nil, err
	}

	// Verify templates.
	if errs := proto.ValidateEmailTemplates(emailer.Templater); errs != nil {
		for _, err := range errs {
			fmt.Printf("error: %s", err)
		}
		return nil, fmt.Errorf("template validation failed: %s...", errs[0].Error())
	}

	return emailer, nil
}
