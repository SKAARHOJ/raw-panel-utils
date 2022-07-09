package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func getTimeString() string {
	t := time.Now()
	return fmt.Sprintf("%02d:%02d:%02d.%d", t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000/1000/100)
}

func prettyHexPrint(data []byte) string {
	output := ""
	for _, b := range data {
		output += hex.EncodeToString([]byte{b}) + " "
	}

	return strings.TrimSpace(output)
}
