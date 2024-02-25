package plugin

import (
	"io"
	"log"
	"os"

	"github.com/mailru/easyjson"
)

func Main(m func(in *PluginContext) (out []byte, err error)) {
	log.SetOutput(os.Stderr)

	rawBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	inp := PluginContext{}
	err = easyjson.Unmarshal(rawBytes, &inp)
	if err != nil {
		log.Fatal(err)
	}

	rawBytes, err = m(&inp)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(rawBytes)
}
