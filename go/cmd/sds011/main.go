// sds011 is a simple reader for the SDS011 Air Quality Sensor. It
// outputs data in TSV to standard output (timestamp formatted
// according to RFC3339, PM2.5 levels, PM10 levels).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/ryszard/sds011/go/sds011"
)

var portPath = flag.String("port_path", "/dev/ttyUSB0", "serial port path")

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr,
			`sds011 reads data from the SDS011 sensor and sends them to stdout as TSV.

The columns are: an RFC3339 timestamp, the PM2.5 level, the PM10 level.`)
		fmt.Fprintf(os.Stderr, "\n\nUsage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
}
func main() {
	flag.Parse()

	options := serial.OpenOptions{
		PortName:        *portPath,
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	defer port.Close()
	sensor := sds011.New(port)

	for {
		point, err := sensor.Get()
		if err != nil {
			log.Printf("ERROR: sensor.Get: %v", err)
			continue
		}
		fmt.Fprintf(os.Stdout, "%v\t%v\t%v\n", point.Timestamp.Format(time.RFC3339), point.PM25, point.PM10)
	}
}
