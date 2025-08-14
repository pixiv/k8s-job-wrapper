#!/bin/bash

#
# Entrypoint of dev tools
#
# 開発のためのツールの呼び出しとバージョン管理をまとめる

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly name="$1"
if [[ -z "$name" ]] ; then
    echo >&2 "$(basename "$0"): no tool name!"
    exit 1
fi
shift

# Install the tool
readonly toold="${d}/tools"
find_package() {
    grep "$1" "${toold}/go.mod" | grep -v 'indirect' | xargs
}

readonly bind="${d}/../bin"
mkdir -p "${bind}"
readonly binary="${bind}/${name}"
case "$name" in
    "go-licenses")
        # tools/ に追加すると go version が古いせいかビルドが通らないので分離している
        "${d}/setup-go-licenses.sh"
        ;;
    "pandoc")
        "${d}/setup-pandoc.sh" "$PANDOC_VERSION"
        ;;
    "setup-envtest")
        # setup-envtest は他のライブラリのバージョンに依存するので専用のセットアップが必要
        "${d}/setup-envtest.sh"
        # 特別に引数を渡して実行することはない
        exit
        ;;
    "kubectl")
        # kubectl は tool に取り込むと依存関係が増えすぎるのでバイナリを落としてくる
        "${d}/setup-kubectl.sh" "$KUBECTL_VERSION"
        ;;
    *)
        # 呼び出し元のディレクトリが重要なツールがあるので
        # go tool から直接呼び出すと意図しない結果になる場合がある
        if [[ ! -x "$binary" ]] ; then
            echo >&2 "$(basename "$0"): build ${binary}..."
            go -C "$toold" build -o "$binary" "$(find_package "$name")"
        fi
        ;;
esac
# Run the tool
"$binary" "$@"
