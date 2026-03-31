package client

import _ "embed"

// embeddedRootCA содержит встроенный CA сертификат,
// который используется если в конфигурации не указан ca_file.
//
// ВАЖНО:
// - сертификат должен быть в PEM формате
// - путь относительно этого файла
//
//go:embed certs/procurator-root-ca.pem
var embeddedRootCA []byte
