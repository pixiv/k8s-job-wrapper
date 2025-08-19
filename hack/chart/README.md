# chart

You can generate the Helm chart for the CRD by running the following command: `./make.sh ${CHART_VERSION} ${CONTROLLER_IMAGE_REPO} ${CONTROLLER_IMAGE_TAG}`.

## How it works

This project uses the [Kubebuilder Helm Plugin](https://book.kubebuilder.io/plugins/available/helm-v1-alpha) to generate Helm charts.
The [`./make.sh`](./make.sh) script then replaces values such as the chart version and the controller image.

The `Chart.yaml` file is not modified directly. Instead, it is overwritten with a new file generated from its [template](./Chart.yaml.tmpl).

## Artifacts

Running the script as `./make.sh ${CHART_VERSION} ${CONTROLLER_IMAGE_REPO} ${CONTROLLER_IMAGE_TAG} ${DESTINATION}` will generate the file `${DESTINATION}/k8s-job-wrapper-${CHART_VERSION}.tgz`.

If you omit the `DESTINATION` argument, the file will be generated in the current directory.
