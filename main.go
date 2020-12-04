package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/yuki-eto/go-radiko"
)

func main() {
	var (
		user string
		pass string
		station string
	)
	flag.StringVar(&user, "user", "", "username")
	flag.StringVar(&pass, "pass", "", "password")
	flag.StringVar(&station, "station", "LFR", "station code")
	flag.Parse()

	client, err := radiko.New("")
	if err != nil {
		log.Fatalf("%+v", err)
	}

	if user != "" && pass != "" {
		login, err := client.Login(context.Background(), user, pass)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		if login.StatusCode() != 200 {
			log.Fatalf("Failed to login premium member.\nInvalid status code: %d",
				login.StatusCode())
		}
	}

	authToken, err := client.AuthorizeToken(context.Background())
	if err != nil {
		log.Fatalf("%+v", err)
	}

	mplayer := exec.Command(
		"mplayer",
		"-cache", "32",
		"-http-header-fields", fmt.Sprintf(`"X-Radiko-Authtoken: %s"`, authToken),
		fmt.Sprintf("http://c-radiko.smartstream.ne.jp/%s/_definst_/simul-stream.stream/playlist.m3u8", station),
	)

	cmd := exec.Command(
		"bash",
		"-c",
		mplayer.String(),
	)
	log.Print(cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer stdout.Close()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer stderr.Close()

	readerFunc := func(reader io.Reader) {
		bufReader := bufio.NewReader(reader)
		for {
			msg, err := bufReader.ReadString('\n')
			if err != nil {
				return
			}
			msg = strings.TrimSpace(msg)
			fmt.Println(msg)
		}
	}
	go readerFunc(stdout)
	go readerFunc(stderr)

	if err := cmd.Run(); err != nil {
		log.Fatalf("%+v", err)
	}
}
