set -eux

sudo rm -f /tmp/tweet_fan_out_metrics

# Build ap_benchmark binary
make clean || true
make build

# Run test for this user
./main -instance https://one.localhost -instance-second https://two.localhost -load 100 -followers-fed 50 -parallel -show-graph
cat /tmp/tweet_fan_out_metrics
#cat /tmp/tweet_fan_out_metrics | awk '{print $15}' | sed 's/\ms$//' | gnuplot -p -e 'set xtics 1; set ylabel "duration (ms)"; set xlabel "Parallel request no"; plot "/dev/stdin" using 1 title "Duration" w linespoints pt 7'
