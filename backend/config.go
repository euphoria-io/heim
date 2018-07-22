package backend

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"strconv"
	"time"

	"github.com/savaki/geoip2"

	"gopkg.in/yaml.v2"

	"euphoria.io/heim/aws/kms"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/cluster/etcd"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/templates"
	"euphoria.io/scope"
)

var (
	Config = ServerConfig{
		CommonEmailParams: &proto.DefaultCommonEmailParams,
	}

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

	flag.StringVar(&Config.StaticPath, "static", "", "path to static files")

	flag.StringVar(&Config.Cluster.ServerID, "id", env("HEIM_ID", ""), "")
	flag.StringVar(&Config.Cluster.EtcdHome, "etcd", env("HEIM_ETCD_HOME", ""),
		"etcd path for cluster coordination")
	flag.StringVar(&Config.Cluster.EtcdHost, "etcd-host", env("HEIM_ETCD", ""),
		"address of a peer in etcd cluster")

	flag.StringVar(&Config.DB.DSN, "psql", env("HEIM_DSN", ""), "dsn url of heim postgres database")
	count, _ := strconv.Atoi(env("HEIM_DB_MAX_CONNECTIONS", "0"))
	flag.IntVar(&Config.DB.MaxConnCount, "psql-max-connections", count, "maximum db connection count")

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
	flag.BoolVar(&Config.SetInsecureCookies, "set-insecure-cookies", false, "allow non-https cookies")

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
	*proto.CommonEmailParams `yaml:"site"`

	AllowRoomCreation     bool          `yaml:"allow_room_creation"`
	NewAccountMinAgentAge time.Duration `yaml:"new_account_min_agent_age"`
	RoomEntryMinAgentAge  time.Duration `yaml:"room_entry_min_agent_age"`
	SetInsecureCookies    bool          `yaml:"set_insecure_cookies"`

	StaticPath string `yaml:"static_path"`

	Cluster ClusterConfig  `yaml:"cluster,omitempty"`
	Console ConsoleConfig  `yaml:"console,omitempty"`
	DB      DatabaseConfig `yaml:"database"`
	KMS     KMSConfig      `yaml:"kms"`
	Email   EmailConfig    `yaml:"email"`
	GeoIP   GeoIPConfig    `yaml:"geoip"`
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
	pageTemplater, err := LoadPageTemplates(filepath.Join(cfg.StaticPath, "pages"))
	if err != nil {
		return nil, fmt.Errorf("page templates: %s", err)
	}

	// Load and verify page templates.
	c, err := cfg.Cluster.EtcdCluster(ctx)
	if err != nil {
		return nil, err
	}

	kms, err := cfg.KMS.Get()
	if err != nil {
		return nil, err
	}

	emailTemplater, emailDeliverer, err := cfg.Email.Get(cfg)
	if err != nil {
		return nil, err
	}

	heim := &proto.Heim{
		Context:        ctx,
		Cluster:        c,
		PeerDesc:       cfg.Cluster.DescribeSelf(),
		KMS:            kms,
		EmailDeliverer: emailDeliverer,
		EmailTemplater: emailTemplater,
		GeoIP:          cfg.GeoIP.Api(),
		PageTemplater:  pageTemplater,
		SiteName:       cfg.SiteName,
		StaticPath:     cfg.StaticPath,
	}

	backend, err := cfg.GetBackend(heim)
	if err != nil {
		return nil, err
	}

	emojiPath := filepath.Join(cfg.StaticPath, "emoji.json")
	if err = proto.LoadEmoji(emojiPath); err != nil {
		fmt.Printf("error loading %s: %s\n", emojiPath, err)
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
	switch c.EtcdHost {
	case "":
		return nil, fmt.Errorf("cluster: etcd-host must be specified")
	case "mock":
		return &cluster.TestCluster{}, nil
	default:
		return etcd.EtcdCluster(ctx, c.EtcdHome, c.EtcdHost, c.DescribeSelf())
	}
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
	DSN          string `yaml:"dsn"`
	MaxConnCount int    `yaml:"max-connection-count,omitempty"`
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
	AuthMethod string `yaml:"auth_method"` // must be "", "CRAM-MD5", or "PLAIN"
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Identity   string `yaml:"identity"`
	UseTLS     bool   `yaml:"use_tls"`
	Templates  string `yaml:"templates"`
}

func (ec *EmailConfig) Get(cfg *ServerConfig) (*templates.Templater, emails.Deliverer, error) {
	proto.DefaultCommonEmailParams = *cfg.CommonEmailParams
	localDomain := cfg.CommonEmailParams.EmailDomain
	cfg.CommonEmailParams.CommonData.LocalDomain = localDomain

	// Load templates and configure email sender.
	templater := &templates.Templater{}
	// TODO: replace -static with a better sense of a static root
	if errs := templater.Load(filepath.Join(cfg.StaticPath, "..", "email")); errs != nil {
		return nil, nil, errs[0]
	}

	// Verify templates.
	if errs := proto.ValidateEmailTemplates(templater); errs != nil {
		for _, err := range errs {
			fmt.Printf("error: %s\n", err)
		}
		return nil, nil, fmt.Errorf("template validation failed: %s...", errs[0].Error())
	}

	// Set up deliverer.
	fmt.Printf("setting up deliverer for %#v\n", ec)
	switch ec.Server {
	case "":
		return templater, nil, nil
	case "$stdout":
		return templater, &mockDeliverer{Writer: os.Stdout}, nil
	default:
		var sslHost string
		if ec.UseTLS {
			var err error
			sslHost, _, err = net.SplitHostPort(ec.Server)
			if err != nil {
				return nil, nil, err
			}
		}

		var auth smtp.Auth
		switch strings.ToUpper(ec.AuthMethod) {
		case "":
		case "CRAM-MD5":
			auth = smtp.CRAMMD5Auth(ec.Username, ec.Password)
		case "PLAIN":
			if !ec.UseTLS {
				return nil, nil, fmt.Errorf("PLAIN authentication requires TLS")
			}
			auth = smtp.PlainAuth(ec.Identity, ec.Username, ec.Password, sslHost)
		}

		deliverer := emails.NewSMTPDeliverer(localDomain, ec.Server, sslHost, auth)
		return templater, deliverer, nil
	}
}

type mockDeliverer struct {
	io.Writer
}

func (d *mockDeliverer) LocalName() string { return "localhost" }

func (d *mockDeliverer) Deliver(ctx scope.Context, ref *emails.EmailRef) error {
	fmt.Fprintf(d, "mock delivery of email from %s to %s:\n", ref.SendFrom, ref.SendTo)
	d.Write(ref.Message)
	fmt.Fprintf(d, "----- END OF MESSAGE -----\n")
	return nil
}

type GeoIPConfig struct {
	UserID     string `yaml:"user-id"`
	LicenseKey string `yaml:"license-key"`
}

func (c *GeoIPConfig) Api() *geoip2.Api { return geoip2.New(c.UserID, c.LicenseKey) }
