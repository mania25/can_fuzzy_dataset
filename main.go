package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Constants for data generation
const (
	TotalRecords  = 3838860 // Total number of CAN frames to generate
	NormalCount   = 3347013 // Number of normal messages
	InjectedCount = 491847  // Number of injected messages
	DataLength    = 8       // DLC fixed to 8 bytes
)

// Counters to track the number of normal and injected messages generated
var normalMessages, injectedMessages int

// Predefined DBC-like data for normal CAN messages with fluctuating ranges
var DBC = map[uint32]func() [8]byte{
	0x100: func() [8]byte { return [8]byte{byte(toggleOnOff()), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} },      // EngineOnOff (fluctuates between on/off)
	0x101: func() [8]byte { return [8]byte{byte(toggleOnOff()), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} },      // FrontLight (fluctuates between on/off)
	0x200: func() [8]byte { return [8]byte{byte(fluctuate(80, 100)), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} }, // EngineTempSensor (80 - 100 °C)
	0x201: func() [8]byte { return [8]byte{byte(fluctuate(60, 90)), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} },  // InjectorTimingSensor (60 - 90 ms)
	0x202: func() [8]byte { return [8]byte{byte(fluctuate(90, 100)), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} }, // OxygenSensor (90 - 100%)
	0x203: func() [8]byte { return [8]byte{byte(fluctuate(60, 80)), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} },  // FuelTankLevel (60 - 80%)
	0x204: func() [8]byte { return [8]byte{byte(fluctuate(40, 60)), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} },  // ThrottlePosition (40 - 60%)
	0x205: func() [8]byte {
		return [8]byte{byte(fluctuate(2500, 3000) >> 8), byte(fluctuate(2500, 3000) & 0xFF), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}, // EngineRPM (2500 - 3000 RPM)
}

// Helper function to generate random fluctuations within a range
func fluctuate(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// Helper function to randomly toggle on/off (1 for on, 0 for off)
func toggleOnOff() int {
	if rand.Float64() < 0.5 {
		return 1 // on
	}
	return 0 // off
}

// Function to generate CAN data with exact counts for normal and injected messages
func generateCANData() (uint32, [8]byte, string) {
	var canID uint32
	var data [8]byte
	var flag string

	if injectedMessages < InjectedCount && (normalMessages >= NormalCount || rand.Float64() < 0.5) {
		// Generate injected message
		canID = uint32(rand.Intn(0x300-0x206) + 0x206) // Random ID outside DBC range
		for i := 0; i < DataLength; i++ {
			data[i] = byte(rand.Intn(256))
		}
		flag = "T"
		injectedMessages++
	} else if normalMessages < NormalCount {
		// Generate normal message with fluctuating sensor data
		dbcKeys := make([]uint32, 0, len(DBC))
		for k := range DBC {
			dbcKeys = append(dbcKeys, k)
		}
		canID = dbcKeys[rand.Intn(len(dbcKeys))]
		data = DBC[canID]() // Call function to generate fluctuating data
		flag = "R"
		normalMessages++
	}

	return canID, data, flag
}

// Function to generate and save dataset as a CSV file
func generateDataset(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Initialize progress bar
	bar := progressbar.NewOptions(TotalRecords,
		progressbar.OptionSetDescription("Generating CAN dataset"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "#",
			SaucerPadding: "-",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// Generate CAN data and write to CSV
	for i := 0; i < TotalRecords; i++ {
		timestamp := formatTimestamp() // Generate UNIX timestamp with microsecond precision
		canID, data, flag := generateCANData()

		record := []string{
			timestamp,
			fmt.Sprintf("%X", canID), // CAN ID in hex without "0x" prefix
			strconv.Itoa(DataLength),
		}

		// Convert data to hex string
		for _, b := range data {
			record = append(record, fmt.Sprintf("%02X", b))
		}

		record = append(record, flag)
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("could not write record: %v", err)
		}

		bar.Add(1) // Update progress bar
	}

	return nil
}

// Function to format timestamp as UNIX time with microsecond precision
func formatTimestamp() string {
	now := time.Now()
	seconds := now.Unix()
	microseconds := now.UnixMicro() - (seconds * 1e6)
	return fmt.Sprintf("%d.%06d", seconds, microseconds)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	filename := "Fuzzy_dataset.csv"
	if err := generateDataset(filename); err != nil {
		fmt.Printf("Error generating dataset: %v\n", err)
	} else {
		fmt.Printf("\nDataset generated successfully and saved to %s\n", filename)
	}
}
