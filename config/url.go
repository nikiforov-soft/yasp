package config

import (
	"net/url"
)

type Url struct {
	*url.URL
}

func (u *Url) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	parsedUrl, err := url.Parse(s)
	u.URL = parsedUrl
	return err
}
