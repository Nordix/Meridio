#! /bin/bash

collect_namespaced_resource() {
    mkdir -p "$full_output_path/$1/describe"
    mkdir -p "$full_output_path/$1/yaml"
    kubectl describe $1 $3 -n $2 > "$full_output_path/$1/describe/$2.$3.txt" 2>/dev/null
    kubectl get $1 $3 -n $2 -o yaml > "$full_output_path/$1/yaml/$2.$3.yaml" 2>/dev/null
}

collect_not_namespaced_resource() {
    mkdir -p "$full_output_path/$1/describe"
    mkdir -p "$full_output_path/$1/yaml"
    kubectl describe $1 $2 > "$full_output_path/$1/describe/$2.txt" 2>/dev/null
    kubectl get $1 $2 -o yaml > "$full_output_path/$1/yaml/$2.yaml" 2>/dev/null
}

collect_resource() {
    resources=$(kubectl get $1 -o wide --all-namespaces 2>/dev/null)
    echo "$resources" > "$full_output_path/$1/all.txt"
    resources_no_header=$(echo "$resources" | sed '1d')
    while IFS= read -r resource; do
        if [ -z "$resource" ]; then
            continue
        fi
        if [[ "$2" == "true" ]]; then
            name=$(echo "$resource" | awk '{print $2}')
            namespace=$(echo "$resource" | awk '{print $1}')
            collect_namespaced_resource $1 $namespace $name
        else
            name=$(echo "$resource" | awk '{print $1}')
            collect_not_namespaced_resource $1 $name
        fi
    done <<< "$resources_no_header"
}

collect_top() {
    echo "collecting top ..."
    kubectl top pods --all-namespaces >> "$full_output_path/top.txt" 2>/dev/null
}

collect_logs() {
    pods=$(kubectl get pods --all-namespaces --no-headers=true)
    while IFS= read -r pod; do
        name=$(echo "$pod" | awk '{print $2}')
        namespace=$(echo "$pod" | awk '{print $1}')
        containers=$(kubectl get pods $name -n $namespace -o jsonpath="{.spec.containers[*].name}")
        init_containers=$(kubectl get pods $name -n $namespace -o jsonpath="{.spec.initContainers[*].name}")
        echo "collecting logs of $name.$namespace ..."
        mkdir -p "$full_output_path/pods/logs"
        mkdir -p "$full_output_path/pods/logs/previous"
        for container in $containers; do
            kubectl logs $name -n $namespace -c $container > "$full_output_path/pods/logs/$namespace.$name.$container.log"
            kubectl logs $name -n $namespace -c $container --previous=true > "$full_output_path/pods/logs/previous/$namespace.$name.$container.log" 2>/dev/null
        done
        for container in $init_containers; do
            kubectl logs $name -n $namespace -c $container > "$full_output_path/pods/logs/$namespace.$name.$container.log"
            kubectl logs $name -n $namespace -c $container --previous=true > "$full_output_path/pods/logs/previous/$namespace.$name.$container.log" 2>/dev/null
        done
    done <<< "$pods"
}

collect_all() {
    resources=$(kubectl api-resources --verbs=get)
    kubectl api-resources -o wide > "$full_output_path/api-resources.txt"
    resources_no_header=$(echo "$resources" | sed '1d')
    while IFS= read -r resource; do
        namespaced=$(echo "$resource" | awk '{print $(NF-1)}')
        resource_name=$(echo "$resource" | awk '{print $1}')
        mkdir -p "$full_output_path/$resource_name"
        echo "collecting $resource_name ..."
        collect_resource $resource_name $namespaced
    done <<< "$resources_no_header"
    collect_top
    collect_logs
}

timestamp=$(date +%s)

OUTPUT_ID=${OUTPUT_ID:-$timestamp}
OUTPUT_PATH=${OUTPUT_PATH:-"_output"}
collector_output_path=$OUTPUT_PATH"/log_collector"
full_output_path=$collector_output_path"/"$OUTPUT_ID

echo $OUTPUT_ID
echo $OUTPUT_PATH
echo $collector_output_path
echo $full_output_path

rm -rf $full_output_path
mkdir -p $OUTPUT_PATH
mkdir -p $collector_output_path
mkdir -p $full_output_path

collect_all

rm -rf $OUTPUT_PATH/log_collector_$OUTPUT_ID.tgz
tar -cvzf $OUTPUT_PATH/log_collector_$OUTPUT_ID.tgz $full_output_path > /dev/null 2>&1
rm -rf $full_output_path
