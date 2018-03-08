package webpush_package

// potentilly good reference
// https://stackoverflow.com/questions/24472895/how-to-sign-manifest-json-for-safari-push-notifications-using-golang

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/deliverydudes/go-library/utils/logger"
)

type PushPackageConfig struct {
	Website      WebsiteConfig
	IconPath     string
	Certificates CertificatesConfig
}

// websiteName - The website name. This is the heading used in Notification Center.
// websitePushID - The Website Push ID, as specified in your developer account.
// allowedDomains - An array of websites that are allowed to request permission from the user.
// urlFormatString -The URL to go to when the notification is clicked. Use %@ as a placeholder for arguments you fill in when delivering your notification. This URL must use the http or https scheme; otherwise, it is invalid.
// authenticationToken - A string that helps you identify the user. It is included in later requests to your web service. This string must 16 characters or greater.
// webServiceURL - The location used to make requests to your web service. The trailing slash should be omitted.
type WebsiteConfig struct {
	WebsiteName         string   `json:"websiteName"`
	WebsitePushID       string   `json:"websitePushID"`
	AllowedDomains      []string `json:"allowedDomains"`
	UrlFormatString     string   `json:"urlFormatString"`
	AuthenticationToken string   `json:"authenticationToken"`
	WebServiceUrl       string   `json:"webServiceUrl"`
}

type CertificatesConfig struct {
	Key    string
	Signer string
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

	// Generate the base files and return the tempPath
	tempPath := c.makePackageFiles()

	// Make the manifest
	manifestJSON := c.generateManifestJSON(tempPath)
	err := ioutil.WriteFile(tempPath+"/"+manifestJSON.name, manifestJSON.data, 0644)
	checkErr(err)

	// Sign manifest
	signature := c.Certificates.generateManifestSignature(manifestJSON.data)
	// Write signature to package
	err = ioutil.WriteFile(tempPath+"/signature", signature, 0644)
	checkErr(err)

	// Zip up the files
	buffer, err := RecursiveZip(tempPath)
	checkErr(err)

	defer os.RemoveAll(tempPath) // clean up

	return buffer, err
}

func (c *PushPackageConfig) makePackageFiles() string {
	// New random path
	name := strings.Replace(c.Website.WebsiteName, " ", "", -1)
	tempPath, err := ioutil.TempDir("", name+".pushpackage")
	checkErr(err)

	// Copy icons to tempPath
	c.copyIcons(tempPath)

	// Get WebsiteJSON
	websiteJSON := c.generateWebsiteJSON()

	// Create a new file
	file := filepath.Join(tempPath, websiteJSON.name)
	err = ioutil.WriteFile(file, websiteJSON.data, 0644)
	checkErr(err)

	return tempPath
}

func (c *PushPackageConfig) generateManifestJSON(tempPath string) jsonFile {

	// generate json manifest
	jsonLines := []string{}
	jsonLines = append(jsonLines, "{")

	err := filepath.Walk(tempPath, func(path string, f os.FileInfo, err error) error {
		if !f.Mode().IsDir() {
			file, err := os.Open(path)
			checkErr(err)

			// read the file data
			data := make([]byte, f.Size())
			_, err = file.Read(data)
			checkErr(err)

			// Get the sha1 hash of the file
			sha := sha1.Sum(data)

			// Make the json line for the manifest
			jsonLines = append(jsonLines, "\""+path+":"+string(sha[:20])+"\"")
		}
		return nil
	})

	checkErr(err)

	jsonLines = append(jsonLines, "}")

	jsonData, err := json.Marshal(jsonLines)
	checkErr(err)

	return jsonFile{
		name: "manifest.json",
		data: jsonData,
	}
}

func (c *CertificatesConfig) generateManifestSignature(message []byte) []byte {
	// ToDo - read local key file
	keyfile := c.Key
	signed, err := openssl(message, "cms", "-sign", "-signer", keyfile)
	checkErr(err)

	return signed
}

func (c *PushPackageConfig) generateWebsiteJSON() jsonFile {
	// ToDo - Marshal the website config JSON
	data, err := json.Marshal(c.Website)
	checkErr(err)

	return jsonFile{
		data: data,
		name: "website.json",
	}
}

func (c *PushPackageConfig) copyIcons(tempPath string) {
	// copy all files from from c.iconPath to tempPath
	// Create the icons folder
	err := os.Mkdir(filepath.Join(tempPath, "icon.iconset"), os.ModePerm)
	checkErr(err)

	filepath.Walk(c.IconPath, func(fileName string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			dest := filepath.Join(tempPath, "icon.iconset/"+f.Name())
			copy(fileName, dest)
		}
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

// Copy src to dst. Overwrites exiting files, does not copy attributes
func copy(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	checkErr(err)
	err = ioutil.WriteFile(dst, data, 0644)
	return err
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func RecursiveZip(pathToZip string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	myZip := zip.NewWriter(buf)
	err := filepath.Walk(pathToZip, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		checkErr(err)
		relPath := strings.TrimPrefix(filePath, filepath.Dir(pathToZip))
		logger.Printf("Zipping %s\t(%d bytes)", info.Name(), info.Size())
		zipFile, err := myZip.Create(relPath)
		checkErr(err)
		fsFile, err := os.Open(filePath)
		checkErr(err)
		_, err = io.Copy(zipFile, fsFile)
		checkErr(err)
		return nil
	})
	checkErr(err)
	err = myZip.Close()
	checkErr(err)
	return buf, err
}
