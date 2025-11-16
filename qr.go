package qr

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/nfnt/resize"
)

// Default size of the QR code - 601px is the breakpoint for small screens on @media queries.
// This will display as full size on mobile devices without compression.
const DefaultQRSize int = 600

var (
	ErrUnsupportedDataType error = errors.New("unsupported data type")
	ErrRequired            error = errors.New("input is required")
)

type DataType string

const (
	TypeURL   = "url"
	TypeTel   = "tel"
	TypeSMS   = "sms"
	TypeEmail = "email"
)

type CodeRequest struct {
	DataType string `json:"data_type,omitempty"`
	Text     string `json:"text,omitempty"`
}

func GenerateCode(ctx context.Context, req CodeRequest) (string, error) {
	if req.Text == "" {
		return "", ErrRequired
	}
	var textToEncode string
	switch req.DataType {
	case TypeURL:
		textToEncode = req.Text
	case TypeTel:
		textToEncode = "tel:" + req.Text
	case TypeEmail:
		textToEncode = "mailto:" + req.Text
	case TypeSMS:
		textToEncode = "smsto:" + req.Text
	default:
		return "", ErrUnsupportedDataType
	}

	qrcode, err := qr.Encode(textToEncode, qr.H, qr.Auto)
	if err != nil {
		return "", fmt.Errorf("failed to encode qr code: %w", err)
	}

	qrcode, err = barcode.Scale(qrcode, DefaultQRSize, DefaultQRSize)
	if err != nil {
		return "", fmt.Errorf("failed to resize qr code: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, qrcode); err != nil {
		return "", fmt.Errorf("failed to encode to png: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(buf.Bytes()), nil
}

func GenerateGopherCode(ctx context.Context, req CodeRequest) (string, error) {
	if req.Text == "" {
		return "", ErrRequired
	}
	var textToEncode string
	switch req.DataType {
	case TypeURL:
		textToEncode = req.Text
	case TypeTel:
		textToEncode = "tel:" + req.Text
	case TypeEmail:
		textToEncode = "mailto:" + req.Text
	case TypeSMS:
		textToEncode = "smsto:" + req.Text
	default:
		return "", ErrUnsupportedDataType
	}

	qrcode, err := qr.Encode(textToEncode, qr.H, qr.Auto)
	if err != nil {
		return "", fmt.Errorf("failed to encode qr code: %w", err)
	}

	qrcode, err = barcode.Scale(qrcode, DefaultQRSize, DefaultQRSize)
	if err != nil {
		return "", fmt.Errorf("failed to resize qr code: %w", err)
	}

	logoFilePath := "gopherize.png"
	logoFile, err := os.Open(logoFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open logo image: %w", err)
	}
	defer logoFile.Close()

	logoImage, _, err := image.Decode(logoFile)
	if err != nil {
		return "", fmt.Errorf("failed to decode logo image: %w", err)
	}

	// Create a new RGBA image from the QR code
	b := qrcode.Bounds()
	m := image.NewRGBA(b)
	draw.Draw(m, b, qrcode, image.Point{}, draw.Src)

	// Define the size of the square cutout in the middle.
	// This should be small enough to not obstruct too much of the QR code.
	cutoutSize := 120

	// Resize logo to fit in the cutout
	logoImage = resize.Thumbnail(uint(cutoutSize), uint(cutoutSize), logoImage, resize.Lanczos3)

	// Calculate the position for the square cutout (center).
	cutoutPos := image.Pt((b.Dx()-cutoutSize)/2, (b.Dy()-cutoutSize)/2)
	bgRect := image.Rect(
		cutoutPos.X,
		cutoutPos.Y,
		cutoutPos.X+cutoutSize,
		cutoutPos.Y+cutoutSize,
	)
	draw.Draw(m, bgRect, image.NewUniform(color.White), image.Point{}, draw.Src)

	// Calculate logo position to be centered within the cutout.
	logoPos := image.Pt(
		cutoutPos.X+(cutoutSize-logoImage.Bounds().Dx())/2,
		cutoutPos.Y+(cutoutSize-logoImage.Bounds().Dy())/2,
	)

	// Draw the logo over the white rectangle
	draw.Draw(m, logoImage.Bounds().Add(logoPos), logoImage, image.Point{}, draw.Over)

	var buf bytes.Buffer
	if err := png.Encode(&buf, m); err != nil {
		return "", fmt.Errorf("failed to encode to png: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(buf.Bytes()), nil
}
