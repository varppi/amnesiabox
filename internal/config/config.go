package config

import (
	"flag"
	"log"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

type Configuration struct {
	Open           bool
	Listener       string
	Cert           string
	Key            string
	Password       string
	Sizelimit      int
	Hidehosted     bool
	Disablecaptcha bool
}

func GetConfig() (*Configuration, error) {
	viper.SetConfigFile("amnesiabox.conf")
	viper.SetConfigType("env")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &Configuration{
		Listener:       "127.0.0.1:8080",
		Cert:           "",
		Key:            "",
		Open:           false,
		Password:       "",
		Sizelimit:      10485760,
		Hidehosted:     false,
		Disablecaptcha: false,
	}

	viper.Unmarshal(conf)

	commandLineParameters := make(map[string]any)
	commandLineParameters["listener"] = flag.String("l", "", "listener (127.0.0.1:8080)")
	commandLineParameters["cert"] = flag.String("cert", "", "ssl public certificate path")
	commandLineParameters["key"] = flag.String("key", "", "ssl private key path")
	commandLineParameters["open"] = flag.Bool("open", false, "make anyone be able to run a site without password")
	commandLineParameters["password"] = flag.String("password", "", "global password that you need to upload a site")
	commandLineParameters["sizelimit"] = flag.Int("sizelimit", 10485760, "size limit for the file uploads")
	commandLineParameters["hidehosted"] = flag.Bool("hidehosted", false, "enable or disable showing sites hosted")
	commandLineParameters["disablecaptcha"] = flag.Bool("disable-captcha", false, "disables login and upload captcha")
	flag.Parse()

	for key, val := range commandLineParameters {
		switch val.(type) {
		case *string:
			val := *(val.(*string))
			if val == "" {
				continue
			}
			reflect.ValueOf(conf).Elem().FieldByName(strings.ToUpper(string(key[0])) + key[1:]).SetString(val)
		case *bool:
			val := *(val.(*bool))
			if !val {
				continue
			}
			reflect.ValueOf(conf).Elem().FieldByName(strings.ToUpper(string(key[0])) + key[1:]).SetBool(val)
		case *int:
			val := *(val.(*int))
			if val == 10485760 {
				continue
			}
			reflect.ValueOf(conf).Elem().FieldByName(strings.ToUpper(string(key[0])) + key[1:]).SetInt(int64(val))
		}
	}

	if !conf.Open && conf.Password == "" {
		log.Fatal("please set a global password (--password), or turn on the open mode")
	}

	return conf, nil
}
