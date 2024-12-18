package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/David-Antunes/gone/internal/network"
	"net/http"
	"time"
)

func ParseRequest(r *http.Request, apiRequestStruct any) error {

	d := json.NewDecoder(r.Body)
	req := apiRequestStruct
	err := d.Decode(&req)

	if err != nil {
		return err
	}
	return nil
}

func SendResponse(w http.ResponseWriter, apiRequestStruct any) {

	resp, err := json.Marshal(apiRequestStruct)

	if err != nil {
		fmt.Println("Error marshalling response.")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Write(resp)
}

func SendError(w http.ResponseWriter, apiRequestStruct any) {

	resp, err := json.Marshal(apiRequestStruct)

	if err != nil {
		fmt.Println("Error marshalling response.")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, string(resp), http.StatusBadRequest)
}

func ParseLinkProps(latency float64, bandwidth int, jitter float64, dropRate float64, weight int) (network.LinkProps, error) {
	if latency < 0 {
		return network.LinkProps{}, errors.New("latency can't be lower than 0 ms")
	} else if bandwidth < 12000 {
		return network.LinkProps{}, errors.New("bandwidth can't be lower than 1.5 kbps")
	} else if jitter < 0 {
		return network.LinkProps{}, errors.New("jitter can't be lower than 0")
	} else if dropRate < 0 || dropRate > 1 {
		return network.LinkProps{}, errors.New("drop rate must be between 0 and 1")
	} else if weight < 0 {
		return network.LinkProps{}, errors.New("weight can't be lower than 0")
	}
	return network.LinkProps{
		Latency:   time.Duration(latency * float64(time.Millisecond) / 2.0),
		FLatency:  latency,
		Bandwidth: bandwidth / 8,
		Jitter:    jitter,
		DropRate:  dropRate,
		Weight:    weight,
	}, nil
}

func ParseLinkPropsInternal(latency time.Duration, bandwidth int, jitter float64, dropRate float64, weight int) (network.LinkProps, error) {
	if latency < time.Millisecond*0 {
		return network.LinkProps{}, errors.New("latency can't be lower than 0 ms")
	} else if bandwidth < 10 {
		return network.LinkProps{}, errors.New("bandwidth can't be lower than 1.5 kbps")
	} else if jitter < 0 {
		return network.LinkProps{}, errors.New("jitter can't be lower than 0")
	} else if dropRate < 0 || dropRate > 1 {
		return network.LinkProps{}, errors.New("drop rate must be between 0 and 1")
	} else if weight < 0 {
		return network.LinkProps{}, errors.New("weight can't be lower than 0")
	}
	return network.LinkProps{
		Latency:   latency / 2.0,
		FLatency:  float64(latency),
		Bandwidth: bandwidth / 8,
		Jitter:    jitter,
		DropRate:  dropRate,
		Weight:    weight,
	}, nil

}
