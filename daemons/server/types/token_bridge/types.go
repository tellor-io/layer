package types

import (
	"bytes"
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
	return &DepositReports{}
}

func (d *DepositReports) GetReports() []DepositReport {
	return d.Reports
}

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

func (d *DepositReports) GetOldestReport() DepositReport {
	return d.Reports[0]
}
