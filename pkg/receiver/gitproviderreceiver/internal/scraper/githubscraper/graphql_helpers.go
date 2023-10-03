package githubscraper

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/Khan/genqlient/graphql"
)

// TODO: getRepoData and getBranchData should be a singular function since we can
// get all the data at once & pagenate through it. Doing so will reduce subsequent
// calls to GraphQL and allow for more performant metrics building.
func getRepoData(
	ctx context.Context,
	client graphql.Client,
	searchQuery string,
	ownertype string,
	// here we use a pointer to a string so that graphql will receive null if the
	// value is not set since the after: $repoCursor is optional to graphql
	repoCursor *string,
	// since we're using a interface{} here we do type checking when data
	// is returned to the calling function
) (interface{}, error) {
	data, err := getRepoDataBySearch(ctx, client, searchQuery, repoCursor)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func getNumPages(p float64, n float64) int {
	numPages := math.Ceil(n / p)

	return int(numPages)
}

func add[T ~int | ~float64](a, b T) T {
	return a + b
}

func sub[T ~int | ~float64](a, b T) T {
	return a - b
}

// breaks up a slice into chunks of size chunkSize or as close to it as possible
func chunkSlice[T any](slice []T, chunkSize int) [][]T {
	var chunks [][]T

	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// Ensure that the type of owner is user or organization
func checkOwnerTypeValid(ownertype string) (bool, error) {
	if ownertype == "org" || ownertype == "user" {
		return true, nil
	}
	return false, errors.New("ownertype must be either org or user")
}

// Check to ensure that the login user (org name or user id) exists or
// can be logged into.
func (ghs *githubScraper) checkOwnerExists(ctx context.Context, client graphql.Client, owner string) (exists bool, ownerType string, err error) {

	loginResp, err := checkLogin(ctx, client, ghs.cfg.GitHubOrg)

	exists = false
	ownerType = ""

	// These types are used later to generate the default string for the search query
	// and thus must match the convention for user: and org: searches in GitHub
	if loginResp.User.Login == owner {
		exists = true
		ownerType = "user"
	} else if loginResp.Organization.Login == owner {
		exists = true
		ownerType = "org"
	}

	if exists {
		err = nil
	}

	return
}

// Returns the default search query string based on input of owner type
// and GitHubOrg name with a default of archived:false to ignore archived repos
func genDefaultSearchQuery(ownertype string, ghorg string) string {
	return fmt.Sprintf("%s:%s archived:false", ownertype, ghorg)
}
