package common

import (
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"
	"os"
	"strings"
)

const (
	MIMETYPE_HEADER_LEN = 1024
)

type MIMETYPE_EXTENSION struct {
	MimeType, Ext string
}

var (
	MIMETYPE_APPLICATION_DICOM                                                         = MIMETYPE_EXTENSION{"application/dicom", "dcm"}
	MIMETYPE_APPLICATION_EPUB_ZIP                                                      = MIMETYPE_EXTENSION{"application/epub+zip", "epub"}
	MIMETYPE_APPLICATION_GEO_JSON                                                      = MIMETYPE_EXTENSION{"application/geo+json", "geojson"}
	MIMETYPE_APPLICATION_GML_XML                                                       = MIMETYPE_EXTENSION{"application/gml+xml", "gml"}
	MIMETYPE_APPLICATION_GPX_XML                                                       = MIMETYPE_EXTENSION{"application/gpx+xml", "gpx"}
	MIMETYPE_APPLICATION_GZIP                                                          = MIMETYPE_EXTENSION{"application/gzip", "gz"}
	MIMETYPE_APPLICATION_JAR                                                           = MIMETYPE_EXTENSION{"application/jar", "jar"}
	MIMETYPE_APPLICATION_JAVASCRIPT                                                    = MIMETYPE_EXTENSION{"application/javascript", "js"}
	MIMETYPE_APPLICATION_JSON                                                          = MIMETYPE_EXTENSION{"application/json", "json"}
	MIMETYPE_APPLICATION_MSWORD                                                        = MIMETYPE_EXTENSION{"application/msword", "doc"}
	MIMETYPE_APPLICATION_OCTET_STREAM                                                  = MIMETYPE_EXTENSION{"application/octet-stream", ""}
	MIMETYPE_APPLICATION_OGG                                                           = MIMETYPE_EXTENSION{"application/ogg", "ogg"}
	MIMETYPE_APPLICATION_PDF                                                           = MIMETYPE_EXTENSION{"application/pdf", "pdf"}
	MIMETYPE_APPLICATION_POSTSCRIPT                                                    = MIMETYPE_EXTENSION{"application/postscript", "ps"}
	MIMETYPE_APPLICATION_VND_GARMIN_TCX_XML                                            = MIMETYPE_EXTENSION{"application/vnd.garmin.tcx+xml", "tcx"}
	MIMETYPE_APPLICATION_VND_GOOGLE_EARTH_KML_XML                                      = MIMETYPE_EXTENSION{"application/vnd.google-earth.kml+xml", "kml"}
	MIMETYPE_APPLICATION_VND_MS_EXCEL                                                  = MIMETYPE_EXTENSION{"application/vnd.ms-excel", "xls"}
	MIMETYPE_APPLICATION_VND_MS_POWERPOINT                                             = MIMETYPE_EXTENSION{"application/vnd.ms-powerpoint", "ppt"}
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_PRESENTATIONML_PRESENTATION = MIMETYPE_EXTENSION{"application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx"}
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_SPREADSHEETML_SHEET         = MIMETYPE_EXTENSION{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"}
	MIMETYPE_APPLICATION_VND_OPENXMLFORMATS_OFFICEDOCUMENT_WORDPROCESSINGML_DOCUMENT   = MIMETYPE_EXTENSION{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"}
	MIMETYPE_APPLICATION_WASM                                                          = MIMETYPE_EXTENSION{"application/wasm", "wasm"}
	MIMETYPE_APPLICATION_X_7Z_COMPRESSED                                               = MIMETYPE_EXTENSION{"application/x-7z-compressed", "7z"}
	MIMETYPE_APPLICATION_X_CHROME_EXTENSION                                            = MIMETYPE_EXTENSION{"application/x-chrome-extension", "crx"}
	MIMETYPE_APPLICATION_X_JAVA_APPLET                                                 = MIMETYPE_EXTENSION{"application/x-java-applet; charset=binary", "class"}
	MIMETYPE_APPLICATION_X_PHOTOSHOP                                                   = MIMETYPE_EXTENSION{"application/x-photoshop", "psd"}
	MIMETYPE_APPLICATION_X_PYTHON                                                      = MIMETYPE_EXTENSION{"application/x-python", "py"}
	MIMETYPE_APPLICATION_X_SHOCKWAVE_FLASH                                             = MIMETYPE_EXTENSION{"application/x-shockwave-flash", "swf"}
	MIMETYPE_APPLICATION_X_TAR                                                         = MIMETYPE_EXTENSION{"application/x-tar", "tar"}
	MIMETYPE_APPLICATION_ZIP                                                           = MIMETYPE_EXTENSION{"application/zip", "zip"}
	MIMETYPE_AUDIO_AIFF                                                                = MIMETYPE_EXTENSION{"audio/aiff", "aiff"}
	MIMETYPE_AUDIO_AMR                                                                 = MIMETYPE_EXTENSION{"audio/amr", "amr"}
	MIMETYPE_AUDIO_APE                                                                 = MIMETYPE_EXTENSION{"audio/ape", "ape"}
	MIMETYPE_AUDIO_BASIC                                                               = MIMETYPE_EXTENSION{"audio/basic", "au"}
	MIMETYPE_AUDIO_FLAC                                                                = MIMETYPE_EXTENSION{"audio/flac", "flac"}
	MIMETYPE_AUDIO_MIDI                                                                = MIMETYPE_EXTENSION{"audio/midi", "midi"}
	MIMETYPE_AUDIO_MP4                                                                 = MIMETYPE_EXTENSION{"audio/mp4", "mp4"}
	MIMETYPE_AUDIO_MPEG                                                                = MIMETYPE_EXTENSION{"audio/mpeg", "mp3"}
	MIMETYPE_AUDIO_MUSEPACK                                                            = MIMETYPE_EXTENSION{"audio/musepack", "mpc"}
	MIMETYPE_AUDIO_WAV                                                                 = MIMETYPE_EXTENSION{"audio/wav", "wav"}
	MIMETYPE_AUDIO_X_M4A                                                               = MIMETYPE_EXTENSION{"audio/x-m4a", "m4a"}
	MIMETYPE_FONT_WOFF                                                                 = MIMETYPE_EXTENSION{"font/woff", "woff"}
	MIMETYPE_FONT_WOFF2                                                                = MIMETYPE_EXTENSION{"font/woff2", "woff2"}
	MIMETYPE_IMAGE_BMP                                                                 = MIMETYPE_EXTENSION{"image/bmp", "bmp"}
	MIMETYPE_IMAGE_GIF                                                                 = MIMETYPE_EXTENSION{"image/gif", "gif"}
	MIMETYPE_IMAGE_JPEG                                                                = MIMETYPE_EXTENSION{"image/jpeg", "jpg"}
	MIMETYPE_IMAGE_PNG                                                                 = MIMETYPE_EXTENSION{"image/png", "png"}
	MIMETYPE_IMAGE_SVG_XML                                                             = MIMETYPE_EXTENSION{"image/svg+xml", "svg"}
	MIMETYPE_IMAGE_TIFF                                                                = MIMETYPE_EXTENSION{"image/tiff", "tiff"}
	MIMETYPE_IMAGE_WEBP                                                                = MIMETYPE_EXTENSION{"image/webp", "webp"}
	MIMETYPE_IMAGE_X_ICON                                                              = MIMETYPE_EXTENSION{"image/x-icon", "ico"}
	MIMETYPE_MODEL_VND_COLLADA_XML                                                     = MIMETYPE_EXTENSION{"model/vnd.collada+xml", "dae"}
	MIMETYPE_MODEL_X3D_XML                                                             = MIMETYPE_EXTENSION{"model/x3d+xml", "x3d"}
	MIMETYPE_TEXT_HTML                                                                 = MIMETYPE_EXTENSION{"text/html; charset=utf-8", "html"}
	MIMETYPE_TEXT_PLAIN                                                                = MIMETYPE_EXTENSION{"text/plain", "txt"}
	MIMETYPE_TEXT_RTF                                                                  = MIMETYPE_EXTENSION{"text/rtf", "rtf"}
	MIMETYPE_TEXT_XML                                                                  = MIMETYPE_EXTENSION{"text/xml; charset=utf-8", "xml"}
	MIMETYPE_TEXT_X_LUA                                                                = MIMETYPE_EXTENSION{"text/x-lua", "lua"}
	MIMETYPE_TEXT_X_PERL                                                               = MIMETYPE_EXTENSION{"text/x-perl", "pl"}
	MIMETYPE_TEXT_X_PHP                                                                = MIMETYPE_EXTENSION{"text/x-php; charset=utf-8", "php"}
	MIMETYPE_TEXT_X_TCL                                                                = MIMETYPE_EXTENSION{"text/x-tcl", "tcl"}
	MIMETYPE_VIDEO_3GPP                                                                = MIMETYPE_EXTENSION{"video/3gpp", "3gp"}
	MIMETYPE_VIDEO_3GPP2                                                               = MIMETYPE_EXTENSION{"video/3gpp2", "3g2"}
	MIMETYPE_VIDEO_MP4                                                                 = MIMETYPE_EXTENSION{"video/mp4", "mp4"}
	MIMETYPE_VIDEO_MPEG                                                                = MIMETYPE_EXTENSION{"video/mpeg", "mpeg"}
	MIMETYPE_VIDEO_QUICKTIME                                                           = MIMETYPE_EXTENSION{"video/quicktime", "mov"}
	MIMETYPE_VIDEO_WEBM                                                                = MIMETYPE_EXTENSION{"video/webm", "webm"}
	MIMETYPE_VIDEO_X_FLV                                                               = MIMETYPE_EXTENSION{"video/x-flv", "flv"}
	MIMETYPE_VIDEO_X_MATROSKA                                                          = MIMETYPE_EXTENSION{"video/x-matroska", "mkv"}
	MIMETYPE_VIDEO_X_MSVIDEO                                                           = MIMETYPE_EXTENSION{"video/x-msvideo", "avi"}
)

func DetectMimeType(buf []byte) (string, string) {
	t, err := filetype.Match(buf)
	if err == nil {
		return t.MIME.Value, t.Extension
	}

	mime, ext := mimetype.Detect(buf)

	if len(mime) > 0 {
		return mime, ext
	}

	return MIMETYPE_APPLICATION_OCTET_STREAM.MimeType, ""
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
