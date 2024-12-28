package wakeonlan

import (
	"fmt"
	"net"
)

// Broadcast sends Wake-on-LAN packets.
//
// The MAC must be in EUI-48/MAC-48 format.
func Broadcast(mac string) error {
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("parse mac: %w", err)
	}

	if len(hwAddr) != 6 {
		return fmt.Errorf("unsupported mac format")
	}

	// Magic packet is a frame that contains anywhere within its payload 6 bytes of
	// all 255 (FF FF FF FF FF FF in hexadecimal), followed by sixteen repetitions of
	// the target computer's 48-bit MAC address, for a total of 102 bytes.
	// See https://en.wikipedia.org/wiki/Wake-on-LAN
	packet := make([]byte, 0, 102)
	packet = append(packet, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...)
	for i := 0; i < 16; i++ {
		packet = append(packet, hwAddr...)
	}

	raddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:7")
	if err != nil {
		return fmt.Errorf("resolve udp addr: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return fmt.Errorf("dial udp: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write(packet); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
