resource "exoscale_ai_model" "my_ai_model" {
  zone = "at-vie-2"
  name = "Qwen/Qwen3-0.6B"
}

resource "exoscale_ai_deployment" "my_ai_deployment" {
  zone      = "at-vie-2"
  name      = "my-ai-deployment"
  model_id  = exoscale_ai_model.my_ai_model.id
  gpu_type  = "gpu3080ti"
  gpu_count = 1
  replicas  = 1
}

# API key granted access to a single deployment
resource "exoscale_ai_api_key" "my_ai_api_key" {
  zone = "at-vie-2"
  name = "my-ai-api-key"

  deployment_ids = [
    exoscale_ai_deployment.my_ai_deployment.id,
  ]
}

# API key granted access to all the public models
resource "exoscale_ai_api_key" "my_public_models_api_key" {
  zone       = "at-vie-2"
  name       = "my-public-models-api-key"
  all_models = true
}
