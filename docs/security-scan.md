# Security Scan

[Grype](https://github.com/anchore/grype), [Nancy](https://github.com/sonatype-nexus-community/nancy) and [Trivy](https://github.com/aquasecurity/trivy) are available in the makefile to scan the Meridio images and dependencies and report the vulnerabilities. The make commands are the following:
- `make grype`
- `make nancy`
- `make trivy`

Once scanned, the reports are available in json format under the `_output` directory. To convert them in a more readable format, the `./hack/parse_security_scan.sh` script can be executed, the new report will be available in `_output/report.txt`.

here are some other helpful commands:
```
# Create table from Grype json report:
cat _output/grype_proxy_latest.json | jq -r '.matches[] | [.vulnerability.id, .vulnerability.severity, .artifact.name, .artifact.metadata.mainModule] | @tsv' | column -t

# Create table from Nancy json report:
cat _output/nancy.json | jq -r '.vulnerable[] | {Coordinates} + (try .Vulnerabilities[] | {ID}) | [.ID, .Coordinates] | @tsv' | column -t

# Create table from Trivy json report:
cat _output/trivy_proxy_latest.json | jq -r '.Results[] | {Target} + (try .Vulnerabilities[] | {VulnerabilityID, PkgName, Severity}) | [.VulnerabilityID, .Severity, .PkgName, .Target] | @tsv' | column -t

# Count number of CVEs:
cat _output/list.txt | grep -v "^$" | awk '{print $1}' | sort | uniq | wc -l

# List of CVEs:
cat _output/list.txt | grep -v "^$" | awk '{print $1}' | sort | uniq | sed ':a;N;$!ba;s/\n/ ; /g'

# Count number of CVEs with High severity:
cat _output/list.txt | grep -v "^$" | grep -i "high" | awk '{print $1}' | sort | uniq | wc -l

# List High severity CVEs:
cat _output/list.txt | grep -v "^$" | grep -i "high" | awk '{print $1}' | sort | uniq | sed ':a;N;$!ba;s/\n/ ; /g'
```
