package types

import (
	"bytes"
	"fmt"
	"sync"
)

type DepositReport struct {
	QueryData []byte
	Value     []byte
}

type DepositReports struct {
	sync.Mutex
	Reports []DepositReport
}

func NewDepositReports() *DepositReports {
	return &DepositReports{
		Reports: make([]DepositReport, 0),
	}
}

// GetReports returns the list of pending deposits.
func (d *DepositReports) GetReports() []DepositReport {
	d.Lock()
	defer d.Unlock()
	return d.Reports
}

// AddReport adds a new deposit report to the list of pending deposits.
func (d *DepositReports) AddReport(report DepositReport) {
	d.Lock()
	defer d.Unlock()
	d.Reports = append(d.Reports, report)
}

func (d *DepositReports) RemoveReport(report DepositReport) {
	d.Lock()
	defer d.Unlock()
	for i, r := range d.Reports {
		if bytes.Equal(r.QueryData, report.QueryData) && bytes.Equal(r.Value, report.Value) {
			d.Reports = append(d.Reports[:i], d.Reports[i+1:]...)
			break
		}
	}
}

func (d *DepositReports) GetOldestReport() (DepositReport, error) {
	d.Lock()
	defer d.Unlock()
	if len(d.Reports) == 0 {
		return DepositReport{}, fmt.Errorf("no pending deposits")
	}
	oldest := d.Reports[0]
	d.Reports = d.Reports[1:]
	return oldest, nil
}
