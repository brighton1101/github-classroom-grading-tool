package main

import "fmt"
import "runtime"
import "os"
import "os/exec"
import "io"
import "errors"
import "context"
import "strings"
import "encoding/csv"
import "log"
import "bufio"
import "flag"

import "github.com/google/go-github/v32/github"
import "golang.org/x/oauth2"
import "github.com/joho/godotenv"

/**
 * Open url in web browser
 * https://gist.github.com/hyg/9c4afcd91fe24316cbf0
 */
func StartBrowser(url string) error {
    var err error
    switch runtime.GOOS {
    case "linux":
        err = exec.Command("xdg-open", url).Start()
    case "windows":
        err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "darwin":
        err = exec.Command("open", url).Start()
    }
    return err
}

/**
 * Fetches GITHUB_AUTH_TOKEN string from env
 */
func GithubTokenFromEnv() (string, error) {
    return fromEnv("GITHUB_AUTH_TOKEN")
}

/**
 * Fetches GITHUB_CLASSROOM_ORG string from env
 */
func GithubOrgFromEnv() (string, error) {
    return fromEnv("GITHUB_CLASSROOM_ORG")
}

/**
 * Gets GRADING_LOGGING_DEST string from env
 */
func LoggingDestFromEnv() (string, error) {
    return fromEnv("GRADING_LOGGING_DEST")
}

/**
 * Accessor function to get env var from system.
 */
func fromEnv(name string) (string, error) {
    res, prs := os.LookupEnv(name)
    if !prs {
        err := fmt.Sprintf("Problem getting %s from environment. Make sure it's set.", name)
        return "", errors.New(err)
    }
    return res, nil
}

/**
 * Gets Authorized Github HTTP Client
 */
func GithubClient(ctx context.Context, token string) *github.Client {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(ctx, ts)
    client := github.NewClient(tc)
    return client
}

/**
 * Fetches all repos in a given org.
 */
func OrgRepos(ctx context.Context, client *github.Client, org string) ([]*github.Repository, error) {

    // slice to contain all repos, and err object
    var allRepos []*github.Repository

    // Set options to load 100 repos at a time
    // This is bc the # of repos in a given classroom org
    // tends to be in the scale of 100s (at least), as there
    // is usually one repo per assignment
    opt := &github.RepositoryListByOrgOptions{
        ListOptions: github.ListOptions{PerPage: 100},
    }
    for {
        repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
        if err != nil {
            return allRepos, err
        }
        allRepos = append(allRepos, repos...)
        if resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }
    return allRepos, nil
}

func RepoNameByPrefixAndUser(pref, user string) string {
    return fmt.Sprintf("%s%s", pref, user)
}

/**
 * Github classroom repos are created with the following naming standard:
 * `{some assignment prefix}{username}`, with a good example being
 * `assignment-3-brighton1101`
 *
 * This method operates under that assumption: there is some assignment
 * prefix that needs to be combined with the users' username. The repo
 * then lives under the org
 */
func RepoByPrefixAndUser(ctx context.Context, client *github.Client, org, pref, user string) (*github.Repository, error) {
    rname := RepoNameByPrefixAndUser(pref, user)
    repo, _, err := client.Repositories.Get(ctx, org, rname)
    return repo, err
}

// Wrapper struct for posting issues
type PostIssueOptions struct {
    OrgName string
    RepoName string
    Header string
    Body string
}

/**
 * Posts an issue to a Github repo.
 */
func PostIssue(ctx context.Context, client *github.Client, opts *PostIssueOptions) error {
    issueopt := &github.IssueRequest{
        Title: &opts.Header,
        Body: &opts.Body,
    }
    _, _, err := client.Issues.Create(ctx, opts.OrgName, opts.RepoName, issueopt)
    return err
}

func RepoUrl(repo *github.Repository) string {
    return *repo.
}

/**
 * Assignment repos for specific assignments can be identified by a prefix in the
 * repository name. Given the desired prefix, return all repos with names
 * that contain that prefix
 */
func FilterReposByPref(repos []*github.Repository, pref string) []*github.Repository {
    var prefRepos []*github.Repository
    for _, repo := range(repos) {
        if strings.Contains(*repo.Name, pref) {
            prefRepos = append(prefRepos, repo)
        }
    }
    return prefRepos
}

/**
 * Reads a username map from a given csv file. Operates under the assumption that
 * the 0 index will contain the name, and the 1 index will contain the username. Returns
 * two maps, for either direction.
 */
func ReadUsernameMap(path string) (map[string]string, map[string]string, error) {
    f, err := os.Open(path)
    if err != nil {
        emsg := fmt.Sprintf("Could not load file from given path: %s", path)
        return nil, nil, errors.New(emsg)
    }
    defer f.Close()
    reader := csv.NewReader(f)
    name_username := map[string]string{}
    username_name := map[string]string{}
    for {
        row, err := reader.Read()
        if err == io.EOF {
            return name_username, username_name, nil
        } else if err != nil {
            return nil, nil, err
        }
        name := row[0]
        username := row[1]
        name_username[name] = username
        username_name[username] = name
    }
}

/**
 * Gather input from user, terminated by the endin char
 */
func GatherInput() (string, error) {
    reader := bufio.NewReader(os.Stdin)
    in, err := reader.ReadString('\n')
    if err != nil {
        return "", err
    }
    // This is ugly. There's probably a better way to do this,
    // but eh it's the way I knew how to do it haha
    in = strings.Replace(in, "\n", "", -1)
    return in, nil
}

func main() {

    // First we gather two pieces of information from the user
    // 1. The assignment prefix (ie, 'assignment-2-') AND
    // 2. EITHER the students' full name, as saved in the csv
    //    that maps names to usernames, or the students' github
    //    username itself
    prefix := flag.String("p", "",
        "The assignment prefix to usernames for github classroom ie 'assignment-2-'")
    name := flag.String("n", "",
        "The students' full name to use")
    username := flag.String("un", "",
        "The students' username to use")
    flag.Parse()

    // Only allow users one option or the other
    if name != "" && username != "" {
        fmt.Println("Using -n and -un together is not allowed. Please only specify one.")
        return
    }

    // Load environment vars from .env file
    err := godotenv.Load()
    if err != nil {
        fmt.Println(derr)
        return
    }

    // Set up logging based on GRADING_LOGGING_DEST var
    logdir, err := LoggingDestFromEnv()
    if err != nil {
        fmt.Println(err)
        return
    }
    f, err := os.OpenFile(fmt.Sprintf("%s/%s.log", logdir, asnmtprefix), os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
        fmt.Println(err)
        return
    }
    defer f.Close()
    log.SetOutput(f)

    // Read mappings between usernames and passwords
    name_username, username_name, err := ReadUsernameMap("test.csv")
    if err != nil {
        fmt.Println(err)
        return
    }

    // If name is not defined, get the name from username
    if name == "" {
        name, prs := username_name[username]
    }
    if !prs {
        fmt.Println("Username not defined in csv file")
        return
    }

    // If username is not defined, get the username from name
    if username == "" {
        username, prs := name_username[name]
    }
    if !prs {
        fmt.Println("Name not defined in csv file")
        return
    }

    // Get orgname from env var
    orgname, oerr  := GithubOrgFromEnv()
    if oerr != nil {
        fmt.Println(oerr)
        return
    }

    // Get Github token from env var
    token, terr := GithubTokenFromEnv()
    if terr != nil {
        fmt.Println(terr)
        return
    }

    // Get context and github client
    // I think? these should be treated as singletons or
    // at least singleton-like
    ctx := context.Background()
    client := GithubClient(ctx, token)

    // Get repo for user based on org, prefix, and username
    repo, err := RepoByPrefixAndUser(ctx, client, orgname, prefix, username)
    if err != nil {
        fmt.Println(err)
        return
    }

    // Get repo url from repo
    url := RepoUrl(repo)



    in, err := GatherInput()
    if in != "lolol" {
        fmt.Println(in)
        return
    }

    

    
    log.Println("test")

    fmt.Println(token)

    

    // Get repos by org
    // Github paginates to 30 results
    repos, err := OrgRepos(ctx, client, orgname)
    if err != nil {
        fmt.Println(err)
        return
    }
    nrepos := FilterReposByPref(repos, "assignment-3-")
    fmt.Println(len(nrepos))

    /*
    // Test post issue
    issueopts := &PostIssueOptions{
        OrgName: "brighton1101",
        RepoName: "hacksc2020",
        Header: "maybe delete",
        Body: "justatest",
    }
    PostIssue(ctx, client, issueopts) */

    
    fmt.Println(m1)
}
