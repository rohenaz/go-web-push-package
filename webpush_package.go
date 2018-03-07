package webpush_package

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type PushPackageConfig struct {
	website      websiteConfig
	iconPath     string
	certificates certificatesConfig
}

// websiteName - The website name. This is the heading used in Notification Center.
// websitePushID - The Website Push ID, as specified in your developer account.
// allowedDomains - An array of websites that are allowed to request permission from the user.
// urlFormatString -The URL to go to when the notification is clicked. Use %@ as a placeholder for arguments you fill in when delivering your notification. This URL must use the http or https scheme; otherwise, it is invalid.
// authenticationToken - A string that helps you identify the user. It is included in later requests to your web service. This string must 16 characters or greater.
// webServiceURL - The location used to make requests to your web service. The trailing slash should be omitted.
type websiteConfig struct {
	websiteName         string   `json:"websiteName"`
	websitePushID       string   `json:"websitePushID"`
	allowedDomains      []string `json:"allowedDomains"`
	urlFormatString     string   `json:"urlFormatString"`
	authenticationToken string   `json:"authenticationToken"`
	webServiceUrl       string   `json:"webServiceUrl"`
}

type certificatesConfig struct {
	key    string
	signer string
}

// Each key value pair in manifest file is a file path and its hash
type manifestRecord struct {
	filePath string
	hash     string
}

type jsonFile struct {
	data []byte
	name string
}

func (c *PushPackageConfig) GeneratePackage() (*bytes.Buffer, error) {

	// potentilly good reference
	// https://stackoverflow.com/questions/24472895/how-to-sign-manifest-json-for-safari-push-notifications-using-golang

	// Generate the files and return the tempPath
	tempPath := c.makePackageFiles()

	// Create a new zip archive.
	buf := new(bytes.Buffer)
	// z := zip.NewWriter(buf)

	// Make the manifest
	manifestJSON := c.generateManifestJSON(tempPath)
	err := ioutil.WriteFile(tempPath+"/"+manifestJSON.name, manifestJSON.data, 0644)
	if err != nil {
		panic(err)
	}

	// Sign manifest
	signature := c.certificates.generateManifestSignature(manifestJSON.data)
	// Write signature to package
	err = ioutil.WriteFile(tempPath+"/signature", signature, 0644)
	if err != nil {
		panic(err)
	}
	// ToDo - zip package

	// Zip the files
	// "github.com/pierrre/archivefile/zip"
	// tmpDir, err := ioutil.TempDir("", "package_zip")
	// if err != nil {
	// 	panic(err)
	// }
	// defer func() {
	// 	_ = os.RemoveAll(tmpDir)
	// }()
	//
	// outFilePath := filepath.Join(tmpDir, "foo.zip")
	//
	// progress := func(archivePath string) {
	// 	fmt.Println(archivePath)
	// }
	//
	// err = ArchiveFile("testdata/foo", outFilePath, progress)
	// if err != nil {
	// 	panic(err)
	// }
	return buf, err
}

func (c *PushPackageConfig) makePackageFiles() string {
	// New random path
	tempPath := "/temp/" + randomString(10)

	// Get the icons
	// This func should create them locally
	c.copyIcons(tempPath)

	// Get WebsiteJSON
	websiteJSON := c.generateWebsiteJSON()
	err := ioutil.WriteFile(tempPath+"/"+websiteJSON.name, websiteJSON.data, 0644)
	if err != nil {
		panic(err)
	}

	return tempPath
}

func (c *PushPackageConfig) generateManifestJSON(tempPath string) jsonFile {

	// 1. loop over files in the temp path
	// Sha1s of all files in there so far (icons and website.json)

	jsonLines := []string{}
	jsonLines = append(jsonLines, "{")
	filepath.Walk(tempPath, func(path string, f os.FileInfo, err error) error {

		// read the file and get the sha1
		file, err := os.Open(path) // For read access.
		if err != nil {
			log.Fatal(err)
		}

		// read the file data
		data := make([]byte, f.Size())
		_, err = file.Read(data)
		if err != nil {
			log.Fatal(err)
		}

		// Get the hash
		sha := sha1.Sum(data)

		// Make the json line for the manifest
		jsonLines = append(jsonLines, "\""+path+":"+string(sha[:20])+"\"")

		return nil
	})

	jsonLines = append(jsonLines, "}")

	jsonData, err := json.Marshal(jsonLines)
	if err != nil {
		panic(err)
	}

	return jsonFile{
		name: "manifest.json",
		data: jsonData,
	}

}

func (c *certificatesConfig) generateManifestSignature(message []byte) []byte {
	// ToDo - read local key file
	keyfile := c.key
	signed, err := openssl(message, "cms", "-sign", "-signer", keyfile)
	if err != nil {
		panic(err)
	}

	return signed
}

func (c *PushPackageConfig) generateWebsiteJSON() jsonFile {
	// ToDo - Marshal the website config JSON
	data, err := json.Marshal(c.website)
	if err != nil {
		panic(err)
	}

	return jsonFile{
		data: data,
		name: "website.json",
	}
}

func (c *PushPackageConfig) copyIcons(tempPath string) {
	// copy all files from from c.iconPath to tempPath
	filepath.Walk(c.iconPath, func(fileName string, f os.FileInfo, err error) error {
		copy(c.iconPath+"/"+fileName, tempPath+"/"+fileName)
		return nil
	})
}

// Wraps openssl
// This comes from here https://play.golang.org/p/TyLE8UFQGc
func openssl(stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.Command("openssl", args...)

	in := bytes.NewReader(stdin)
	out := &bytes.Buffer{}
	errs := &bytes.Buffer{}

	cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, errs

	if err := cmd.Run(); err != nil {
		if len(errs.Bytes()) > 0 {
			return nil, fmt.Errorf("error running %s (%s):\n %v", cmd.Args, err, errs.String())
		}
		return nil, err
	}

	return out.Bytes(), nil
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// see this https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func randomString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
