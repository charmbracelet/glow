package client

import (
	"errors"
	"time"

	"github.com/charmbracelet/charm/kv"
)

type Client struct {
	kv *kv.KV
}

func NewClient() (*Client, error) {
	kv, err := kv.OpenWithDefaults("charm.sh.glow", "./data")
	if err != nil {
		return nil, err
	}
	return &Client{kv: kv}, nil
}

// ErrorPageOutOfBounds is an error for an invalid page number.
var ErrorPageOutOfBounds = errors.New("page must be a value of 1 or greater")

// MarkdownsByCreatedAtDesc sorts markdown documents by date in descending
// order. It implements sort.Interface for []Markdown based on the CreatedAt
// field.
type MarkdownsByCreatedAtDesc []*Markdown

// Sort implementation for MarkdownByCreatedAt.
func (m MarkdownsByCreatedAtDesc) Len() int           { return len(m) }
func (m MarkdownsByCreatedAtDesc) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m MarkdownsByCreatedAtDesc) Less(i, j int) bool { return m[i].CreatedAt.After(m[j].CreatedAt) }

// Markdown is the struct that contains the markdown and note data. If
// EncryptKeyID is not blank, the content should be assumed to be encrypted.
// Once decrypted, that field will be blanked.
type Markdown struct {
	ID           int       `json:"id"`
	EncryptKeyID string    `json:"encrypt_key_id"`
	Note         string    `json:"note"`
	Body         string    `json:"body,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetNews returns the Glow paginated news results.
func (cc *Client) GetNews(page int) ([]*Markdown, error) {
	if page < 1 {
		return nil, ErrorPageOutOfBounds
	}
	var news []*Markdown
	// err := cc.makeAPIRequest("GET", fmt.Sprintf("news?page=%d", page), nil, &news)
	// if err != nil {
	// 	return nil, err
	// }
	return news, nil
}

// GetNewsMarkdown returns the Markdown struct for the given news markdown ID.
func (cc *Client) GetNewsMarkdown(markdownID int) (*Markdown, error) {
	var md Markdown
	// err := cc.makeAPIRequest("GET", fmt.Sprintf("news/%d", markdownID), nil, &md)
	// if err != nil {
	// 	return nil, err
	// }
	return &md, nil
}

// GetStash returns the paginated user stash for the authenticated Charm user.
func (cc *Client) GetStash(page int) ([]*Markdown, error) {
	if page < 1 {
		return nil, ErrorPageOutOfBounds
	}

	var stash []*Markdown
	// auth, err := cc.Auth()
	// if err != nil {
	// 	return nil, err
	// }

	// err = cc.makeAPIRequest("GET", fmt.Sprintf("%s/stash?page=%d", auth.CharmID, page), nil, &stash)
	// if err != nil {
	// 	return nil, err
	// }
	// for i, md := range stash {
	// 	dm, err := cc.decryptMarkdown(md)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	stash[i] = dm
	// }

	return stash, nil
}

// GetStashMarkdown returns the Markdown struct for the given stash markdown ID.
func (cc *Client) GetStashMarkdown(markdownID int) (*Markdown, error) {
	var md Markdown
	// auth, err := cc.Auth()
	// if err != nil {
	// 	return nil, err
	// }

	// err = cc.makeAPIRequest("GET", fmt.Sprintf("%s/stash/%d", auth.CharmID, markdownID), nil, &md)
	// if err != nil {
	// 	return nil, err
	// }
	// mdDec, err := cc.decryptMarkdown(&md)
	// if err != nil {
	// 	return nil, err
	// }

	// return mdDec, nil
	return &md, nil
}

// StashMarkdown encrypts and stashes a new markdown file with note.
func (cc *Client) StashMarkdown(note string, body string) (*Markdown, error) {
	// auth, err := cc.Auth()
	// if err != nil {
	// 	return nil, err
	// }
	// md := &Markdown{Note: note, Body: body}
	// md, err = cc.encryptMarkdown(md)
	// if err != nil {
	// 	return nil, err
	// }
	// var mde Markdown
	// err = cc.makeAPIRequest("POST", fmt.Sprintf("%s/stash", auth.CharmID), md, &mde)
	// if err != nil {
	// 	return nil, err
	// }
	// newMd, err := cc.decryptMarkdown(&mde)
	// if err != nil {
	// 	return nil, err
	// }
	// return newMd, nil
	var newMd *Markdown
	return newMd, nil
}

// DeleteMarkdown deletes the stash markdown for the given ID.
func (cc *Client) DeleteMarkdown(markdownID int) error {
	// auth, err := cc.Auth()
	// if err != nil {
	// 	return err
	// }

	// return cc.makeAPIRequest("DELETE", fmt.Sprintf("%s/stash/%d", auth.CharmID, markdownID), nil, nil)
	return nil
}

// SetMarkdownNote updates the note for a given stash markdown ID.
func (cc *Client) SetMarkdownNote(markdownID int, note string) error {
	// auth, err := cc.Auth()
	// if err != nil {
	// 	return err
	// }

	// md, err := cc.GetStashMarkdown(markdownID)
	// if err != nil {
	// 	return err
	// }
	// md.Note = note
	// md, err = cc.encryptMarkdown(md)
	// if err != nil {
	// 	return err
	// }

	// return cc.makeAPIRequest("PUT", fmt.Sprintf("%s/stash/%d", auth.CharmID, markdownID), md, nil)
	return nil
}
