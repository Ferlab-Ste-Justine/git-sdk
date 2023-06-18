package git

import (
	"errors"
	"fmt"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
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
Clone the given reference of a given repo in a memory filesystem.
A reference to the generated filesystem as well as the repository is returned.
*/
func MemCloneGitRepo(url string, ref string, depth int, pk *ssh.PublicKeys) (*GitRepository, *MemoryStore, error) {
	storer := memory.NewStorage()
	fs := memfs.New()
	store := MemoryStore{storer, &fs}

	repo, cloneErr := gogit.Clone(storer, fs, &gogit.CloneOptions{
		Auth:              pk,
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