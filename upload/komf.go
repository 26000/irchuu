package upload

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/paths"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var downloadRegex = regexp.MustCompile("<a href=\".*\">(.*)</a>")

// Komf uploads a Telegram media file to a komf hosting. It doesn't check
// if the file was already uploaded, that is handeled by pomfs.
func Komf(bot *tgbotapi.BotAPI, id string, c *config.Telegram) (url string, err error) {
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
		return uploadLocalFileKomf(localUrl, c)
	}
	return uploadRemoteFileKomf(file, localUrl, id, fileStrings[len(fileStrings)-1], c)
}

// uploadLocalFileKomf actually uploads the file to a komf using HTTP POST with
// multipart/form-data mime. It also reads the whole file to memory because of
// the current implementation of Go's multipart.
func uploadLocalFileKomf(file string, c *config.Telegram) (url string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	ff, err := w.CreateFormFile("file", f.Name())
	if err != nil {
		return
	}

	// i know, copying it to RAM was never a good idea...
	_, err = io.Copy(ff, f)
	if err != nil {
		return
	}

	err = w.WriteField("date", c.KomfDate)
	if err != nil {
		return
	}

	w.Close()

	r, err := http.Post(makeKomfUrl(c.Komf), w.FormDataContentType(), &b)
	if err != nil {
		return
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	url = makeKomfDownloadUrl(string(body))
	return
}

// uploadRemoteFileKomf downloads a file from Telegram and uploads it to a
// komf using HTTP POST with multipart/form-data mime. It also reads the
// whole file to memory because of the current implementation of Go's multipart.
func uploadRemoteFileKomf(file string, localUrl string, id string, name string, c *config.Telegram) (url string, err error) {
	downloadable, err := http.Get(file)
	defer downloadable.Body.Close()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	ff, err := w.CreateFormFile("file", name) // create the multipart file
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
	err = w.WriteField("date", c.KomfDate)
	if err != nil {
		return
	}

	w.Close()

	r, err := http.Post(makeKomfUrl(c.Komf), w.FormDataContentType(), &b)
	if err != nil {
		return
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	url = makeKomfDownloadUrl(string(body))

	return
}

// makePomfUrl just appends "upload" to a komf site link.
func makeKomfUrl(komf string) string {
	if len(komf) < 2 {
		return ""
	}
	if komf[len(komf)-1] == '/' {
		return komf + "upload"
	}
	return komf + "/upload"
}

// makeKomfDownloadUrl appends a file path to a komf site link.
func makeKomfDownloadUrl(file string) string {
	matches := downloadRegex.FindStringSubmatch(file)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
