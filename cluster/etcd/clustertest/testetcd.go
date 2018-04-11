package clustertest // import "euphoria.io/heim/cluster/etcd/clustertest"

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"euphoria.io/heim/cluster"
	"euphoria.io/heim/cluster/etcd"
	"euphoria.io/scope"
)

func pickPort() (int, error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	parts := strings.Split(ln.Addr().String(), ":")
	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, err
	}
	if err := ln.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

func StartEtcd() (*EtcdServer, error) {
	path, err := exec.LookPath("etcd")
	if err != nil {
		return nil, nil
	}

	d, err := ioutil.TempDir("", "etcd_test")
	if err != nil {
		return nil, fmt.Errorf("start etcd: tempdir error: %s", err)
	}

	port, err := pickPort()
	if err != nil {
		return nil, fmt.Errorf("start etcd: port selection error: %s", err)
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	cmd := exec.Command(
		path,
		"--force-new-cluster",
		"--data-dir", d,
		"--listen-client-urls", url,
		"--listen-peer-urls", "http://localhost:0",
		"--advertise-client-urls", url,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("start etcd: pipe error: %s", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start etcd: exec error: %s", err)
	}

	s := &EtcdServer{
		d:    d,
		cmd:  cmd,
		addr: url,
	}

	ch := make(chan string)
	go s.consumeStderr(stderr, ch)

	timeout := time.After(10 * time.Second)
	select {
	case <-timeout:
		defer s.Shutdown()
		return nil, fmt.Errorf("start etcd: timeout waiting for listen addr")
	case addr := <-ch:
		if addr == "" {
			defer s.Shutdown()
			return nil, fmt.Errorf("start etcd: failed to start")
		}
	}

	return s, nil
}

type EtcdServer struct {
	t    testing.TB
	d    string
	cmd  *exec.Cmd
	addr string
}

func (s *EtcdServer) consumeStderr(stderr io.ReadCloser, ch chan<- string) {
	// Read until we see a line like this:
	// 2015/02/24 09:52:20 etcdserver: published {Name:default ClientURLs:[http://localhost:2379
	// http://localhost:4001]} to cluster 7e27652122e8b2ae

	marker := "etcdserver: published {Name:default ClientURLs:["
	r := bufio.NewReader(stderr)
	atStart := true
	for {
		line, isPrefix, err := r.ReadLine()
		if err != nil {
			fmt.Printf("read error: %s\n", err)
			close(ch)
			return
		}
		if atStart {
			lineStr := string(line)
			fmt.Printf("%s\n", lineStr)
			if idx := strings.Index(lineStr, marker); idx >= 0 {
				idx += len(marker)
				if space := strings.IndexRune(lineStr[idx:], ' '); space >= 0 {
					// Found a client URL, send it back over the channel
					// and go into blind consumption mode.
					ch <- lineStr[idx : idx+space]
					close(ch)
					break
				}
			}
		}
		atStart = !isPrefix
	}

	// Consume the remainder.
	io.Copy(ioutil.Discard, stderr)
}

func (s *EtcdServer) Shutdown() error {
	defer os.RemoveAll(s.d)
	if err := s.cmd.Process.Kill(); err != nil {
		return err
	}
	if err := s.cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func (s *EtcdServer) Join(root, id, era string) cluster.Cluster {
	desc := &cluster.PeerDesc{
		ID:  id,
		Era: era,
	}
	c, err := etcd.EtcdCluster(scope.New(), root, s.addr, desc)
	if err != nil {
		panic(fmt.Sprintf("error joining cluster: %s", err))
	}
	return c
}
