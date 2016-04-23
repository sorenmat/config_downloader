package main

import (
	"os"
	"fmt"
	"log"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"net/http"
	"github.com/alecthomas/kingpin"
	"crypto/tls"
	"crypto/x509"
	"strconv"
)

func createConfig(host string, caFile string, certFile string, keyFile string) *api.Config {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Unable to load cert '%v' or key '%v' with error '%v'\n", certFile, keyFile, err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatalf("Unable to load cacert '%v' with error '%v'\n", caFile, err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()

	config := api.DefaultConfig()
	config.HttpClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	config.Address = host
	config.Scheme = "https"
	return config
}

func main() {
	caFile := kingpin.Flag("caFile", "the file holding the CA certificate.").Required().String()
	certFile := kingpin.Flag("certFile", "the file holding the client certificate.").Required().String()
	keyFile := kingpin.Flag("keyFile", "the file holding the client key.").Required().String()

	host := kingpin.Arg("host", "consul host to talk to without the protocol (https)").Required().String()
	key := kingpin.Arg("key", "Name of the key that contains the config").Required().String()
	baseDir := kingpin.Arg("baseDir", "Directory where config files should be written").Required().String()
	perm := kingpin.Arg("permissions", "File permissions").Default("0744").String()

	kingpin.Version("0.0.1")
	kingpin.Parse()

	// Get a new client
	client, err := api.NewClient(createConfig(*host, *caFile, *certFile, *keyFile))
	if err != nil {
		panic(err)
	}

	// Get a handle to the KV API
	kv := client.KV()
	if err != nil {
		panic(err)
	}
	fmt.Println("Trying to fetch: " + *key)
	pair, _, err := kv.Get(*key, nil)

	configPath := *baseDir + "/" + pair.Key
	filePermission, err := strconv.ParseUint(*perm, 8, 32)
	err = os.MkdirAll(configPath, os.FileMode(uint32(filePermission)))
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(configPath + "/config.properties", pair.Value, os.FileMode(uint32(filePermission)))
	if err != nil {
		log.Fatal(err)
	}
}
