# go-web-push-package
[WIP] Generates a push package for use with Apple's Push Notification Service

## Prerequisites
Uses openssl to sign the manifest

## installation

  $ go get github.com/rohenaz/go-web-push-package

## usage

```
Define your package config as follows:

PushPackageConfig {
	website {
    "websiteName": "Test",
    "websitePushID": "web.example.test",
    "allowedDomains": ["https://example.com"],
    "urlFormatString": "https://example.com/%@",
    "authenticationToken": "19f8d7a6e9fb8a7f6d9330dabe",
    "webServiceURL": "https://example.com",
  }
	iconPath: 'path/to/iconFolder'
	certificates {
    signer: 'certificates/cert.pem',
		key: 'certificates/key.pem',
  }
}
```
Generage the package and return the archive

`buffer := Config.GeneratePackage()`

Write your response

```
w.Header().Set("Content-type", "application/zip")
w.Write(buf.Bytes())
```

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
