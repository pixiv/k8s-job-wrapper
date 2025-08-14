# docs

CRD のドキュメントを `./make.sh ${DEST_DIR} ${K8S_VERSION}` で生成できます

## 仕組み

[crd-ref-docs](https://github.com/elastic/crd-ref-docs) を使って [CRDのスキーマ](../../api/v1) から [markdown](docs.md) を生成し [pandoc](https://github.com/jgm/pandoc) を使って html に変換します.

スタイルは公開されている CSS をベースに [カスタマイズ](docs.css) しています

## ローカルで確認

`$DEST_DIR` に全部入りの `index.html` が作られるので、それをブラウザで開きます.

## マニフェストの例の更新

[examples/results](examples/results) は `config/samples` の内容からコントローラーが生成するリソースです.
これは手動で書き換える必要があります.

``` shell
# PWD=プロジェクトのルート
./kind.sh
make sample
```

こうしてから `kubectl get job 作られたJOBの名前 -o yaml` などとして取得したものを [examples/results](examples/results) にコミットします

Examples の部分の記事は [examples/generate.sh](examples/generate.sh) で作っています
