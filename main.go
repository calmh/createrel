package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var (
	errNotFound = errors.New("not found")
)

type release struct {
	ID         int
	Tag        string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	PreRelease bool   `json:"prerelease"`
}

func main() {
	token := flag.String("t", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	verbose := flag.Bool("v", false, "Verbose")
	flag.Parse()

	log.SetFlags(0)

	if *token == "" {
		log.Fatalln("Please export GITHUB_TOKEN=\"<your token here>\" or use the -t flag")
	}

	if flag.NArg() != 2 {
		log.Fatalln("Usage: createrel [options] <user/repo> <tag>")
	}

	repo := flag.Arg(0)
	tag := flag.Arg(1)
	pre := strings.Contains(tag, "-")

	msg, err := getTagMessage(tag)
	if err != nil {
		log.Fatalln("Getting tag message:", err)
	}

	if *verbose {
		if pre {
			fmt.Println("*** Pre-release ***")
		}
		fmt.Println(tag)
		fmt.Println()
		fmt.Println(msg)
	}

	rel := release{
		Tag:        tag,
		Name:       tag,
		Body:       msg,
		Draft:      false,
		PreRelease: pre,
	}

	if err := createRelease(*token, repo, rel); err != nil {
		if strings.Contains(err.Error(), "already_exists") {
			// no error
			if *verbose {
				log.Println("*** Release already exists")
			}
			return
		}
		log.Fatalln("Failed to create release:", err)
	}
}

func getTagMessage(tag string) (string, error) {
	cmd := exec.Command("git", "tag", "-n99", "-l", tag)
	bs, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	br := bufio.NewScanner(bytes.NewReader(bs))
	br.Scan() // tag name and subject
	start := false
	msg := new(bytes.Buffer)
	for br.Scan() {
		line := br.Bytes()
		if bytes.HasPrefix(line, []byte("    ")) {
			line = line[4:]
		}
		if !start && len(line) == 0 {
			continue
		}
		start = true
		fmt.Fprintf(msg, "%s\n", line)
	}

	return msg.String(), nil
}

func createRelease(token, repo string, rel release) error {
	data, err := json.Marshal(rel)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/repos/%s/releases", repo), bytes.NewReader(data))
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		bs, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, bs)
	}
	return nil
}
