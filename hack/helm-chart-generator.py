"""
This is a script to generate a helm chart if helm is prefered over make.

How to use:
$ python3 ./hack/helm-chart-generator.py [--image <full-operator-image-path>] [--version <version>] [--mutating <boolean>]

The --image arguement should be specified without option, if not specified, it's "controller" by default.
The version arguement should fulfill Semantic Versioning 2.0.0. The default value is 0.0.1
A helm folder will then be created under Meridio-Operator repo root, which is ready to be used by helm.
"""
import argparse
import subprocess
import re
import shutil
import semver
from pathlib import Path

parser = argparse.ArgumentParser(description='Varialbes to build helm chart')
parser.add_argument('--image', type=str, nargs='?', default='controller',
                    help='operator image full path without version')
parser.add_argument('--version', type=str, nargs='?', default='0.0.1',
                    help='version of the helm chart, must follow semver 2')
parser.add_argument('--mutating', type=str, nargs='?', default='false',
                    help='enable mutating webhook or not')

args = parser.parse_args()
print("operator image: " + args.image)
# check if version is valid
semver.VersionInfo.parse(args.version)
print("operator version: " + args.version)
print("mutating webhook enabled: " + args.mutating)

# silent mode to supress the echo of make command
out = subprocess.run(["make", "-s", "print-manifests",
 'IMG=' + args.image + ":" + args.version,
 "ENABLE_MUTATING_WEBHOOK=" + args.mutating],
                     capture_output=True).stdout.decode("utf-8")
# split the output
contents = out.split('\n---\n')

# initialize the directories
helmdir = "helm"
templatedir = helmdir+"/templates"
if Path(helmdir).exists():
    # remove all the files
    shutil.rmtree(helmdir)
    if Path(templatedir).exists():
        shutil.rmtree(templatedir)
Path(templatedir).mkdir(parents=True, exist_ok=True)

# create files by "kind" and "name", and write contents to files
for content in contents:
    # compose file name, using:
    # first "kind" found in the manifest. For example: Namespace
    kind = re.findall('(?<=\nkind: )\S+', content)[0]
    # skip namespace file
    if kind == "Namespace":
        continue
    # first "name" found in the manifest. For example: meridio
    name = re.findall('(?<=\n  name: )\S+', content)[0]
    filename = kind + "-" + name + ".yaml"

    # fix the identation
    # content = subprocess.run(["yq", "-Y", "."], input=content, stdout=subprocess.PIPE, encoding='utf-8').stdout
    # replace namespace in content
    content = re.sub("meridio-operator-system",
                     '{{.Release.Namespace}}', content)
    # suffix cluster-scoped resources with namespace
    if kind == "MutatingWebhookConfiguration" or kind == "ValidatingWebhookConfiguration":
        content = re.sub(name, name + '-{{.Release.Namespace}}', content)
    # write contents to the files
    with open(templatedir+"/"+filename, 'w') as f:
        f.write("---\n" + content)

# copy Chart.yaml to helm chart
chartfile = "hack/Chart.yaml"
if Path(chartfile).exists():
    subprocess.run(["cp", chartfile, helmdir])
else:
    exit(1)

# tar helm chart
subprocess.run(["helm", "package", helmdir, "--version", args.version])
