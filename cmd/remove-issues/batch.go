package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type batchXML struct {
	XMLName   string      `xml:"http://www.loc.gov/ndnp batch"`
	Name      string      `xml:"name,attr"`
	Awardee   string      `xml:"awardee,attr"`
	AwardYear string      `xml:"awardYear,attr"`
	Issues    []*issueXML `xml:"issue"`
	Reels     []*reelXML  `xml:"reel"`
	SkipDirs  []string    `xml:"-"`
}
type issueXML struct {
	LCCN      string `xml:"lccn,attr"`
	IssueDate string `xml:"issueDate,attr"`
	Edition   string `xml:"editionOrder,attr"`
	Path      string `xml:",innerxml"`
	Skip      bool   `xml:"-"`
}
type reelXML struct {
	ReelNum string `xml:"reelNumber,attr"`
	Path    string `xml:",innerxml"`
}

// ParseBatch reads the given XML file and processes it into batch data.  The
// skipKeys are converted into directories that should be skipped from the
// copy, and stored in the returned structure's SkipDirs field.
func ParseBatch(pth string, skipKeys []string) (*batchXML, error) {
	var data, err = ioutil.ReadFile(pth)
	if err != nil {
		return nil, err
	}
	var b = new(batchXML)
	err = xml.Unmarshal(data, b)
	if err != nil {
		return nil, err
	}
	if len(b.Issues) == 0 {
		return nil, fmt.Errorf("parsed data has no issues")
	}

	var keyToDir = make(map[string]*issueXML)
	for _, issue := range b.Issues {
		var key = keyfix(issue.LCCN + "/" + issue.IssueDate + issue.Edition)
		keyToDir[key] = issue
	}

	for _, key := range skipKeys {
		key = keyfix(key)
		var issue = keyToDir[key]
		if issue == nil {
			return nil, fmt.Errorf("issuekey %q not in batch", key)
		}

		var dir, _ = filepath.Split(issue.Path)
		issue.Skip = true
		b.SkipDirs = append(b.SkipDirs, dir)
	}

	var newIssues []*issueXML
	for _, i := range b.Issues {
		if i.Skip {
			continue
		}

		newIssues = append(newIssues, i)
	}

	b.Issues = newIssues
	return b, nil
}

func (b *batchXML) WriteBatchXML(pth string) error {
	var dir, _ = filepath.Split(pth)
	var err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	var data []byte
	data, err = xml.MarshalIndent(b, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pth, data, 0644)
}

func keyfix(key string) string {
	key = strings.Replace(key, "-", "", -1)
	key = strings.Replace(key, "_", "", -1)
	return key
}
