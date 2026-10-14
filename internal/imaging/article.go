package imaging

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"net/http"
)

const MaxArticlePixels = 24_000_000

var ErrInvalidImage = errors.New("invalid or oversized image")

// ArticleInputDimensions checks the real file signature and bounded dimensions.
// Full decoding is done by cwebp before upload; malformed payloads never reach S3.
func ArticleInputDimensions(data []byte) (int, int, string, error) {
	format := http.DetectContentType(data)
	switch format {
	case "image/jpeg", "image/png", "image/gif":
		cfg, decoded, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width < 1 || cfg.Height < 1 || decoded == "" || int64(cfg.Width)*int64(cfg.Height) > MaxArticlePixels {
			return 0, 0, "", ErrInvalidImage
		}
		w, h := cfg.Width, cfg.Height
		if format == "image/jpeg" {
			switch jpegOrientation(data) {
			case 5, 6, 7, 8:
				w, h = h, w
			}
		}
		return w, h, format, nil
	case "image/webp":
		w, h, err := WebPDimensions(data)
		if err != nil || int64(w)*int64(h) > MaxArticlePixels {
			return 0, 0, "", ErrInvalidImage
		}
		return w, h, format, nil
	default:
		return 0, 0, "", ErrInvalidImage
	}
}

// NormalizeArticleInput applies JPEG EXIF orientation before cwebp, which
// otherwise encodes the unrotated pixel matrix.
func NormalizeArticleInput(data []byte) ([]byte, int, int, error) {
	w, h, format, err := ArticleInputDimensions(data)
	if err != nil {
		return nil, 0, 0, err
	}
	if format == "image/gif" {
		frame, err := gif.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, ErrInvalidImage
		}
		var out bytes.Buffer
		if err := png.Encode(&out, frame); err != nil {
			return nil, 0, 0, err
		}
		return out.Bytes(), w, h, nil
	}
	if format != "image/jpeg" {
		return data, w, h, nil
	}
	orientation := jpegOrientation(data)
	if orientation < 2 || orientation > 8 {
		return data, w, h, nil
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, ErrInvalidImage
	}
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			var dx, dy int
			switch orientation {
			case 2:
				dx, dy = sw-1-x, y
			case 3:
				dx, dy = sw-1-x, sh-1-y
			case 4:
				dx, dy = x, sh-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = sh-1-y, x
			case 7:
				dx, dy = sh-1-y, sw-1-x
			case 8:
				dx, dy = y, sw-1-x
			}
			dst.Set(dx, dy, src.At(src.Bounds().Min.X+x, src.Bounds().Min.Y+y))
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, 0, 0, err
	}
	return out.Bytes(), w, h, nil
}

func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}
	for pos := 2; pos+4 <= len(data) && data[pos] == 0xff; {
		marker := data[pos+1]
		if marker == 0xda || marker == 0xd9 {
			break
		}
		length := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if length < 2 || pos+2+length > len(data) {
			break
		}
		if marker == 0xe1 {
			app := data[pos+4 : pos+2+length]
			if len(app) >= 14 && bytes.Equal(app[:6], []byte("Exif\x00\x00")) {
				tiff := app[6:]
				var order binary.ByteOrder
				switch string(tiff[:2]) {
				case "II":
					order = binary.LittleEndian
				case "MM":
					order = binary.BigEndian
				}
				if order != nil && order.Uint16(tiff[2:4]) == 42 {
					offset := int(order.Uint32(tiff[4:8]))
					if offset >= 0 && offset+2 <= len(tiff) {
						count := int(order.Uint16(tiff[offset : offset+2]))
						for i := 0; i < count; i++ {
							at := offset + 2 + i*12
							if at+12 > len(tiff) {
								break
							}
							if order.Uint16(tiff[at:at+2]) == 0x112 && order.Uint16(tiff[at+2:at+4]) == 3 && order.Uint32(tiff[at+4:at+8]) == 1 {
								v := int(order.Uint16(tiff[at+8 : at+10]))
								if v >= 1 && v <= 8 {
									return v
								}
							}
						}
					}
				}
			}
		}
		pos += 2 + length
	}
	return 1
}

// WebPDimensions supports VP8, VP8L and VP8X RIFF chunks. cwebp verifies
// decodability on upload; the verifier also checks the stored object's header.
func WebPDimensions(b []byte) (int, int, error) {
	if len(b) < 30 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" || int64(binary.LittleEndian.Uint32(b[4:8]))+8 != int64(len(b)) {
		return 0, 0, ErrInvalidImage
	}
	size := int(binary.LittleEndian.Uint32(b[16:20]))
	if size < 10 || size > len(b)-20 {
		return 0, 0, ErrInvalidImage
	}
	p := b[20 : 20+size]
	var w, h int
	switch string(b[12:16]) {
	case "VP8 ":
		if len(p) < 10 || !bytes.Equal(p[3:6], []byte{0x9d, 0x01, 0x2a}) {
			return 0, 0, ErrInvalidImage
		}
		w = int(binary.LittleEndian.Uint16(p[6:8]) & 0x3fff)
		h = int(binary.LittleEndian.Uint16(p[8:10]) & 0x3fff)
	case "VP8L":
		if p[0] != 0x2f {
			return 0, 0, ErrInvalidImage
		}
		w = int(1 + uint32(p[1]) + uint32(p[2]&0x3f)<<8)
		h = int(1 + uint32(p[2]>>6) + uint32(p[3])<<2 + uint32(p[4]&0x0f)<<10)
	case "VP8X":
		w = 1 + int(p[4]) + int(p[5])<<8 + int(p[6])<<16
		h = 1 + int(p[7]) + int(p[8])<<8 + int(p[9])<<16
	default:
		return 0, 0, ErrInvalidImage
	}
	if w < 1 || h < 1 || int64(w)*int64(h) > MaxArticlePixels {
		return 0, 0, ErrInvalidImage
	}
	return w, h, nil
}

func IsSixteenNine(w, h int) bool { return w > 0 && h > 0 && int64(w)*9 == int64(h)*16 }
