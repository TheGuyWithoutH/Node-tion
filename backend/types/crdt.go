package types

import "fmt"

// -----------------------------------------------------------------------------
// CRDTOperationsMessage

// NewEmpty implements types.Message.
func (c CRDTOperationsMessage) NewEmpty() Message {
	return &CRDTOperationsMessage{}
}

// Name implements types.Message.
func (c CRDTOperationsMessage) Name() string {
	return "crdtoperations"
}

// String implements types.Message.
func (c CRDTOperationsMessage) String() string {
	return fmt.Sprintf("crdtoperations{%d operations}", len(c.Operations))
}

// HTML implements types.Message.
func (c CRDTOperationsMessage) HTML() string { return c.String() }
