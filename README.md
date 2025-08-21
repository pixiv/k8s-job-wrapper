# k8s-job-wrapper

Provides a CRD to make `Job` and `CronJob` easier to use through templating.

> [!WARNING]
> This repository does not actively accept external contributions.

## Prerequisites

- Kubernetes v1.32.2+ cluster

## Usage

[DOcumentation for the CRD](https://pixiv.github.io/k8s-job-wrapper/).

### Install

``` shell
helm install [RELEASE_NAME] oci://ghcr.io/pixiv/k8s-job-wrapper/charts/k8s-job-wrapper
```

To show the configurable values for the chart:

``` shell
helm show values oci://ghcr.io/pixiv/k8s-job-wrapper/charts/k8s-job-wrapper
```

## Contribution

### Prerequisites

- [direnv](https://github.com/direnv/direnv)
- [kind](https://github.com/kubernetes-sigs/kind)
- go version v1.25.0+
- docker version 28.3.2+

### Development

The repository layout conforms to the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).
Run `make help` to list all available development tasks.

#### Deploy

To deploy controller to your local development environment, run `make deploy-local`.

#### Tools

See the [hack](./hack) for development tools and scripts.
Development tool versions are managed in [.env](./.github/.env) and [go tool](./hack/tools).

To reinstall the development tools, run `make clean`.
Afterwards, they will be automatically reinstalled the next time you invoke them.
For more details, please see [tools.sh](./hack/tools.sh).

#### Documentation for the CRD

See the [docs](./hack/docs) for documentation for the CRD.
To generate the documentation, run `make docs`.

#### Helm chart

See the [chart](./hack/chart) for Helm chart for the CRD.
To create the Helm chart for your local development environment, run `make chart-local`.

#### How to upgrade Go

Edit following files:

- [.env](./.github/.env)
- [Dockerfile](./Dockerfile)
- [go.mod](./go.mod)
- [tools/go.mod](./hack/tools/go.mod)
- [go-licenses/go.mod](./hack/go-licenses/go.mod)

### Release

When you create and push a new tag, the following actions will be executed:

- Build and push the controller image to ghcr.
- Generate and push the Helm chart to ghcr.
- Create a release.
- Deploy the documentation for CRD to GitHub Pages.

Confirm that CI on the main branch is green before creating a tag.

## License

[Apache 2.0 License](./LICENSE).
