package main

import (
	"bytes"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	_ "image/jpeg" // Required for image decoding
	_ "image/png"  // Required for image decoding
	"io/ioutil"
	"math"
	"os"
)

// CompareImages compares two image files and returns true if they are identical.
func CompareImages(imgPath1, imgPath2 string) (bool, error) {
	// Open the first image file
	imgFile1, err := os.Open(imgPath1)
	if err != nil {
		return false, fmt.Errorf("failed to open image file 1: %v", err)
	}
	defer imgFile1.Close()

	// Open the second image file
	imgFile2, err := os.Open(imgPath2)
	if err != nil {
		return false, fmt.Errorf("failed to open image file 2: %v", err)
	}
	defer imgFile2.Close()

	// Decode both images
	img1, format1, err := image.Decode(imgFile1)
	if err != nil {
		return false, fmt.Errorf("failed to decode image file 1: %v", err)
	}

	img2, format2, err := image.Decode(imgFile2)
	if err != nil {
		return false, fmt.Errorf("failed to decode image file 2: %v", err)
	}

	// Normalize formats (convert to RGBA)
	img1RGBA := imageToRGBA(img1, format1)
	img2RGBA := imageToRGBA(img2, format2)

	// Compare the image bounds (dimensions)
	if img1RGBA.Bounds() != img2RGBA.Bounds() {
		return false, nil // Images have different dimensions
	}
	hash1 := ComputeImageHash(img1)
	hash2 := ComputeImageHash(img2)
	if comparePixelsWithTolerance(img1RGBA, img2RGBA, 5) || hash1 == hash2 {
		return true, nil
	}

	return false, nil
}

// CompareImageBytes compares two image files based on their byte content.
func CompareImageBytes(imgPath1, imgPath2 string) (bool, error) {
	// Read the content of the first image
	img1Bytes, err := ioutil.ReadFile(imgPath1)
	if err != nil {
		return false, fmt.Errorf("failed to read image file 1: %v", err)
	}

	// Read the content of the second image
	img2Bytes, err := ioutil.ReadFile(imgPath2)
	if err != nil {
		return false, fmt.Errorf("failed to read image file 2: %v", err)
	}

	// Compare the byte contents of the two images
	if bytes.Equal(img1Bytes, img2Bytes) {
		return true, nil // Images are identical
	}
	return false, nil
}

// HandleImageComparisonAndDeletion compares the downloaded images and deletes one if they are identical.
func HandleImageComparisonAndDeletion(imgPath1, imgPath2 string) error {
	// Compare the images
	identical, err := CompareImages(imgPath1, imgPath2)
	if err != nil {
		return fmt.Errorf("error comparing images: %v", err)
	}

	iden, err := CompareImageBytes(imgPath1, imgPath2)
	if err != nil {
		return fmt.Errorf("error comparing bytes images: %v", err)
	}

	if identical || iden {
		// Images are identical, delete one of them
		err = os.Remove(imgPath1)
		if err != nil {
			return fmt.Errorf("failed to delete image file 1: %v", err)
		}
		err = os.Remove(imgPath2)
		if err != nil {
			return fmt.Errorf("failed to delete image file 2: %v", err)
		}
		fmt.Printf("Image %s was deleted because it is identical to %s\n", imgPath2, imgPath1)
	} else {
		fmt.Println("Images are different, both will be kept.")
	}

	return nil
}

// imageToRGBA converts an image to RGBA format for consistent comparison.
func imageToRGBA(img image.Image, format string) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return rgba
}

// comparePixelsWithTolerance compares two RGBA images with a given tolerance level.
func comparePixelsWithTolerance(img1, img2 *image.RGBA, tolerance int) bool {
	for y := 0; y < img1.Bounds().Dy(); y++ {
		for x := 0; x < img1.Bounds().Dx(); x++ {
			c1 := img1.RGBAAt(x, y)
			c2 := img2.RGBAAt(x, y)

			if !colorsAreSimilar(c1, c2, tolerance) {
				return false // Images differ
			}
		}
	}
	return true // Images are similar
}

// colorsAreSimilar checks if two colors are similar within a given tolerance.
func colorsAreSimilar(c1, c2 color.RGBA, tolerance int) bool {
	return math.Abs(float64(c1.R-c2.R)) <= float64(tolerance) &&
		math.Abs(float64(c1.G-c2.G)) <= float64(tolerance) &&
		math.Abs(float64(c1.B-c2.B)) <= float64(tolerance) &&
		math.Abs(float64(c1.A-c2.A)) <= float64(tolerance) // Consider alpha channel if necessary
}

// ComputeImageHash generates a perceptual hash for an image.
func ComputeImageHash(img image.Image) uint64 {
	// Resize the image to a smaller size for hashing
	const hashSize = 8
	resizedImg := resizeImage(img, hashSize)

	// Convert to grayscale
	grayImg := image.NewGray(resizedImg.Bounds())
	for x := 0; x < resizedImg.Bounds().Max.X; x++ {
		for y := 0; y < resizedImg.Bounds().Max.Y; y++ {
			c := resizedImg.At(x, y)
			grayImg.Set(x, y, color.GrayModel.Convert(c))
		}
	}

	// Compute the average pixel value
	var total uint64
	pixels := make([]uint8, hashSize*hashSize)

	for y := 0; y < hashSize; y++ {
		for x := 0; x < hashSize; x++ {
			pix := grayImg.GrayAt(x, y)
			pixels[y*hashSize+x] = pix.Y
			total += uint64(pix.Y)
		}
	}
	avg := total / (hashSize * hashSize)

	// Generate hash based on pixel values relative to the average
	var hash uint64
	for _, pixel := range pixels {
		if uint64(pixel) >= avg {
			hash |= 1
		}
		hash <<= 1
	}
	return hash
}

// resizeImage resizes the image to a specified width and height using bilinear interpolation.
func resizeImage(img image.Image, newSize int) image.Image {
	return resize.Resize(uint(newSize), uint(newSize), img, resize.Bilinear)
}
