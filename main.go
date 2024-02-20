package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/pkg/env"
)

var (
	address   string
	authToken = env.Get("WEBHOOK_AUTH_TOKEN", "")
	insecure  bool
	dryRun    bool
)

type Remote struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Insecure  bool
}

func main() {
	flag.StringVar(&address, "address", ":8080", "bind to a specific ADDRESS:PORT, ADDRESS can be an IP or hostname")
	flag.BoolVar(&insecure, "insecure", false, "Disable TLS verification for all the remote sites")
	flag.BoolVar(&dryRun, "dry-run", false, "Enable dry run mode")
	flag.Parse()

	envs := env.List("REMOTE_ENDPOINT_")
	remoteTargets := make(map[string]Remote, len(envs))
	for _, k := range envs {
		targetName := strings.TrimPrefix(k, "REMOTE_ENDPOINT_")
		r := Remote{
			Endpoint:  env.Get("REMOTE_ENDPOINT_"+targetName, ""),
			AccessKey: env.Get("REMOTE_ACCESS_"+targetName, ""),
			SecretKey: env.Get("REMOTE_SECRET_"+targetName, ""),
			Insecure:  env.Get("REMOTE_INSECURE_"+targetName, strconv.FormatBool(insecure)) == "true",
		}
		if r.AccessKey == "" {
			log.Fatalf("REMOTE_ACCESS_%s not set", targetName)
		}
		if r.SecretKey == "" {
			log.Fatalf("REMOTE_SECRET_%s not set", targetName)
		}
		remoteTargets[targetName] = r
	}
	if len(remoteTargets) == 0 {
		log.Fatal("no remote sites provided")
	}

	var remoteClients []*minio.Client
	for name, target := range remoteTargets {
		remoteClient, err := getS3Client(target.Endpoint, target.AccessKey, target.SecretKey, target.Insecure)
		if err != nil {
			log.Fatalf("unable to create s3 client for %v; %v", name, err)
		}
		fmt.Printf("Configured remote site; name: %v, host: %v\n", name, remoteClient.EndpointURL().Host)
		remoteClients = append(remoteClients, remoteClient)
	}

	fmt.Println("Started listening...\n")

	err := http.ListenAndServe(address, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authToken != "" {
			if authToken != r.Header.Get("Authorization") {
				http.Error(w, "authorization header missing", http.StatusBadRequest)
				return
			}
		}
		switch r.Method {
		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("unable to read the body; %v\n", err)
				http.Error(w, "error reading response body", http.StatusBadRequest)
				return
			}
			var jsonData map[string]interface{}
			if err = json.Unmarshal(body, &jsonData); err != nil {
				log.Printf("unable to unmarshal the body; %v\n", err)
				http.Error(w, "error marshalling response body", http.StatusBadRequest)
				return
			}
			if eventName, ok := jsonData["EventName"].(string); !ok || eventName != "s3:ObjectRemoved:NoOP" {
				return
			}
			records, ok := jsonData["Records"].([]interface{})
			if !ok || len(records) == 0 {
				log.Println("missing records in the request body")
				http.Error(w, "missing records in the request body", http.StatusBadRequest)
				return
			}
			record := records[0].(map[string]interface{})
			s3Data, ok := record["s3"].(map[string]interface{})
			if !ok {
				log.Println("missing records in the request body")
				http.Error(w, "missing s3 data in the request body", http.StatusBadRequest)
				return
			}
			var bucket string
			if bucketData, ok := s3Data["bucket"].(map[string]interface{}); ok {
				bucket, _ = bucketData["name"].(string)
			}
			var object, versionID string
			if objectData, ok := s3Data["object"].(map[string]interface{}); ok {
				object, _ = objectData["key"].(string)
				versionID, _ = objectData["versionId"].(string)
			}
			if bucket == "" || object == "" {
				return
			}
			var failed bool
			if !dryRun {
				for _, remoteClient := range remoteClients {
					if err := remoteClient.RemoveObject(context.Background(), bucket, object, minio.RemoveObjectOptions{VersionID: versionID, ForceDelete: true}); err != nil {
						failed = true
						log.Printf("unable to delete the object: %v from site %v; %v\n", object, remoteClient.EndpointURL().Host, err)
					}
				}
			}
			if failed {
				// Purposefully not returning an error here, as there could be scenarios
				// where the buckets won't be available in different sites, we can just
				// ignore the errors and pass by
				return
			}
			if versionID == "" {
				fmt.Printf("Deleted %v/%v\n", bucket, object)
			} else {
				fmt.Printf("Deleted %v/%v; version: %v\n", bucket, object, versionID)
			}
		default:
		}
	}))
	if err != nil {
		log.Fatal(err)
	}
}

func getS3Client(endpoint string, accessKey string, secretKey string, insecure bool) (*minio.Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	secure := strings.EqualFold(u.Scheme, "https")
	transport, err := minio.DefaultTransport(secure)
	if err != nil {
		return nil, err
	}
	if transport.TLSClientConfig != nil {
		transport.TLSClientConfig.InsecureSkipVerify = insecure
	}
	s3Client, err := minio.New(u.Host, &minio.Options{
		Creds:     credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:    secure,
		Transport: transport,
	})
	if err != nil {
		return nil, err
	}
	return s3Client, nil
}
