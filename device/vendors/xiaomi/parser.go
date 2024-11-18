package xiaomi

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/pschlump/AesCCM"
)

var (
	ErrBindKeyRequired = errors.New("bind key required")
)

func ParseBLEFrame(rawData []byte, bindKeyCallback func(mac string) ([]byte, error)) (*Frame, error) {
	reader := bufio.NewReader(bytes.NewReader(rawData))
	frameControl, version, err := parseFrameControlAndVersion(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read frame control and version: %w", err)
	}

	productId, err := parseProductId(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read product id: %w", err)
	}

	frameCounter, err := parseFrameCounter(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read product id: %w", err)
	}

	var macAddress string
	if frameControl.HasMacAddress {
		macAddress, err = parseMacAddress(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read mac address: %w", err)
		}
	}

	var capabilities CapabilityFlags
	if frameControl.HasCapabilities {
		capabilities, err = parseCapabilities(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read capabilities: %w", err)
		}
	}

	var eventReader io.Reader
	if frameControl.IsEncrypted {
		eventOffset := 5
		if frameControl.HasMacAddress {
			eventOffset += 6
		}
		if frameControl.HasCapabilities {
			eventOffset += 1
		}

		var bindKey []byte
		if frameControl.HasMacAddress {
			bindKey, err = bindKeyCallback(formatMacAddress(macAddress))
			if err != nil {
				return nil, fmt.Errorf("failed to bind key: %w", err)
			}
		}

		decryptedPayload, err := parseEncryptedPayload(rawData, bindKey, eventOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to read encrypted payload: %w", err)
		}
		eventReader = bufio.NewReader(bytes.NewReader(decryptedPayload))
	} else {
		eventReader = reader
	}

	var eventType uint16
	var eventLength uint8
	var event Event
	if frameControl.HasEvent {
		eventType, err = parseEventType(eventReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read event type: %w", err)
		}

		eventLength, err = parseEventLength(eventReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read event length: %w", err)
		}

		event, err = parseEventData(eventType, eventReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read event data: %w", err)
		}
	}

	return &Frame{
		FrameControlFlags: frameControl,
		Version:           version,
		ProductId:         productId,
		FrameCounter:      frameCounter,
		Capabilities:      capabilities,
		MacAddress:        macAddress,
		EventType:         eventType,
		EventLength:       eventLength,
		Event:             event,
	}, nil
}

func parseFrameControlAndVersion(reader io.Reader) (FrameControlFlags, byte, error) {
	var frameControl uint16
	if err := binary.Read(reader, binary.LittleEndian, &frameControl); err != nil {
		return FrameControlFlags{}, 0, err
	}
	version := byte(frameControl >> (8 + 4))

	return FrameControlFlags{
		IsFactoryNew:    (frameControl & frameControlBitIsFactoryNew) != 0,
		IsConnected:     (frameControl & frameControlBitIsConnected) != 0,
		IsCentral:       (frameControl & frameControlBitIsCentral) != 0,
		IsEncrypted:     (frameControl & frameControlBitIsEncrypted) != 0,
		HasMacAddress:   (frameControl & frameControlBitHasMacAddress) != 0,
		HasCapabilities: (frameControl & frameControlBitHasCapabilities) != 0,
		HasEvent:        (frameControl & frameControlBitHasEvent) != 0,
		HasCustomData:   (frameControl & frameControlBitHasCustomData) != 0,
		HasSubtitle:     (frameControl & frameControlBitHasSubtitle) != 0,
		HasBinding:      (frameControl & frameControlBitHasBinding) != 0,
	}, version, nil
}

func parseProductId(reader io.Reader) (uint16, error) {
	var productId uint16
	if err := binary.Read(reader, binary.LittleEndian, &productId); err != nil {
		return 0, err
	}
	return productId, nil
}

func parseFrameCounter(reader io.Reader) (byte, error) {
	var frameCounter uint8
	if err := binary.Read(reader, binary.LittleEndian, &frameCounter); err != nil {
		return 0, err
	}
	return frameCounter, nil
}

func parseMacAddress(reader io.Reader) (string, error) {
	macBuffer := make([]byte, 6)
	if _, err := io.ReadFull(reader, macBuffer); err != nil {
		return "", err
	}
	for i, j := 0, len(macBuffer)-1; i < j; i, j = i+1, j-1 {
		macBuffer[i], macBuffer[j] = macBuffer[j], macBuffer[i]
	}
	return hex.EncodeToString(macBuffer), nil
}

func parseCapabilities(reader io.Reader) (CapabilityFlags, error) {
	var capabilities uint8
	if err := binary.Read(reader, binary.LittleEndian, &capabilities); err != nil {
		return CapabilityFlags{}, err
	}
	return CapabilityFlags{
		Connectable: (capabilities & capabilityBitConnectable) != 0,
		Central:     (capabilities & capabilityBitCentral) != 0,
		Secure:      (capabilities & capabilityBitSecure) != 0,
		IO:          (capabilities & capabilityBitIO) != 0,
	}, nil
}

func parseEncryptedPayload(data, bindKey []byte, eventOffset int) ([]byte, error) {
	if len(bindKey) == 0 {
		return nil, ErrBindKeyRequired
	}

	msgLength := len(data)
	eventLength := msgLength - eventOffset
	if eventLength < 3 {
		return nil, errors.New("not enough data to decrypt")
	}

	var ciphertext []byte
	ciphertext = append(ciphertext, data[11:len(data)-7]...) // payload
	ciphertext = append(ciphertext, data[len(data)-4:]...)   // token

	var nonce []byte
	nonce = append(nonce, data[5:11]...)                    // reverse MAC
	nonce = append(nonce, data[2:5]...)                     // sensor type
	nonce = append(nonce, data[len(data)-7:len(data)-4]...) // counter

	aesCipher, err := aes.NewCipher(bindKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to initializes aes cipher: %w", err)
	}

	ccm, err := aesccm.NewCCM(aesCipher, 4, 12)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ccm cipher")
	}

	var additionalData = []byte{0x11}
	dst, err := ccm.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return dst, nil
}

func parseEventType(reader io.Reader) (uint16, error) {
	var eventType uint16
	if err := binary.Read(reader, binary.LittleEndian, &eventType); err != nil {
		return 0, err
	}
	return eventType, nil
}

func parseEventLength(reader io.Reader) (byte, error) {
	var eventLength uint8
	if err := binary.Read(reader, binary.LittleEndian, &eventLength); err != nil {
		return 0, err
	}
	return eventLength, nil
}

func parseEventData(eventType uint16, reader io.Reader) (Event, error) {
	switch EventType(eventType) {
	case EventTypeTemperature:
		var eventData int16
		if err := binary.Read(reader, binary.LittleEndian, &eventData); err != nil {
			return nil, err
		}
		return &EventTemperature{
			Temperature: float64(eventData) / 10.0,
		}, nil
	case EventTypeHumidity:
		var eventData uint16
		if err := binary.Read(reader, binary.LittleEndian, &eventData); err != nil {
			return nil, err
		}
		return &EventHumidity{
			Humidity: float64(eventData) / 10.0,
		}, nil
	case EventTypeIlluminance:
		eventData := make([]byte, 3)
		if _, err := io.ReadFull(reader, eventData); err != nil {
			return nil, err
		}
		return &EventIlluminance{
			Illuminance: uint(eventData[0]) | uint(eventData[1])<<8 | uint(eventData[2])<<16,
		}, nil
	case EventTypeMoisture:
		var eventData uint8
		if err := binary.Read(reader, binary.LittleEndian, &eventData); err != nil {
			return nil, err
		}
		return &EventMoisture{
			Moisture: eventData,
		}, nil
	case EventTypeFertility:
		var eventData int16
		if err := binary.Read(reader, binary.LittleEndian, &eventData); err != nil {
			return nil, err
		}
		return &EventFertility{
			Fertility: eventData,
		}, nil
	case EventTypeBattery:
		var eventData uint8
		if err := binary.Read(reader, binary.LittleEndian, &eventData); err != nil {
			return nil, err
		}
		return &EventBattery{
			Battery: eventData,
		}, nil
	case EventTypeTemperatureAndHumidity:
		var temperature int16
		if err := binary.Read(reader, binary.LittleEndian, &temperature); err != nil {
			return nil, err
		}

		var humidity uint16
		if err := binary.Read(reader, binary.LittleEndian, &humidity); err != nil {
			return nil, err
		}

		return &EventTemperatureAndHumidity{
			Temperature: float64(temperature) / 10.0,
			Humidity:    float64(humidity) / 10.0,
		}, nil
	default:
		return nil, fmt.Errorf("unknown event type: %d", eventType)
	}
}
