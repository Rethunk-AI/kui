package sshtunnel

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds parsed qemu+ssh connection parameters.
type SSHConfig struct {
	Host    string
	Port    int
	User    string
	Keyfile string
}

// ParseQemuSSH parses a qemu+ssh URI and returns SSH connection parameters.
// Returns nil if the URI is not qemu+ssh.
func ParseQemuSSH(uri string) (*SSHConfig, error) {
	raw := strings.TrimSpace(uri)
	if raw == "" || !strings.HasPrefix(raw, "qemu+ssh://") {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse qemu+ssh uri: %w", err)
	}
	if parsed.Scheme != "qemu+ssh" {
		return nil, nil
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("qemu+ssh uri missing host: %q", raw)
	}
	port := 22
	if p := parsed.Port(); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			port = n
		}
	}
	user := "root"
	if parsed.User != nil {
		if u := parsed.User.Username(); u != "" {
			user = u
		}
	}
	keyfile := strings.TrimSpace(parsed.Query().Get("keyfile"))
	return &SSHConfig{Host: host, Port: port, User: user, Keyfile: keyfile}, nil
}

// DialRemote establishes an SSH connection and dials the given address on the remote host.
// The address is interpreted on the remote host (e.g. "127.0.0.1:5900").
// cfg.Keyfile must be set. Caller must close the returned connection.
func DialRemote(ctx context.Context, cfg *SSHConfig, network, addr string) (net.Conn, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ssh config required")
	}
	if strings.TrimSpace(cfg.Keyfile) == "" {
		return nil, fmt.Errorf("keyfile is required for qemu+ssh")
	}
	key, err := os.ReadFile(cfg.Keyfile)
	if err != nil {
		return nil, fmt.Errorf("read keyfile %q: %w", cfg.Keyfile, err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	sshAddr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", sshAddr)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", sshAddr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshAddr, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake: %w", err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	remoteConn, err := client.Dial(network, addr)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("dial %s on remote: %w", addr, err)
	}
	return &sshTunnelConn{Conn: remoteConn, client: client}, nil
}

type sshTunnelConn struct {
	net.Conn
	client *ssh.Client
}

func (c *sshTunnelConn) Close() error {
	err := c.Conn.Close()
	_ = c.client.Close()
	return err
}
