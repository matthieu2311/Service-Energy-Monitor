package controller

import (
	"data_api/server/model"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ReadEnergy reads energy consumption from RAPL
func readEnergy() (float64, error) {
	data, err := os.ReadFile("/sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj")
	if err != nil {
		return 0, err
	}

	energyStr := strings.TrimSpace(string(data))
	energy, err := strconv.ParseFloat(energyStr, 64)
	if err != nil {
		return 0, err
	}

	return energy, nil
}

// Calls readEnergy every 5 seconds (can be changed) and compute the energy consumed during those 5 seconds.
// It then creates a point in time with this value, and add it to the channel.
func MonitorEnergy(pointsChan chan model.Point, wg *sync.WaitGroup) {

	defer wg.Done()
	prevValue := 0.0
	var dif float64

	for {
		value, err := readEnergy()
		if err != nil {
			log.Fatal(err)
		}
		if prevValue != 0 {
			dif = value - prevValue //We have to substract to get the uJ consumed in the meantime
			prevValue = value
		} else {
			prevValue = value
			continue
		}
		p := model.Point{Timestamp: time.Now().UTC(), Value: dif * 1e-6}
		fmt.Println(p)
		pointsChan <- p

		time.Sleep(5 * time.Second)
	}
}
