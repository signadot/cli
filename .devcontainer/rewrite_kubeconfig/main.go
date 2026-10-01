package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	in, out, localhostHome string
)

func main() {
	flag.StringVar(&in, "in", os.ExpandEnv("$HOME/.kube-localhost/config"), "input kube config file")
	flag.StringVar(&out, "out", os.ExpandEnv("$HOME/.kube/config"), "output kube config file")
	log.Printf("running with in=%q out=%q and $LOCALHOST_HOME=%q", in, out, os.Getenv("LOCALHOST_HOME"))
	flag.Parse()

	cfg, err := clientcmd.LoadFromFile(in)
	if err != nil {
		log.Fatal(err)
	}
	d, _ := filepath.Split(out)
	if err := os.MkdirAll(d, 0755); err != nil {
		log.Fatal(err)
	}
	for _, cluster := range cfg.Clusters {
		if cluster.ProxyURL != "" {
			cluster.ProxyURL = rewriteHost(cluster.ProxyURL)
			continue
		}
		cluster.Server = rewriteHost(cluster.Server)
		cluster.CertificateAuthority = rewriteHome(cluster.CertificateAuthority)
	}
	for _, user := range cfg.AuthInfos {
		user.ClientCertificate = rewriteHome(user.ClientCertificate)
		user.ClientKey = rewriteHome(user.ClientKey)
	}
	if err := clientcmd.WriteToFile(*cfg, out); err != nil {
		log.Fatal(err)
	}
}

func rewriteHost(s string) string {
	s = strings.ReplaceAll(s, "127.0.0.1", "host.docker.internal")
	s = strings.ReplaceAll(s, "localhost", "host.docker.internal")
	return s
}

func rewriteHome(s string) string {
	s = strings.ReplaceAll(s, os.ExpandEnv("$LOCALHOST_HOME"), os.ExpandEnv("$HOME"))
	s = strings.ReplaceAll(s, ".minikube", ".minikube-localhost")
	return s
}
