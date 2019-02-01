package main

import (
	"encoding/csv"
	"io"
	"math/rand"
	"net/url"
	"os"
)

type Proxy *url.URL
type Proxies []Proxy

var proxies = new(Proxies)
var lastIndexProxy int = 0

func (p *Proxies) AddFromFile(filePatch string) error {
	file, err := os.Open(filePatch)

	if err != nil {
		return err
	}

	defer file.Close()
	reader := csv.NewReader(file)

	for {
		record, err := reader.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		} else if len(record) != 1 {
			continue
		}

		proxyUrlRaw := "//" + record[0]
		proxyUrl, err := url.Parse(proxyUrlRaw)

		if err != nil {
			return err
		}

		if proxyUrl != nil {
			*p = append(*p, proxyUrl)
		}
	}

	return nil
}

func (p *Proxies) Get() Proxy {
	if len(*p) == lastIndexProxy {
		lastIndexProxy = 0
	}

	proxy := (*p)[lastIndexProxy]
	lastIndexProxy++

	return proxy
}

func (p *Proxies) GetRandom() Proxy {
	if len(*p) == 0 {
		return nil
	}

	index := rand.Intn(len(*p))
	proxy := (*p)[index]

	return proxy
}
