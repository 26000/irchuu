// upload contains functions to upload files on variuos file hostings.
package upload

import (
	"bytes"
	"encoding/json"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/paths"
	"gopkg.in/telegram-bot-api.v4"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
)

// Pomf uploads a Telegram media file to a pomf-like hosting. It doesn't check
// if the file was already uploaded, that is handeled by pomfs.
func Pomf(bot *tgbotapi.BotAPI, id string, c *config.Telegram) (url string, err error) {
	file, err := bot.GetFileDirectURL(id)
	if err != nil {
		return
	}
	fileStrings := strings.Split(file, "/")
	fileName := strings.Split(fileStrings[len(fileStrings)-1], ".")

	var ext string
	if len(fileName) > 1 {
		ext = "." + fileName[len(fileName)-1]
	}
	localUrl := path.Join(c.DataDir, id+ext)
	// if it is already downloaded, just upload the local copy
	if paths.Exists(localUrl) {
		return uploadLocalFilePomf(localUrl, c)
	} else {
		return uploadRemoteFilePomf(file, localUrl, id, fileStrings[len(fileStrings)-1], c)
	}
}

// uploadLocalFilePomf actually uploads the file to a pomf clone using HTTP POST with
// multipart/form-data mime. It also reads the whole file to memory because of
// the current implementation of Go's multipart.
func uploadLocalFilePomf(file string, c *config.Telegram) (url string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	ff, err := w.CreateFormFile("files[]", f.Name())
	if err != nil {
		return
	}

	// i know, copying it to RAM was never a good idea...
	_, err = io.Copy(ff, f)
	if err != nil {
		return
	}

	w.Close()

	var pr pomfResult
	r, err := http.Post(makePomfUrl(c.Pomf), w.FormDataContentType(), &b)
	if err != nil {
		return
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &pr)
	if err != nil {
		return
	}
	url = pr.Url()
	return
}

// uploadRemoteFilePomf downloads a file from Telegram and uploads it to a
// pomf clone using HTTP POST with multipart/form-data mime. It also reads the
// whole file to memory because of the current implementation of Go's multipart.
func uploadRemoteFilePomf(file string, localUrl string, id string, name string, c *config.Telegram) (url string, err error) {
	downloadable, err := http.Get(file)
	defer downloadable.Body.Close()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	ff, err := w.CreateFormFile("files[]", name) // create the multipart file
	if err != nil {
		return
	}

	if c.DownloadMedia {
		// if we need to both upload and download it, we use io.MultiWriter
		res, err := os.Create(localUrl)
		if err != nil {
			return "", err
		}
		writer := io.MultiWriter(res, ff)

		// i know, copying it to RAM was never a good idea...
		_, err = io.Copy(writer, downloadable.Body)
		res.Close()
		if err != nil {
			return "", err
		}
	} else {
		// just io.Copy otherwise
		// i know, copying it to RAM was never a good idea...
		_, err = io.Copy(ff, downloadable.Body)
		if err != nil {
			return
		}
	}
	w.Close()

	var pr pomfResult
	r, err := http.Post(makePomfUrl(c.Pomf), w.FormDataContentType(), &b)
	if err != nil {
		return
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &pr)
	if err != nil {
		return
	}
	url = pr.Url()
	return
}

// pomfResult is the data pomf returns in JSON.
type pomfResult struct {
	Success bool
	Files   []pomfile
}

// pomfile is a file in pomfResult.
type pomfile struct {
	Hash string
	Name string
	Url  string
	Size int
}

// Url returns the URL of the first file in pomfResult.
func (r *pomfResult) Url() string {
	if len(r.Files) != 0 {
		return r.Files[0].Url
	}
	return ""
}

// makePomfUrl just appends "upload.php" to a pomf clone link.
func makePomfUrl(pomf string) string {
	if len(pomf) < 2 {
		return ""
	}
	if pomf[len(pomf)-1] == '/' {
		return pomf + "upload.php"
	}
	return pomf + "/upload.php"
}
