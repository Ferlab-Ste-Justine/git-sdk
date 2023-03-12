# About

Git sdk to support the gitops workflows of our internal tooling.

It uses the go-git library, sacrificing on overall flexibity to simplify our specific use-cases of that library as much as possible.

The functionality of this sdk may change in a backward-incompatible way as our needs evolve.

# Features

Currently, the sdk focuses on the following use-cases:
- Cloning and/or pulling on a repo depending on the current state of the target repository
- Verifying that the top commit of a repository was signed by a key from a trusted list
- Adding and commiting on a group of files if any are changed
- Pushing to the origin of a repository returned by a function argument with retries if there are conflicts