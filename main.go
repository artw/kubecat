package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

var (
	codec = scheme.Codecs.UniversalDeserializer()

	etcdEndpoints = flag.String("etcd-endpoint", getEnv("ETCDCTL_ENDPOINTS", "localhost:2379"), "etcd endpoint")

	etcdKey string

	// mTLS
	caFile   = flag.String("cafile", getEnv("ETCDCTL_CACERT", ""), "path to the CA certificate file")
	certFile = flag.String("certfile", getEnv("ETCDCTL_CERT", ""), "path to the client certificate file")
	keyFile  = flag.String("keyfile", getEnv("ETCDCTL_KEY", ""), "path to the client key file")
	write    = flag.Bool("write", false, "write to etcd")
	help     = flag.Bool("help", false, "display help")
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	corev1.AddToScheme(scheme.Scheme)

	// Parsing etcdKey as the default argument
	flag.Parse()
	if flag.NArg() > 0 {
		etcdKey = flag.Arg(0)
	}

	if etcdKey == "" || *help {
		fmt.Println("kubecat usage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(*caFile)
	if err != nil {
		log.Fatal(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}

	// Connect to etcd
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: strings.Split(*etcdEndpoints, ","), // Comma-separated endpoints
		TLS:       tlsConfig,
	})
	if err != nil {
		log.Fatal(err)
	}

	defer cli.Close()

	// Crash on 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = cli.Sync(ctx)
	if err != nil {
		log.Fatalf("failed to connect to etcd: %v", err)
	}

	// Write the stdin to the key as value
	if *write {
		if err := readYAMLAndWriteToEtcd(cli, etcdKey); err != nil {
			log.Fatal(err)
		}
		// Read the value instead
	} else {
		obj, err := fetchObject(cli, etcdKey)
		if err != nil {
			log.Fatal(err)
		}

		yamlBytes, err := yaml.Marshal(obj)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(yamlBytes))
	}
}

func fetchObject(cli *clientv3.Client, key string) (runtime.Object, error) {
	resp, err := cli.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	obj, _, err := codec.Decode(resp.Kvs[0].Value, nil, nil)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func readYAMLAndWriteToEtcd(cli *clientv3.Client, key string) error {
	// Read YAML from stdin.
	yamlBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	// Deserialize the YAML into a runtime.Object.
	obj, _, err := codec.Decode(yamlBytes, nil, nil)
	if err != nil {
		return err
	}

	// Serialize the object into protobuf format.
	protobufBytes, err := runtime.Encode(scheme.Codecs.LegacyCodec(scheme.Scheme.PrioritizedVersionsAllGroups()...), obj)
	if err != nil {
		return err
	}

	// Write the protobuf data to etcd.
	_, err = cli.Put(context.Background(), key, string(protobufBytes))
	return err
}
