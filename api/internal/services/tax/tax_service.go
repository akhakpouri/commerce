package tax

import (
	dto "commerce/api/internal/dto/tax"
	"errors"
	"sort"
)

type TaxServiceI interface {
GetStates() []string
	Calculate(amount float64, state string) (*float64, error)
}

type TaxService struct {
}

func NewTaxService() TaxServiceI {
	return &TaxService{}
}

// Calculate implements [TaxServiceI].
func (t *TaxService) Calculate(amount float64, state string) (*float64, error) {
	taxState, found := stateTaxes[state]

	if !found {
		return nil, errors.New("State was not found.")
	}
	taxAmount := (taxState.Amount * amount)
	return &taxAmount, nil
}

// GetStates implements [TaxServiceI].
func (t *TaxService) GetStates() []string {
	keys := make([]string, 0, len(stateTaxes))
	for key := range stateTaxes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

var stateTaxes = map[string]dto.Tax{
	"AL": {State: "AL", Amount: 0.04},
	"AK": {State: "AK", Amount: 0.00},
	"AZ": {State: "AZ", Amount: 0.056},
	"AR": {State: "AR", Amount: 0.065},
	"CA": {State: "CA", Amount: 0.0725},
	"CO": {State: "CO", Amount: 0.029},
	"CT": {State: "CT", Amount: 0.0635},
	"DE": {State: "DE", Amount: 0.00},
	"FL": {State: "FL", Amount: 0.06},
	"GA": {State: "GA", Amount: 0.04},
	"HI": {State: "HI", Amount: 0.04},
	"ID": {State: "ID", Amount: 0.06},
	"IL": {State: "IL", Amount: 0.0625},
	"IN": {State: "IN", Amount: 0.07},
	"IA": {State: "IA", Amount: 0.06},
	"KS": {State: "KS", Amount: 0.065},
	"KY": {State: "KY", Amount: 0.06},
	"LA": {State: "LA", Amount: 0.0445},
	"ME": {State: "ME", Amount: 0.055},
	"MD": {State: "MD", Amount: 0.06},
	"MA": {State: "MA", Amount: 0.0625},
	"MI": {State: "MI", Amount: 0.06},
	"MN": {State: "MN", Amount: 0.06875},
	"MS": {State: "MS", Amount: 0.07},
	"MO": {State: "MO", Amount: 0.04225},
	"MT": {State: "MT", Amount: 0.00},
	"NE": {State: "NE", Amount: 0.055},
	"NV": {State: "NV", Amount: 0.0685},
	"NH": {State: "NH", Amount: 0.00},
	"NJ": {State: "NJ", Amount: 0.06625},
	"NM": {State: "NM", Amount: 0.04875},
	"NY": {State: "NY", Amount: 0.04},
	"NC": {State: "NC", Amount: 0.0475},
	"ND": {State: "ND", Amount: 0.05},
	"OH": {State: "OH", Amount: 0.0575},
	"OK": {State: "OK", Amount: 0.045},
	"OR": {State: "OR", Amount: 0.00},
	"PA": {State: "PA", Amount: 0.06},
	"RI": {State: "RI", Amount: 0.07},
	"SC": {State: "SC", Amount: 0.06},
	"SD": {State: "SD", Amount: 0.045},
	"TN": {State: "TN", Amount: 0.07},
	"TX": {State: "TX", Amount: 0.0625},
	"UT": {State: "UT", Amount: 0.0595},
	"VT": {State: "VT", Amount: 0.06},
	"VA": {State: "VA", Amount: 0.053},
	"WA": {State: "WA", Amount: 0.065},
	"WV": {State: "WV", Amount: 0.06},
	"WI": {State: "WI", Amount: 0.05},
	"WY": {State: "WY", Amount: 0.04},
	"DC": {State: "DC", Amount: 0.06},
}
