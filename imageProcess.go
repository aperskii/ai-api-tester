package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"
)

// ImageProcessor handles the image processing logic.
type ImageProcessor struct {
	Params       RequestParam
	Client1      *ServerClient
	Client2      *ServerClient
	CSVWriter    *CSVWriter
	imageSaveDir string
	mu           sync.Mutex
	imgPath1     string
	imgPath2     string
}

// NewImageProcessor creates a new ImageProcessor instance.
func NewImageProcessor(params RequestParam, client1URL, client2URL, imageUrl1Out, imageUrl2Out, csvFile, imageSaveDir string) (*ImageProcessor, error) {
	csvWriter, err := NewCSVWriter(csvFile)
	if err != nil {
		return nil, err
	}
	return &ImageProcessor{
		Params: params,
		Client1: &ServerClient{
			URL:    client1URL + params.Route,
			OutUrl: imageUrl1Out,
		},
		Client2: &ServerClient{
			URL:    client2URL + params.Route,
			OutUrl: imageUrl2Out,
		},
		CSVWriter:    csvWriter,
		imageSaveDir: imageSaveDir,
	}, nil
}

// ProcessImage processes a single image.
func (ip *ImageProcessor) ProcessImage(imagePath, imageName string) error {
	// Map to track unique differences
	differenceMap := make(map[string]struct{})

	// Use WaitGroup to handle concurrent server requests
	var wg sync.WaitGroup
	var resp1, resp2 *ResponseData
	var err1, err2 error
	wg.Add(2)

	// Fetch response from Server 1
	go func() {
		defer wg.Done()
		resp1, err1 = ip.Client1.SendImageAndGetResponse(imagePath, ip.Params)
	}()

	// Fetch response from Server 2
	go func() {
		defer wg.Done()
		resp2, err2 = ip.Client2.SendImageAndGetResponse(imagePath, ip.Params)
	}()

	wg.Wait()

	// Check for errors in requests
	if err1 != nil {
		return fmt.Errorf("request to Server 1 failed: %v", err1)
	}
	if err2 != nil {
		return fmt.Errorf("request to Server 2 failed: %v", err2)
	}

	// Process image comparison based on Params.Type
	switch ip.Params.Type {
	case "classical_pdf", "qr_pdf":
		ip.processImageType(resp1, resp2, differenceMap)
	case "qr_text_pdf":
		ip.compareQrText(resp1.Blocks.Qr2015Bvg.Text, resp2.Blocks.Qr2015Bvg.Text, differenceMap)
	case "text_block":
		ip.compareQrText(resp1.Blocks.Text.Text, resp2.Blocks.Text.Text, differenceMap)
	case "image":
		ip.processImageProperties(resp1, resp2, differenceMap)
	}

	// Write differences to CSV
	for diff := range differenceMap {
		if err := ip.CSVWriter.WriteDifference(imageName, diff); err != nil {
			log.Printf("Failed to write differences for image %s to CSV: %v", imageName, err)
		}
	}

	return nil
}

// processImageType handles the comparison for image types and PDF responses.
func (ip *ImageProcessor) processImageType(resp1, resp2 *ResponseData, differenceMap map[string]struct{}) {
	if len(resp1.Blocks.Photo.Properties) == 0 {
		differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client1.URL)] = struct{}{}
	} else if len(resp2.Blocks.Photo.Properties) == 0 {
		differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client2.URL)] = struct{}{}
	} else {
		wg := &sync.WaitGroup{}
		wg.Add(2)

		// Download images concurrently
		go ip.downloadAndCompareImage(resp1.Blocks.Photo.Properties, ip.Client1.OutUrl, "server1_", differenceMap, wg)
		go ip.downloadAndCompareImage(resp2.Blocks.Photo.Properties, ip.Client2.OutUrl, "server2_", differenceMap, wg)

		wg.Wait()

		// Compare the responses from the two servers
		diffs := compareProperties(resp1.Blocks.Photo.Properties, resp2.Blocks.Photo.Properties)
		ip.compareQrText(resp1.Blocks.Qr2015.Text, resp2.Blocks.Qr2015.Text, differenceMap)
		for _, diff := range diffs {
			differenceMap[diff] = struct{}{}
		}
	}
}

// processImageProperties handles comparison for the "image" type.
func (ip *ImageProcessor) processImageProperties(resp1, resp2 *ResponseData, differenceMap map[string]struct{}) {
	if len(resp1.Photo.Properties) == 0 {
		differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client1.URL)] = struct{}{}
	} else if len(resp2.Photo.Properties) == 0 {
		differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client2.URL)] = struct{}{}
	} else {
		wg := &sync.WaitGroup{}
		wg.Add(2)

		// Download images concurrently
		go ip.downloadAndCompareImage(resp1.Photo.Properties, ip.Client1.OutUrl, "server1_", differenceMap, wg)
		go ip.downloadAndCompareImage(resp2.Photo.Properties, ip.Client2.OutUrl, "server2_", differenceMap, wg)

		wg.Wait()

		// Compare the responses from the two servers
		diffs := compareProperties(resp1.Photo.Properties, resp2.Photo.Properties)
		for _, diff := range diffs {
			differenceMap[diff] = struct{}{}
		}
	}
}

// downloadAndCompareImage handles the image download and updates differenceMap if necessary.
func (ip *ImageProcessor) downloadAndCompareImage(properties map[string]interface{}, serverURL, filePrefix string, differenceMap map[string]struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	// Download the image
	imgPath, err := handleImageDownload(properties, serverURL, ip.imageSaveDir, filePrefix)
	if err != nil {
		differenceMap[fmt.Sprintf("Failed to download image from %s: %v", serverURL, err)] = struct{}{}
		return
	}

	// Save the downloaded image path
	ip.mu.Lock()
	defer ip.mu.Unlock()

	if filePrefix == "server1_" {
		ip.imgPath1 = imgPath
	} else {
		ip.imgPath2 = imgPath
	}

	// If both images are downloaded, compare and delete duplicates
	if ip.imgPath1 != "" && ip.imgPath2 != "" {
		err = HandleImageComparisonAndDeletion(ip.imgPath1, ip.imgPath2)
		if err != nil {
			differenceMap[fmt.Sprintf("Error during image comparison: %v", err)] = struct{}{}
		}
	}
}

// handleImageDownload handles the image download based on the response properties.
func handleImageDownload(properties map[string]interface{}, serverURL, imageSaveDir, filePrefix string) (string, error) {
	if downloadLink, ok := properties["download_link"].(string); ok && downloadLink != "" {
		imageURL := serverURL + "/" + downloadLink
		resp, err := http.Get(imageURL)
		if err != nil {
			return "", fmt.Errorf("failed to download image from %s: %v", serverURL, err)
		}
		defer resp.Body.Close()

		// Check if the directory exists
		if _, err := os.Stat(imageSaveDir); os.IsNotExist(err) {
			// Create the directory if it does not exist
			err := os.MkdirAll(imageSaveDir, os.ModePerm)
			if err != nil {
				return "", fmt.Errorf("failed to create directory %s: %v", imageSaveDir, err)
			}
			fmt.Printf("Directory %s created successfully\n", imageSaveDir)
		}
		imagePath := filepath.Join(imageSaveDir, fmt.Sprintf("%s_%s", filePrefix, filepath.Base(downloadLink)))
		file, err := os.Create(imagePath)
		if err != nil {
			return "", fmt.Errorf("failed to create image file: %v", err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to save image file: %v", err)
		}
		fmt.Printf("Image saved to %s\n", imagePath)
		return imagePath, nil
	} else {
		return "", fmt.Errorf("no valid download link found for image from %s", serverURL)
	}
}

// compareProperties compares two maps of properties, excluding "download_link".
func compareProperties(props1, props2 map[string]interface{}) []string {
	var differences []string
	for key, value1 := range props1 {
		if key == "download_link" {
			continue
		}
		value2, ok := props2[key]
		if !ok {
			// If the key doesn't exist in the second map, log the difference
			differences = append(differences, fmt.Sprintf("Key %s missing in second map", key))
			continue
		}
		// Compare based on value type
		diff := compareValues(key, value1, value2)
		if diff != "" {
			differences = append(differences, diff)
		}
	}
	for key := range props2 {
		if key == "download_link" {
			continue
		}
		if _, ok := props1[key]; !ok {
			differences = append(differences, fmt.Sprintf("Key %s missing in first response", key))
		}
	}
	return differences
}

// compareValues compares two values and returns a formatted difference string if they differ.
func compareValues(key string, val1, val2 interface{}) string {
	switch v1 := val1.(type) {
	case float64:
		if v2, ok := val2.(float64); ok {
			if !compareFloats(v1, v2, 1.0) { // Compare based on the integer part (precision 1.0)
				return fmt.Sprintf("Difference in %s: %v vs %v", key, v1, v2)
			}
		} else {
			return fmt.Sprintf("Key %s type mismatch: %T vs %T", key, val1, val2)
		}
	case string:
		if v2, ok := val2.(string); ok {
			if v1 != v2 {
				return fmt.Sprintf("Difference in %s: %v vs %v", key, v1, v2)
			}
		} else {
			return fmt.Sprintf("Key %s type mismatch: %T vs %T", key, val1, val2)
		}
	default:
		// For other types, use reflect.DeepEqual for comparison
		if !reflect.DeepEqual(val1, val2) {
			return fmt.Sprintf("Difference in %s: %v vs %v", key, val1, val2)
		}
	}
	return ""
}

// compareFloats compares two float64 values based on a given precision.
func compareFloats(f1, f2, precision float64) bool {
	return math.Abs(f1-f2) < precision
}

// compareQrText compares two text blocks (strings) and adds differences to the differenceMap.
func (ip *ImageProcessor) compareQrText(text1, text2 string, differenceMap map[string]struct{}) {
	// Check if both texts are empty or missing
	if text1 == "" && text2 == "" {
		return // No differences, both are empty or missing
	}

	// If one of the texts is missing or empty, record the difference
	if text1 == "" {
		differenceMap["Text missing in first response"] = struct{}{}
		return
	}
	if text2 == "" {
		differenceMap["Text missing in second response"] = struct{}{}
		return
	}

	// Compare the strings. If they differ, record the difference
	if text1 != text2 {
		differenceMap[fmt.Sprintf("Text difference: '%s' vs '%s'", text1, text2)] = struct{}{}
	}
}
