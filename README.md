# Podtail

Podtail is a Go port of Johan Haleby's [kubetail](https://github.com/johanhaleby/kubetail) utilty which now also allows Windows users to aggregate (tail/follow) logs from multiple pods into one stream.

This is the same as running "kubectl logs -f <pod>" but for multiple pods.

## Installation
Podtail currently wraps the `kubectl` command which you need to have installed and available on the path in order to use `podtail`.

Please refer to the Kubernetes documentation for information on how to install `kubectl` for your specific OS.

### OSX
You use Brew to install the client as follows:

    $ brew tap johnmccabe/podtail && brew install podtail

### Windows
Just download the [podtail.exe](TODO LINK) file and copy it to a location on your `%PATH%`.

### Linux
Just download and rename the `podtail` binary and you're good to go.

```
curl -O TODO link
```

## Usage

```
podtail --help
```