package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/yyoshiki41/go-radiko"
)

func main() {
	var (
		user   string
		pass   string
		isList bool
	)
	flag.StringVar(&user, "user", "", "username")
	flag.StringVar(&pass, "pass", "", "password")
	flag.BoolVar(&isList, "list", false, "show station list")
	flag.Parse()

	client, err := radiko.New("")
	if err != nil {
		log.Fatalf("%+v", err)
	}

	if isList {
		infos, err := getStationPrograms(client)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		for _, s := range infos {
			log.Print(s)
		}
		return
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

	station := flag.Arg(0)
	if station == "" {
		log.Print("please select station ID")
		return
	}

	info, err := getStationProgram(client, station)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	log.Print(info)

	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			newInfo, err := getStationProgram(client, station)
			if err != nil {
				log.Printf("%+v", err)
				continue
			}
			if info != newInfo {
				log.Print(newInfo)
				info = newInfo
			}
		}
	}()

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

	if err := cmd.Run(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func getStationProgram(client *radiko.Client, id string) (string, error) {
	stations, err := client.GetNowPrograms(context.Background())
	if err != nil {
		return "", err
	}
	for _, s := range stations {
		if id != s.ID {
			continue
		}
		return getStationString(s), nil
	}
	return "", errors.New("cannot find station")
}

func getStationPrograms(client *radiko.Client) ([]string, error) {
	stations, err := client.GetNowPrograms(context.Background())
	if err != nil {
		return nil, err
	}
	var infos []string
	for _, s := range stations {
		infos = append(infos, getStationString(s))
	}
	return infos, nil
}

func getStationString(s radiko.Station) string {
	prog := s.Scd.Progs.Progs[0]
	const parseFormat = "20060102150405"
	const displayFormat = "15:04:05"
	from, _ := time.Parse(parseFormat, prog.Ft)
	to, _ := time.Parse(parseFormat, prog.To)
	return fmt.Sprintf(
		"%s: %s / %s [%s - %s]",
		s.ID,
		s.Name,
		prog.Title,
		from.Format(displayFormat),
		to.Format(displayFormat),
	)
}
