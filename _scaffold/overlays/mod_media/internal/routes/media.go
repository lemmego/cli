package routes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lemmego/api/app"
	"github.com/lemmego/media"
)

// MediaRoutes is the worked example: one upload that is decoded, resized and
// stripped of its metadata before it is stored.
func MediaRoutes(a app.App) {
	a.Router().Post("/media/upload", uploadImage)
}

// uploadImage ingests one image and answers with the paths it wrote.
//
// **The submitted bytes are never stored.** That is the whole reason to go
// through the pipeline rather than StoreUploadOn: every rendition is
// decoded and re-encoded, which discards the metadata it arrived with — so a
// photograph does not publish the coordinates it was taken at, and a
// 4096-pixel upload does not get served into a 48-pixel slot on every page.
//
// That includes the full-size one, which is kept here as a re-encoded
// rendition rather than as the upload. The two other policies are a
// decision, not a convenience: media.RawOriginal stores the submitted bytes
// and therefore their metadata, and media.NoOriginal writes the conversions
// only — which is what an avatar wants, where no page shows the upload at
// its own size and keeping it is storage spent on nothing.
func uploadImage(c app.Context) error {
	manager, ok := app.Lookup[*media.Manager](c.App())
	if !ok {
		return fmt.Errorf("no media manager: add plugins.Media() to LoadProviders")
	}

	file, header, err := c.FormFile("image")
	if err != nil {
		return c.SetStatus(422).JSON(app.M{"error": "send one file as the `image` field"})
	}
	defer file.Close()

	// The stored name is derived here and never taken from the upload. A
	// client-supplied filename decides a path, and the extension is decided
	// by the encoder each rendition actually used — which for an automatic
	// format is not known until the pixels have been read.
	base := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	result, err := manager.Ingest(c.RequestContext(), file, media.Request{
		Dir:  "uploads",
		Base: base,

		// Two renditions, which is the shape almost every application wants:
		// one to display and one for a list or an avatar. The thumbnail is
		// what makes a feed cheap, and generating it on upload rather than on
		// request is what keeps it cheap.
		// Left at the default, which re-encodes. Named rather than omitted
		// because it is the field that decides whether EXIF survives.
		Original: media.CleanOriginal,

		Conversions: []media.Conversion{
			{Name: "lg", Width: 2000, Fit: media.FitInside},
			{Name: "sm", Width: 400, Fit: media.FitCover, Height: 400},
		},
	})
	if err != nil {
		// Includes the refusals that are the point of the guard: an image
		// whose header claims more pixels than the ceiling allows is
		// rejected before anything is allocated, because a few kilobytes of
		// deflate can ask for ten gigabytes of RGBA.
		return c.SetStatus(422).JSON(app.M{"error": err.Error()})
	}

	// Width and height are returned so a template can set them on the <img>.
	// Without them the page reflows as each image lands, which is the layout
	// shift a Lighthouse score complains about and a reader experiences as
	// the text jumping while they read it.
	variants := make([]app.M, 0, len(result.Variants))
	for _, v := range result.Variants {
		variants = append(variants, app.M{
			"name":   v.Name,
			"path":   v.Path,
			"width":  v.Width,
			"height": v.Height,
		})
	}

	return c.JSON(app.M{
		"uploaded": header.Filename,
		"variants": variants,
	})
}
