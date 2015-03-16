package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	githuboauth2 "golang.org/x/oauth2/github"
	"gopkg.in/alecthomas/kingpin.v1"
)

func init() {
}

func deleteGist(ids []string) error {
	for _, id := range ids {
		if _, err := client.Gists.Delete(id); err != nil {
			return err
		}
	}

	return nil
}

func createGist(files []string) error {
	gist := &github.Gist{Files: map[github.GistFilename]github.GistFile{}}
	gist.Public = createCmdPublic
	gist.Description = createCmdDescription

	for _, f := range files {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		s := string(b)
		gf := github.GistFile{
			Content: &s,
		}
		gist.Files[github.GistFilename(path.Base(f))] = gf
	}

	if len(files) == 0 {
		edit(gist)
	}

	gist, _, err := client.Gists.Create(gist)
	if err != nil {
	}

	fmt.Printf("%s\n", *gist.ID)

	return nil
}

func fork() error {
	return errors.New("Not implemented yet.")
}

func uploadGist(files []string) error {
	return nil
}

func downloadGist(id string, files []string) error {
	gist, _, err := client.Gists.Get(id)
	if err != nil {
		return err
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	if *downloadCmdDest != "" {
		dir = *downloadCmdDest
	}

	for filename, gf := range gist.Files {
		if len(files) > 0 {
			if !stringInSlice(string(filename), files) {
				continue
			}
		}

		f, err := os.Create(path.Join(dir, string(filename)))
		if err != nil {
			return err
		}

		defer f.Close()

		fmt.Printf("%s\n", f.Name())
		bf := bufio.NewWriter(f)
		bf.WriteString(*gf.Content)
		bf.Flush()
	}

	return nil
}

func infoGist(id string) error {
	gist, _, err := client.Gists.Get(id)
	if err != nil {
		return err
	}

	t := template.New("")

	if t, err = t.Parse(`ID:	     {{.ID}}
Owner:	     {{.Owner.Name}} [{{.Owner.Login}}]
Description: {{.Description}}
Availability:{{ if .Public}}Public{{else}}Private{{end}}
Url:	     {{.HTMLURL }}
Created At:  {{.CreatedAt}}
Updated At:  {{.UpdatedAt}}

Files:
{{ range $filename, $file := .Files }}{{$filename}}
{{end}}
`); err != nil {
		return err
	}

	if err = t.Execute(os.Stdout, gist); err != nil {
		return err
	}

	return nil
}

func edit(gist *github.Gist) error {
	dir, err := ioutil.TempDir("", "gister-")
	if err != nil {
		return err
	}

	DebugPrintln("Using temporary folder %s.", dir)

	defer func() {
		DebugPrintln("Cleaning up folder %s.", dir)

		if err := os.RemoveAll(dir); err != nil {

		}
	}()

	args := []string{}

	for filename, gf := range gist.Files {
		f, err := os.Create(path.Join(dir, string(filename)))
		if err != nil {
			return err
		}

		defer f.Close()

		bf := bufio.NewWriter(f)
		if _, err = bf.WriteString(*gf.Content); err != nil {
			return err
		}

		bf.Flush()

		DebugPrintln("Created local copy of gistfile %s.", f.Name())
		args = append(args, f.Name())
	}

	currentPath, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.Chdir(dir); err != nil {
		return err
	}

	DebugPrintln("Starting editor %s with args %#v.", *editor, args)

	cmd := exec.Command(*editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return err
	}

	DebugPrintln("Editor finished.")

	gist.Files = map[github.GistFilename]github.GistFile{}

	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		b, err := ioutil.ReadFile(p)
		s := string(b)
		gf := github.GistFile{Content: &s}

		filename := github.GistFilename(path.Base(p))
		gist.Files[filename] = gf
		return nil
	})

	os.Chdir(currentPath)

	return nil
}

func editGist(id string) error {
	DebugPrintln("Retrieving gist %s", id)

	gist, _, err := client.Gists.Get(id)
	if err != nil {
		return err
	}

	edit(gist)

	DebugPrintln("Updating gist %v.", *gist.ID)

	gist, _, err = client.Gists.Edit(*gist.ID, gist)
	if err != nil {
		return err
	}

	return nil
}

func nz(val *string) string {
	if val == nil {
		return ""
	} else {
		return *val
	}
}

func listGists(user string) error {
	gists, _, err := client.Gists.List(user, &github.GistListOptions{})
	if err != nil {
		return err
	}

	for _, gist := range gists {
		fmt.Printf("%s/%-21s%s\n", *gist.Owner.Login, *gist.ID, nz(gist.Description))
	}

	return nil
}

func searchGists(user string, keyword string) error {
	gists, _, err := client.Gists.List(user, &github.GistListOptions{})
	if err != nil {
		return err
	}

	for _, gist := range gists {
		if strings.Index(nz(gist.Description), keyword) == -1 {
			continue
		}

		fmt.Printf("%s/%-21s%s\n", *gist.Owner.Login, *gist.ID, nz(gist.Description))
	}

	return nil
}

var client *github.Client

var (
	debug                = kingpin.Flag("debug", "enable debug mode").Default("false").Bool()
	editor               = kingpin.Flag("editor", "editor to use").OverrideDefaultFromEnvar("EDITOR").Required().String()
	accessToken          = kingpin.Flag("token", "github token").OverrideDefaultFromEnvar("GITHUB_TOKEN").Required().String()
	listCmd              = kingpin.Command("list", "list gists")
	listCmdUser          = listCmd.Flag("user", "Owner").String()
	searchCmd            = kingpin.Command("search", "search gists")
	searchCmdUser        = searchCmd.Flag("user", "Owner").String()
	searchCmdKeyword     = searchCmd.Arg("keyword", "keyword to search").Required().String()
	downloadCmd          = kingpin.Command("download", "download gist")
	downloadCmdId        = downloadCmd.Flag("gist", "gist id").Required().String()
	downloadCmdDest      = downloadCmd.Flag("dest", "destination directory").String()
	downloadCmdFiles     = downloadCmd.Arg("files", "gist files").Strings()
	createCmd            = kingpin.Command("create", "create new gist with specified files")
	createCmdFiles       = createCmd.Arg("files", "").Strings()
	createCmdDescription = kingpin.Flag("description", "description").Default("").String()
	createCmdPublic      = kingpin.Flag("public", "public gist").Default("false").Bool()
	infoCmd              = kingpin.Command("info", "show info gist")
	infoCmdId            = infoCmd.Flag("gist", "gist id").Required().String()
	editCmd              = kingpin.Command("edit", "edit gist using editor")
	editCmdId            = editCmd.Flag("gist", "gist id").Required().String()
	deleteCmd            = kingpin.Command("delete", "delete specified gist")
	deleteCmdIds         = deleteCmd.Flag("gist", "gist id").Short('g').Required().Strings()
)

func NewGithubClient() *github.Client {
	config := oauth2.Config{Endpoint: githuboauth2.Endpoint, Scopes: []string{"gist"}}
	token := &oauth2.Token{AccessToken: *accessToken, TokenType: "bearer"}
	client := github.NewClient(config.Client(oauth2.NoContext, token))
	return client
}

func main() {
	cmd := kingpin.Parse()

	client = NewGithubClient()

	var err error
	switch cmd {
	case "search":
		err = searchGists(*searchCmdUser, *searchCmdKeyword)
	case "list":
		err = listGists(*listCmdUser)
	case "delete":
		err = deleteGist(*deleteCmdIds)
	case "download":
		files := []string{}
		if downloadCmdFiles != nil {
			files = append(files, *downloadCmdFiles...)
		}
		err = downloadGist(*downloadCmdId, files)
	case "create":
		err = createGist(*createCmdFiles)
	case "info":
		err = infoGist(*infoCmdId)
	case "edit":
		err = editGist(*editCmdId)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occured: \n%s\n", err)
		os.Exit(1)
	}
}
