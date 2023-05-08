package main

import (
	// "bufio"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
)

func fileExists(f string) bool {
	if _, err := os.Stat(f); errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		return true
	}
}

func fileNotExists(f string) bool {
	return !fileExists(f)
}

func configDelete() error {
	f := getConfigFullName()
	if fileExists(f) {
		err := os.Remove(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func getConfigDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Unable to work out where your home directory is!")
	}

	return fmt.Sprintf("%s/.config/", h)
}

func getConfigFile() string {
	return "xfer.conf"
}

func getConfigFullName() string {
	return fmt.Sprintf("%s%s", getConfigDir(), getConfigFile())
}

func saveConfig(c *xferConfig) error {

	// create dir if not there already
	if !fileExists(getConfigDir()) {
		e := os.Mkdir(getConfigDir(), 0755)
		if e != nil {
			return e
		}
	}

	// write out config
	e := os.WriteFile(getConfigFullName(), []byte(c.ServerEndpoint), 0600)
	if e != nil {
		return e
	}

	// done
	return nil
}

func loadConfig() (*xferConfig, error) {

	c := new(xferConfig)

	if fileNotExists(getConfigFullName()) {
		// fmt.Println("Enter your transfer.sh server endpoint (e.g. https://transfer.sh): ")

		// reader := bufio.NewReader(os.Stdin)
		// ReadString will block until the delimiter is entered
		
		/*
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Sorry, i couldn't understand what you typed: %s", err)
		}

		// remove trailing delimeters
		if runtime.GOOS == "windows" {
			input = strings.Replace(input, "\r\n", "", -1)
		} else {
			input = strings.Replace(input, "\n", "", -1)
		}

		// add suffix if required
		if !strings.HasSuffix(input, "/") {
			input += "/"
		}
		
		*/
		
		// create default config based on input
		c.ServerEndpoint = "https://transfer.sh/"
		err := saveConfig(c)
		if err != nil {
			log.Fatalf("Failed to save your configuration file: %s", err)
		}
	} else {
		// read in config
		b, err := os.ReadFile(getConfigFullName())
		if err != nil {
			return nil, err
		}
		c.ServerEndpoint = string(b)
	}

	// done
	return c, nil
}

// Proper key injected at build time
var EncryptionKey string = "dummy"

type xferConfig struct {
	ServerEndpoint string
}

var Version = "development"
var PS = ""

func init() {
	if runtime.GOOS == "windows" {
		PS = "\\"
	} else {
		PS = "/"
	}
}

func main() {

	switch len(os.Args) {
	case 1:
		// help
		fmt.Printf("transfer.sh (version %s) - help:\n\nSimply pass in the filename you wish to upload!\n", Version)
	case 2:
		// get argument
		a := os.Args[1]

		// check if special command
		if strings.ToLower(a) == "/reset" {
			err := configDelete()
			if err != nil {
				log.Fatalf("Error resetting configuration file: %s", err)
			}
			fmt.Println("Configuration reset!")
			os.Exit(0)
		}

		// upload file
		link, token, err := upload(a)
		if err != nil {
			log.Fatalf("Failed to upload file: %s", err)
		}

		fmt.Printf("\nLink: %s\nDelete token: %s\n", link, token)
	default:
		log.Fatalf("Only one parameter expected, I.E. the file name to upload")
	}

}

// upload
// returns link, token and error
func upload(fpath string) (string, string, error) {

	c, err := loadConfig()
	if err != nil {
		return "", "", err
	}

	// find out filename
	f := path.Base(fpath)

	// set endpoint
	ep := c.ServerEndpoint + f

	data, err := os.Open(path.Dir(fpath) + PS + f)
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	req, err := http.NewRequest("PUT", ep, data)
	if err != nil {
		return "", "", err
	}
	// req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	// non-2xx
	if res.StatusCode < 200 && res.StatusCode >= 300 {
		log.Fatalf("received invalid status code from server %d (%s)", res.StatusCode, http.StatusText(res.StatusCode))
	}

	// 2xx
	hdr := res.Header.Get("x-url-delete")
	tmp := strings.Split(hdr, "/")
	if len(tmp) > 1 {
		token := tmp[len(tmp)-1]
		link := hdr[0 : len(hdr)-len(token)-1]
		return link, token, nil
	} else {
		return "", "", errors.New("invalid or missing x-url-delete response from server")
	}

}
