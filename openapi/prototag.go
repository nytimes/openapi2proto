package openapi

import (
	"encoding/json"
	"strconv"
)

type protoTag int

func (pt *protoTag) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		return json.Unmarshal(b, (*int)(pt))
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	i, err := strconv.Atoi(s)
	if (err != nil) {
		return err
	}

	*pt = protoTag(i)

	return nil
}
