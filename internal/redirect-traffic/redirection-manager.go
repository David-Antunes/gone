package redirect_traffic

import (
	"errors"
	"sync"
)

type RedirectManager struct {
	mu         sync.Mutex
	sniffers   map[string]*SniffComponent
	intercepts map[string]*InterceptComponent
}

type SniffComponent struct {
	Id        string
	MachineId string
	Socket    *RedirectionSocket
	Component Component
}

type InterceptComponent struct {
	Id        string
	MachineId string
	Socket    *RedirectionSocket
	Component Component
}

type Component interface {
	ID() string
}

func NewRedirectManager() *RedirectManager {
	return &RedirectManager{
		mu:         sync.Mutex{},
		sniffers:   make(map[string]*SniffComponent),
		intercepts: make(map[string]*InterceptComponent),
	}
}

func (rm *RedirectManager) AddSniffer(id string, sniffer *SniffComponent) error {
	rm.mu.Lock()
	if _, ok := rm.sniffers[id]; !ok {
		rm.sniffers[id] = sniffer
	} else {
		rm.mu.Unlock()
		return errors.New("sniffer already exists")
	}
	rm.mu.Unlock()
	return nil
}
func (rm *RedirectManager) AddIntercept(id string, intercept *InterceptComponent) error {
	rm.mu.Lock()
	if _, ok := rm.intercepts[id]; !ok {
		rm.intercepts[id] = intercept
	} else {
		rm.mu.Unlock()
		return errors.New("intercept already exists")
	}
	rm.mu.Unlock()
	return nil
}

func (rm *RedirectManager) RemoveSniffer(id string) error {
	rm.mu.Lock()
	if _, ok := rm.sniffers[id]; ok {
		delete(rm.sniffers, id)
	} else {
		rm.mu.Unlock()
		return errors.New("sniffer does not exist")
	}
	rm.mu.Unlock()
	return nil
}

func (rm *RedirectManager) RemoveIntercept(id string) error {
	rm.mu.Lock()
	if _, ok := rm.intercepts[id]; ok {
		delete(rm.intercepts, id)
	} else {
		rm.mu.Unlock()
		return errors.New("intercept does not exist")
	}
	rm.mu.Unlock()
	return nil
}

func (rm *RedirectManager) GetSniffer(id string) (*SniffComponent, error) {
	rm.mu.Lock()
	if sniff, ok := rm.sniffers[id]; ok {
		rm.mu.Unlock()
		return sniff, nil
	} else {
		rm.mu.Unlock()
		return nil, errors.New("sniffer does not exist")
	}
}

func (rm *RedirectManager) GetIntercept(id string) (*InterceptComponent, error) {
	rm.mu.Lock()
	if intercept, ok := rm.intercepts[id]; ok {
		rm.mu.Unlock()
		return intercept, nil
	} else {
		rm.mu.Unlock()
		return nil, errors.New("intercept does not exist")
	}
}

func (rm *RedirectManager) ListSniffers() map[string]SniffComponent {
	rm.mu.Lock()
	list := make(map[string]SniffComponent)

	for k, v := range rm.sniffers {
		list[k] = *v
	}
	rm.mu.Unlock()
	return list
}
func (rm *RedirectManager) ListIntercepts() map[string]InterceptComponent {
	rm.mu.Lock()
	list := make(map[string]InterceptComponent)

	for k, v := range rm.intercepts {
		list[k] = *v
	}

	rm.mu.Unlock()
	return list
}
