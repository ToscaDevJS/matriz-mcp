package mcpserver

import (
	"bytes"
	"image"
	"image/png"

	"github.com/disintegration/imaging"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// thumbnailContent builds the ImageContent every image-producing tool returns (§5.12).
// Full-resolution bytes NEVER travel through the protocol: they blow up the context
// window and cost tokens for no benefit.
//
// maxEdge is 512 px by default (decision of this handoff): large enough for a
// model to judge composition, colour and obvious artefacts; small enough that
// four of them in one response stay manageable.
func thumbnailContent(img image.Image, maxEdge int) (*mcp.ImageContent, error) {
	if maxEdge <= 0 {
		maxEdge = 512
	}
	small := imaging.Fit(img, maxEdge, maxEdge, imaging.Lanczos)
	var buf bytes.Buffer
	if err := png.Encode(&buf, small); err != nil {
		return nil, err
	}
	return &mcp.ImageContent{
		Data:     buf.Bytes(),
		MIMEType: "image/png",
	}, nil
}
