package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

var (
	conf    = flag.String("conf", os.Getenv("HOME")+"/.melt", "Path to the config")
	meltKey = flag.String("key", "", "Define the key of the link to send to melt host")
	head    = flag.String("head", "", "Specify a header that will be in top of the input")
	hip     = flag.Bool("hip", false, "Enable Use of HipChat")
	hipRoom = flag.String("room", "", "HipChat: Name of the room to send the input to on HipChat")
	//hipTitle = flag.String("title", "", "HipChat: Title of your message in HipChat")
	hipMelt = flag.Bool("melt", false, "HipChat: Melt content and give a link to content")

	//	command      = flag.String("command", "", "Enter a command to execute")
)

const (
	MeltDefault    = "https://melt.grindvoll.org"
	HipChatDefault = "https://api.hipchat.com/v2"
)

func init() {
	flag.Parse()
}

type ConfigData struct {
	MeltHost     string
	HipChatHost  string
	HipChatToken string
}

type MeltResponse struct {
	Key     string `json:"key"`
	Data    string `json:"data"`
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

//	{"message":"'"$MESSAGE"'","message_format":"html","color":"yellow","notify":false}'
type HipChatData struct {
	Format  string `json:"message_format"`
	Color   string `json:"color"`
	Notify  bool   `json:"notify"`
	Message string `json:"message"`
}

type HipResponse map[string]interface{}

/*type HipResponse struct {
	Code    int
	Message string
	Type    string
}
*/

func readConfig(f string) *ConfigData {
	c := new(ConfigData)
	var file *os.File
	defer file.Close()

	file, err := os.Open(f)
	if os.IsNotExist(err) {
		c.MeltHost = MeltDefault
		c.HipChatHost = HipChatDefault

		file, err := os.Create(f)
		if err != nil {
			log.Fatalln(err)
		}

		fEnc := json.NewEncoder(file)
		fEnc.Encode(&c)

	} else if err != nil {
		log.Fatalln(err)
	} else {

		cDec := json.NewDecoder(file)
		err = cDec.Decode(c)
		if err != nil {
			log.Fatalln(err)
		}
	}

	return c
}

// readStdin will read data from stdin so we can safely output the data in json later

func readStdin() []byte {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln(err)
	}
	return in
}

func addHead(head string) io.Reader {
	byteHead := []byte(head + "\n")
	return io.MultiReader(bytes.NewReader(byteHead), os.Stdin)

}

func meltPost(key, head string, c *ConfigData) (url string) {
	resp := new(http.Response)
	var postMeltTo string
	var Data io.Reader

	if key == "" {
		postMeltTo = c.MeltHost + "/documents"

	} else {
		postMeltTo = c.MeltHost + "/documents/custom/" + key
	}

	if head != "" {
		Data = addHead(head)
	} else {
		Data = os.Stdin

	}

	resp, err := http.Post(postMeltTo, "text/plain", Data)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalln(err)
	}
	rDec := json.NewDecoder(resp.Body)
	m := new(MeltResponse)

	err = rDec.Decode(m)
	if err != nil {
		log.Fatalln(err)
	}

	if m.Ok != true {
		log.Fatalln("Error:", m.Message)
	}

	url = c.MeltHost + "/" + m.Key
	return
}

func hipRoomPost(room, key, head string, short bool, c *ConfigData) {
	resp := new(http.Response)
	var hip HipChatData
	var vHip = make(url.Values)
	vHip.Add("auth_token", c.HipChatToken)

	postHipTo, err := url.Parse(c.HipChatHost + "/room/" + room + "/notification")
	if err != nil {
		log.Fatalln(err)
	}
	postHipTo.RawQuery = vHip.Encode()

	hip.Color = "yellow"
	hip.Format = "text"
	hip.Notify = false

	if head != "" {
		hip.Message = fmt.Sprintf("%s\n", head)
	}

	if short {
		hip.Message += meltPost(key, head, c)
	} else {
		s, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalln(err)
		}
		hip.Message += string(s)
	}

	bHip, err := json.Marshal(&hip)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err = http.Post(postHipTo.String(), "application/json", bytes.NewReader(bHip))

	defer resp.Body.Close()
	if err != nil {
		log.Fatalln(err)
	}

	rDec := json.NewDecoder(resp.Body)
	h := make(HipResponse)

	rDec.Decode(&h)

	//	fmt.Println(postHipTo)
	if len(h) > 0 {
		fmt.Println(h)
	}

}

func main() {

	Config := readConfig(*conf)
	//fmt.Printf("%s", Config.HipChatToken)
	//fmt.Println(getRoom())
	//	fmt.Println(string(readStdin()))
	if *hip {
		if *hipRoom != "" {
			hipRoomPost(*hipRoom, *meltKey, *head, *hipMelt, Config)
		} else {
			log.Fatalln("HipChat: No room specified")
			flag.PrintDefaults()
		}

	} else {
		fmt.Println(meltPost(*meltKey, *head, Config))
	}
}
