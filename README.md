```sh
Usage of ./delete-webhook:
  -address string
    	bind to a specific ADDRESS:PORT, ADDRESS can be an IP or hostname (default ":8080")
  -dry-run
    	Enable dry run mode
  -insecure
    	Disable TLS verification
  -remote-access-key string
    	S3 Access Key of the remote target
  -remote-endpoint string
    	S3 endpoint URL of the remote target
  -remote-secret-key string
    	S3 Secret Key of the remote target
```

Example :-

```sh
./delete-webhook --address 127.0.0.1:8080 --remote-endpoint https://play.minio.io:9000 --remote-access-key <access> --remote-secret-key <secret> --dry-run
```
