package main

import (
	"flag"
	"fmt"
	"time"

	"bufio"
	"os"
	"strconv"
	"strings"
)

var W = flag.Int("w", 5000000, "warning limit in bytes")
var C = flag.Int("c", 10000000, "critical limit in bytes")
var Sleep = flag.Duration("s", 10*time.Second, "sleep time in seconds")
var Inter = flag.String("i", "*", "interface")
var Stats = flag.Bool("S", false, "runtime stats for debugging")

type NetStat struct {
	Dev  []string
	Stat map[string]*DevStat
}

type DevStat struct {
	Name string
	Rx   uint64
	Tx   uint64
	Rbps int
	Tbps int
}

func ReadLines(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}
	return ret, nil
}

func getStats() (ret NetStat) {
	lines, _ := ReadLines("/proc/net/dev")

	ret.Dev = make([]string, 0)
	ret.Stat = make(map[string]*DevStat)

	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.Fields(strings.TrimSpace(fields[1]))
		if *Inter != "*" && *Inter != key {
			continue
		}

		c := new(DevStat)
		c.Name = key

		r, err := strconv.ParseInt(value[0], 10, 64)
		if err != nil {
			break
		}
		c.Rx = uint64(r)

		t, err := strconv.ParseInt(value[8], 10, 64)
		if err != nil {
			break
		}
		c.Tx = uint64(t)

		ret.Dev = append(ret.Dev, key)
		ret.Stat[key] = c
	}
	return
}

func main() {
	flag.Parse()

	var stat0 NetStat
	var stat1 NetStat
	var delta NetStat

	delta.Dev = make([]string, 0)
	delta.Stat = make(map[string]*DevStat)

	start := time.Now()

	stat0 = getStats()
	time.Sleep(*Sleep)
	stat1 = getStats()
	sleepfloat := time.Duration.Seconds(*Sleep)

	for _, value := range stat0.Dev {
		t0, ok := stat0.Stat[value]
		if ok {
			dev, ok := delta.Stat[value]
			if !ok {
				delta.Stat[value] = new(DevStat)
				dev = delta.Stat[value]
				delta.Dev = append(delta.Dev, value)
			}
			t1, ok := stat1.Stat[value]
			dev.Rx = t1.Rx - t0.Rx
			dev.Tx = t1.Tx - t0.Tx
			dev.Rbps = int(float64(dev.Rx) / sleepfloat)
			dev.Tbps = int(float64(dev.Tx) / sleepfloat)
		}
	}

	totaldevs := len(delta.Dev) - 1

	status := "OK"
	exitcode := 0
	for _, iface := range delta.Dev {
		stat := delta.Stat[iface]
		if stat.Rbps > *C || stat.Tbps > *C {
			status = "CRITICAL"
			exitcode = 2
		} else if stat.Rbps > *W || stat.Tbps > *W {
			if status == "OK" {
				status = "WARNING"
				exitcode = 1
			}
		}
	}
	fmt.Printf("BANDWIDTH %v: ", status)

	for k, iface := range delta.Dev {
		stat := delta.Stat[iface]
		if k == totaldevs {
			fmt.Printf("%v(Rx %v Tx %v)", iface, Vsize(stat.Rx, sleepfloat), Vsize(stat.Tx, sleepfloat))
		} else {
			fmt.Printf("%v(Rx %v Tx %v) ", iface, Vsize(stat.Rx, sleepfloat), Vsize(stat.Tx, sleepfloat))
		}
	}

	fmt.Printf(";|")

	for k, iface := range delta.Dev {
		stat := delta.Stat[iface]
		if k == totaldevs {
			fmt.Printf("%v_Rx=%v[B];%v;%v;; %v_Tx=%v[B];%v;%v;;", iface, stat.Rbps, *W, *C, iface, stat.Tbps, *W, *C)
		} else {
			fmt.Printf("%v_Rx=%v[B];%v;%v;; %v_Tx=%v[B];%v;%v;; ", iface, stat.Rbps, *W, *C, iface, stat.Tbps, *W, *C)
		}
	}

	fmt.Printf("\n")

	elapsed := time.Since(start)
	if *Stats {
		overhead := elapsed - *Sleep
		fmt.Printf("\n")
		fmt.Printf("%10s: %v\n", "Start", start)
		fmt.Printf("%10s: %v\n", "Elapsed", elapsed)
		fmt.Printf("%10s: %v\n", "Sleep", *Sleep)
		fmt.Printf("%10s: %v\n", "Overhead", overhead)
		fmt.Printf("%10s: %v\n", "Devices", totaldevs+1)
	}
	os.Exit(exitcode)
}

func Vsize(bytes uint64, delta float64) (ret string) {
	var tmp float64 = float64(bytes) / delta
	var s string

	bytes = uint64(tmp)

	switch {
	case bytes < uint64(2<<9):

	case bytes < uint64(2<<19):
		tmp = tmp / float64(2<<9)
		s = "K"

	case bytes < uint64(2<<29):
		tmp = tmp / float64(2<<19)
		s = "M"

	case bytes < uint64(2<<39):
		tmp = tmp / float64(2<<29)
		s = "G"

	case bytes < uint64(2<<49):
		tmp = tmp / float64(2<<39)
		s = "T"

	}
	ret = fmt.Sprintf("%.2f%sbyte/s", tmp, s)
	return
}
