package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"time"
)

// Convert DOCX to text
func ConvertDocx(r io.Reader) (string, map[string]string, error) {
	meta := make(map[string]string)
	var textHeader, textBody, textFooter string

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", nil, fmt.Errorf("error unzipping data: %v", err)
	}

	// Regular expression for XML files to include in the text parsing
	reHeaderFile, _ := regexp.Compile("^word/header[0-9]+.xml$")
	reFooterFile, _ := regexp.Compile("^word/footer[0-9]+.xml$")

	for _, f := range zr.File {
		if f.Name == "docProps/core.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", nil, fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
			}
			defer rc.Close()

			info, err := XMLToMap(rc)
			if err != nil {
				return "", nil, fmt.Errorf("error parsing '%v': %v", f.Name, err)
			}

			if tmp, ok := info["modified"]; ok {
				if t, err := time.Parse(time.RFC3339, tmp); err == nil {
					meta["ModifiedDate"] = fmt.Sprintf("%d", t.Unix())
				}
			}
			if tmp, ok := info["created"]; ok {
				if t, err := time.Parse(time.RFC3339, tmp); err == nil {
					meta["CreatedDate"] = fmt.Sprintf("%d", t.Unix())
				}
			}
		} else if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", nil, fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
			}
			defer rc.Close()

			textBody, err = DocxXMLToText(rc)
			if err != nil {
				return "", nil, fmt.Errorf("error parsing '%v': %v", f.Name, err)
			}
		} else if reHeaderFile.MatchString(f.Name) {
			rc, err := f.Open()
			if err != nil {
				return "", nil, fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
			}
			defer rc.Close()

			header, err := DocxXMLToText(rc)
			if err != nil {
				return "", nil, fmt.Errorf("error parsing header file '%v': %v", f.Name, err)
			}
			textHeader += header + "\n"
		} else if reFooterFile.MatchString(f.Name) {
			rc, err := f.Open()
			if err != nil {
				return "", nil, fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
			}
			defer rc.Close()

			footer, err := DocxXMLToText(rc)
			if err != nil {
				return "", nil, fmt.Errorf("error parsing footer file '%v': %v", f.Name, err)
			}
			textFooter += footer + "\n"
		}
	}

	return textHeader + "\n" + textBody + "\n" + textFooter, meta, nil
}

func DocxXMLToText(r io.Reader) (string, error) {
	return XMLToText(r, []string{"br", "p", "tab"}, []string{"instrText", "script"}, true)
}
