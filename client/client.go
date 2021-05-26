package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/charm/kv"
	"github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
)

// Client provides the Glow interface to the Charm Cloud
type Client struct {
	kv *kv.KV
}

// NewClient creates a new Client with the default settings
func NewClient() (*Client, error) {
	kv, err := kv.OpenWithDefaults("charm.sh.glow", "./data")
	if err != nil {
		return nil, err
	}
	err = kv.Sync()
	if err != nil {
		return nil, err
	}
	return &Client{kv: kv}, nil
}

var stashPrefix = []byte("stash_")

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
	ID        string    `json:"id"`
	Note      string    `json:"note"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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
func (cc *Client) GetNewsMarkdown(markdownID string) (*Markdown, error) {
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
	limit := 50
	startOffset := (page * limit) - limit
	endOffset := page * limit
	var stash []*Markdown
	err := cc.kv.View(func(txn *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.Prefix = stashPrefix
		it := txn.NewIterator(opt)
		defer it.Close()
		for it.Seek(stashPrefix); it.ValidForPrefix(stashPrefix); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				md := &Markdown{}
				err := json.Unmarshal(v, md)
				if err != nil {
					return err
				}
				stash = append(stash, md)
				return nil
			})
			if err != nil {
				return err
			}
			if len(stash) > endOffset {
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if startOffset >= len(stash) {
		return []*Markdown{}, nil
	}
	if endOffset > len(stash) {
		return stash[startOffset:], nil
	}
	return stash[startOffset:endOffset], nil
}

// GetStashMarkdown returns the Markdown struct for the given stash markdown ID.
func (cc *Client) GetStashMarkdown(markdownID string) (*Markdown, error) {
	var md Markdown
	d, err := cc.kv.Get([]byte(markdownID))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(d, &md)
	if err != nil {
		return nil, err
	}
	return &md, nil
}

// StashMarkdown encrypts and stashes a new markdown file with note.
func (cc *Client) StashMarkdown(note string, body string) (*Markdown, error) {
	gid := uuid.New().String()
	md := &Markdown{Note: note, Body: body, ID: gid, CreatedAt: time.Now()}
	err := cc.saveMarkdown(md)
	if err != nil {
		return nil, err
	}
	return md, nil
}

// DeleteMarkdown deletes the stash markdown for the given ID.
func (cc *Client) DeleteMarkdown(markdownID string) error {
	txn, err := cc.kv.NewTransaction(true)
	if err != nil {
		return err
	}
	mid, sid := markdownKeys(markdownID)
	err = txn.Delete(mid)
	if err != nil {
		return err
	}
	err = txn.Delete(sid)
	if err != nil {
		return err
	}
	return cc.kv.Commit(txn, func(err error) {
		if err != nil {
			log.Printf("Badger commit error: %s", err)
		}
	})
}

// SetMarkdownNote updates the note for a given stash markdown ID.
func (cc *Client) SetMarkdownNote(markdownID string, note string) error {
	md, err := cc.GetStashMarkdown(markdownID)
	if err != nil {
		return err
	}
	md.Note = note
	return cc.saveMarkdown(md)
}

func (cc *Client) saveMarkdown(md *Markdown) error {
	mid, sid := markdownKeys(md.ID)
	txn, err := cc.kv.NewTransaction(true)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	err = json.NewEncoder(buf).Encode(md)
	if err != nil {
		return err
	}
	err = txn.Set(mid, buf.Bytes())
	if err != nil {
		return err
	}
	buf = bytes.NewBuffer(nil)
	smd := &Markdown{ID: md.ID, Note: md.Note, CreatedAt: md.CreatedAt}
	err = json.NewEncoder(buf).Encode(smd)
	if err != nil {
		return err
	}
	err = txn.Set(sid, buf.Bytes())
	if err != nil {
		return err
	}
	return cc.kv.Commit(txn, func(err error) {
		if err != nil {
			log.Printf("Badger commit error: %s", err)
		}
	})
}

func markdownKeys(markdownID string) ([]byte, []byte) {
	return []byte(markdownID), []byte(fmt.Sprintf("%s%s", stashPrefix, markdownID))
}
