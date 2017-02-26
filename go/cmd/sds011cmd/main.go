package main

import (
	"flag"
	"fmt"
	"strconv"

	log "github.com/golang/glog"
	"github.com/ryszard/sds011/go/sds011"
)

var portPath = flag.String("port_path", "/dev/ttyUSB0", "serial port path")

func main() {
	flag.Parse()

	sensor, err := sds011.New(*portPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sensor.Close()

	switch cmd := flag.Arg(0); cmd {
	case "cycle":
		dutyCycle, err := sensor.Cycle()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(dutyCycle)
	case "set_cycle":
		v, err := strconv.Atoi(flag.Arg(1))
		if err != nil {
			log.Fatalf("bad number: %v", flag.Arg(1))
		}
		if err := sensor.SetCycle(uint8(v)); err != nil {
			log.Fatal(err)
		}

	default:
		log.Errorf("flag.Args: %v", flag.Args())
	}
}
