package main

import (
	"encoding/json"
	"os"
	"os/exec"
)

func executeRead(argumentJSON string) (string, error) {
	var args struct {
		FilePath string `json:"file_path"`
	}
	err := json.Unmarshal([]byte(argumentJSON), &args)
	if err != nil {
		return "", err
	}
	// Read the file contents
	fileContents, err := os.ReadFile(args.FilePath)
	if err != nil {
		return "", err
	}

	return string(fileContents), err
}

func executeWrite(argumentJSON string) error {
	var args struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	err := json.Unmarshal([]byte(argumentJSON), &args)
	if err != nil {
		return err
	}
	// Write the file
	err = os.WriteFile(args.FilePath, []byte(args.Content), 0666)
	return err
}

func executeBash(argumentJSON string) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	err := json.Unmarshal([]byte(argumentJSON), &args)
	if err != nil {
		return "", err
	}

	// Do bash
	cmd := exec.Command("bash", "-c", args.Command)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), err
}
