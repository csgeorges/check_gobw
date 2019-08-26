package main

import (
	"check_gobw/config"

	"flag"
	"fmt"
	"math"
	"time"

	"bufio"
	"os"
	"strconv"
	"strings"
)

var W = flag.Int("w", 50, "warning limit as percentage")
var C = flag.Int("c", 100, "critical limit as percentage")
var Sleep = flag.Duration("s", 10*time.Second, "sleep time in seconds")
var Inter = flag.String("i", "*", "interface")
var Stats = flag.Bool("S", false, "runtime stats for debugging")
var B = flag.Bool("B", false, "switch to using bytes, default is bits")
var Version = flag.Bool("v", false, "version information")

type NetStat struct {
	Dev  []string
	Stat map[string]*DevStat
}

type DevStat struct {
	Name   string
	Speed  int
	Rx     uint64
	Tx     uint64
	RBitps float64
	TBitps float64
}

func ReadLines(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		if *Stats {
			fmt.Printf("Error: %v\n", err)
		}
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

		if key == "lo" {
			continue
		}

		if *Stats {
			fmt.Printf("%10s: %v\n", "Interface", key)
			fmt.Printf("%10s: %v\n", "Stats", value)
		}

		c := new(DevStat)
		c.Name = key

		speedfile := fmt.Sprintf("/sys/class/net/%v/speed", key)
		tempspeed, _ := ReadLines(speedfile)
		tempspeedint, _ := strconv.Atoi(tempspeed[0])

		c.Speed = tempspeedint * 1000000

		if *Stats {
			fmt.Printf("%10s: %v\n", "Speedfile", speedfile)
			fmt.Printf("%10s: %v\n", "Speed", c.Speed)
		}

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

	if *Version {
		exitcode := 0
		fmt.Printf("%10s: %s\n%10s: %s\n%10s: %s\n", "VERSION", config.VERSION, "GITHASH", config.GITHASH, "BUILD DATE", config.BUILDSTAMP)
		os.Exit(exitcode)
	}

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
			dev.RBitps = (float64(dev.Rx) * 8) / sleepfloat
			dev.TBitps = (float64(dev.Tx) * 8) / sleepfloat
			dev.Speed = t1.Speed
		}
	}

	status := "OK"
	exitcode := 0
	totaldevs := len(delta.Dev) - 1
	if totaldevs+1 == 0 {
		status = "UNKNOWN"
		exitcode = 3
		fmt.Printf("BANDWIDTH %v: Unable to determine network interfaces.\n", status)
		os.Exit(exitcode)
	}

	for _, iface := range delta.Dev {
		stat := delta.Stat[iface]
		// calculate the percentage of the interface
		warning := (*W * stat.Speed) / 100
		critical := (*C * stat.Speed) / 100

		if int(stat.RBitps) > critical || int(stat.TBitps) > critical {
			status = "CRITICAL"
			exitcode = 2
		} else if int(stat.RBitps) > warning || int(stat.TBitps) > warning {
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
		warning := (*W * stat.Speed) / 100
		critical := (*C * stat.Speed) / 100

		if k == totaldevs {
			fmt.Printf("%v_Rx=%.2fB/s;%v;%v;; %v_Tx=%.2fB/s;%v;%v;;", iface, stat.RBitps, warning, critical, iface, stat.TBitps, warning, critical)
		} else {
			fmt.Printf("%v_Rx=%.2fB/s;%v;%v;; %v_Tx=%.2fB/s;%v;%v;; ", iface, stat.RBitps, warning, critical, iface, stat.TBitps, warning, critical)
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
	var tmp float64
	var suffix string
	var s string

	if *B {
		tmp = float64(bytes) / delta
		suffix = "Byte"
		b := uint64(tmp)
		switch {
		case b < uint64(2<<9):
		case b < uint64(2<<19):
			tmp = tmp / float64(2<<9)
			s = "K"
		case b < uint64(2<<29):
			tmp = tmp / float64(2<<19)
			s = "M"
		case b < uint64(2<<39):
			tmp = tmp / float64(2<<29)
			s = "G"
		case b < uint64(2<<49):
			tmp = tmp / float64(2<<39)
			s = "T"
		}
	} else {
		tmp = float64(bytes*8) / delta
		suffix = "bit"
		b := tmp
		switch {
		case b < math.Pow10(3):
		case b < math.Pow10(6):
			tmp = tmp / math.Pow10(3)
			s = "K"
		case b < math.Pow10(9):
			tmp = tmp / math.Pow10(6)
			s = "M"
		case b < math.Pow10(12):
			tmp = tmp / math.Pow10(9)
			s = "G"
		case b < math.Pow10(15):
			tmp = tmp / math.Pow10(12)
			s = "T"
		}
	}
	ret = fmt.Sprintf("%.2f%s%s/s", tmp, s, suffix)
	return
}
