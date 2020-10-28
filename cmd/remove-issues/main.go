package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func usageError(msg string, args ...interface{}) {
	var fmsg = fmt.Sprintf(msg, args...)
	fmt.Printf("\033[31;1mERROR: %s\033[0m\n", fmsg)
	fmt.Printf(`
Usage: %s <source directory> <destination directory> <issue key>...

The source directory should either be the pristine dark archive, or a copy
thereof (though the TIFF files won't matter, as they aren't copied to the
destination).  Once complete, the destination will contain an ONI-ingestable
batch.

One or more issue keys must be present.  If any key is given but isn't in the
source batch, this tool will report it and exit without processing any other
keys, even if they're valid.
`, os.Args[0])
	os.Exit(1)
}

// FixContext is just the app's directory/lccn context so we don't have global
// variables puked out everywhere but we also don't pass a million args around
// to everything
type FixContext struct {
	SourceDir string
	DestDir   string
	IssueKeys []string
	SkipDirs  []string
}

// getArgs does some sanity-checking and sets the source/dest args
func getArgs() *FixContext {
	if len(os.Args) < 4 {
		usageError("Missing one or more arguments")
	}

	var fc = &FixContext{
		SourceDir: os.Args[1],
		DestDir:   os.Args[2],
		IssueKeys: os.Args[3:],
	}
	var err error
	fc.SourceDir, err = filepath.Abs(fc.SourceDir)
	if err != nil {
		usageError("Source (%s) is invalid: %s", fc.SourceDir, err)
	}
	fc.DestDir, err = filepath.Abs(fc.DestDir)
	if err != nil {
		usageError("Source (%s) is invalid: %s", fc.DestDir, err)
	}

	var info os.FileInfo
	info, err = os.Stat(fc.SourceDir)
	if err != nil {
		usageError("Source (%s) is invalid: %s", fc.SourceDir, err)
	}
	if !info.IsDir() {
		usageError("Source (%s) is invalid: not a directory", fc.SourceDir)
	}

	_, err = os.Stat(fc.DestDir)
	if err == nil || !os.IsNotExist(err) {
		usageError("Destination (%s) already exists", fc.DestDir)
	}

	return fc
}

func main() {
	var fixContext = getArgs()

	// Read the batch XML to get a list of issue directories to skip
	var batchPath = filepath.Join(fixContext.SourceDir, "data", "batch.xml")
	var newBatchPath = filepath.Join(fixContext.DestDir, "data", "batch.xml")
	var err error
	fixContext.SkipDirs, err = fixBatchXML(batchPath, newBatchPath, fixContext.IssueKeys)
	if err != nil {
		log.Fatalf("Unable to read batch XML file %q: %s", batchPath, err)
	}

	// Crawl all files and determine the action necessary.  NOTE: this may not be
	// the ideal number of workers.  On an SSD, it seems to work much faster than
	// lower numbers.  One of the following must be true, but I dunno which:
	// - Go's IO is really bad when not parallelized
	// - My code is doing more CPU-intense logic than it seems like it should
	// - SSD write queuing is just super amazing
	var queue = NewWorkQueue(fixContext, 2*runtime.NumCPU())
	var walker = NewWalker(fixContext, queue)
	err = walker.Walk()
	if err != nil {
		log.Fatalf("Error trying to copy/fix the batch: %s\n", err)
	}

	// Wait for the queue to complete all actions/jobs
	queue.Wait()
}

type batchXML struct {
	Issues []*issueXML `xml:"issue"`
}
type issueXML struct {
	LCCN      string `xml:"lccn,attr"`
	IssueDate string `xml:"issueDate,attr"`
	Edition   string `xml:"editionOrder,attr"`
	Path      string `xml:",innerxml"`
}

// fixBatchXML is a bad function - it really does two things: rewrites the
// batch XML in the new batch and gathers the list of directories to skip for
// the workers.  It's easier this way, but definitely not clean.
func fixBatchXML(batchPath, newBatchPath string, keysToDelete []string) ([]string, error) {
	var batchInfo, err = parseBatchXML(batchPath)
	if err != nil {
		return nil, err
	}

	var dirsToSkip []string
	dirsToSkip, err = getSkipDirs(batchInfo, keysToDelete)
	if err != nil {
		return nil, err
	}

	return dirsToSkip, err
}

func getSkipDirs(batchInfo *batchXML, keysToDelete []string) ([]string, error) {
	var keyToDir = make(map[string]string)
	for _, issue := range batchInfo.Issues {
		var key = keyfix(issue.LCCN + "/" + issue.IssueDate + issue.Edition)
		var dir, _ = filepath.Split(issue.Path)
		keyToDir[key] = dir
	}
	log.Printf("INFO - batch.xml contains %d issues", len(keyToDir))

	var dirs []string
	for _, key := range keysToDelete {
		key = keyfix(key)
		var dir = keyToDir[key]
		if dir == "" {
			return nil, fmt.Errorf("key %q was not found in the batch.xml file", key)
		}
		log.Printf("INFO - Mapping input key %q to directory %q", key, dir)
		dirs = append(dirs, dir)
	}

	return dirs, nil
}

func parseBatchXML(pth string) (*batchXML, error) {
	var data, err = ioutil.ReadFile(pth)
	if err != nil {
		return nil, err
	}
	var batchInfo = new(batchXML)
	err = xml.Unmarshal(data, batchInfo)
	if err != nil {
		log.Fatalf("Unable to parse batch XML file %q: %s", pth, err)
		return nil, err
	}

	return batchInfo, nil
}

func keyfix(key string) string {
	key = strings.Replace(key, "-", "", -1)
	key = strings.Replace(key, "_", "", -1)
	return key
}
