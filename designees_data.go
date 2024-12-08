package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type searchResponse struct {
	Total  int        `json:"total"`
	Data   []Designee `json:"data"`
	Status int        `json:"Status"`
	Title  string     `json:"Title"`
}

type Designee struct {
	DesigneeNumber string `json:"designeeNumber"`
	PhoneNumber    string `json:"phoneNumber"`
	Address        struct {
		Address1 string `json:"address1"`
		Address2 string `json:"address2"`
		City     string `json:"city"`
		State    struct {
			Name string `json:"name"`
		} `json:"state"`
		Country struct {
			Name string `json:"name"`
		} `json:"country"`
		ZipCode         string `json:"zipCode"`
		PhoneNumber     string `json:"phoneNumber"`
		AddressFullName string `json:"addressFullName"`
	} `json:"address"`
	FullName        string `json:"fullName"`
	FunctionCodes   string `json:"functionCodes"`
	Email           string `json:"email"`
	CompleteAddress string `json:"completeAddress"`
}

type DesigneesData struct {
	designees map[string]Designee // uuid -> Designee
}

func NewDesigneesData(designeeTypeID int, refreshCache bool) (d *DesigneesData, err error) {
	d = &DesigneesData{designees: make(map[string]Designee)}
	if refreshCache {
		err = d.fetchFromFAA(designeeTypeID)
		if err != nil {
			return nil, fmt.Errorf("d.fetchFromFAA error: %v", err)
		}
	}
	hasData, err := d.loadFromLocalStorage(designeeTypeID)
	if err != nil {
		return nil, fmt.Errorf("d.loadFromLocalStorage error: %v", err)
	}
	if !hasData {
		err = d.fetchFromFAA(designeeTypeID)
		if err != nil {
			return nil, fmt.Errorf("d.fetchFromFAA error: %v", err)
		}
	}
	return d, nil
}

func (d *DesigneesData) getDataLocalStoragePath(designeeTypeID int) (filePath string, err error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("os.UserCacheDir error: %v", err)
	}
	fp := filepath.Join(cacheDir, fmt.Sprintf("designees-%d.json", designeeTypeID))
	return fp, nil
}

func (d *DesigneesData) ingestSearchResponse(resp *searchResponse) error {
	if (resp.Status != 0 && resp.Status != 200) || len(resp.Title) > 0 {
		return fmt.Errorf("searchResponse error. Status: %d, Title: %s", resp.Status, resp.Title)
	}
	if len(resp.Data) == 0 {
		return errors.New("empty data")
	}
	if len(resp.Data) != resp.Total {
		return fmt.Errorf("unexpected number of items in searchResponse %d != %d", resp.Total, len(resp.Data))
	}
	for _, designee := range resp.Data {
		d.designees[uuid.New().String()] = designee
	}
	return nil
}

type faaSearchQuery struct {
	PageModel struct {
		First int `json:"first"`
		Rows  int `json:"rows"`
	} `json:"pageModel"`
	CountryID        int  `json:"countryId"`
	DesigneeTypeID   int  `json:"designeeTypeId"`
	IsLocationSearch bool `json:"isLocationSearch"`
}

func (d *DesigneesData) fetchFromFAA(designeeTypeID int) (err error) {
	var query faaSearchQuery
	query.CountryID = 184
	query.DesigneeTypeID = 24
	query.IsLocationSearch = true
	query.PageModel.First = 0
	query.PageModel.Rows = 65536 // big enough to cover all DPEs
	j, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("json.Marshal error: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost,
		"https://designee.faa.gov/designeeapi/api/Cloa/Search/", bytes.NewReader(j))
	if err != nil {
		return fmt.Errorf("http.NewRequest error: %v", err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http.DefaultClient.Do error: %v", err)
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp("", "*.json")
	if err != nil {
		return fmt.Errorf("os.CreateTemp error: %v", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	log.Printf("saving FAA result (designeeTypeID=%d) into temp file: %s",
		designeeTypeID, f.Name())
	var sr searchResponse
	err = json.NewDecoder(io.TeeReader(resp.Body, f)).Decode(&sr)
	if err != nil {
		return fmt.Errorf("json Decode error: %v", err)
	}
	err = d.ingestSearchResponse(&sr)
	if err != nil {
		return fmt.Errorf("ingestSearchResponse error: %v", err)
	}

	{ // Data is good. Move data file over.
		f.Close()
		p, err := d.getDataLocalStoragePath(designeeTypeID)
		if err != nil {
			return fmt.Errorf("getDataLocalStoragePath error: %v", err)
		}
		log.Printf("searchResponse(designeeTypeID=%d) is good. Moving result into: %s",
			designeeTypeID, p)
		err = os.Rename(f.Name(), p)
		if err != nil {
			return fmt.Errorf("os.Rename error: %v", err)
		}
	}
	return nil
}

func (d *DesigneesData) loadFromLocalStorage(designeeTypeID int) (hasData bool, err error) {
	p, err := d.getDataLocalStoragePath(designeeTypeID)
	if err != nil {
		return false, fmt.Errorf("getDataLocalStoragePath error: %v", err)
	}
	log.Printf("loadFromLocalStorage(designeeTypeID=%d) from local file: %s",
		designeeTypeID, p)
	f, err := os.Open(p)
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err == nil:
	default:
		return false, fmt.Errorf("os.Open error: %v", err)
	}
	defer f.Close()
	var sr searchResponse
	err = json.NewDecoder(f).Decode(&sr)
	if err != nil {
		return false, fmt.Errorf("json Decode error: %v", err)
	}
	err = d.ingestSearchResponse(&sr)
	if err != nil {
		return false, fmt.Errorf("ingestSearchResponse error: %v", err)
	}
	return true, nil
}
