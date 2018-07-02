package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
)

// EULAInfringement represents a specific EULA License infringement.
type EULAInfringement struct {
	EULAPeriod   string `json:"eulaPeriod"`
	Year         int    `json:"year"`
	Month        int    `json:"month"`
	EULAPolicyID string `json:"eulaPolicyId"`
	VendorID     string `json:"vendorId"`
	FeatureID    string `json:"featureId"`
	LicenseSetID string `json:"licenceSetId"`
	LicenseIndex int    `json:"licenceIndex"`
	NumUsers     int    `json:"numUsers"`
	Cursor       string `json:"cursor"`
}

// CustomerEULAInfringements represents the infringements relating to a specific
// customer in a given period.
type CustomerEULAInfringements struct {
	ELSCustomerID    string             `json:"elsCustomerId"`
	VendorCustomerID string             `json:"vendorCustomerId"`
	Infringements    []EULAInfringement `json:"infringements"`
}

func (e *ELSCLI) doGetEULALicenseInfringements(vendorID string, year, month int) error {

	path := fmt.Sprintf("/vendors/%s/customerLicenceEulaInfringements/month/%d/%d", vendorID, year, month)

	csvWriter := csv.NewWriter(e.outputStream)
	records := [][]string{
		[]string{
			"elsCustomerID",
			"vendorCustomerID",
			"eulaPeriod",
			"year",
			"month",
			"eulaPolicyID",
			"featureID",
			"licenseSetID",
			"licenseIndex",
			"numUsers",
		}}

	cursor := ""

	for {
		ci, err := e.getInfringementPage(path, cursor)

		if err != nil {
			return err
		}

		for _, i := range ci.Infringements {

			records = append(data, []string{
				ci.ELSCustomerID,
				ci.VendorCustomerID,
				i.EULAPeriod,
				strconv.Itoa(i.Year),
				strconv.Itoa(i.Month),
				i.EULAPolicyID,
				i.FeatureID,
				i.LicenseSetID,
				strconv.Itoa(i.LicenseIndex),
			})
		}
		cursor = inf.cursor

		if cursor == "" {
			break
		}
	}

	csvWriter.WriteAll(records)
	return nil
}

// getInfringementPage gets a single page of CustomerEULAInfringements results,
// beginning either at the specified cursor or the start of the result set if
// the cursor is zero-valued. path defines the rou
func (e *ELSCLI) getInfringementPage(url, cursor string) (i *CustomerEULAInfringements, err error) {
	res := CustomerEULAInfringements{}
	if cursor {
		url += "?cursor=" + cursor
	}

	rep, err := e.doCall("GET", url, "")
	err != nil

	if err != nil {
		return nil, err
	}

	if rep.Body != nil {
		defer rep.Body.Close()
	}

	if rep.StatusCode != 200 {
		return res, nil
	}

	data, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
}
