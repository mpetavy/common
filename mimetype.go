package common

import (
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"
	"os"
	"path/filepath"
	"strings"
)

const (
	MIMETYPE_HEADER_LEN = 1024
)

type MIMETYPE_EXTENSION struct {
	MimeType, Ext string
}

var MIMETYPES []MIMETYPE_EXTENSION

var (
	MIMETYPE_APPLICATION_DICOM,
	MIMETYPE_APPLICATION_EPUB_ZIP,
	MIMETYPE_APPLICATION_GEO_JSON,
	MIMETYPE_APPLICATION_GML_XML,
	MIMETYPE_APPLICATION_GPX_XML,
	MIMETYPE_APPLICATION_GZIP,
	MIMETYPE_APPLICATION_JAR,
	MIMETYPE_APPLICATION_JAVASCRIPT,
	MIMETYPE_APPLICATION_JSON,
	MIMETYPE_APPLICATION_MSWORD,
	MIMETYPE_APPLICATION_OCTET_STREAM,
	MIMETYPE_APPLICATION_OGG,
	MIMETYPE_APPLICATION_PDF,
	MIMETYPE_APPLICATION_POSTSCRIPT,
	MIMETYPE_APPLICATION_VND_GARMIN_TCX_XML,
	MIMETYPE_APPLICATION_VND_GOOGLE_EARTH_KML_XML,
	MIMETYPE_APPLICATION_VND_MS_EXCEL,
	MIMETYPE_APPLICATION_VND_MS_POWERPOINT,
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_PRESENTATIONML_PRESENTATION,
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_SPREADSHEETML_SHEET,
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_WORDPROCESSINGML_DOCUMENT,
	MIMETYPE_APPLICATION_WASM,
	MIMETYPE_APPLICATION_X_7Z_COMPRESSED,
	MIMETYPE_APPLICATION_X_CHROME_EXTENSION,
	MIMETYPE_APPLICATION_X_JAVA_APPLET,
	MIMETYPE_APPLICATION_X_PHOTOSHOP,
	MIMETYPE_APPLICATION_X_PYTHON,
	MIMETYPE_APPLICATION_X_SHOCKWAVE_FLASH,
	MIMETYPE_APPLICATION_X_TAR,
	MIMETYPE_APPLICATION_ZIP,
	MIMETYPE_AUDIO_AIFF,
	MIMETYPE_AUDIO_AMR,
	MIMETYPE_AUDIO_APE,
	MIMETYPE_AUDIO_BASIC,
	MIMETYPE_AUDIO_FLAC,
	MIMETYPE_AUDIO_MIDI,
	MIMETYPE_AUDIO_MP4,
	MIMETYPE_AUDIO_MPEG,
	MIMETYPE_AUDIO_MUSEPACK,
	MIMETYPE_AUDIO_WAV,
	MIMETYPE_AUDIO_X_M4A,
	MIMETYPE_FONT_WOFF,
	MIMETYPE_FONT_WOFF2,
	MIMETYPE_IMAGE_BMP,
	MIMETYPE_IMAGE_GIF,
	MIMETYPE_IMAGE_JPEG,
	MIMETYPE_IMAGE_PNG,
	MIMETYPE_IMAGE_SVG_XML,
	MIMETYPE_IMAGE_TIFF,
	MIMETYPE_IMAGE_WEBP,
	MIMETYPE_IMAGE_X_ICON,
	MIMETYPE_MODEL_VND_COLLADA_XML,
	MIMETYPE_MODEL_X3D_XML,
	MIMETYPE_TEXT_HTML,
	MIMETYPE_TEXT_CSS,
	MIMETYPE_TEXT_PLAIN,
	MIMETYPE_TEXT_RTF,
	MIMETYPE_TEXT_XML,
	MIMETYPE_TEXT_X_LUA,
	MIMETYPE_TEXT_X_PERL,
	MIMETYPE_TEXT_X_PHP,
	MIMETYPE_TEXT_X_TCL,
	MIMETYPE_VIDEO_3GPP,
	MIMETYPE_VIDEO_3GPP2,
	MIMETYPE_VIDEO_MP4,
	MIMETYPE_VIDEO_MPEG,
	MIMETYPE_VIDEO_QUICKTIME,
	MIMETYPE_VIDEO_WEBM,
	MIMETYPE_VIDEO_X_FLV,
	MIMETYPE_VIDEO_X_MATROSKA,
	MIMETYPE_VIDEO_X_MSVIDEO MIMETYPE_EXTENSION
)

func init() {
	MIMETYPE_APPLICATION_DICOM = registerMimeType("application/dicom", "dcm")
	MIMETYPE_APPLICATION_EPUB_ZIP = registerMimeType("application/epub+zip", "epub")
	MIMETYPE_APPLICATION_GEO_JSON = registerMimeType("application/geo+json", "geojson")
	MIMETYPE_APPLICATION_GML_XML = registerMimeType("application/gml+xml", "gml")
	MIMETYPE_APPLICATION_GPX_XML = registerMimeType("application/gpx+xml", "gpx")
	MIMETYPE_APPLICATION_GZIP = registerMimeType("application/gzip", "gz")
	MIMETYPE_APPLICATION_JAR = registerMimeType("application/jar", "jar")
	MIMETYPE_APPLICATION_JAVASCRIPT = registerMimeType("application/javascript", "js")
	MIMETYPE_APPLICATION_JSON = registerMimeType("application/json", "json")
	MIMETYPE_APPLICATION_MSWORD = registerMimeType("application/msword", "doc")
	MIMETYPE_APPLICATION_OCTET_STREAM = registerMimeType("application/octet-stream", "")
	MIMETYPE_APPLICATION_OGG = registerMimeType("application/ogg", "ogg")
	MIMETYPE_APPLICATION_PDF = registerMimeType("application/pdf", "pdf")
	MIMETYPE_APPLICATION_POSTSCRIPT = registerMimeType("application/postscript", "ps")
	MIMETYPE_APPLICATION_VND_GARMIN_TCX_XML = registerMimeType("application/vnd.garmin.tcx+xml", "tcx")
	MIMETYPE_APPLICATION_VND_GOOGLE_EARTH_KML_XML = registerMimeType("application/vnd.google-earth.kml+xml", "kml")
	MIMETYPE_APPLICATION_VND_MS_EXCEL = registerMimeType("application/vnd.ms-excel", "xls")
	MIMETYPE_APPLICATION_VND_MS_POWERPOINT = registerMimeType("application/vnd.ms-powerpoint", "ppt")
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_PRESENTATIONML_PRESENTATION = registerMimeType("application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx")
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_SPREADSHEETML_SHEET = registerMimeType("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx")
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_WORDPROCESSINGML_DOCUMENT = registerMimeType("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx")
	MIMETYPE_APPLICATION_WASM = registerMimeType("application/wasm", "wasm")
	MIMETYPE_APPLICATION_X_7Z_COMPRESSED = registerMimeType("application/x-7z-compressed", "7z")
	MIMETYPE_APPLICATION_X_CHROME_EXTENSION = registerMimeType("application/x-chrome-extension", "crx")
	MIMETYPE_APPLICATION_X_JAVA_APPLET = registerMimeType("application/x-java-applet; charset=binary", "class")
	MIMETYPE_APPLICATION_X_PHOTOSHOP = registerMimeType("application/x-photoshop", "psd")
	MIMETYPE_APPLICATION_X_PYTHON = registerMimeType("application/x-python", "py")
	MIMETYPE_APPLICATION_X_SHOCKWAVE_FLASH = registerMimeType("application/x-shockwave-flash", "swf")
	MIMETYPE_APPLICATION_X_TAR = registerMimeType("application/x-tar", "tar")
	MIMETYPE_APPLICATION_ZIP = registerMimeType("application/zip", "zip")
	MIMETYPE_AUDIO_AIFF = registerMimeType("audio/aiff", "aiff")
	MIMETYPE_AUDIO_AMR = registerMimeType("audio/amr", "amr")
	MIMETYPE_AUDIO_APE = registerMimeType("audio/ape", "ape")
	MIMETYPE_AUDIO_BASIC = registerMimeType("audio/basic", "au")
	MIMETYPE_AUDIO_FLAC = registerMimeType("audio/flac", "flac")
	MIMETYPE_AUDIO_MIDI = registerMimeType("audio/midi", "midi")
	MIMETYPE_AUDIO_MP4 = registerMimeType("audio/mp4", "mp4")
	MIMETYPE_AUDIO_MPEG = registerMimeType("audio/mpeg", "mp3")
	MIMETYPE_AUDIO_MUSEPACK = registerMimeType("audio/musepack", "mpc")
	MIMETYPE_AUDIO_WAV = registerMimeType("audio/wav", "wav")
	MIMETYPE_AUDIO_X_M4A = registerMimeType("audio/x-m4a", "m4a")
	MIMETYPE_FONT_WOFF = registerMimeType("font/woff", "woff")
	MIMETYPE_FONT_WOFF2 = registerMimeType("font/woff2", "woff2")
	MIMETYPE_IMAGE_BMP = registerMimeType("image/bmp", "bmp")
	MIMETYPE_IMAGE_GIF = registerMimeType("image/gif", "gif")
	MIMETYPE_IMAGE_JPEG = registerMimeType("image/jpeg", "jpg")
	MIMETYPE_IMAGE_PNG = registerMimeType("image/png", "png")
	MIMETYPE_IMAGE_SVG_XML = registerMimeType("image/svg+xml", "svg")
	MIMETYPE_IMAGE_TIFF = registerMimeType("image/tiff", "tiff")
	MIMETYPE_IMAGE_WEBP = registerMimeType("image/webp", "webp")
	MIMETYPE_IMAGE_X_ICON = registerMimeType("image/x-icon", "ico")
	MIMETYPE_MODEL_VND_COLLADA_XML = registerMimeType("model/vnd.collada+xml", "dae")
	MIMETYPE_MODEL_X3D_XML = registerMimeType("model/x3d+xml", "x3d")
	MIMETYPE_TEXT_HTML = registerMimeType("text/html; charset=utf-8", "html")
	MIMETYPE_TEXT_PLAIN = registerMimeType("text/css", "css")
	MIMETYPE_TEXT_PLAIN = registerMimeType("text/plain", "txt")
	MIMETYPE_TEXT_RTF = registerMimeType("text/rtf", "rtf")
	MIMETYPE_TEXT_XML = registerMimeType("text/xml; charset=utf-8", "xml")
	MIMETYPE_TEXT_X_LUA = registerMimeType("text/x-lua", "lua")
	MIMETYPE_TEXT_X_PERL = registerMimeType("text/x-perl", "pl")
	MIMETYPE_TEXT_X_PHP = registerMimeType("text/x-php; charset=utf-8", "php")
	MIMETYPE_TEXT_X_TCL = registerMimeType("text/x-tcl", "tcl")
	MIMETYPE_VIDEO_3GPP = registerMimeType("video/3gpp", "3gp")
	MIMETYPE_VIDEO_3GPP2 = registerMimeType("video/3gpp2", "3g2")
	MIMETYPE_VIDEO_MP4 = registerMimeType("video/mp4", "mp4")
	MIMETYPE_VIDEO_MPEG = registerMimeType("video/mpeg", "mpeg")
	MIMETYPE_VIDEO_QUICKTIME = registerMimeType("video/quicktime", "mov")
	MIMETYPE_VIDEO_WEBM = registerMimeType("video/webm", "webm")
	MIMETYPE_VIDEO_X_FLV = registerMimeType("video/x-flv", "flv")
	MIMETYPE_VIDEO_X_MATROSKA = registerMimeType("video/x-matroska", "mkv")
	MIMETYPE_VIDEO_X_MSVIDEO = registerMimeType("video/x-msvideo", "avi")
}

func registerMimeType(mimeType, ext string) MIMETYPE_EXTENSION {
	mt := MIMETYPE_EXTENSION{
		MimeType: mimeType,
		Ext:      ext,
	}

	MIMETYPES = append(MIMETYPES, mt)

	return mt
}

func DetectMimeType(filename string, buf []byte) MIMETYPE_EXTENSION {
	if filename != "" {
		ext := filepath.Ext(filename)
		if ext != "" {
			if strings.HasPrefix(ext, ".") {
				ext = ext[1:]
			}

			ext = strings.ToLower(ext)

			for _, mt := range MIMETYPES {
				if mt.Ext == ext {
					return mt
				}
			}
		}
	}

	t, err := filetype.Match(buf)
	if t.MIME.Value != "" && err == nil {
		return MIMETYPE_EXTENSION{t.MIME.Value, t.Extension}
	}

	mime, ext := mimetype.Detect(buf)

	if len(mime) > 0 {
		return MIMETYPE_EXTENSION{mime, ext}
	}

	return MIMETYPE_APPLICATION_OCTET_STREAM
}

func ReadHeader(path string) ([]byte, error) {
	byteSlice := make([]byte, MIMETYPE_HEADER_LEN)

	file, err := os.Open(path)
	if err != nil {
		return byteSlice[0:0], err
	}
	defer file.Close()

	bytesRead, err := file.Read(byteSlice)
	if err != nil {
		return byteSlice[0:0], err
	}

	return byteSlice[:bytesRead], nil
}

func IsImageMimeType(s string) bool {
	return strings.HasPrefix(s, "image/")
}
