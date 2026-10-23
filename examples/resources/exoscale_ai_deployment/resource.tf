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

  inference_engine_parameters = [
    "--max-model-len=4096",
  ]
}
