# Public model
resource "exoscale_ai_model" "my_ai_model" {
  zone = "at-vie-2"
  name = "Qwen/Qwen3-0.6B"
}

# Gated model, downloaded with a Hugging Face token
variable "huggingface_token" {
  type      = string
  sensitive = true
}

resource "exoscale_ai_model" "my_gated_ai_model" {
  zone              = "at-vie-2"
  name              = "meta-llama/Llama-3.2-1B-Instruct"
  huggingface_token = var.huggingface_token
}
