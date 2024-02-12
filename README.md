```sh
Usage of ./delete-webhook:
  -address string
    	bind to a specific ADDRESS:PORT, ADDRESS can be an IP or hostname (default ":8080")
  -dry-run
    	Enable dry run mode
  -insecure
    	Disable TLS verification for all the remote sites
```

Example :-

```sh
export REMOTE_ENDPOINT_site1=http://127.0.0.1:9002
export REMOTE_ACCESS_site1=<access-key>
export REMOTE_SECRET_site1=<secret-key>
export REMOTE_INSECURE_site1=true # optional; default: false.
./delete-webhook --address 127.0.0.1:8080
```
