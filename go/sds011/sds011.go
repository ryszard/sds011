// Copyright 2017 Ryszard Szopa <ryszard.szopa@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package sds011 implements a library to read the protocol of SDS011,
// an air quality sensor than can work with Raspberry Pi.
package sds011

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// Message is what we get on the wire from the sensor. See
// http://inovafitness.com/software/SDS011%20laser%20PM2.5%20sensor%20specification-V1.3.pdf.
type message struct {
	Header      byte
	CommanderNo byte
	PM25        int16
	PM10        int16
	ID          int16
	CheckSum    byte
	Tail        byte
}

// A Point represents a single reading from the sensor.
type Point struct {
	PM25      float64   `datastore:"pm25"`
	PM10      float64   `datastore:"pm10"`
	Timestamp time.Time `datastore:"timestamp"`
}

func (point *Point) String() string {
	return fmt.Sprintf("PM2.5: %v μg/m³ PM10: %v μg/m³", point.PM25, point.PM10)
}

// IsCorrect returns nil if the messages checksum matches, an error
// otherwise.
func (m *message) IsCorrect() error {

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, m.PM25); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, m.PM10); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, m.ID); err != nil {
		return err
	}
	var checkSum byte
	b := buf.Bytes()
	for i := 0; i < 6; i++ {
		checkSum += b[i]
	}

	if checkSum != m.CheckSum {
		return fmt.Errorf("bad checksum: %#v", m)
	}
	return nil
}

// Sensor represents an SDS011 sensor.
type Sensor struct {
	reader io.Reader
}

// New returns a sensor that will read its data from the provided
// reader.
func New(reader io.Reader) *Sensor {
	return &Sensor{reader: reader}
}
func (sensor *Sensor) Get() (point *Point, err error) {
	data := new(message)
	if err := binary.Read(sensor.reader, binary.LittleEndian, data); err != nil {
		return nil, fmt.Errorf("binary.Read: %v", err)
	}

	if err := data.IsCorrect(); err != nil {
		return nil, err
	}

	return &Point{PM25: float64(data.PM25) / 10, PM10: float64(data.PM10) / 10, Timestamp: time.Now()}, nil
}
