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

	clientv3 "go.etcd.io/etcd/client/v3"
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

var (
	codec = scheme.Codecs.UniversalDeserializer()

	etcdEndpoint = flag.String("etcd-endpoint", "localhost:2379", "etcd endpoint")
	etcdKey      = flag.String("etcd-key", "", "etcd key of the object")

	// New flags for mTLS
	caFile   = flag.String("cafile", "", "path to the CA certificate file")
	certFile = flag.String("certfile", "", "path to the client certificate file")
	keyFile  = flag.String("keyfile", "", "path to the client key file")
	write    = flag.Bool("write", false, "write to etcd")
)

func main() {
	corev1.AddToScheme(scheme.Scheme)

	flag.Parse()

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
		Endpoints: []string{*etcdEndpoint},
		TLS:       tlsConfig,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	// Write the stdin to the key as value
	if *write {
		if err := readYAMLAndWriteToEtcd(cli, *etcdKey); err != nil {
			log.Fatal(err)
		}
		// Read the value instead
	} else {
		obj, err := fetchObject(cli, *etcdKey)
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
