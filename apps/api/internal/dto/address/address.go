package address

import "commerce/internal/shared/models"

type Address struct {
	Id         uint   `json:"id"`
	UserId     uint   `json:"user_id"`
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	IsDefault  bool   `json:"is_default"`
}

func FromModel(address *models.Address) *Address {
	return &Address{
		Id:         address.Id,
		UserId:     address.UserId,
		Street:     address.Street,
		City:       address.City,
		State:      address.State,
		PostalCode: address.PostalCode,
		Country:    address.Country,
		IsDefault:  address.IsDefault,
	}
}

func ToModel(address *Address) *models.Address {
	return &models.Address{
		UserId:     address.UserId,
		Street:     address.Street,
		City:       address.City,
		State:      address.State,
		PostalCode: address.PostalCode,
		Country:    address.Country,
		IsDefault:  address.IsDefault,
	}
}
