package github

import "fmt"

func GetAuthenticatedClient() (*Client, error) {
	token, err := GetGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("github authentication: %w", err)
	}
	return NewClient(token), nil
}
