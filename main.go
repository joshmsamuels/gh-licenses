package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

const githubAPIURL = "https://api.github.com"

type License struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RepoResponse struct {
	RepoLicense License `json:"license"`
}

func main() {
	// Gets all the arguments excuding the program name
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Printf("At least one filename must be supplied as an argument")
		os.Exit(1)
	}

	licenses := make(map[string]License)

	for _, filename := range args {
		newLicenses := getLicensesForFile(filename)

		for key, val := range newLicenses {
			licenses[key] = val
		}
	}

	prettyPrintLicenses(licenses)
}

func getGithubLicense(repo string) (License, error) {
	ownerProj := repo[len("github.com/"):]
	resp, err := http.Get(fmt.Sprintf("%s/repos/%s", githubAPIURL, ownerProj))
	if err != nil {
		return License{}, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)

	var repoResp RepoResponse
	err = json.Unmarshal(data, &repoResp)
	if err != nil {
		return License{}, err
	}

	return repoResp.RepoLicense, nil
}

func stripNewline(text string) string {
	if len(text) > 0 && text[len(text)-1] == '\n' {
		return text[:len(text)-1]
	}

	return text
}

func getGithubRepos(text string) ([]string, error) {
	regex, err := regexp.Compile(`github\.com[^\s]+`)
	if err != nil {
		return nil, err
	}

	return regex.FindAllString(text, -1), nil
}

func getLicensesForFile(filename string) map[string]License {
	licenses := make(map[string]License)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file %s. Error: %v\n", "go.mod", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := stripNewline(scanner.Text())

		repos, err := getGithubRepos(line)
		if err != nil {
			fmt.Printf("Error getting github repos %v\n", err)
		}

		for _, repo := range repos {
			license, err := getGithubLicense(repo)
			if err != nil {
				fmt.Printf("Error getting license info %v", err)
			}

			licenses[license.Key] = license
		}
	}

	return licenses
}

func (license *License) print() {
	fmt.Printf("Name: %s\n\tKey: %s\n\tURL: %s\n", license.Name, license.Key, license.URL)
}

func prettyPrintLicenses(licenses map[string]License) {
	for _, license := range licenses {
		license.print()
	}
}
