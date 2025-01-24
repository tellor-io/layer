package types

import (
	"bytes"
	"fmt"
	"sync"
)

type QueryData struct {
	QueryData []byte
}

type DepositTips struct {
	sync.Mutex
	Tips []QueryData
}

func NewDepositTips() *DepositTips {
	return &DepositTips{
		Tips: make([]QueryData, 0),
	}
}

// GetReports returns the list of pending deposits.
func (d *DepositTips) GetTips() []QueryData {
	d.Lock()
	defer d.Unlock()
	return d.Tips
}

// AddReport adds a new deposit report to the list of pending deposits.
func (d *DepositTips) AddTip(tip QueryData) {
	fmt.Printf("Adding tip: %v\n", tip)
	d.Lock()
	defer d.Unlock()
	d.Tips = append(d.Tips, tip)
}

func (d *DepositTips) RemoveTip(tip QueryData) {
	d.Lock()
	defer d.Unlock()
	for i, t := range d.Tips {
		if bytes.Equal(t.QueryData, tip.QueryData) {
			d.Tips = append(d.Tips[:i], d.Tips[i+1:]...)
			break
		}
	}
}

func (d *DepositTips) RemoveOldestTip() {
	d.Lock()
	defer d.Unlock()
	if len(d.Tips) == 0 {
		return
	}
	d.Tips = d.Tips[1:]
}

func (d *DepositTips) GetOldestTip() (QueryData, error) {
	d.Lock()
	defer d.Unlock()
	if len(d.Tips) == 0 {
		return QueryData{}, fmt.Errorf("no pending deposits")
	}
	oldest := d.Tips[0]
	return oldest, nil
}
