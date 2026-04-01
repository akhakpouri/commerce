package helpers

import "strconv"

func ParseParamToUint(param string) (*uint, error) {
	id, err := strconv.ParseUint(param, 10, 64)

	if err != nil {
		return nil, err
	}
	val := uint(id)
	return &val, nil
}
