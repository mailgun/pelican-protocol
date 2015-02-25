package pelican

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	mailgun "github.com/mailgun/mailgun-go"
)

// send archive over mail

type MailgunConfig struct {
	ApiKey     string   `json:"apikey"`
	Domain     string   `json:"domain"`
	FromEmail  string   `json:"from-email"`  // e.g. "Jill McMail <jill@example.com>"
	RecipEmail []string `json:"recip-email"` // e.g. ["Jill McMail <jill@examaple.com", "Joe McMail <joe@example.com>"]
}

func (c *MailgunConfig) Load(path string) error {

	if !FileExists(path) {
		return fmt.Errorf("could not open because no such file: '%s'", path)
	}

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(dat, c)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("readjson() read as json (raw bytes): '%s'\n\n", string(dat))

	return err
}

func ReadMailgunConfig(path string) *MailgunConfig {
	c := &MailgunConfig{}
	err := c.Load(path)
	panicOn(err)
	return c

}

func (c *MailgunConfig) Save(path string) error {
	by, err := json.Marshal(c)
	panicOn(err)
	err = ioutil.WriteFile(path, by, 0600)
	panicOn(err)
	return err
}

func (cfg *MailgunConfig) SendPPPBackupMailgunMail(body string) (statusMsg string, id string, err error) {

	now := time.Now()
	nowfmt := now.Format(time.RFC3339)
	now2 := now.Format(time.RFC850)

	subject := fmt.Sprintf("pelical-protocol-archive %s / %s", nowfmt, now2)

	publicApiKey := ""
	mg := mailgun.NewMailgun(cfg.Domain, cfg.ApiKey, publicApiKey)
	m := mg.NewMessage(
		cfg.FromEmail,
		subject,
		body,
		cfg.RecipEmail...)
	statusMsg, id, err = mg.Send(m)
	panicOn(err)
	return statusMsg, id, err
}
