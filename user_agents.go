package main

import (
	"encoding/csv"
	"io"
	"math/rand"
	"os"
)

type UserAgent *string
type UserAgents []UserAgent

var userAgents = new(UserAgents)
var lastIndexUserAgent = 0

func (u *UserAgents) AddFromFile(filePatch string) error {
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

		userAgent := UserAgent(&record[0])
		*u = append(*u, userAgent)
	}

	return nil
}

func (u *UserAgents) Get() UserAgent {
	if len(*u) == lastIndexUserAgent {
		lastIndexUserAgent = 0
	}

	userAgent := (*u)[lastIndexUserAgent]
	lastIndexUserAgent++

	return userAgent
}

func (u *UserAgents) GetRandom() UserAgent {
	if len(*u) == 0 {
		return nil
	}

	index := rand.Intn(len(*u))
	userAgent := (*u)[index]

	return userAgent
}
