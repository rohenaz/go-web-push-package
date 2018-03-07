# go-web-push-package
[WIP] Generates a push package for use with Apple's Push Notification Service

## Prerequisites
Uses openssl to sign the manifest

## installation

  $ go get github.com/rohenaz/go-web-push-package

##usage

```
Define your package config as follows:

type Config struct {
	website      WebsiteConfig
	iconPath     string
	certificates CertificatesConfig
}

type WebsiteConfig struct {
	websiteName         string   `json:"websiteName"`
	websitePushID       string   `json:"websitePushID"`
	allowedDomains      []string `json:"allowedDomains"`
	urlFormatString     string   `json:"urlFormatString"`
	authenticationToken string   `json:"authenticationToken"`
	webServiceUrl       string   `json:"webServiceUrl"`
}

type CertificatesConfig struct {
	key    string
	signer string
}

```
Generage the package and return the archive

`zipPath, zipData := Config.GeneratePackage()`

the above would generate a package looking like this

- icon.iconset/icon_128x128.png
- icon.iconset/icon_128x128@2x.png
- icon.iconset/icon_16x16.png
- icon.iconset/icon_16x16@2x.png
- icon.iconset/icon_32x32.png
- icon.iconset/icon_32x32@2x.png
- manifest.json
- signature
- website.json
