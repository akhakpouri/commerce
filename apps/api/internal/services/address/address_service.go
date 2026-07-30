package address

import (
	dto "commerce/api/internal/dto/address"
	addressrepo "commerce/internal/shared/repositories/address"
	"log/slog"
)

type AddressServiceI interface {
	GetById(id uint) (*dto.Address, error)
	GetAllByUserId(userId uint) ([]*dto.Address, error)
	Save(address *dto.Address) error
	Delete(id uint, hard bool) error
}

type AddressService struct {
	repo addressrepo.AddressRepositoryI
}

func NewAddressService(repo addressrepo.AddressRepositoryI) AddressServiceI {
	return &AddressService{repo: repo}
}

// Delete implements [AddressServiceI].
func (a *AddressService) Delete(id uint, hard bool) error {
	return a.repo.Delete(id, hard)
}

// GetAllByUserId implements [AddressServiceI].
func (a *AddressService) GetAllByUserId(userId uint) ([]*dto.Address, error) {
	models, err := a.repo.GetByUserId(userId)
	if err != nil {
		slog.Error("Error occured getting addresses by user.", "error", err)
		return nil, err
	}
	addresses := []*dto.Address{}
	for _, addr := range models {
		addresses = append(addresses, dto.FromModel(addr))
	}
	return addresses, nil
}

// GetById implements [AddressServiceI].
func (a *AddressService) GetById(id uint) (*dto.Address, error) {
	model, err := a.repo.GetById(id)
	if err != nil {
		slog.Error("Error occured getting addresses by id.", "error", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// Save implements [AddressServiceI].
func (a *AddressService) Save(address *dto.Address) error {
	model := dto.ToModel(address)
	return a.repo.Save(model)
}
