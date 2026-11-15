package plugins

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	mediaprovider "github.com/lemmego/media/provider"
)

// Media registers the image pipeline over one of this application's disks.
//
// The defaults are the module's and are left alone deliberately: the pixel
// ceiling, the byte ceiling and the JPEG quality are the three settings that
// decide whether a hostile upload can exhaust memory, and the module's
// values are the ones its own tests cover.
//
// Worth knowing about what it does, because two of the three are not
// obvious:
//
//   - Orientation is applied before metadata is stripped. A phone does not
//     rotate the pixels it writes; it stores the sensor's landscape frame and
//     adds a tag asking the viewer for a quarter turn, and image/jpeg ignores
//     that tag. Discarding metadata without first applying it turns every
//     portrait photograph on its side.
//   - Metadata is discarded, which is a privacy fix and not a size one: a
//     photograph taken on a phone carries the GPS coordinates it was taken
//     at, and serving the submitted bytes verbatim publishes them.
//   - An animated GIF is passed through rather than re-encoded, because
//     re-encoding one yields a still poster frame.
func Media() app.Provider {
	return &mediaprovider.Provider{
		// public is served at /storage/, which is what an <img> needs. Put
		// anything that should not be readable by whoever has the URL on the
		// default disk instead, which is not served at all.
		Disk: config.MustEnv("MEDIA_DISK", "public"),
	}
}
