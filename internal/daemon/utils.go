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

func ParseLinkProps(latency int, bandwidth int, jitter float64, dropRate float64, weight int) (network.LinkProps, error) {
	if latency < 0 {
		return network.LinkProps{}, errors.New("latency can't be lower than 10 ms")
	} else if bandwidth < 10 {
		return network.LinkProps{}, errors.New("bandwidth can't be lower than 10 kbps")
	} else if jitter < 0 {
		return network.LinkProps{}, errors.New("jitter can't be lower than 0")
	} else if dropRate < 0 || dropRate > 1 {
		return network.LinkProps{}, errors.New("drop rate must be between 0 and 1")
	} else if weight < 0 {
		return network.LinkProps{}, errors.New("weight can't be lower than 0")
	}

	return network.LinkProps{
		Latency:   time.Duration(latency*int(time.Millisecond)) / 2,
		Bandwidth: bandwidth,
		Jitter:    jitter / 2,
		DropRate:  dropRate / 2,
		Weight:    weight,
	}, nil

}
func ParseLinkPropsInternal(latency time.Duration, bandwidth int, jitter float64, dropRate float64, weight int) (network.LinkProps, error) {
	if latency < time.Millisecond*0 {
		return network.LinkProps{}, errors.New("latency can't be lower than 10 ms")
	} else if bandwidth < 10 {
		return network.LinkProps{}, errors.New("bandwidth can't be lower than 10 kbps")
	} else if jitter < 0 {
		return network.LinkProps{}, errors.New("jitter can't be lower than 0")
	} else if dropRate < 0 || dropRate > 1 {
		return network.LinkProps{}, errors.New("drop rate must be between 0 and 1")
	} else if weight < 0 {
		return network.LinkProps{}, errors.New("weight can't be lower than 0")
	}

	return network.LinkProps{
		Latency:   latency * time.Millisecond / 2,
		Bandwidth: bandwidth,
		Jitter:    jitter / 2,
		DropRate:  dropRate / 2,
		Weight:    weight,
	}, nil

}
