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
	"errors"
	"fmt"
	"io"
	"time"

	log "github.com/golang/glog"
	"github.com/jacobsa/go-serial/serial"
)

type command byte
type mode byte

const (
	commandReportMode command = 2
	commandQuery      command = 4
	commandDeviceID   command = 5
	commandWorkState  command = 6
	commandFirmware   command = 7
	commandCycle      command = 8

	modeGet mode = 0
	modeSet mode = 1

	reportModeActive byte = 0
	reportModeQuery  byte = 1

	workStateSleeping  byte = 0
	workStateMeasuring byte = 1
)

// response is what we get on the wire from the sensor. Its meaning
// depends on what it is a reply to.
type response struct {
	Header   byte // always 0xAA
	Command  byte // 0xC0 if in active mode, if reply 0xC5
	Data     [6]byte
	CheckSum byte
	Tail     byte // always 0xAB
}

// IsReply returns true if this response is a reply to a command (as
// opposed to measurements).
func (resp *response) IsReply() bool {
	return resp.Command == 0xC5
}

// PM25 returns the sensor's PM2.5 reading. It will panic if this
// isn't a reply containing the readings.
func (resp *response) PM25() float64 {
	if resp.IsReply() {
		panic(fmt.Sprintf("access to field that doesn't work with this type of response %#v", resp))
	}
	return float64(binary.LittleEndian.Uint16(resp.Data[0:2])) / 10.0
}

// PM10 returns the sensor's PM10 reading. It will panic if this isn't
// a reply containing the readings.
func (resp *response) PM10() float64 {
	if resp.IsReply() {
		panic(fmt.Sprintf("access to field that doesn't work with this type of response %#v", resp))
	}

	return float64(binary.LittleEndian.Uint16(resp.Data[2:4])) / 10.0
}

func (resp *response) checkMatches(cmd command) {
	if resp.Data[0] != byte(cmd) {
		panic(fmt.Sprintf("access to field that doesn't work with this type of response %#v", resp))
	}
}

// ID returns the response's device ID.
func (resp *response) ID() uint16 {
	if resp.IsReply() {
		panic(fmt.Sprintf("access to field that doesn't work with this type of response %#v", resp))
	}

	return binary.LittleEndian.Uint16(resp.Data[4:6])
}

// Firmware returns the version of firmware, as a date (yy-mm-dd). It
// will panic if this is the wrong kind of response.
func (resp *response) Firmware() string {
	resp.checkMatches(commandFirmware)
	return fmt.Sprintf("%02d-%02d-%02d", resp.Data[1], resp.Data[2], resp.Data[3])
}

// DeviceID returns the device id. It will panic if this is the
func (resp *response) DeviceID() string {
	resp.checkMatches(commandDeviceID)
	return fmt.Sprintf("%02d%02d", resp.Data[1], resp.Data[2])
}

func (resp *response) ReportMode() byte {
	resp.checkMatches(commandReportMode)
	return resp.Data[2]
}

func (resp *response) Cycle() uint8 {
	resp.checkMatches(commandCycle)
	return resp.Data[2]
}

func (resp *response) WorkState() byte {
	resp.checkMatches(commandWorkState)
	return resp.Data[2]
}

type request struct {
	Header     byte     // 1 always 0xAA
	SendMarker byte     // 2 always 0xB4
	Command    byte     // 3 command
	Mode       byte     // 4 getting 0, setting 1
	Data       [11]byte // 5-15
	DeviceID   [2]byte  // 16-17 0xFFFF for all device IDs.
	CheckSum   byte     // 18 See makeRequest
	Tail       byte     // 19 always 0xAB
}

func makeRequest(cmd command, mod mode, value byte) *request {
	data := [11]byte{}
	data[0] = value

	req := &request{
		Header:     0xAA,
		SendMarker: 0xB4,
		Command:    byte(cmd),
		Mode:       byte(mod),
		Data:       data,
		DeviceID:   [2]byte{0xFF, 0xFF},
		Tail:       0xAB,
	}
	checksum := int(req.Command) + int(req.Mode)
	for _, v := range data {
		checksum += int(v)
	}
	for _, v := range req.DeviceID {
		checksum += int(v)
	}
	req.CheckSum = byte(checksum % 256)
	return req
}

// IsCorrect returns nil if the responses checksum matches, an error
// otherwise.
func (resp *response) IsCorrect() error {

	var checkSum byte
	for i := 0; i < 6; i++ {
		checkSum += resp.Data[i]
	}

	if checkSum != resp.CheckSum {
		return fmt.Errorf("bad checksum: %#v", resp)
	}
	return nil
}

// A Point represents a single reading from the sensor.
type Point struct {
	PM25      float64
	PM10      float64
	Timestamp time.Time
}

func (point *Point) String() string {
	return fmt.Sprintf("PM2.5: %v μg/m³ PM10: %v μg/m³", point.PM25, point.PM10)
}

// Sensor represents an SDS011 sensor.
type Sensor struct {
	rwc io.ReadWriteCloser
}

func (sensor *Sensor) send(cmd command, mod mode, data byte) error {
	b := new(bytes.Buffer)
	if err := binary.Write(b, binary.LittleEndian, makeRequest(cmd, mod, data)); err != nil {
		return err
	}
	log.V(6).Infof("sending bytes: %#v", b.Bytes())
	_, err := sensor.rwc.Write(b.Bytes())
	return err
}

// receive reads one response from the wire.
func (sensor *Sensor) receive() (*response, error) {
	data := new(response)
	if err := binary.Read(sensor.rwc, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	if err := data.IsCorrect(); err != nil {
		return nil, err
	}
	return data, nil
}

func (sensor *Sensor) receiveReply() (*response, error) {
	// FIXME(ryszard): This should support timeouts.
	for i := 0; i < 10; i++ {
		resp, err := sensor.receive()
		if err != nil {
			return nil, err
		}
		if resp.IsReply() {
			return resp, nil
		}
		log.V(6).Infof("received data, but not a reply: %#v", resp)
	}
	return nil, errors.New("no reply")

}

// ReportMode returns true if the device is in active mode, false if
// in query mode.
func (sensor *Sensor) ReportMode() (bool, error) {
	if err := sensor.send(commandReportMode, modeGet, 0); err != nil {
		return false, err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return false, err
	}
	log.V(6).Infof("ReportMode response: %#v", data)
	return data.ReportMode() == reportModeActive, nil
}

// MakeActive makes the sensor actively report its measurements.
func (sensor *Sensor) MakeActive() error {
	if err := sensor.send(commandReportMode, modeSet, reportModeActive); err != nil {
		return err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return err
	}
	log.V(6).Infof("MakeActive: %#v", data)
	return nil
}

// MakePassive stop the sensor from actively reporting its
// measurements. You will need to send a Query command.
func (sensor *Sensor) MakePassive() error {
	log.V(6).Infof("make passive")
	if err := sensor.send(commandReportMode, modeSet, reportModeQuery); err != nil {
		return err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return err
	}
	log.V(6).Infof("MakePassive response: %#v", data)
	return nil
}

// DeviceID returns the sensor's device ID.
func (sensor *Sensor) DeviceID() (string, error) {
	if err := sensor.send(commandDeviceID, modeGet, 0); err != nil {
		return "", err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return "", err
	}
	log.V(6).Infof("DeviceID: %#v", data)
	return data.DeviceID(), nil

}

// Firmware returns the firmware version (a yy-mm-dd date).
func (sensor *Sensor) Firmware() (string, error) {
	if err := sensor.send(commandFirmware, modeGet, 0); err != nil {
		return "", err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return "", err
	}
	log.V(6).Infof("Firmare: %#v", data)
	return data.Firmware(), nil

}

// Cycle returns the current cycle length in minutes. If it's 0 it
// means that cycle is not set, and the sensor is streaming data
// continuously.
func (sensor *Sensor) Cycle() (uint8, error) {
	if err := sensor.send(commandCycle, modeGet, 0); err != nil {
		return 0, err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return 0, err
	}
	log.V(6).Infof("Cycle: %#v", data)
	return data.Cycle(), nil
}

// SetCycle sets the cycle length. The value is the cycle's length in
// minutes, accepting values from 1 to 30. If you pass it 0 it will
// disable cycle work, and the sensor will just stream data.
func (sensor *Sensor) SetCycle(value uint8) error {
	if value < 0 || value > 30 {
		return fmt.Errorf("duty cycle: bad value %v. Should be between 0 and 30.", value)
	}
	if err := sensor.send(commandCycle, modeSet, value); err != nil {
		return err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return err
	}
	log.V(6).Infof("SetCycle: %#v", data)
	return nil
}

// Query returns one reading.
func (sensor *Sensor) Query() (*Point, error) {
	if err := sensor.send(commandQuery, modeGet, 0); err != nil {
		return nil, err
	}
	return sensor.Get()
}

// IsAwake returns true if the sensor is awake.
func (sensor *Sensor) IsAwake() (bool, error) {
	if err := sensor.send(commandWorkState, modeGet, 0); err != nil {
		return false, err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return false, err
	}
	log.V(6).Infof("IsAwake WorkState: %#v", data)
	return data.WorkState() == workStateMeasuring, nil
}

// Awake awakes the sensor if it is in sleep mode.
func (sensor *Sensor) Awake() error {
	if err := sensor.send(commandWorkState, modeSet, workStateMeasuring); err != nil {
		return err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return err
	}
	log.V(6).Infof("Awake WorkState: %#v", data)
	return nil
}

// Sleep puts the sensor to sleep.
func (sensor *Sensor) Sleep() error {
	if err := sensor.send(commandWorkState, modeSet, workStateSleeping); err != nil {
		return err
	}
	data, err := sensor.receiveReply()
	if err != nil {
		return err
	}
	log.V(6).Infof("WorkState: %#v", data)
	return nil
}

// Close closes the underlying serial port.
func (sensor *Sensor) Close() {
	sensor.rwc.Close()
}

// New returns a sensor that will read data from serial port for which
// the path was provided. It is the responsibility of the caller to
// close the sensor.
func New(portPath string) (*Sensor, error) {
	options := serial.OpenOptions{
		PortName:        portPath,
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	port, err := serial.Open(options)
	if err != nil {
		return nil, err
	}
	return NewSensor(port), nil
}

// NewSensor returns a sensor that will read its data from the provided
// read-write-closer.
func NewSensor(rwc io.ReadWriteCloser) *Sensor {
	return &Sensor{rwc: rwc}
}

// Get will read one measurement. It will block until data is
// available. It only makes sense to call read if the sensor is in
// active mode.
func (sensor *Sensor) Get() (point *Point, err error) {
	data, err := sensor.receive()
	if err != nil {
		return nil, err
	}
	log.V(6).Infof("Query data: %#v", data)
	return &Point{PM25: data.PM25(), PM10: data.PM10(), Timestamp: time.Now()}, nil
}
