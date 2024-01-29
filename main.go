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
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/pkg/env"
)

var (
	address                                          string
	authToken                                        = env.Get("WEBHOOK_AUTH_TOKEN", "")
	remoteEndpoint, remoteAccessKey, remoteSecretKey string
	insecure                                         bool
	dryRun                                           bool
)

func main() {
	flag.StringVar(&address, "address", ":8080", "bind to a specific ADDRESS:PORT, ADDRESS can be an IP or hostname")
	flag.StringVar(&remoteEndpoint, "remote-endpoint", "", "S3 endpoint URL of the remote target")
	flag.StringVar(&remoteAccessKey, "remote-access-key", "", "S3 Access Key of the remote target")
	flag.StringVar(&remoteSecretKey, "remote-secret-key", "", "S3 Secret Key of the remote target")
	flag.BoolVar(&insecure, "insecure", false, "Disable TLS verification")
	flag.BoolVar(&dryRun, "dry-run", false, "Enable dry run mode")

	flag.Parse()

	if remoteEndpoint == "" {
		log.Fatalln("remote endpoint is not provided")
	}

	if remoteAccessKey == "" {
		log.Fatalln("remote access key is not provided")
	}

	if remoteSecretKey == "" {
		log.Fatalln("remote secret key is not provided")
	}

	s3Client := getS3Client(remoteEndpoint, remoteAccessKey, remoteSecretKey, insecure)

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
			var apiData map[string]interface{}
			apiData, ok := jsonData["api"].(map[string]interface{})
			if !ok {
				http.Error(w, "missing api in the request body", http.StatusBadRequest)
				return
			}
			if apiData["name"].(string) != "DeleteObject" || apiData["statusCode"].(float64) != 204 {
				return
			}
			var responseHeaders map[string]interface{}
			responseHeaders, ok = jsonData["responseHeader"].(map[string]interface{})
			if !ok {
				return
			}
			if _, ok := responseHeaders["x-amz-delete-marker"].(string); ok {
				return
			}
			if _, ok := responseHeaders["x-amz-version-id"].(string); ok {
				return
			}
			bucket := apiData["bucket"].(string)
			object := apiData["object"].(string)
			if bucket == "" || object == "" {
				return
			}
			var versionID string
			if reqQ, ok := jsonData["requestQuery"].(map[string]interface{}); ok {
				versionID = reqQ["versionId"].(string)
			}
			if !dryRun {
				if err := s3Client.RemoveObject(context.Background(), bucket, object, minio.RemoveObjectOptions{VersionID: versionID}); err != nil {
					log.Printf("unable to delete the object: %v; %v\n", object, err)
					return
				}
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

func getS3Client(endpoint string, accessKey string, secretKey string, insecure bool) *minio.Client {
	u, err := url.Parse(endpoint)
	if err != nil {
		log.Fatalln(err)
	}

	secure := strings.EqualFold(u.Scheme, "https")
	transport, err := minio.DefaultTransport(secure)
	if err != nil {
		log.Fatalln(err)
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
		log.Fatalln(err)
	}
	return s3Client
}
