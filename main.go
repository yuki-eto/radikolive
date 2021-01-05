package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/yyoshiki41/go-radiko"
)

func main() {
	var (
		user   string
		pass   string
		areaID string
		isList bool
	)
	flag.StringVar(&user, "user", "", "Username")
	flag.StringVar(&pass, "pass", "", "Password")
	flag.StringVar(&areaID, "area", "", "AreaID")
	flag.BoolVar(&isList, "list", false, "Show station list")
	flag.Parse()

	client, err := radiko.New("")
	if err != nil {
		log.Fatalf("%+v", err)
	}

	if areaID != "" {
		client.SetAreaID(areaID)
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

	playlistURL := fmt.Sprintf("http://c-radiko.smartstream.ne.jp/%s/_definst_/simul-stream.stream/playlist.m3u8", station)
	log.Printf("m3u8: %s", playlistURL)

	ticker := time.NewTicker(time.Second * 10)
	go func() {
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

	ffplay := exec.Command(
		"ffmpeg",
		"-headers", fmt.Sprintf("'X-Radiko-Authtoken: %s\r\n'", authToken),
		"-i", playlistURL,
		"-f", "wav",
		"-hls_allow_cache", "0",
		"-live_start_index", "-99999",
		"-",
		"|",
		"ffplay",
		"-i", "-",
	)
	cmd := exec.Command(
		"bash",
		"-c",
		ffplay.String(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
