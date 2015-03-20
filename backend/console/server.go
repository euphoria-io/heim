package console

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"encoding/pem"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type Controller struct {
	listener net.Listener
	config   *ssh.ServerConfig
	backend  proto.Backend
	kms      security.KMS
	cluster  cluster.Cluster

	// TODO: key ssh.PublicKey
	authorizedKeys []ssh.PublicKey
}

func NewController(
	addr string, backend proto.Backend, kms security.KMS, c cluster.Cluster) (*Controller, error) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %s", addr, err)
	}

	ctrl := &Controller{
		listener: listener,
		backend:  backend,
		kms:      kms,
		cluster:  c,
	}

	ctrl.config = &ssh.ServerConfig{
		PublicKeyCallback: ctrl.authorizeKey,
	}

	return ctrl, nil
}

func (ctrl *Controller) authorizeKey(conn ssh.ConnMetadata, key ssh.PublicKey) (
	*ssh.Permissions, error) {

	marshaledKey := key.Marshal()
	for _, authorizedKey := range ctrl.authorizedKeys {
		if bytes.Compare(authorizedKey.Marshal(), marshaledKey) == 0 {
			return &ssh.Permissions{}, nil
		}
	}

	nodes, err := ctrl.cluster.GetDir("console/authorized_keys")
	if err != nil {
		if err == cluster.ErrNotFound {
			return nil, fmt.Errorf("unauthorized")
		}
		return nil, err
	}

	for path, value := range nodes {
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(value))
		if err != nil {
			fmt.Printf("bad authorized key from etcd: %s: %s\n", path, err)
		}
		if bytes.Compare(key.Marshal(), marshaledKey) == 0 {
			return &ssh.Permissions{}, nil
		}
	}

	return nil, fmt.Errorf("unauthorized")
}

func (ctrl *Controller) AddHostKeyFromCluster(host string) error {
	generate := func() (string, error) {
		// Generate an ECDSA key.
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return "", err
		}
		derBytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return "", err
		}
		w := &bytes.Buffer{}
		if err := pem.Encode(w, &pem.Block{Type: "EC PRIVATE KEY", Bytes: derBytes}); err != nil {
			return "", err
		}
		return w.String(), nil
	}
	pemString, err := ctrl.cluster.GetValueWithDefault(fmt.Sprintf("console/%s", host), generate)
	if err != nil {
		return fmt.Errorf("failed to get/generate host key: %s", err)
	}

	signer, err := ssh.ParsePrivateKey([]byte(pemString))
	if err != nil {
		return fmt.Errorf("failed to parse host key: %s", err)
	}

	ctrl.config.AddHostKey(signer)
	return nil
}

func (ctrl *Controller) AddHostKey(path string) error {
	pemBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	key, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return err
	}

	ctrl.config.AddHostKey(key)
	return nil
}

func (ctrl *Controller) AddAuthorizedKeys(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	curLine := 1

	for {
		startLine := curLine
		buf := &bytes.Buffer{}

		for {
			line, isPrefix, err := r.ReadLine()
			if err != nil && err != io.EOF {
				return err
			}
			buf.Write(line)
			curLine++
			if !isPrefix {
				break
			}
		}

		line := bytes.TrimSpace(buf.Bytes())
		if len(line) == 0 {
			break
		}

		key, _, _, _, err := ssh.ParseAuthorizedKey(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			fmt.Printf("%s:%d: not a public key: %s\n", path, startLine, err)
			continue
		}

		ctrl.authorizedKeys = append(ctrl.authorizedKeys, key)
	}

	return nil
}

func (ctrl *Controller) Serve() {
	for {
		conn, err := ctrl.listener.Accept()
		if err != nil {
			panic(fmt.Sprintf("controller accept: %s", err))
		}

		go ctrl.interact(conn)
	}
}

func (ctrl *Controller) interact(conn net.Conn) {
	_, nchs, reqs, err := ssh.NewServerConn(conn, ctrl.config)
	if err != nil {
		return
	}

	go ssh.DiscardRequests(reqs)

	for nch := range nchs {
		if nch.ChannelType() != "session" {
			nch.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		ch, reqs, err := nch.Accept()
		if err != nil {
			return
		}
		go ctrl.filterClientRequests(reqs)
		go ctrl.terminal(ch)
	}
}

func (ctrl *Controller) filterClientRequests(reqs <-chan *ssh.Request) {
	for req := range reqs {
		switch req.Type {
		case "shell":
			req.Reply(len(req.Payload) == 0, nil)
		case "pty-req":
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

func (ctrl *Controller) terminal(ch ssh.Channel) {
	defer ch.Close()

	term := terminal.NewTerminal(ch, "> ")
	for {
		line, err := term.ReadLine()
		if err != nil {
			fmt.Printf("terminal ReadLine: %s\n", err)
			break
		}

		cmd := parse(line)
		fmt.Printf("[control] > %v\n", cmd)
		switch cmd[0] {
		case "":
			continue
		case "quit":
			return
		case "shutdown":
			// TODO: graceful shutdown
			os.Exit(0)
		default:
			runCommand(ctrl, cmd[0], term, cmd[1:])
		}
	}
}

func parse(line string) []string {
	parts := strings.Split(strings.TrimSpace(line), " ")
	if len(parts) == 0 {
		parts[0] = ""
	}
	parts[0] = strings.ToLower(parts[0])
	return parts
}
