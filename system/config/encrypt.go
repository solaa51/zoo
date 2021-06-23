package config

// Encrypt http请求加密处理验证方式
type Encrypt struct {
	Type string     `toml:"type"`
	Keys []struct { //加密使用的键值对
		Key   string `toml:"key"`
		Value string `toml:"value"`
	} `toml:"keys"`
}

//// EncryptKey 加密键值对
//type EncryptKey struct {
//	Key   string `toml:"key"`
//	Value string `toml:"value"`
//}
