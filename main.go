package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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
		fmt.Printf("At least one filename must be supplied as an argument\n")
		os.Exit(1)
	}

	var licenses map[License][]string

	for _, filename := range args {
		newLicenses := getLicenses(filename)
		licenses = mergeMaps(licenses, newLicenses)
	}

	prettyPrintLicenses(licenses)
}

func getLicenses(filename string) map[License][]string {
	isDirectory, err := isDir(filename)
	if err != nil {
		fmt.Printf("Error checking if %s is a directory. Error was %v\n", filename, err)
		return nil
	}

	var licenses map[License][]string

	if isDirectory {
		newLicenses := getLicensesFromDir(filename)
		licenses = mergeMaps(licenses, newLicenses)
	} else {
		newLicenses := getLicensesFromFile(filename)
		licenses = mergeMaps(licenses, newLicenses)
	}

	return licenses
}

func getLicensesFromFile(filename string) map[License][]string {
	licenses := make(map[License][]string)

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
				fmt.Printf("Error getting license info %v\n", err)
			}

			licenses[license] = append(licenses[license], repo)
		}
	}

	return licenses
}

func getLicensesFromDir(dir string) map[License][]string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Printf("Error reading the directory %s. Error was %v\n", dir, err)
		return nil
	}

	var licenses map[License][]string

	for _, file := range files {
		newLicenses := getLicenses(filepath.Join(dir, file.Name()))
		licenses = mergeMaps(licenses, newLicenses)
	}

	return licenses
}

func isDir(filename string) (bool, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return false, err
	}

	return fi.IsDir(), nil
}

func getGithubLicense(repo string) (License, error) {
	ownerProj := repo[len("github.com/"):]

	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/repos/%s", githubAPIURL, ownerProj), nil)

	// To increase the rate limit from 60-5000 (as of the time of this comment),
	// GitHub requires an auth token. For a mix of security and ease of use
	// I decided to use an environment variable for the token.
	// To generate a new token go to https://github.com/settings/tokens.
	if authToken := os.Getenv("GITHUB_AUTH_TOKEN"); authToken != "" {
		req.Header.Set("Authorization", "token "+authToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return License{}, err
	}

	defer resp.Body.Close()

	// TODO: Handle error codes (e.g 400, 403, 404, 500, etc)

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

func (license *License) print() {
	fmt.Printf("Name: %s | Key: %s | URL: %s\n", license.Name, license.Key, license.URL)
}

func prettyPrintLicenses(licenses map[License][]string) {
	for license, repos := range licenses {
		license.print()
		printArr("Repos", repos)

		fmt.Println()
	}
}

func printArr(prompt string, arr []string) {
	fmt.Printf("%s: ", prompt)
	arrLen := len(arr)

	if arrLen == 0 {
		fmt.Println()
		return
	}

	for _, a := range arr[:arrLen-1] {
		fmt.Printf("%s, ", a)
	}

	fmt.Printf("%s\n", arr[arrLen-1])
}

func mergeMaps(map1 map[License][]string, map2 map[License][]string) map[License][]string {
	if map1 == nil && map2 == nil {
		return make(map[License][]string)
	}

	if map1 == nil {
		return map2
	}

	if map2 == nil {
		return map1
	}

	merged := make(map[License][]string)

	// copys all the values from map1 into the new map
	for key, val := range map1 {
		merged[key] = val
	}

	for key, val := range map2 {
		if merged[key] == nil {
			merged[key] = val
			continue
		}

		merged[key] = appendUnique(merged[key], map2[key]...)
	}

	return merged
}

func appendUnique(currentStrings []string, newStrings ...string) []string {
	var newArrStrings []string

	for _, currentString := range currentStrings {
		newArrStrings = append(newArrStrings, currentString)
	}

	for _, newString := range newStrings {
		found := false
		for _, currentString := range currentStrings {
			if newString == currentString {
				found = true
				break
			}
		}

		if !found {
			newArrStrings = append(newArrStrings, newString)
		}
	}

	return newArrStrings
}
