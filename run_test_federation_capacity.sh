#!/bin/bash

# Function to display usage information
usage() {
    echo "Usage: $0 --producer-instance <argA> --consumer-instance <argB> --load <argC>"
    echo "  --producer-instance    Host URL for instance which will produce /inbox request"
    echo "  --consumer-instance    Host URL for instance which will consume /inbox request"
    echo "  --load                 Number of requests to fire"
    echo "  --client               Number of clients on the host"
    exit 1
}

# Check if no arguments were provided
if [ $# -eq 0 ]; then
    usage
fi

client=1

# Parse arguments
while [[ $# -gt 0 ]]; do
    key="$1"

    case $key in
        --producer-instance)
            producerInstance="$2"
            shift # past argument
            shift # past value
            ;;
        --consumer-instance)
            consumerInstance="$2"
            shift # past argument
            shift # past value
            ;;
        --load)
            load="$2"
            shift # past argument
            shift # past value
            ;;
        --client)
            client="$2"
            shift # past argument
            shift # past value
            ;;
        --help)
            usage
            ;;
        *)
            echo "Invalid option: $1" >&2
            usage
            ;;
    esac
done

# Check if all required arguments are provided
if [ -z "$producerInstance" ] || [ -z "$consumerInstance" ] || [ -z "$load" ]; then
    echo "All arguments are required."
    usage
fi

for ((i=1; i<=$client; i++))
do
  ./main -instance $producerInstance -instance-second $consumerInstance -load $load > /dev/null 2>&1 &
done
./main -instance $consumerInstance -load $load &
wait

ps -aux | grep gotosocial | awk '{print $2}'| while read line; do sudo kill -9 $line; done;
sudo rm -f /tmp/tweet_fan_out_metrics
sudo rm /home/ar/work/gotosocial/sqlite_*

