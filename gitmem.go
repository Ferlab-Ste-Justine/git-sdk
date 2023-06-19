package git

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

/*
Container for a memory store. It used to keep a reference to the store and clear it as needed.
The Fs property is a pointer to a billy.Filesystem that can be used to intereract with the filesystem in memory
*/
type MemoryStore struct {
	storage *memory.Storage
	Fs *billy.Filesystem
}

/*
Frees the references to the memory store, allowing the garbage collector to collect it.
*/
func (mem *MemoryStore) Clear() {
	mem.storage = nil
	mem.Fs = nil
}

/*
Returns all the files in the memory filesystem thet fall under a given source path as a map where the keys are the relative path of each file
(relative to the specified source path) and the value is their content. 
You can pass the empty string as a source path if you wish to return the entire content of the memory filesystem.
*/
func (mem *MemoryStore) GetKeyVals(sourcePath string) (map[string]string, error) {
	keys := make(map[string]string)
	err := buildKeySpace(sourcePath, sourcePath, mem, keys)
	return keys, err
}

func stripsourcePath(fPath string, sourcePath string) string {
	if sourcePath == "" {
		return fPath
	}

	if sourcePath[len(sourcePath)-1:] == "/" {
		return strings.TrimPrefix(fPath, sourcePath)
	}

	return strings.TrimPrefix(fPath, sourcePath + "/")
}

func buildKeySpace(fPath string, sourcePath string, store *MemoryStore, keys map[string]string) error {
	files, filesErr := (*store.Fs).ReadDir(fPath)
	if filesErr != nil {
		return filesErr
	}

	for _, file := range files {
		if file.IsDir() {
			err := buildKeySpace(path.Join(fPath, file.Name()), sourcePath, store, keys)
			if err != nil {
				return err
			}
		} else {
			err := func() error {
				fReader, err := (*store.Fs).Open(path.Join(fPath, file.Name()))
				if err != nil {
					return err
				}

				defer fReader.Close()
				
				fContent, fReaderErr := ioutil.ReadAll(fReader)
				if fReaderErr != nil {
					return fReaderErr
				}
				
				keys[path.Join(stripsourcePath(fPath, sourcePath), file.Name())] = string(fContent)

				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

/*
Clone the given reference of a given repo in a memory filesystem.
A reference to the generated filesystem as well as the repository is returned.
*/
func MemCloneGitRepo(url string, ref string, depth int, sshCred *SshCredentials) (*GitRepository, *MemoryStore, error) {
	storer := memory.NewStorage()
	fs := memfs.New()
	store := MemoryStore{storer, &fs}

	repo, cloneErr := gogit.Clone(storer, fs, &gogit.CloneOptions{
		Auth:              sshCred.Keys,
		RemoteName:        "origin",
		URL:               url,
		ReferenceName:     plumbing.NewBranchReferenceName(ref),
		SingleBranch:      true,
		NoCheckout:        false,
		Depth:             depth,
		RecurseSubmodules: gogit.NoRecurseSubmodules,
		Progress:          nil,
		Tags:              gogit.NoTags,
	})
	if cloneErr != nil {
		return &GitRepository{repo}, &store, errors.New(fmt.Sprintf("Error cloning repo in memory: %s", cloneErr.Error()))
	}

	fmt.Println(fmt.Sprintf("Cloned branch \"%s\" of repo \"%s\"", ref, url))
	return &GitRepository{repo}, &store, nil
}