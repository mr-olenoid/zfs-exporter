package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type config struct {
	ioScrapeInterval  int
	arcScrapeInterval int
}

func execute(cmd string, args ...string) string {

	out, err := exec.Command(cmd, args...).Output()

	if err != nil {
		fmt.Printf("%s", err)
	}

	//fmt.Println("Command Successfully Executed")
	output := string(out[:])
	//print(output)
	return output
}

func recordPoolsIO(interval int) {
	for {
		ioStatOut := execute("zpool", "iostat", "-Hpy", strconv.Itoa(interval), "1")
		outLines := strings.Split(ioStatOut, "\n")
		for _, line := range outLines[:len(outLines)-1] {
			stats := strings.Fields(line)
			data, _ := strconv.Atoi(stats[1])
			zpoolAlloc.WithLabelValues(stats[0]).Set(float64(data))
			data, _ = strconv.Atoi(stats[2])
			zpoolFree.WithLabelValues(stats[0]).Set(float64(data))
			data, _ = strconv.Atoi(stats[3])
			zpoolReadIO.WithLabelValues(stats[0]).Set(float64(data))
			data, _ = strconv.Atoi(stats[4])
			zpoolWriteIO.WithLabelValues(stats[0]).Set(float64(data))
			data, _ = strconv.Atoi(stats[5])
			zpoolReadBytes.WithLabelValues(stats[0]).Set(float64(data))
			data, _ = strconv.Atoi(stats[6])
			zpoolWriteBytes.WithLabelValues(stats[0]).Set(float64(data))
		}
	}
}

func readARCStats() {
	for {
		file, err := ioutil.ReadFile("/proc/spl/kstat/zfs/arcstats")
		if err != nil {
			fmt.Println(err)
		}
		arcData := string(file)
		arcDataStrings := strings.Split(arcData, "\n")
		parsedData := map[string]int{}
		for _, line := range arcDataStrings[:len(arcDataStrings)-1] {
			data := strings.Fields(line)
			parsedData[data[0]], _ = strconv.Atoi(data[2])
		}
		//print(arcDataStrings[0])
		//print(parsedData["size"])
		arcSize.Set(float64(parsedData["size"]))
		arcSizeMax.Set(float64(parsedData["c"]))
		arcMRUSize.Set(float64(parsedData["mru_size"]))
		arcMFUSize.Set(float64(parsedData["mfu_size"]))
		arcMemoryThrottle.Set(float64(parsedData["arc_memory_throttle_count"]))
		time.Sleep(15 * time.Second)
	}
}

var (
	zpoolAlloc = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "pool_allocated",
		Help:      "Allocated space for specific pool"},
		[]string{
			"pool_name",
		},
	)

	zpoolFree = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "pool_free",
		Help:      "Free space for specific pool"},
		[]string{
			"pool_name",
		},
	)

	zpoolReadIO = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "pool_read_io",
		Help:      "Read IO for specific pool"},
		[]string{
			"pool_name",
		},
	)

	zpoolWriteIO = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "pool_write_io",
		Help:      "Write IO for specific pool"},
		[]string{
			"pool_name",
		},
	)

	zpoolReadBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "pool_read_bytes",
		Help:      "Read bytes for specific pool"},
		[]string{
			"pool_name",
		},
	)

	zpoolWriteBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "pool_write_bytes",
		Help:      "Write bytes for specific pool"},
		[]string{
			"pool_name",
		},
	)

	arcSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "arc_size",
		Help:      "ARC size in bytes"},
	)

	arcSizeMax = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "arc_size_max",
		Help:      "ARC max size in bytes"},
	)

	arcMRUSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "arc_mru_size",
		Help:      "ARC mru size in bytes"},
	)

	arcMFUSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "arc_mfu_size",
		Help:      "ARC mfu size in bytes"},
	)

	arcMemoryThrottle = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "zfs",
		Name:      "arc_memory_throttle_count",
		Help:      "ARC memory throttle count"},
	)
)

func main() {
	conf := config{15, 15}

	go recordPoolsIO(conf.ioScrapeInterval)
	go readARCStats()
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
