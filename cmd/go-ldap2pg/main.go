package main

import (
	"context"
	"log"

	ldap "github.com/go-ldap/ldap/v3"
	"github.com/jackc/pgx/v4"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

var data string = `
toto: [1, "titi", null, 4.2]
`

type YamlConfig struct {
	Toto []interface{} `yaml:"toto"`
}

func main() {
	var c EnvConfig
	envconfig.MustProcess("", &c)

	y := YamlConfig{}
	err := yaml.Unmarshal([]byte(data), &y)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %s", err)
	}
	log.Println("Len toto", len(y.Toto))
	for i, value := range y.Toto {
		switch t := value.(type) {
		case int:
			log.Printf("toto[%d] %T = %d", i, t, value.(int))
		case string:
			log.Printf("toto[%d] %T = %s", i, t, value.(string))
		default:
			log.Printf("toto[%d] %+v %T, unhandled.", i, value, t)
		}
	}

	pgconn, err := pgx.Connect(context.Background(), "")
	if err != nil {
		log.Fatalf("PostgreSQL connection error: %s", err)
	}
	defer pgconn.Close(context.Background())

	log.Printf("LDAP dial: %s", c.LdapUri)
	ldapconn, err := ldap.DialURL(c.LdapUri)
	if err != nil {
		log.Fatal(err)
	}
	defer ldapconn.Close()
	log.Printf("LDAP simple bind: %s", c.LdapBindDn)
	err = ldapconn.Bind(c.LdapBindDn, c.LdapPassword)
	if err != nil {
		log.Fatal(err)
	}

	wai, err := ldapconn.WhoAmI(nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("authzid", wai.AuthzID)

	var me string
	err = pgconn.QueryRow(context.Background(), "SELECT CURRENT_USER;").Scan(&me)
	if err != nil {
		log.Fatalf("Failed to query: %s", err)
	}

	log.Printf("Running as %s.\n", me)
}

type EnvConfig struct {
	LdapUri      string `envconfig:"LDAPURI"`
	LdapBindDn   string `envconfig:"LDAPBINDDN"`
	LdapPassword string `envconfig:"LDAPPASSWORD"`
}
