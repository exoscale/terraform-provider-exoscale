# AI Dedicated Inference

This example demonstrates how to download
[AI Models](https://community.exoscale.com/product/concrete-ai/dedicated-inference/overview/)
from Hugging Face, serve one of them on GPUs and grant an API key access to it, using the
`exoscale_ai_model`, `exoscale_ai_deployment` and `exoscale_ai_api_key` resources.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

The deployment runs on GPUs and is billed while running: the available GPU types are listed by
`exo dedicated-inference deployment instance-type`, and your organization GPU limits (`exo limits`)
must allow the deployment.

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET \
  -var huggingface_token=$HUGGINGFACE_TOKEN

...

Outputs:

my_api_key_secret = <sensitive>
my_deployment = {
  "deployment_url" = "https://5e1f2a3b-4c5d-4e6f-8a9b-0c1d2e3f4a5b.inference.at-vie-2.exoscale-cloud.com/v1"
  "id" = "5e1f2a3b-4c5d-4e6f-8a9b-0c1d2e3f4a5b"
  "state" = "deploying"
}
my_gated_model = {
  "id" = "2d6b6f5e-8c1a-4a7e-9f3b-1c2d3e4f5a6b"
  "state" = "downloading"
}
my_public_model = {
  "id" = "7a1c9e2b-3d4f-4b5a-8c6d-9e0f1a2b3c4d"
  "state" = "downloading"
}
```

The API key secret is only returned at creation, use it to query the deployment:

```console
curl "$(terraform output -json my_deployment | jq -r .deployment_url)/chat/completions" \
  -H "Authorization: Bearer $(terraform output -raw my_api_key_secret)" \
  -H "Content-Type: application/json" \
  -d '{"model": "Qwen/Qwen3-0.6B", "messages": [{"role": "user", "content": "Hello world!"}]}'
```
