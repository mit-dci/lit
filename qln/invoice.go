package qln

import (
	"fmt"
	"log"
)

func parseInvoice(invoice string) (string, uint64, uint64, error) {
	if len(invoice) > 60 || len(invoice) < 4 {
		// having 1 as invoice length in order to check whether this works
		// error printed by appropriate handlers, no need to print it here.
		return invoice, 0, 0, fmt.Errorf("Invalid invoice length. Quitting!")
	}
	return invoice, 0, 0, nil
}

func (nd *LitNode) PayInvoice(invoice string) (string, error) {
	// log.Println("Calls litrpc. Cool, QUitting")
	destAdr, invoiceAmount, invoiceId, err := parseInvoice(invoice)
	if err != nil {
		return "", err
	}
	log.Printf("Sending %d to address: %s with invoice ID %d\n", invoiceAmount, destAdr, invoiceId)
	return "0x00000000000000000000000000000000", nil
}
