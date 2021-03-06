package common

import (
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"
	"os"
	"path/filepath"
	"strings"
)

const (
	MimetypeHeaderLen = 1024
)

type MimetypeExtension struct {
	MimeType, Ext string
}

var Mimetypes []MimetypeExtension

var (
	MimetypeApplicationDicom,
	MimetypeApplicationEpubZip,
	MimetypeApplicationGeoJson,
	MimetypeApplicationGmlXml,
	MimetypeApplicationGpxXml,
	MimetypeApplicationGzip,
	MimetypeApplicationJar,
	MimetypeApplicationJavascript,
	MimetypeApplicationJson,
	MimetypeApplicationMsword,
	MimetypeApplicationOctetStream,
	MimetypeApplicationOgg,
	MimetypeApplicationPdf,
	MimetypeApplicationPostscript,
	MimetypeApplicationVndGarminTcxXml,
	MimetypeApplicationVndGoogleEarthKmlXml,
	MimetypeApplicationVndMsExcel,
	MimetypeApplicationVndMsPowerpoint,
	MimetypeApplicationVndOpenxmlformatsOfficedocumentPresentationmlPresentation,
	MimetypeApplicationVndOpenxmlformatsOfficedocumentSpreadsheetmlSheet,
	MimetypeApplicationVndOpenxmlformatsOfficedocumentWordprocessingmlDocument,
	MimetypeApplicationWasm,
	MimetypeApplicationX7ZCompressed,
	MimetypeApplicationXChromeExtension,
	MimetypeApplicationXJavaApplet,
	MimetypeApplicationXPhotoshop,
	MimetypeApplicationXPython,
	MimetypeApplicationXShockwaveFlash,
	MimetypeApplicationXTar,
	MimetypeApplicationZip,
	MimetypeAudioAiff,
	MimetypeAudioAmr,
	MimetypeAudioApe,
	MimetypeAudioBasic,
	MimetypeAudioFlac,
	MimetypeAudioMidi,
	MimetypeAudioMp4,
	MimetypeAudioMpeg,
	MimetypeAudioMusepack,
	MimetypeAudioWav,
	MimetypeAudioXM4A,
	MimetypeFontWoff,
	MimetypeFontWoff2,
	MimetypeImageBmp,
	MimetypeImageGif,
	MimetypeImageJpeg,
	MimetypeImagePng,
	MimetypeImageSvgXml,
	MimetypeImageTiff,
	MimetypeImageWebp,
	MimetypeImageXIcon,
	MimetypeModelVndColladaXml,
	MimetypeModelX3DXml,
	MimetypeTextHtml,
	MimetypeTextCss,
	MimetypeTextPlain,
	MimetypeTextRtf,
	MimetypeTextXml,
	MimetypeTextXLua,
	MimetypeTextXPerl,
	MimetypeTextXPhp,
	MimetypeTextXTcl,
	MimetypeVideo3GPP,
	MimetypeVideo3GPP2,
	MimetypeVideoMp4,
	MimetypeVideoMpeg,
	MimetypeVideoQuicktime,
	MimetypeVideoWebm,
	MimetypeVideoXFlv,
	MimetypeVideoXMatroska,
	MimetypeVideoXMsvideo MimetypeExtension
)

func init() {
	MimetypeApplicationDicom = registerMimeType("application/dicom", "dcm")
	MimetypeApplicationEpubZip = registerMimeType("application/epub+zip", "epub")
	MimetypeApplicationGeoJson = registerMimeType("application/geo+json", "geojson")
	MimetypeApplicationGmlXml = registerMimeType("application/gml+xml", "gml")
	MimetypeApplicationGpxXml = registerMimeType("application/gpx+xml", "gpx")
	MimetypeApplicationGzip = registerMimeType("application/gzip", "gz")
	MimetypeApplicationJar = registerMimeType("application/jar", "jar")
	MimetypeApplicationJavascript = registerMimeType("application/javascript", "js")
	MimetypeApplicationJson = registerMimeType("application/json", "json")
	MimetypeApplicationMsword = registerMimeType("application/msword", "doc")
	MimetypeApplicationOctetStream = registerMimeType("application/octet-stream", "")
	MimetypeApplicationOgg = registerMimeType("application/ogg", "ogg")
	MimetypeApplicationPdf = registerMimeType("application/pdf", "pdf")
	MimetypeApplicationPostscript = registerMimeType("application/postscript", "ps")
	MimetypeApplicationVndGarminTcxXml = registerMimeType("application/vnd.garmin.tcx+xml", "tcx")
	MimetypeApplicationVndGoogleEarthKmlXml = registerMimeType("application/vnd.google-earth.kml+xml", "kml")
	MimetypeApplicationVndMsExcel = registerMimeType("application/vnd.ms-excel", "xls")
	MimetypeApplicationVndMsPowerpoint = registerMimeType("application/vnd.ms-powerpoint", "ppt")
	MimetypeApplicationVndOpenxmlformatsOfficedocumentPresentationmlPresentation = registerMimeType("application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx")
	MimetypeApplicationVndOpenxmlformatsOfficedocumentSpreadsheetmlSheet = registerMimeType("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx")
	MimetypeApplicationVndOpenxmlformatsOfficedocumentWordprocessingmlDocument = registerMimeType("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx")
	MimetypeApplicationWasm = registerMimeType("application/wasm", "wasm")
	MimetypeApplicationX7ZCompressed = registerMimeType("application/x-7z-compressed", "7z")
	MimetypeApplicationXChromeExtension = registerMimeType("application/x-chrome-extension", "crx")
	MimetypeApplicationXJavaApplet = registerMimeType("application/x-java-applet; charset=binary", "class")
	MimetypeApplicationXPhotoshop = registerMimeType("application/x-photoshop", "psd")
	MimetypeApplicationXPython = registerMimeType("application/x-python", "py")
	MimetypeApplicationXShockwaveFlash = registerMimeType("application/x-shockwave-flash", "swf")
	MimetypeApplicationXTar = registerMimeType("application/x-tar", "tar")
	MimetypeApplicationZip = registerMimeType("application/zip", "zip")
	MimetypeAudioAiff = registerMimeType("audio/aiff", "aiff")
	MimetypeAudioAmr = registerMimeType("audio/amr", "amr")
	MimetypeAudioApe = registerMimeType("audio/ape", "ape")
	MimetypeAudioBasic = registerMimeType("audio/basic", "au")
	MimetypeAudioFlac = registerMimeType("audio/flac", "flac")
	MimetypeAudioMidi = registerMimeType("audio/midi", "midi")
	MimetypeAudioMp4 = registerMimeType("audio/mp4", "mp4")
	MimetypeAudioMpeg = registerMimeType("audio/mpeg", "mp3")
	MimetypeAudioMusepack = registerMimeType("audio/musepack", "mpc")
	MimetypeAudioWav = registerMimeType("audio/wav", "wav")
	MimetypeAudioXM4A = registerMimeType("audio/x-m4a", "m4a")
	MimetypeFontWoff = registerMimeType("font/woff", "woff")
	MimetypeFontWoff2 = registerMimeType("font/woff2", "woff2")
	MimetypeImageBmp = registerMimeType("image/bmp", "bmp")
	MimetypeImageGif = registerMimeType("image/gif", "gif")
	MimetypeImageJpeg = registerMimeType("image/jpeg", "jpg")
	MimetypeImagePng = registerMimeType("image/png", "png")
	MimetypeImageSvgXml = registerMimeType("image/svg+xml", "svg")
	MimetypeImageTiff = registerMimeType("image/tiff", "tiff")
	MimetypeImageWebp = registerMimeType("image/webp", "webp")
	MimetypeImageXIcon = registerMimeType("image/x-icon", "ico")
	MimetypeModelVndColladaXml = registerMimeType("model/vnd.collada+xml", "dae")
	MimetypeModelX3DXml = registerMimeType("model/x3d+xml", "x3d")
	MimetypeTextHtml = registerMimeType("text/html; charset=utf-8", "html")
	MimetypeTextPlain = registerMimeType("text/css", "css")
	MimetypeTextPlain = registerMimeType("text/plain", "txt")
	MimetypeTextRtf = registerMimeType("text/rtf", "rtf")
	MimetypeTextXml = registerMimeType("text/xml; charset=utf-8", "xml")
	MimetypeTextXLua = registerMimeType("text/x-lua", "lua")
	MimetypeTextXPerl = registerMimeType("text/x-perl", "pl")
	MimetypeTextXPhp = registerMimeType("text/x-php; charset=utf-8", "php")
	MimetypeTextXTcl = registerMimeType("text/x-tcl", "tcl")
	MimetypeVideo3GPP = registerMimeType("video/3gpp", "3gp")
	MimetypeVideo3GPP2 = registerMimeType("video/3gpp2", "3g2")
	MimetypeVideoMp4 = registerMimeType("video/mp4", "mp4")
	MimetypeVideoMpeg = registerMimeType("video/mpeg", "mpeg")
	MimetypeVideoQuicktime = registerMimeType("video/quicktime", "mov")
	MimetypeVideoWebm = registerMimeType("video/webm", "webm")
	MimetypeVideoXFlv = registerMimeType("video/x-flv", "flv")
	MimetypeVideoXMatroska = registerMimeType("video/x-matroska", "mkv")
	MimetypeVideoXMsvideo = registerMimeType("video/x-msvideo", "avi")
}

func registerMimeType(mimeType, ext string) MimetypeExtension {
	mt := MimetypeExtension{
		MimeType: mimeType,
		Ext:      ext,
	}

	Mimetypes = append(Mimetypes, mt)

	return mt
}

func DetectMimeType(filename string, buf []byte) MimetypeExtension {
	if filename != "" {
		ext := filepath.Ext(filename)
		if ext != "" {
			if strings.HasPrefix(ext, ".") {
				ext = ext[1:]
			}

			ext = strings.ToLower(ext)

			for _, mt := range Mimetypes {
				if mt.Ext == ext {
					return mt
				}
			}
		}
	}

	t, err := filetype.Match(buf)
	if t.MIME.Value != "" && err == nil {
		return MimetypeExtension{t.MIME.Value, t.Extension}
	}

	mime := mimetype.Detect(buf)

	if mime != nil {
		return MimetypeExtension{mime.String(), mime.Extension()}
	}

	return MimetypeApplicationOctetStream
}

func ReadHeader(path string) ([]byte, error) {
	byteSlice := make([]byte, MimetypeHeaderLen)

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
