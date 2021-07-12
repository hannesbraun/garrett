package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/dh1tw/gosamplerate"
	"github.com/gabriel-vasile/mimetype"
	"github.com/go-audio/aiff"
	"github.com/go-audio/audio"
	"github.com/hajimehoshi/go-mp3"
	"github.com/mewkiz/flac"
	"github.com/youpy/go-wav"
	"io"
	"math"
	"os"
	"path"
	"strings"
)

// Updates the status label
func updateStatus(statusLabel **widget.Label, text string) {
	if len(text) > 95 {
		// Crop the text to avoid resizing the window
		(*statusLabel).SetText(text[:92] + "...")
	} else {
		(*statusLabel).SetText(text)
	}
}

// Runs the conversions
func convert(files []string, outDir string, sampleRate float64, progress *binding.ExternalFloat, statusLabel **widget.Label) []string {
	failed := make([]string, 0)
	if len(files) <= 0 {
		// No files; nothing to do
		return failed
	}
	progressStep := 1.0 / (3.0 * float64(len(files)))

	for i, file := range files {
		// Update progress
		_ = (*progress).Set(float64(3*i) * progressStep)
		updateStatus(statusLabel, "Decoding "+file)

		// Detect file type
		mimeType, err := mimetype.DetectFile(file)
		if err != nil {
			failed = append(failed, file)
			continue
		}

		// Decode the file
		var track Track
		switch mimeType.String() {
		case "audio/mpeg": // mp3
			track, err = decodeMp3(file)
			if err != nil {
				failed = append(failed, file)
				continue
			}
		case "audio/flac": // flac
			track, err = decodeFlac(file)
			if err != nil {
				failed = append(failed, file)
				continue
			}
		case "audio/wav": // wav
			track, err = decodeWav(file)
			if err != nil {
				failed = append(failed, file)
				continue
			}
		case "audio/aiff": // aiff
			track, err = decodeAiff(file)
			if err != nil {
				failed = append(failed, file)
				continue
			}
		default:
			failed = append(failed, file)
			continue
		}

		_ = (*progress).Set(float64(3*i+1) * progressStep)
		updateStatus(statusLabel, "Resampling "+file)

		// Resample (if necessary)
		resampled := track.data
		if sampleRate != track.sampleRate {
			resampled, err = gosamplerate.Simple(track.data, sampleRate/track.sampleRate, int(track.channels), gosamplerate.SRC_SINC_BEST_QUALITY)
			if err != nil {
				failed = append(failed, file)
				continue
			}
		}

		_ = (*progress).Set(float64(3*i+2) * progressStep)
		updateStatus(statusLabel, "Assembling wave samples")

		// Assemble wav samples
		var samples []wav.Sample
		for i := 0; i < len(resampled); i += 2 {
			var sample [2]int
			// Clipping is necessary since the signal may leave the default range from -1.0 to 1.0 while resampling
			sample[0] = clip16Bit(int(resampled[i] * math.MaxInt16))
			if i+1 < len(resampled) {
				sample[1] = clip16Bit(int(resampled[i+1] * math.MaxInt16))
			} else {
				sample[1] = 0
			}

			samples = append(samples, wav.Sample{Values: sample})
		}

		// Figure out the destination path
		baseName := path.Base(file)
		indexSuffix := strings.LastIndex(baseName, ".")
		if indexSuffix > 0 {
			baseName = baseName[0:indexSuffix]
		}

		updateStatus(statusLabel, "Writing "+path.Join(outDir, baseName+".wav"))

		// Write wav file
		out, err := os.Create(path.Join(outDir, baseName+".wav"))
		if err != nil {
			failed = append(failed, file)
			continue
		}
		wavOut := bufio.NewWriter(out)
		writer := wav.NewWriter(wavOut, uint32(len(samples)), track.channels, uint32(sampleRate), 16)
		err = writer.WriteSamples(samples)
		_ = out.Close()
		if err != nil {
			failed = append(failed, file)
			continue
		}

		_ = out.Close()
	}

	_ = (*progress).Set(1.0)
	updateStatus(statusLabel, "Idle")

	return failed
}

func clip16Bit(sample int) int {
	if sample < -32768 {
		return -32768
	} else if sample > 32767 {
		return 32767
	} else {
		return sample
	}
}

type Track struct {
	data       []float32
	sampleRate float64
	channels   uint16
}

func decodeMp3(file string) (Track, error) {
	f, err := os.Open(file)
	if err != nil {
		return Track{}, err
	}
	defer f.Close()

	d, err := mp3.NewDecoder(f)
	if err != nil {
		return Track{}, err
	}

	buf := make([]byte, d.Length())
	i := int64(0)
	for {
		read, err := d.Read(buf[i:])
		if err == io.EOF {
			break
		} else if err != nil {
			return Track{}, err
		}
		i += int64(read)
	}

	floatBuf := make([]float32, d.Length()/2)
	for i = 0; i < d.Length(); i += 2 {
		var sample int16
		_ = binary.Read(bytes.NewBuffer(buf[i:i+2]), binary.LittleEndian, &sample)
		floatBuf[i/2] = float32(sample) / math.MaxInt16
	}

	sampleRate := float64(d.SampleRate())

	return Track{
		data:       floatBuf,
		sampleRate: sampleRate,
		channels:   2,
	}, nil
}

func decodeFlac(file string) (Track, error) {
	stream, err := flac.Open(file)
	if err != nil {
		return Track{}, err
	}
	defer stream.Close()

	sampleRate := uint32(44100)
	dualChannel := false
	buf := make([]float32, 0)
	for {
		frame, err := stream.ParseNext()
		if err == io.EOF {
			break
		} else if err != nil {
			return Track{}, err
		}

		frameBuf := make([]float32, 0)
		maxIntVal := 1 << (frame.BitsPerSample - 1)
		for i := 0; i < frame.Subframes[0].NSamples; i++ {
			frameBuf = append(frameBuf, float32(frame.Subframes[0].Samples[i])/float32(maxIntVal))
			if len(frame.Subframes) > 1 {
				frameBuf = append(frameBuf, float32(frame.Subframes[1].Samples[i])/float32(maxIntVal))
				dualChannel = true
			}
		}
		buf = append(buf, frameBuf...)
		sampleRate = frame.SampleRate
	}

	channels := uint16(2)
	if !dualChannel {
		channels = 1
	}

	return Track{
		data:       buf,
		sampleRate: float64(sampleRate),
		channels:   channels,
	}, nil
}

func decodeWav(file string) (Track, error) {
	f, err := os.Open(file)
	if err != nil {
		return Track{}, err
	}
	defer f.Close()

	reader := wav.NewReader(f)
	format, err := reader.Format()
	if err != nil {
		return Track{}, err
	}
	channels := format.NumChannels

	floatBuf := make([]float32, 0)
	for {
		// Reading samples
		samples, err := reader.ReadSamples()
		if err == io.EOF {
			break
		}

		// Converting to float32
		for _, sample := range samples {
			// Iterate over channels
			for i := uint(0); i < 2; i++ {
				floatBuf = append(floatBuf, float32(reader.FloatValue(sample, i)))
			}
		}
	}

	outChannels := uint16(2)
	if channels == 1 {
		outChannels = 1
	}

	return Track{
		data:       floatBuf,
		sampleRate: float64(format.SampleRate),
		channels:   outChannels,
	}, nil
}

func decodeAiff(file string) (Track, error) {
	f, err := os.Open(file)
	if err != nil {
		return Track{}, err
	}
	defer f.Close()

	decoder := aiff.NewDecoder(io.ReadSeeker(f))
	if !decoder.IsValidFile() {
		return Track{}, errors.New("invalid aiff file")
	}

	floatBuf := make([]float32, 0)
	intBuf := make([]int, 255)
	ch := 0
	buf := &audio.IntBuffer{Data: intBuf}
	for {
		n, err := decoder.PCMBuffer(buf)
		if n == 0 {
			break
		}
		if err != nil {
			return Track{}, err
		}

		for _, val := range buf.AsFloat32Buffer().Data {
			if ch < 2 {
				floatBuf = append(floatBuf, val)
			}
			ch = (ch + 1) % int(decoder.NumChans)
		}
	}

	outChannels := uint16(2)
	if decoder.NumChans == 1 {
		outChannels = 1
	}

	return Track{
		data:       floatBuf,
		sampleRate: float64(decoder.SampleRate),
		channels:   outChannels,
	}, nil
}
