package vivaldi

import (
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/utils"
	"log"
	"slices"
	"sync"
	"time"
)

type Filter interface {
	FilterCoordinates(string, time.Duration) time.Duration
}

func NewFilter() Filter {
	filterType := utils.ReadConfigString("config.ini", "vivaldi", "filter_type")
	switch filterType {
	case "mp":
		fmt.Println("Using MP filter")
		windowSize, err1 := utils.ReadConfigInt("config.ini", "vivaldi", "h")
		p, err2 := utils.ReadConfigFloat64("config.ini", "vivaldi", "p")
		if err1 != nil || err2 != nil {
			log.Fatalf("Failed to read config: %v", err1)
		}
		return &MPFilter{
			h:       windowSize,
			p:       p,
			windows: make(map[string][]time.Duration),
			mu:      &sync.RWMutex{},
		}
	case "ewma":
		fmt.Println("Using EWMA filter")
		return &EWMAFilter{
			alpha:        0.15,
			currentValue: 0,
		}
	case "raw":
		fmt.Println("Using Raw filter")
		return &RawFilter{}

	default:
		fmt.Println("Invalid filter type: using the default filter")
		return &RawFilter{}
	}
}

type MPFilter struct {
	h       int     // history window size (default 4)
	p       float64 // percentile in 0 - 100 (default 25)
	windows map[string][]time.Duration
	mu      *sync.RWMutex
}

type EWMAFilter struct {
	alpha        float64
	currentValue float64
}

type RawFilter struct {
}

func (mpf *MPFilter) FilterCoordinates(nodeId string, rtt time.Duration) time.Duration {
	mpf.mu.Lock()
	defer mpf.mu.Unlock()
	_, ok := mpf.windows[nodeId]
	if !ok {
		mpf.windows[nodeId] = make([]time.Duration, 0)
	}
	window := mpf.windows[nodeId]
	if len(window) < mpf.h {
		window = append(window, rtt)
		mpf.windows[nodeId] = window
		return rtt
	}
	window = append(window[1:], rtt)
	mpf.windows[nodeId] = window
	samples := make([]time.Duration, mpf.h)
	copy(samples, window)
	slices.Sort(samples)
	i := int(float64(len(samples)) * (mpf.p / 100))
	return samples[i]
}

func (ef *EWMAFilter) FilterCoordinates(nodeId string, rtt time.Duration) time.Duration {
	ef.currentValue = ef.alpha*float64(rtt.Milliseconds()) + (1-ef.alpha)*ef.currentValue
	return time.Duration(ef.currentValue) * time.Millisecond
}

func (rf *RawFilter) FilterCoordinates(nodeId string, rtt time.Duration) time.Duration {
	return rtt
}
