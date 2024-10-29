package main

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/httphdr"
)

// upload base translation.
func (c *twoskyClient) upload() (err error) {
	defer func() { err = errors.Annotate(err, "upload: %w") }()

	uploadURI := c.uri.JoinPath("upload")
	basePath := filepath.Join(localesDir, defaultBaseFile)

	formData := map[string]string{
		"format":   "json",
		"language": string(c.baseLang),
		"filename": defaultBaseFile,
		"project":  c.projectID,
	}

	buf, cType, err := prepareMultipartMsg(formData, basePath)
	if err != nil {
		return fmt.Errorf("preparing multipart msg: %w", err)
	}

	err = send(uploadURI.String(), cType, buf)
	if err != nil {
		return fmt.Errorf("sending multipart msg: %w", err)
	}

	return nil
}

// prepareMultipartMsg prepares translation data for upload.
func prepareMultipartMsg(
	formData map[string]string,
	basePath string,
) (buf *bytes.Buffer, cType string, err error) {
	buf = &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	var fw io.Writer

	for _, k := range slices.Sorted(maps.Keys(formData)) {
		err = w.WriteField(k, formData[k])
		if err != nil {
			return nil, "", fmt.Errorf("writing field %q: %w", k, err)
		}
	}

	file, err := os.Open(basePath)
	if err != nil {
		return nil, "", fmt.Errorf("opening file: %w", err)
	}

	defer func() {
		err = errors.WithDeferred(err, file.Close())
	}()

	h := make(textproto.MIMEHeader)
	h.Set(httphdr.ContentType, aghhttp.HdrValApplicationJSON)

	d := fmt.Sprintf("form-data; name=%q; filename=%q", "file", defaultBaseFile)
	h.Set(httphdr.ContentDisposition, d)

	fw, err = w.CreatePart(h)
	if err != nil {
		return nil, "", fmt.Errorf("creating part: %w", err)
	}

	_, err = io.Copy(fw, file)
	if err != nil {
		return nil, "", fmt.Errorf("copying: %w", err)
	}

	err = w.Close()
	if err != nil {
		return nil, "", fmt.Errorf("closing writer: %w", err)
	}

	return buf, w.FormDataContentType(), nil
}

// send POST request to uriStr.
func send(uriStr, cType string, buf *bytes.Buffer) (err error) {
	client := http.Client{
		Timeout: uploadTimeout,
	}

	req, err := http.NewRequest(http.MethodPost, uriStr, buf)
	if err != nil {
		return fmt.Errorf("bad request: %w", err)
	}

	req.Header.Set(httphdr.ContentType, cType)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("client post form: %w", err)
	}

	defer func() {
		err = errors.WithDeferred(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code is not ok: %q", http.StatusText(resp.StatusCode))
	}

	return nil
}
