# Usage

Install yq
```
wget https://github.com/mikefarah/yq/releases/download/v4.21.1/yq_linux_amd64 -O ~/.local/bin/yq
```

pip install 
```
pip3 install requirements.txt
```

Generate helm chart 
```
python3 ./hack/helm-chart-generator.py [--image <full-operator-image-path>] [--version <version>] [--mutating <boolean>]
```