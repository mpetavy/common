package common

import (
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	MimetypeHeaderLen = 1024
)

type MimetypeExtension struct {
	MimeType, Ext string
}

var Mimetypes = make(map[string]MimetypeExtension)

var (
	MimetypeApplicationDicom                                                     = registerMimeType("application/dicom", "dcm")
	MimetypeApplicationEpubZip                                                   = registerMimeType("application/epub+zip", "epub")
	MimetypeApplicationGeoJson                                                   = registerMimeType("application/geo+json", "geojson")
	MimetypeApplicationGmlXml                                                    = registerMimeType("application/gml+xml", "gml")
	MimetypeApplicationGpxXml                                                    = registerMimeType("application/gpx+xml", "gpx")
	MimetypeApplicationGzip                                                      = registerMimeType("application/gzip", "gz")
	MimetypeApplicationJar                                                       = registerMimeType("application/jar", "jar")
	MimetypeApplicationJavascript                                                = registerMimeType("application/javascript", "js")
	MimetypeApplicationJson                                                      = registerMimeType("application/json", "json")
	MimetypeApplicationMsword                                                    = registerMimeType("application/msword", "doc")
	MimetypeApplicationOctetStream                                               = registerMimeType("application/octet-stream", "bin")
	MimetypeApplicationOgg                                                       = registerMimeType("application/ogg", "ogg")
	MimetypeApplicationPdf                                                       = registerMimeType("application/pdf", "pdf")
	MimetypeApplicationPostscript                                                = registerMimeType("application/postscript", "ps")
	MimetypeApplicationVndGarminTcxXml                                           = registerMimeType("application/vnd.garmin.tcx+xml", "tcx")
	MimetypeApplicationVndGoogleEarthKmlXml                                      = registerMimeType("application/vnd.google-earth.kml+xml", "kml")
	MimetypeApplicationVndMsExcel                                                = registerMimeType("application/vnd.ms-excel", "xls")
	MimetypeApplicationVndMsPowerpoint                                           = registerMimeType("application/vnd.ms-powerpoint", "ppt")
	MimetypeApplicationVndOpenxmlformatsOfficedocumentPresentationmlPresentation = registerMimeType("application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx")
	MimetypeApplicationVndOpenxmlformatsOfficedocumentSpreadsheetmlSheet         = registerMimeType("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx")
	MimetypeApplicationVndOpenxmlformatsOfficedocumentWordprocessingmlDocument   = registerMimeType("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx")
	MimetypeApplicationWasm                                                      = registerMimeType("application/wasm", "wasm")
	MimetypeApplicationX7ZCompressed                                             = registerMimeType("application/x-7z-compressed", "7z")
	MimetypeApplicationXChromeExtension                                          = registerMimeType("application/x-chrome-extension", "crx")
	MimetypeApplicationXJavaApplet                                               = registerMimeType("application/x-java-applet; charset=binary", "class")
	MimetypeApplicationXPhotoshop                                                = registerMimeType("application/x-photoshop", "psd")
	MimetypeApplicationXPython                                                   = registerMimeType("application/x-python", "py")
	MimetypeApplicationXShockwaveFlash                                           = registerMimeType("application/x-shockwave-flash", "swf")
	MimetypeApplicationXTar                                                      = registerMimeType("application/x-tar", "tar")
	MimetypeApplicationXWWWFormUrlencoded                                        = registerMimeType("application/x-www-form-urlencoded", "")
	MimetypeApplicationZip                                                       = registerMimeType("application/zip", "zip")
	MimetypeAudioAiff                                                            = registerMimeType("audio/aiff", "aiff")
	MimetypeAudioAmr                                                             = registerMimeType("audio/amr", "amr")
	MimetypeAudioApe                                                             = registerMimeType("audio/ape", "ape")
	MimetypeAudioBasic                                                           = registerMimeType("audio/basic", "au")
	MimetypeAudioFlac                                                            = registerMimeType("audio/flac", "flac")
	MimetypeAudioMidi                                                            = registerMimeType("audio/midi", "midi")
	MimetypeAudioMp4                                                             = registerMimeType("audio/mp4", "mp4")
	MimetypeAudioMpeg                                                            = registerMimeType("audio/mpeg", "mp3")
	MimetypeAudioMusepack                                                        = registerMimeType("audio/musepack", "mpc")
	MimetypeAudioWav                                                             = registerMimeType("audio/wav", "wav")
	MimetypeAudioXM4A                                                            = registerMimeType("audio/x-m4a", "m4a")
	MimetypeFontWoff                                                             = registerMimeType("font/woff", "woff")
	MimetypeFontWoff2                                                            = registerMimeType("font/woff2", "woff2")
	MimetypeImageBmp                                                             = registerMimeType("image/bmp", "bmp")
	MimetypeImageGif                                                             = registerMimeType("image/gif", "gif")
	MimetypeImageJpeg                                                            = registerMimeType("image/jpeg", "jpg")
	MimetypeImagePng                                                             = registerMimeType("image/png", "png")
	MimetypeImageSvgXml                                                          = registerMimeType("image/svg+xml", "svg")
	MimetypeImageTiff                                                            = registerMimeType("image/tiff", "tiff")
	MimetypeImageWebp                                                            = registerMimeType("image/webp", "webp")
	MimetypeImageXIcon                                                           = registerMimeType("image/x-icon", "ico")
	MimetypeModelVndColladaXml                                                   = registerMimeType("model/vnd.collada+xml", "dae")
	MimetypeModelX3DXml                                                          = registerMimeType("model/x3d+xml", "x3d")
	MimetypeTextHtml                                                             = registerMimeType("text/html; charset=utf-8", "html")
	MimetypeTextCss                                                              = registerMimeType("text/css", "css")
	MimetypeTextPlain                                                            = registerMimeType("text/plain", "txt")
	MimetypeTextRtf                                                              = registerMimeType("text/rtf", "rtf")
	MimetypeTextXml                                                              = registerMimeType("text/xml; charset=utf-8", "xml")
	MimetypeTextXLua                                                             = registerMimeType("text/x-lua", "lua")
	MimetypeTextXPerl                                                            = registerMimeType("text/x-perl", "pl")
	MimetypeTextXPhp                                                             = registerMimeType("text/x-php; charset=utf-8", "php")
	MimetypeTextXTcl                                                             = registerMimeType("text/x-tcl", "tcl")
	MimetypeVideo3GPP                                                            = registerMimeType("video/3gpp", "3gp")
	MimetypeVideo3GPP2                                                           = registerMimeType("video/3gpp2", "3g2")
	MimetypeVideoMp4                                                             = registerMimeType("video/mp4", "mp4")
	MimetypeVideoMpeg                                                            = registerMimeType("video/mpeg", "mpeg")
	MimetypeVideoQuicktime                                                       = registerMimeType("video/quicktime", "mov")
	MimetypeVideoWebm                                                            = registerMimeType("video/webm", "webm")
	MimetypeVideoXFlv                                                            = registerMimeType("video/x-flv", "flv")
	MimetypeVideoXMatroska                                                       = registerMimeType("video/x-matroska", "mkv")
	MimetypeVideoXMsvideo                                                        = registerMimeType("video/x-msvideo", "avi")
)

func registerMimeType(mimeType, ext string) MimetypeExtension {
	mt := MimetypeExtension{
		MimeType: mimeType,
		Ext:      ext,
	}

	Mimetypes[mimeType] = mt

	return mt
}

func DetectMimeType(filename string, buf []byte) (MimetypeExtension, error) {
	if filename != "" {
		ext := filepath.Ext(filename)
		if ext != "" {
			if strings.HasPrefix(ext, ".") {
				ext = ext[1:]
			}

			ext = strings.ToLower(ext)

			for _, mt := range Mimetypes {
				if mt.Ext == ext {
					return mt, nil
				}
			}

			if buf == nil {
				var err error

				buf, err = os.ReadFile(filename)
				if Error(err) {
					return MimetypeApplicationOctetStream, err
				}
			}
		}
	}

	t, err := filetype.Match(buf)
	if t.MIME.Value != "" && err == nil {
		return MimetypeExtension{t.MIME.Value, t.Extension}, nil
	}

	mime := mimetype.Detect(buf)

	if mime != nil {
		return Mimetypes[mime.String()], nil
	}

	return MimetypeApplicationOctetStream, fmt.Errorf("cannot detect mime type")
}

func ReadFileHeader(path string) ([]byte, error) {
	byteSlice := make([]byte, MimetypeHeaderLen)

	file, err := os.Open(path)
	if err != nil {
		return byteSlice[0:0], err
	}
	defer func() {
		Error(file.Close())
	}()

	bytesRead, err := file.Read(byteSlice)
	if err != nil {
		return byteSlice[0:0], err
	}

	return byteSlice[:bytesRead], nil
}

func IsImageMimeType(mimeType string) bool {
	if strings.Contains(mimeType, ";") {
		mimeType = mimeType[:strings.Index(mimeType, ";")]
	}

	b := strings.HasPrefix(mimeType, "image/")

	DebugFunc("%s: %v", mimeType, b)

	return b
}

func IsTextMimeType(mimeType string) bool {
	if strings.Contains(mimeType, ";") {
		mimeType = mimeType[:strings.Index(mimeType, ";")]
	}

	b := slices.Contains([]string{
		MimetypeApplicationGmlXml.MimeType,
		MimetypeApplicationGpxXml.MimeType,
		MimetypeApplicationJavascript.MimeType,
		MimetypeApplicationJson.MimeType,
		MimetypeTextHtml.MimeType,
		MimetypeTextCss.MimeType,
		MimetypeTextPlain.MimeType,
		MimetypeTextRtf.MimeType,
		MimetypeTextXml.MimeType,
		MimetypeTextXLua.MimeType,
		MimetypeTextXPerl.MimeType,
		MimetypeTextXPhp.MimeType,
		MimetypeTextXTcl.MimeType,
		MimetypeApplicationGmlXml.MimeType,
		MimetypeApplicationXWWWFormUrlencoded.MimeType,
	}, mimeType)

	DebugFunc("%s: %v", mimeType, b)

	return b
}
