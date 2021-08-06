# yq

`yq` lets you read YAML files easily on the terminal. You can find key/values easily.

## Motivation

Reading yaml configurations for k8s file becomes ardent through the terminal. `yq` helps reading/searching through the YAML easy. It uses [tview](https://github.com/rivo/tview) and is inspired by [tson](https://github.com/skanehira/tson).

[![asciicast](https://asciinema.org/a/KHvEQmiSnBNWOiGsPKE4v9Om3.png)](https://asciinema.org/a/KHvEQmiSnBNWOiGsPKE4v9Om3)

## Installation


### Grab the latest binary

```shell
$ cd "$(mktemp -d)"
$ curl -sL "https://github.com/arriqaaq/yq/releases/download/v0.1.0/yq_0.1.0_$(uname)_amd64.tar.gz" | tar xz
$ mv yq /usr/local/bin
# yq should be available now in your $PATH
```

## Usage

```shell
NAME:
   yq 

USAGE:
# from file
$ yq < test.json

# from kubectl
$ kubectl get pod kube-dns -n kube-system -oyaml | yq
```

## Bindings
### YAMLtree 

| key    | description                    |
|--------|--------------------------------|
| s      | hide/show current node              |
| S      | collaspe/expand all value nodes           |
| / or f | search nodes                   |
| q | quit                   |
