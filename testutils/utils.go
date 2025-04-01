package testutils

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"net/http"
	"path"
	"text/template"
	"time"

	"code.gitea.io/sdk/gitea"
)

var (
	//go:embed app.ini.tpl
	configTemplate string
)

func removeIfExists(path string) (error) {
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		return nil
	}

	return os.RemoveAll(path)
}

type TeardownTestGitea func() error

type GiteaOpts struct {
	Workdir  string
	BindIp   string
	BindPort int64
	Password string
	Email    string
	SshPub   string
	Repos    []string
}

type GiteaTemplate struct {
	Options GiteaOpts
	User    string
}

func (opt *GiteaOpts) GetUrl() string {
	return fmt.Sprintf("http://%s:%d", opt.BindIp, opt.BindPort)
}

func WaitOnServer(url string) error {
	var err error
	var res *http.Response
	
	for i := 0; i < 120; i++ {
		cli := http.Client{}
		res, err = cli.Get(fmt.Sprintf("%s/api/v1/version", url))
		if err == nil {
			res.Body.Close()
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
	
	return err
}

type TestGiteaInfo struct {
	User string
	RepoUrls []string
	KnownHostsFile string
}

/*
Launch a gitea server, running on http://127.0.0.1:3000
A teardown method is returned to shut it down.
It is assumed that a recent gitea binary is located in the binary path.
*/
func LaunchTestGitea(opts GiteaOpts) (TeardownTestGitea, TestGiteaInfo, error) {
	err := removeIfExists(opts.Workdir)
	if err != nil {
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	err = os.MkdirAll(path.Join(opts.Workdir, "workdir"), 0770)
	if err != nil {
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	err = os.MkdirAll(path.Join(opts.Workdir, "conf"), 0770)
	if err != nil {
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	tmpl, tErr := template.New("template").Parse(configTemplate)
	if tErr != nil {
		return func() error {return nil}, TestGiteaInfo{}, tErr
	}

	currUser, currUserErr := user.Current()
	if currUserErr != nil {
		return func() error {return nil}, TestGiteaInfo{}, currUserErr
	}

	var tmplBuff bytes.Buffer
	exErr := tmpl.Execute(&tmplBuff, &GiteaTemplate{opts, currUser.Username})
	if exErr != nil {
		return func() error {return nil}, TestGiteaInfo{}, exErr
	}

	confFilePath := path.Join(opts.Workdir, "conf", "app.ini")
	
	err = os.WriteFile(confFilePath, tmplBuff.Bytes(), 0770)
	if err != nil {
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	cmd := exec.Command(
		"gitea", "--config", confFilePath,
		"--work-path", path.Join(opts.Workdir, "workdir"),
	)

	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr

	teardown := func() error {
		err := cmd.Process.Kill()
		if err == nil {
			cmd.Process.Wait()
		} else {
			return err
		}

		return removeIfExists(opts.Workdir)
	}

	err = cmd.Start()
	if err != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	err = WaitOnServer(opts.GetUrl())
	if err != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	//gitea admin user create --username magnitus --password localgiteapassword --email eric_vallee@webificservices.com -c /home/magnitus/Projects/test/gitea/conf/app.ini
	createAdminCmd := exec.Command(
		"gitea", "admin", "user", "create",
		"--username", currUser.Username,
		"--password", opts.Password,
		"--email", opts.Email,
		"-c", confFilePath,
		"--admin",
	)

	//createAdminCmd.Stdout = os.Stdout
	//createAdminCmd.Stderr = os.Stderr

	err = createAdminCmd.Start()
	if err != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	err = createAdminCmd.Wait()
	if err != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	cli, cliErr := gitea.NewClient(opts.GetUrl(), gitea.SetBasicAuth(currUser.Username, opts.Password))
	if cliErr != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, cliErr
	}

	_, _, err = cli.AdminCreateUserPublicKey(currUser.Username, gitea.CreateKeyOption{
		Title: "Test Key",
		Key: opts.SshPub,
		ReadOnly: false,
	})
	if err != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	sshUrls := []string{}
	for _, repo := range opts.Repos {
		repo, _, repoErr := cli.CreateRepo(gitea.CreateRepoOption{
			Name: repo,
			Description: repo,
			Private: true,
			AutoInit: true,
			DefaultBranch: "main",
		})
		if repoErr != nil {
			teardown()
			return func() error {return nil}, TestGiteaInfo{}, repoErr
		}

		sshUrls = append(sshUrls, repo.SSHURL)
	}

	sshKeyPub, sshKeyPubErr := os.ReadFile(path.Join(opts.Workdir, "data", "ssh", "gitea.rsa.pub"))
	if sshKeyPubErr != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, sshKeyPubErr
	}

	err = os.MkdirAll(path.Join(opts.Workdir, "ssh"), 0770)
	if err != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, err
	}

	knownHostsPath := path.Join(opts.Workdir, "ssh", "known_hosts")
	knownHostsErr := os.WriteFile(knownHostsPath, []byte(fmt.Sprintf("[%s]:2222 %s", opts.BindIp, sshKeyPub)), 0770)
	if knownHostsErr != nil {
		teardown()
		return func() error {return nil}, TestGiteaInfo{}, knownHostsErr
	}

	return teardown, TestGiteaInfo{
		User: currUser.Username, 
		RepoUrls: sshUrls, 
		KnownHostsFile: knownHostsPath,
	}, nil
}


func SetupDefaultTestEnvironment() (TeardownTestGitea, TestGiteaInfo, string, error) {
	workDir, workDirErr := os.Getwd()
	if workDirErr != nil {
		return func() error {return nil}, TestGiteaInfo{}, "", errors.New(fmt.Sprintf("Error occured launching getting current working directory: %s", workDirErr.Error()))
	}

    sshPub, sshPubErr := os.ReadFile(path.Join(workDir, "test", "keys", "id_rsa.pub"))
	if sshPubErr != nil {
		return func() error {return nil}, TestGiteaInfo{}, "", errors.New(fmt.Sprintf("Error occured reading user's public ssh key: %s", sshPubErr.Error()))
	}

	testdir := path.Join(workDir, "testdir")
	tearDown, giteaInfo, launchErr := LaunchTestGitea(GiteaOpts{
		Workdir: testdir,
		BindIp: "127.0.0.1",
		BindPort: 3000,
		Password: "test",
		Email: "test@test.test",
		SshPub: string(sshPub),
		Repos: []string{"test"},
	})

	if launchErr != nil {
		return func() error {return nil}, TestGiteaInfo{}, "", errors.New(fmt.Sprintf("Error occured launching test gitea server: %s", launchErr.Error()))
	}

	teardownTemp := func() {
		err := tearDown()
		if err != nil {
			fmt.Println(errors.New(fmt.Sprintf("Errors occured tearing down gitea cluster: %s", err.Error())))
		}
	}

	testReposPath := path.Join(workDir, "test_repos")
	testReposErr := os.MkdirAll(testReposPath, 0770)
	if testReposErr != nil {
		teardownTemp()
		return func() error {return nil}, TestGiteaInfo{}, "", errors.New(fmt.Sprintf("Error occured creating test repos directory: %s", testReposErr.Error()))
	}

	return func() error {
		err := tearDown()
		if err != nil {
			return err
		}

		err = os.RemoveAll(testReposPath)
		if err != nil {
			return err
		}

		return nil
	}, giteaInfo, testReposPath, nil
}