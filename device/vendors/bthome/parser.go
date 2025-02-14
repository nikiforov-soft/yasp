package bthome

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	encryptionFlag         = 1 << 0
	triggerBasedDeviceFlag = 1 << 2
	versionFlag            = (1 << 3) - 1
)

// Parse - Parses bthome encoded data
// https://bthome.io/format/#sensor-data
func Parse(data []byte) (*SensorData, error) {
	if len(data) < 3 {
		return nil, fmt.Errorf("bthome: invalid data length: %d", len(data))
	}

	r := bytes.NewReader(data)
	header, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("bthome: failed to read header id: %w", err)
	}
	capabilityFlags := readCapabilities(header)
	if capabilityFlags.Version != 2 {
		return nil, fmt.Errorf("bthome: unsupported bthome version: %d", capabilityFlags.Version)
	}

	var sensorData SensorData
	sensorData.CapabilityFlags = capabilityFlags
	for {
		propertyType, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("bthome: failed to read property type: %d", propertyType)
		}

		switch propertyType {
		case 0x00: // packet id
			/**
			* https://bthome.io/format/#misc-data
			* The packet id is optional and can be used to filter duplicate data.
			* This allows you to send multiple advertisements that are exactly the same, to improve the chance that the advertisement arrives.
			* BTHome receivers should only process the advertisement if the packet id is different compared to the previous one.
			* The packet id is a value between 0 (0x00) and 255 (0xFF), and should be increased on every change in data.
			* Note that most home automation software already have some other filtering for unchanged data.
			* The use of a packet id is therefore often not needed.
			**/
			packetId, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			sensorData.PacketId = packetId
		case 0x01: // battery %
			battery, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("bthome: failed to read battery: %w", err)
			}
			sensorData.BatteryPercent = battery
		case 0x05: // illuminance lux
			illuminance, err := readIlluminance(r)
			if err != nil {
				return nil, fmt.Errorf("bthome: failed to read illuminance: %w", err)
			}
			sensorData.IlluminanceLux = illuminance
		case 0x21: // motion state
			motion, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("bthome: failed to read motion state: %w", err)
			}
			sensorData.MotionState = motion
		case 0x2D: // window state
			windowState, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("bthome: failed to read window state: %w", err)
			}
			sensorData.WindowState = windowState
		case 0x2E: // humidity %
			humidity, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("bthome: failed to read humidity: %w", err)
			}
			sensorData.HumidityPercent = humidity
		case 0x3A: // button
			var buttonEvent uint16
			if err := binary.Read(r, binary.BigEndian, &buttonEvent); err != nil {
				return nil, fmt.Errorf("bthome: failed to read button event: %w", err)
			}
			sensorData.ButtonEvent = buttonEvent
		case 0x3F: // rotation °
			var rotation int16
			if err := binary.Read(r, binary.BigEndian, &rotation); err != nil {
				return nil, fmt.Errorf("bthome: failed to read rotation: %w", err)
			}
			sensorData.RotationDegrees = float32(rotation) / 10.
		case 0x45: // temperature °C
			var temperature int16
			if err := binary.Read(r, binary.BigEndian, &temperature); err != nil {
				return nil, fmt.Errorf("bthome: failed to read temperature: %w", err)
			}
			sensorData.TemperatureCelsius = float32(temperature) / 10.
		default:
			return nil, fmt.Errorf("bthome: unknown property type: %d", propertyType)
		}
	}

	return &sensorData, nil
}

func readCapabilities(capabilityFlags byte) CapabilityFlags {
	encryption := (capabilityFlags & encryptionFlag) != 0
	triggerBasedDevice := (capabilityFlags & triggerBasedDeviceFlag) != 0
	version := (capabilityFlags >> 5) & versionFlag
	return CapabilityFlags{
		Encryption:         encryption,
		TriggerBasedDevice: triggerBasedDevice,
		Version:            version,
	}
}

func readIlluminance(r io.Reader) (float32, error) {
	var component12 uint16
	err := binary.Read(r, binary.BigEndian, &component12)
	if err != nil {
		return 0, err
	}

	var component3 byte
	err = binary.Read(r, binary.BigEndian, &component3)
	if err != nil {
		return 0, err
	}

	illuminance := uint32(component12)<<8 | uint32(component3)
	return float32(illuminance) / 100., nil
}
