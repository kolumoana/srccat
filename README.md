# srccat

`srccat` はディレクトリ内のソースコードファイルを表示するコマンドラインツールです。
`.gitignore` ルールを尊重し、不要なファイルを除外しながら、指定されたディレクトリ内のファイルの内容をリストアップしたり表示したりします。

## 特徴

- ディレクトリ内のソースコードファイルを再帰的に処理
- `.gitignore` ルールの尊重
- 大きなファイルやバイナリファイルの自動除外
- カスタム除外パターンのサポート
- 複数の出力フォーマット（テキスト、JSON、ファイル名リスト）
- 並行処理によるパフォーマンスの最適化

## インストール

```
go install github.com/kolumoana/srccat
```

## 使用方法

基本的な使用方法:

```
srccat --dir /path/to/your/directory
```

オプション:

- `--dir`, `-d`: 処理するディレクトリ（必須）
- `--format`, `-f`: 出力フォーマット（"text"、"json"、デフォルトは "text"）
- `--list`, `-l`: ファイル名のリストのみを出力
- `--exclude`, `-e`: カスタム除外パターン（複数指定可能）

## 例

1. ディレクトリ内のファイルを表示:

```
srccat --dir /path/to/your/project
```

2. JSON 形式で出力:

```
srccat --dir /path/to/your/project --format json
```

3. ファイル名のリストのみを表示:

```
srccat --dir /path/to/your/project --list
```

4. カスタム除外パターンを使用:

```
srccat --dir /path/to/your/project --exclude "*.css" --exclude "*.md"
```

## 除外ルール

`srccat` は以下のファイルとディレクトリを自動的に除外します:

- `.git`, `node_modules`, `build`, `dist`, `out`, `.cache`, `.tmp`, `.vscode`, `.idea`, `.next`, `public`, `.terraform` ディレクトリ
- `.json`, `.log`, `.bak`, `~`, `.env`, `.DS_Store`, `package-lock.json`, `yarn.lock`, `.d.ts`, `config.mjs`, `.lock.hcl`, `.ico`, `.tfstate`, `.backup`, `.pptx`, `.ppt`, `.doc`, `.docx`, `.xls`, `.xlsx` ファイル
- `.gitignore` ファイル
- 1MB を超える大きなファイル
- バイナリファイル

カスタム除外パターンを使用して、さらにファイルを除外することができます。
