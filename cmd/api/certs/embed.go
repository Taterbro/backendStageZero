package certs

import _ "embed"

//go:embed ca.pem
var CaCert []byte
