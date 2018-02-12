package ffmpeg

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ProbeContainer struct {
	Streams []ProbeStream `json:"streams"`
	Format  ProbeFormat   `json:"format"`
}

type ProbeStream struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name"`
	Profile       string            `json:"profile"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	CodecType     string            `json:"codec_type"`
	BitRate       string            `json:"bit_rate"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	Tags          map[string]string `json:"tags"`
}

type ProbeFormat struct {
	Filename         string            `json:"filename"`
	NBStreams        int               `json:"nb_streams"`
	NBPrograms       int               `json:"nb_programs"`
	FormatName       string            `json:"format_name"`
	FormatLongName   string            `json:"format_long_name"`
	StartTimeSeconds float64           `json:"start_time,string"`
	DurationSeconds  float64           `json:"duration,string"`
	Size             uint64            `json:"size,string"`
	BitRate          uint64            `json:"bit_rate,string"`
	ProbeScore       float64           `json:"probe_score"`
	Tags             map[string]string `json:"tags"`
}

func (f ProbeFormat) StartTime() time.Duration {
	return time.Duration(f.StartTimeSeconds * float64(time.Second))
}

func (f ProbeFormat) Duration() time.Duration {
	return time.Duration(f.DurationSeconds * float64(time.Second))
}

type ProbeData struct {
	Format *ProbeFormat `json:"format,omitempty"`
}

func Probe(filename string) (*ProbeContainer, error) {
	cmd := exec.Command("ffprobe", "-show_format", "-show_streams", filename, "-print_format", "json", "-v", "quiet")
	cmd.Stderr = os.Stderr

	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var v ProbeContainer
	err = json.NewDecoder(r).Decode(&v)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// ProbeKeyframes scans for keyframes in a file and returns a list of timestamps at which keyframes were found.
func ProbeKeyframes(filename string) ([]time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-select_streams", "v",
		"-show_entries", "packet=pts_time,flags",
		"-v", "quiet",
		"-of", "csv",
		filename)
	cmd.Stderr = os.Stderr

	rawReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	keyframes := []time.Duration{}
	scanner := bufio.NewScanner(rawReader)
	for scanner.Scan() {
		// Each line has the format "packet,4.223000,K_"
		line := strings.Split(scanner.Text(), ",")
		if line[2][0] == 'K' {
			pts, err := strconv.ParseFloat(line[1], 64)
			if err != nil {
				return nil, err
			}
			keyframes = append(keyframes, time.Duration(pts*float64(time.Second)))
		}
	}

	return keyframes, nil
}
