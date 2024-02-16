```sh
Usage of ./delete-webhook:
  -address string
    	bind to a specific ADDRESS:PORT, ADDRESS can be an IP or hostname (default ":8080")
  -dry-run
    	Enable dry run mode
  -insecure
    	Disable TLS verification for all the remote sites
```

## Configuration

### STEP 1: Run the webhook with remote sites configured

Set the following environment variables to configure remote sites

```sh
export REMOTE_ENDPOINT_<site-name>=<endpoint>
export REMOTE_ACCESS_<site-name>=<access-key>
export REMOTE_SECRET_<site-name>=<secret-key>
export REMOTE_INSECURE_<site-name>=true # optional; default: false.
```

#### Example :-

```sh
# configure site 1
export REMOTE_ENDPOINT_site1=https://127.0.0.1:9002
export REMOTE_ACCESS_site1=<access-key>
export REMOTE_SECRET_site1=<secret-key>
export REMOTE_INSECURE_site1=true
# configure site2
export REMOTE_ENDPOINT_site2=https://127.0.0.1:9004
export REMOTE_ACCESS_site2=<access-key>
export REMOTE_SECRET_site2=<secret-key>
export REMOTE_INSECURE_site2=true
# configure site3
# configure site4
# ...
# configure siten
./delete-webhook --address 127.0.0.1:8080
```

### STEP 2: Configure bucket notifications for monitoring DELETE events

reference: https://min.io/docs/minio/linux/administration/monitoring/publish-events-to-webhook.html#minio-bucket-notifications-publish-webhook

```sh
mc admin config set <alias> notify_webhook:<target-id> enable=on endpoint=<webhook-endpoint>
mc admin service restart <alias>
# The ARN will be printed in the logs and also will be seen in `mc admin info <alias> --json | jq .info.sqsARN`
mc event add <alias>/<bucket> <ARN> --event delete
```

#### Example :-

```sh
mc admin config set myminio notify_webhook:1 enable=on endpoint=http://127.0.0.1:8080/
mc admin service restart myminio
mc event add myminio/testb arn:minio:sqs::1:webhook --event delete
```
