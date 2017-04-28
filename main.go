// Command socks-tunnel listens on -listen and forwards requests to -dial using
// the -via port that a SOCKS5 proxy is running on.
//
// So, if my ~/.ssh/config looks like this:
//
//     Host uv
//           HostName ultraviolet-iga-di-1.ops.palantir.local
//           ProxyCommand ssh -oGSSAPIAuthentication=yes gamma-bastion2.palantircloud.com nc %h %p
//           DynamicForward localhost:9090
//
// I can then `ssh uv`, then from a new, local shell, run:
//
//     socks-tunnel -listen localhost:8000 -via 9090 -dial ultraviolet-sga-someapp-1.ops.palantir.local:8443
//
// And then I can hit localhost:8000 from my local machine as if I'm in PCloud
// hitting ^that service, possibly using inspect-tls to see what's up if I'm
// seeing TLS issues, etc...
//
// Note: this makes simple TCP connection and does not do TLS
// handshakes/decryption - so in this example, if that 8443 port was serving
// HTTPS, then localhost:8000 is also HTTPS, and connecting to 8000 would be
// doing the TLS handshake thing with the 8443 service and seeing the cert that
// the 8443 service provides. This tool blindly copies those bytes.
package main

import (
	"flag"
	"io"
	"log"
	"net"

	"golang.org/x/net/proxy"
)

var (
	dial   string
	via    string
	listen string
)
var d interface {
	Dial(network, address string) (net.Conn, error)
}

func main() {
	flag.StringVar(&dial, "dial", "color-foo-bar-123:4567", "address in pcloud to dial")
	flag.StringVar(&via, "via", "8080", "socksproxy port")
	flag.StringVar(&listen, "listen", "localhost:8000", "local address to forward to")
	flag.Parse()
	var err error
	d, err = proxy.SOCKS5("tcp", "localhost:"+via, nil, &net.Dialer{})
	if err != nil {
		panic(err)
	}
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		panic(err)
	}
	log.Printf("Listening on %s\n", ln.Addr())
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	upstream, err := d.Dial("tcp", dial)
	if err != nil {
		panic(err)
	}
	defer upstream.Close()
	go io.Copy(upstream, conn)
	io.Copy(conn, upstream)
}
