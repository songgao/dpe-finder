package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/jftuga/geodist"
)

type Location struct {
	Latitude  float64
	Longitude float64
}

type Geo struct {
	latlong map[string]Location // DesigneeID -> Location
}

func NewGeo(d *DesigneesData) (g *Geo, err error) {
	g = &Geo{latlong: make(map[string]Location)}
	found, missing := 0, 0
	for designeeID, designee := range d.designees {
		shortZipCode := designee.Address.ZipCode
		if len(designee.Address.ZipCode) >= 5 {
			shortZipCode = designee.Address.ZipCode[:5]
		}
		loc, ok := zipCodes[shortZipCode]
		if ok {
			g.latlong[designeeID] = loc
			found++
		} else {
			missing++
			log.Printf("no location found for designeeID=%q, zipCode=%q(%q)", designeeID, designee.Address.ZipCode, shortZipCode)
		}
	}
	log.Printf("finished zip code -> location lookup. found %d; missing %d; for total %d designees", found, missing, len(d.designees))
	return g, nil
}

type DesigneeIDWithDistance struct {
	designeeID string
	miles      float64
}

func (g *Geo) RankDesigneesByDistance(zipCode string) (ranked []DesigneeIDWithDistance, err error) {
	originLoc, ok := zipCodes[zipCode]
	if !ok {
		return nil, fmt.Errorf("origin zip code %s not found", zipCode)
	}
	all := make([]DesigneeIDWithDistance, 0, len(g.latlong))
	for designeeID, loc := range g.latlong {
		miles, _, err := geodist.VincentyDistance(geodist.Coord{
			Lat: originLoc.Latitude,
			Lon: originLoc.Longitude,
		}, geodist.Coord{
			Lat: loc.Latitude,
			Lon: loc.Longitude,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to compute Vincenty Distance to designeeID=%s: %v", designeeID, err)
		}
		all = append(all, DesigneeIDWithDistance{designeeID: designeeID, miles: miles})
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].miles < all[j].miles
	})
	return all, nil
}
