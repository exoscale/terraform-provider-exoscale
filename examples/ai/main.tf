# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone     = "at-vie-2"
  my_gpu_type = "gpu3080ti"
}

variable "huggingface_token" {
  type      = string
  sensitive = true
}

# Public model
resource "exoscale_ai_model" "my_public_model" {
  zone = local.my_zone
  name = "Qwen/Qwen3-0.6B"
}

# Gated model (requires accepting the model license on Hugging Face)
resource "exoscale_ai_model" "my_gated_model" {
  zone              = local.my_zone
  name              = "meta-llama/Llama-3.2-1B-Instruct"
  huggingface_token = var.huggingface_token
}

# Deployment of the public model (GPU billed while running)
resource "exoscale_ai_deployment" "my_deployment" {
  zone      = local.my_zone
  name      = "my-deployment"
  model_id  = exoscale_ai_model.my_public_model.id
  gpu_type  = local.my_gpu_type
  gpu_count = 1
  replicas  = 1

  inference_engine_parameters = [
    "--max-model-len=4096",
  ]
}

# Outputs
output "my_deployment" {
  value = {
    id             = exoscale_ai_deployment.my_deployment.id
    state          = exoscale_ai_deployment.my_deployment.state
    deployment_url = exoscale_ai_deployment.my_deployment.deployment_url
  }
}

output "my_public_model" {
  value = {
    id    = exoscale_ai_model.my_public_model.id
    state = exoscale_ai_model.my_public_model.state
  }
}

output "my_gated_model" {
  value = {
    id    = exoscale_ai_model.my_gated_model.id
    state = exoscale_ai_model.my_gated_model.state
  }
}
