# chart

CRD の Helm チャートを `./make.sh ${CHART_VERSION} ${CONTROLLER_IMAGE_REPO} ${CONTROLLER_IMAGE_TAG}` で生成できます

## 仕組み

[kubebuilder の Helm Plugin](https://book.kubebuilder.io/plugins/available/helm-v1-alpha) を使って Helm チャートを生成し [make.sh](./make.sh) によってチャートのバージョンや controller のイメージなどを置換します

Chart.yaml については置換ではなく [テンプレート](./Chart.yaml.tmpl) から生成したもので上書きします
## 成果物

`./make.sh ${CHART_VERSION} ${CONTROLLER_IMAGE_REPO} ${CONTROLLER_IMAGE_TAG} ${DESTINATION}` と実行した場合は `${DESTINATION}/k8s-job-wrapper-${CHART_VERSION}.tgz` が生成されます

DESTINATION を省略した場合はカレントディレクトリに生成されます
