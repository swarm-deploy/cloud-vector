# cloud-vector

`cloud-vector` - proxy for [Vector](https://github.com/vectordotdev/vector) for support cloud logging, focused on Docker Swarm.

`cloud-vector` supports send logs per log group through label `logging.group_id`

## Use with Cloud.ru Logging

1. Copy [docker-compose.yaml](docker-compose.yaml)
2. Fill project id and log group id
3. [Create Service Account in Cloud.ru IAM](https://cloud.ru/docs/console_api/ug/topics/guides__service_accounts_create?source-platform=Evolution)
4. Add secrets for Cloud.ru IAM `vector-cloudru-iam-client-id` and `vector-cloudru-iam-client-secret`
5. Deploy stack
