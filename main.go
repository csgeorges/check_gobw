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

var W = flag.Int("w", 50000, "warning limit in bytes")
var C = flag.Int("c", 100000, "critical limit in bytes")
var S = flag.Duration("s", 5*time.Second, "sleep time in seconds")
var Inter = flag.String("i", "*", "interface")

type NetStat struct {
	Dev  []string
	Stat map[string]*DevStat
}

type DevStat struct {
	Name string
	Rx   uint64
	Tx   uint64
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

	//start := time.Now()

	stat0 = getStats()
	time.Sleep(*S)
	stat1 = getStats()
	sleepfloat := time.Duration.Seconds(*S)

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
			//fmt.Printf("t0: %v\n", t0)
			//fmt.Printf("t1: %v\n", t1)
			//fmt.Printf("%v\n", sleeptime)
			//fmt.Printf("Rx: %v\n", Vsize(dev.Rx, sleepfloat))
			//fmt.Printf("Tx: %v\n", Vsize(dev.Tx, sleepfloat))
		}
	}

	for _, iface := range delta.Dev {
		stat := delta.Stat[iface]
		fmt.Printf("%v(Rx:%v/Tx:%v)|%v|%v\n", iface, Vsize(stat.Rx, sleepfloat), Vsize(stat.Tx, sleepfloat), stat.Rx, stat.Tx)
	}

	//elapsed := time.Since(start)
	//fmt.Printf("%v\n", stat0)
	//fmt.Printf("%v\n", stat1)
	//fmt.Printf("Start: %v\nElapsed: %v\n", start, elapsed)
}

func Vsize(bytes uint64, delta float64) (ret string) {
	var tmp float64 = float64(bytes) / delta
	var s string = ""

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
	ret = fmt.Sprintf("%.2f%sBps", tmp, s)
	return
}
