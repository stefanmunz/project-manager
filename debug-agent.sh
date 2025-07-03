#!/bin/bash
# Debug agent that shows exactly what arguments it receives

echo "=== DEBUG AGENT ==="
echo "Number of arguments: $#"
echo ""

for i in $(seq 1 $#); do
    echo "Argument $i:"
    echo "---"
    eval echo "\$$i"
    echo "---"
    echo ""
done

echo "Raw command line: $0 $@"
echo ""

# Also log to file for inspection
{
    echo "[$(date +"%Y-%m-%d %H:%M:%S")] Debug agent called"
    echo "Arguments: $#"
    for i in $(seq 1 $#); do
        echo "Arg[$i]: $(eval echo "\$$i")"
    done
    echo "---"
} >> debug-agent.log

echo "Debug info logged to debug-agent.log"
exit 0