package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh"
)

type SSHDialer struct {
	client *ssh.Client
}

func (sshDialer *SSHDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	return sshDialer.client.Dial("tcp", addr)
}

func MysqlUseSsh(dsnName string, dsn string) {
	// 格式如下:
	// 1.密码连接 "root:pass@127.0.0.1:22"
	// 2.密钥文件 "root@127.0.0.1:22/data/aa.key"

	// 1.密码连接
	re := regexp.MustCompile(`^([^:]+):([^@]+)@([^:]+):([^/]+)$`)
	matches := re.FindStringSubmatch(dsn)
	if len(matches) == 5 {
		// 使用密码连接
		config := &ssh.ClientConfig{
			User: matches[1],
			Auth: []ssh.AuthMethod{
				ssh.Password(matches[2]),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境中应使用ssh.FixedHostKey(hostKey)来确保安全性
		}

		conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", matches[3], matches[4]), config)
		if err != nil {
			log.Fatalf("无法连接SSH: %v", err)
		}

		// defer conn.Close()
		mysql.RegisterDialContext(dsnName, (&SSHDialer{conn}).Dial)
	}

	// 2.密钥文件连接
	re = regexp.MustCompile(`^([^:]+)@([^:]+):([^/]+)(/.+)$`)
	matches = re.FindStringSubmatch(dsn)

	if len(matches) == 5 {
		// SSH连接配置
		privateKey, err := os.ReadFile("./" + matches[4])
		if err != nil {
			log.Fatalf("无法读取私钥: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			log.Fatalf("无法解析私钥: %v", err)
		}

		config := &ssh.ClientConfig{
			User: matches[1],
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境中应使用ssh.FixedHostKey(hostKey)来确保安全性
		}

		// 连接到SSH服务器
		conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", matches[2], matches[3]), config)
		if err != nil {
			log.Fatalf("无法连接SSH: %v", err)
		}
		// defer conn.Close()
		mysql.RegisterDialContext(dsnName, (&SSHDialer{conn}).Dial)
	}
}
