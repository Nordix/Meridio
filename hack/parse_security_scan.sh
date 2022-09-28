#! /bin/bash

rm -f ./_output/report.txt
rm -f ./_output/list.txt

for f in ./_output/*.json
do 
    if [[ $f == *"grype"* ]]
    then
        parsed=$(cat $f | jq -r '.matches[] | [.vulnerability.id, .vulnerability.severity, .artifact.name, .artifact.metadata.mainModule] | @tsv' | column -t)
        echo $f >> ./_output/report.txt
        echo "$parsed" >> ./_output/report.txt
        echo "" >> ./_output/report.txt
        echo "$parsed" >> ./_output/list.txt
    elif [[ $f == *"nancy"* ]]
    then
        parsed=$(cat $f | jq -r '.vulnerable[] | {Coordinates} + (try .Vulnerabilities[] | {ID}) | [.ID, .Coordinates] | @tsv' | column -t)
        echo $f >> ./_output/report.txt
        echo "$parsed" >> ./_output/report.txt
        echo "" >> ./_output/report.txt
        echo "$parsed" >> ./_output/list.txt
    elif [[ $f == *"trivy"* ]]
    then
        parsed=$(cat $f | jq -r '.Results[] | {Target} + (try .Vulnerabilities[] | {VulnerabilityID, PkgName, Severity}) | [.VulnerabilityID, .Severity, .PkgName, .Target] | @tsv' | column -t)
        echo $f >> ./_output/report.txt
        echo "$parsed" >> ./_output/report.txt
        echo "" >> ./_output/report.txt
        echo "$parsed" >> ./_output/list.txt
    fi
done
