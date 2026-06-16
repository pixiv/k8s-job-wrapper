variable "IMAGE_TAG" {
  default = "dev"
}

variable "IMAGE_REGISTRY" {
  default = "my-registry.local:5000/k8s-job-wrapper"
}

variable "GO_VERSION" {
  default = "1.25"
}

variable "KUBECTL_VERSION" {
  default = "v1.32.8"
}

function "gentags" {
  params = []
  result = ["${IMAGE_NAME}:${IMAGE_TAG}"]
}

group "default" {
  targets = ["manager"]
}

target "manager" {
  context = "."
  dockerfile = "Dockerfile"
  tags = gentags()
  args = {
    GO_VERSION = GO_VERSION
    KUBECTL_VERSION = KUBECTL_VERSION
  }
}
