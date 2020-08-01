package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Petition interface {
	ID() int64
	Action() string
	SignatureCount() int64
	OpenedAt() time.Time
}

type petition struct {
	id             int64
	action         string
	signatureCount int64
	openedAt       time.Time
}

type Client interface {
	List() ([]Petition, error)
}

type client struct {
	logger  *logrus.Logger
	baseURL string
}

type petitionsLinks struct {
	Next *string `json:"next,omitempty"`
}

type petitionsDataItemAttributes struct {
	Action         string    `json:"action"`
	SignatureCount int64     `json:"signature_count"`
	OpenedAt       time.Time `json:"opened_at"`
}

type petitionsDataItem struct {
	Type       string                      `json:"type"`
	ID         int64                       `json:"id"`
	Attributes petitionsDataItemAttributes `json:"attributes"`
}

type petitionsResponse struct {
	Links petitionsLinks      `json:"links"`
	Data  []petitionsDataItem `json:"data"`
}

func New(baseURL string, logger *logrus.Logger) Client {
	c := &client{
		baseURL: baseURL,
		logger:  logger,
	}

	return c
}

func (p *petition) ID() int64             { return p.id }
func (p *petition) Action() string        { return p.action }
func (p *petition) OpenedAt() time.Time   { return p.openedAt }
func (p *petition) SignatureCount() int64 { return p.signatureCount }

func (c *client) List() ([]Petition, error) {
	petitions := []Petition{}

	firstURL := c.baseURL + "/petitions.json?state=open"
	nextURL := &firstURL

	for {
		if nextURL == nil {
			break
		}

		c.logger.WithField("url", nextURL).Info("http-get-petitions")
		resp, err := http.Get(*nextURL)

		defer func() {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}()

		if err != nil {
			c.logger.WithField("url", nextURL).Error(err)
			return petitions, err
		}

		if resp.StatusCode != 200 {
			err = fmt.Errorf("expected 200")
			c.logger.
				WithField("url", nextURL).
				WithField("code", resp.StatusCode).
				Error(err)

			return petitions, err
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.logger.WithField("url", nextURL).Error(err)
			return petitions, err
		}

		var parsed petitionsResponse
		err = json.Unmarshal(b, &parsed)
		if err != nil {
			c.logger.WithField("url", nextURL).Error(err)
			return petitions, err
		}

		nextURL = parsed.Links.Next

		for _, p := range parsed.Data {
			petitions = append(petitions, &petition{
				id:             p.ID,
				action:         p.Attributes.Action,
				signatureCount: p.Attributes.SignatureCount,
				openedAt:       p.Attributes.OpenedAt,
			})
		}
	}

	return petitions, nil
}
