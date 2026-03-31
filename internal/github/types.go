package github

type ContentResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}

type ContentRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Sha     string `json:"sha,omitempty"`
	Branch  string `json:"branch,omitempty"`
}

type DirectoryItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url,omitempty"`
}

type DeleteRequest struct {
	Message string `json:"message"`
	Sha     string `json:"sha"`
	Branch  string `json:"branch,omitempty"`
}

type UserEntry struct {
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
	AddedAt   string `json:"added_at"`
}

type SSHKeyInfo struct {
	Username string
	KeyName  string
	Size     int
}
