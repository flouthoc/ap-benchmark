# gnuplot -p plot_script.gp
set title "Connected Clients vs Throughput"
set xlabel "Number of Clients"
set ylabel "Throughput (requests/second)"
#set xtics (0, 2, 4, 8, 16, 32, 64, 128, 256, 512)
set logscale x 2
plot "no-relay2" using 2:1 with linespoints title "Vanilla Setup ( Without Relay )", \
     "relay2" using 2:1 with linespoints title "Setup Using Relay "

