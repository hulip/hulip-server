package ffmpeg

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"
)

var AudioEncoderPresets = map[string]EncoderParams{
	"64k-audio":  {audioBitrate: 64000, Codecs: "mp4a.40.2"},
	"128k-audio": {audioBitrate: 128000, Codecs: "mp4a.40.2"},
}

func NewAudioTranscodingSession(
	stream StreamRepresentation,
	startTime time.Duration,
	segmentStartIndex int,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	encoderParams := stream.Representation.encoderParams

	args := []string{}
	if startTime != 0 {
		args = append(args, []string{
			// -ss being before -i is important for fast seeking
			"-ss", fmt.Sprintf("%.3f", startTime.Seconds()),
		}...)
	}

	args = append(args, []string{
		"-i", buildFfmpegUrlFromFileLocator(stream.Stream.FileLocator),
		"-copyts",
		"-start_at_zero",
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "aac", "-ac", "2", "-ab", strconv.Itoa(encoderParams.audioBitrate),
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentStartIndex),
		"-hls_time", fmt.Sprintf("%.3f", SegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
	}...)

	// If we are not starting with the first segment, indicate that the fragment
	// is discontinuous
	if segmentStartIndex != 0 {
		args = append(args, []string{
			"-hls_ts_options", "movflags=dash+frag_discont",
		}...)
	} else {
		args = append(args, []string{
			"-hls_ts_options", "movflags=dash",
		}...)
	}

	// We serve our own manifest, so we don't really care about this.
	args = append(args, path.Join(outputDir, "generated_by_ffmpeg.m3u"))

	cmd := exec.Command("ffmpeg", args...)
	log.Infoln("ffmpeg started with", cmd.Args)

	logSink := getTranscodingLogSink("ffmpeg_transcode_audio")
	cmd.Stderr = logSink

	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{
		cmd:               cmd,
		Stream:            stream,
		OutputDir:         outputDir,
		SegmentStartIndex: segmentStartIndex,
	}, nil
}

func GetTranscodedAudioRepresentation(stream Stream, representationId string, encoderParams EncoderParams) StreamRepresentation {
	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: representationId,
			BitRate:          encoderParams.audioBitrate,
			Container:        "audio/mp4",
			Codecs:           encoderParams.Codecs,
			Transcoded:       true,
		},
	}
}

func buildAudioSegmentDurations(interval Interval, segmentDuration time.Duration) [][]Segment {
	sessions := [][]Segment{}

	session := []Segment{}
	currentTimestamp := interval.StartTimestamp
	segmentId := 0

	segmentDurationDts := DtsTimestamp(segmentDuration.Seconds() * float64(interval.TimeBase))

	for currentTimestamp < interval.EndTimestamp {
		if len(session) >= segmentsPerSession {
			sessions = append(sessions, session)
			session = []Segment{}
		}

		session = append(session, Segment{
			Interval{
				interval.TimeBase,
				currentTimestamp,
				currentTimestamp + segmentDurationDts,
			},
			segmentId,
		})

		segmentId++
		currentTimestamp += segmentDurationDts
	}

	// Append the last segment to the end of the interval
	session = append(session, Segment{
		Interval{
			interval.TimeBase,
			currentTimestamp,
			interval.EndTimestamp,
		},
		segmentId,
	})
	sessions = append(sessions, session)

	return sessions

}
