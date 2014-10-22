// Copyright (C) 2014 Miquel Sabaté Solà <mikisabate@gmail.com>
// This file is licensed under the MIT license.
// See the LICENSE file.

package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mssola/dym"
)

type Topic struct {
	Id         string    `json:"id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Contents   string    `json:"contents,omitempty"`
	Created_at time.Time `json:"created_at,omitempty"`
	Markdown   string    `json:"markdown,omitempty"`
}

func unknownTopic(name string) {
	var topics []Topic
	var names []string

	readTopics(&topics)
	for _, v := range topics {
		names = append(names, v.Name)
	}

	msg := fmt.Sprintf("td: the topic '%v' does not exist.", name)
	similars := dym.Similar(names, name)
	if len(similars) == 0 {
		fmt.Printf(msg)
	} else {
		msg += "\n\nDid you mean one of these?\n"
		for _, v := range similars {
			msg += "\t" + v + "\n"
		}
		fmt.Printf(msg)
	}
}

func Edit() error {
	cmd := exec.Command(editor())
	cmd.Dir = filepath.Join(home(), dirName, newDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Diff() error {
	// TODO
	return nil
}

func Fetch() error {
	// Perform the HTTP request.
	res, err := getResponse("GET", "/topics", nil)
	if err != nil {
		return fromError(err)
	}

	// Parse the given topics.
	var topics []Topic
	body, _ := ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &topics); err != nil {
		return fromError(err)
	}

	// And save the results.
	save(topics)
	fmt.Printf("Topics updated.\n")
	return nil
}

func List() error {
	var topics []Topic
	readTopics(&topics)
	for _, v := range topics {
		fmt.Printf("%v\n", v.Name)
	}
	return nil
}

func Push() error {
	var success, fails []string
	var topics []Topic
	readTopics(&topics)

	total := len(topics)
	for k, v := range topics {
		// Print the status.
		fmt.Printf("\rPushing... %v/%v\r", k+1, total)

		// Get the contents.
		file := filepath.Join(home(), dirName, newDir, v.Name+".md")
		body, _ := ioutil.ReadFile(file)
		t := &Topic{Contents: string(body)}
		if t.Contents == "" {
			success = append(success, v.Name)
			continue
		}

		// Perform the request.
		body, _ = json.Marshal(t)
		path := "/topics/" + v.Id
		_, err := getResponse("PUT", path, bytes.NewReader(body))
		if err == nil {
			success = append(success, v.Name)
		} else {
			fails = append(fails, v.Name)
		}
	}

	// And finally update the file system.
	update(success, fails)
	return nil
}

func Status() error {
	// TODO
	return nil
}

func Create(name string) error {
	// Perform the HTTP request.
	t := &Topic{Name: name}
	body, _ := json.Marshal(t)
	res, err := getResponse("POST", "/topics", bytes.NewReader(body))
	if err != nil {
		return fromError(err)
	}

	// Parse the newly created topic and add it to the list.
	body, _ = ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &t); err != nil {
		return fromError(err)
	}
	addTopic(t)
	return nil
}

func Delete(name string) error {
	var topics, actual []Topic
	var id string

	// Get the list of topics straight.
	readTopics(&topics)
	for _, v := range topics {
		if v.Name == name {
			id = v.Id
		} else {
			actual = append(actual, v)
		}
	}
	if id == "" {
		unknownTopic(name)
		os.Exit(1)
	}

	// Perform the HTTP request.
	_, err := getResponse("DELETE", "/topics/"+id, nil)
	if err != nil {
		return fromError(err)
	}

	// On the system.
	writeTopics(actual)
	file := filepath.Join(home(), dirName, oldDir, name+".md")
	os.RemoveAll(file)
	file = filepath.Join(home(), dirName, newDir, name+".md")
	os.RemoveAll(file)
	return nil
}

func Rename(oldName, newName string) error {
	var topics []Topic
	var id, name string

	readTopics(&topics)
	for k, v := range topics {
		if v.Name == oldName {
			id = v.Id
			topics[k].Name = newName
		}
	}
	if id == "" {
		unknownTopic(name)
		os.Exit(1)
	}

	// Perform the HTTP Request.
	t := &Topic{Name: newName}
	body, _ := json.Marshal(t)
	_, err := getResponse("PUT", "/topics/"+id, bytes.NewReader(body))
	if err != nil {
		return fromError(err)
	}

	// Update the system.
	writeTopics(topics)
	file := filepath.Join(home(), dirName, oldDir)
	os.Rename(filepath.Join(file, oldName), filepath.Join(file, newName))
	file = filepath.Join(home(), dirName, newDir)
	os.Rename(filepath.Join(file, oldName), filepath.Join(file, newName))
	return nil
}
