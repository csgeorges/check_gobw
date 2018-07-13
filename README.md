# check_gobw

check_gobw is a nagios bandwidth checker written in golang, it uses data from /proc/net/dev to calculate the bandwidth

## Install

- Use the included `build.sh` file to build a 64 and 32 bit version.
- Copy the binary file into your nagios plugins directory

## Usage

```
Usage of ./check_gobw:
  -B    switch to using bytes, default is bits
  -S    runtime stats for debugging
  -c int
        critical limit in bytes (default 10000000)
  -i string
        interface (default "*")
  -s duration
        sleep time in seconds (default 10s)
  -w int
        warning limit in bytes (default 5000000)
```

## Todo

- Double check math is correct for conversions.
- Create graph template for perfdata that is provided.
- Ability for interface selection to be multiple interfaces.
