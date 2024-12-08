package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Uage: %s <origin zip code>\n", os.Args[0])
		os.Exit(1)
	}
	originZipCode := os.Args[1]

	d, err := NewDesigneesData(24, false)
	if err != nil {
		log.Fatalf("NewDesigneesData error: %v", err)
	}
	geo, err := NewGeo(d)
	if err != nil {
		log.Fatalf("NewGeo error: %v", err)
	}

	ranked, err := geo.RankDesigneesByDistance(originZipCode)
	if err != nil {
		log.Fatalf("geo.RankDesigneesByDistance error: %v", err)
	}

	fmt.Printf("=== DPEs ranked by distance to %s ===\n\n", originZipCode)
	for _, item := range ranked {
		designee := d.designees[item.designeeID]
		fmt.Printf("%.1f sm\n", item.miles)
		fmt.Printf("Designee Name: %s\n", designee.FullName)
		fmt.Printf("Designee City: %s\n", designee.Address.City+", "+designee.Address.State.Name)
		fmt.Printf("Designee Phone Number: %s\n", designee.PhoneNumber)
		if len(designee.Address.PhoneNumber) != 0 && designee.PhoneNumber != designee.Address.PhoneNumber {
			fmt.Printf("Designee Address Phone Number: %s\n", designee.Address.PhoneNumber)
		}
		fmt.Printf("Designee Email: %s\n", designee.Email)
		fmt.Printf("Designee Function Codes: %s\n", designee.FunctionCodes)
		fmt.Println()
	}
}
