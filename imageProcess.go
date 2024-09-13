package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)

// ImageProcessor handles the image processing logic.
type ImageProcessor struct {
	Params       RequestParam
	Client1      *ServerClient
	Client2      *ServerClient
	CSVWriter    *CSVWriter
	imageSaveDir string
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

	resp1, err := ip.Client1.SendImageAndGetResponse(imagePath, ip.Params)
	if err != nil {
		return fmt.Errorf("request to Server 1 failed: %v", err)
	}
	fmt.Printf("Responce Server : %s\n  %v\n", ip.Client1.URL, resp1)

	resp2, err := ip.Client2.SendImageAndGetResponse(imagePath, ip.Params)
	if err != nil {
		return fmt.Errorf("request to Server 2 failed: %v", err)
	}
	fmt.Printf("Responce Server : %s\n  %v\n", ip.Client2.URL, resp2)

	if ip.Params.Type == "classical_pdf" || ip.Params.Type == "qr_pdf" {
		if len(resp1.Blocks.Photo.Properties) == 0 {
			differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client1.URL)] = struct{}{}
		} else if len(resp2.Blocks.Photo.Properties) == 0 {
			differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client2.URL)] = struct{}{}
		} else {
			err = handleImageDownload(resp1.Blocks.Photo.Properties, ip.Client1.OutUrl, ip.imageSaveDir, "server1_")
			if err != nil {
				return fmt.Errorf("Failed to download background segmentation: %v\n", err)
			}
			err = handleImageDownload(resp2.Blocks.Photo.Properties, ip.Client2.OutUrl, ip.imageSaveDir, "server2_")
			if err != nil {
				return fmt.Errorf("Failed to download background segmentation: %v\n", err)
			}
			// Compare the responses from the two servers
			diffs := compareProperties(resp1.Blocks.Photo.Properties, resp2.Blocks.Photo.Properties)
			for _, diff := range diffs {
				differenceMap[diff] = struct{}{}
			}
		}
	} else if ip.Params.Type == "qr_text_pdf" {
		diffs := compareBlocks(resp1.Blocks.Qr2015Bvg.Text, resp2.Blocks.Qr2015Bvg.Text)
		// Compare the "Text" field in both responses
		for _, diff := range diffs {
			differenceMap[diff] = struct{}{}
		}
	} else if ip.Params.Type == "text_block" {
		diffs := compareBlocks(resp1.Blocks.Text.Text, resp2.Blocks.Text.Text)
		// Compare the "Text" field in both responses
		for _, diff := range diffs {
			differenceMap[diff] = struct{}{}
		}
	} else if ip.Params.Type == "image" {
		// Check if either response properties are empty and log it
		if len(resp1.Photo.Properties) == 0 {
			differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client1.URL)] = struct{}{}
		} else if len(resp2.Photo.Properties) == 0 {
			differenceMap[fmt.Sprintf("Response from: %s - has empty properties", ip.Client2.URL)] = struct{}{}
		} else {
			err = handleImageDownload(resp1.Photo.Properties, ip.Client1.OutUrl, ip.imageSaveDir, "server1_")
			if err != nil {
				return fmt.Errorf("Failed to download background segmentation: %v\n", err)
			}
			err = handleImageDownload(resp2.Photo.Properties, ip.Client2.OutUrl, ip.imageSaveDir, "server2_")
			if err != nil {
				return fmt.Errorf("Failed to download background segmentation: %v\n", err)
			}
			// Compare the responses from the two servers
			diffs := compareProperties(resp1.Photo.Properties, resp2.Photo.Properties)
			for _, diff := range diffs {
				differenceMap[diff] = struct{}{}
			}
		}
	}
	// Write differences to the CSV
	for diff := range differenceMap {
		if err := ip.CSVWriter.WriteDifference(imageName, diff); err != nil {
			log.Printf("Failed to write differences for image %s to CSV: %v", imageName, err)
		}
	}

	return nil
}

// compareProperties compares two maps of properties, excluding "download_link".
func compareProperties(props1, props2 map[string]interface{}) []string {
	var differences []string
	for key, value1 := range props1 {
		if key == "download_link" {
			continue
		}
		if value2, ok := props2[key]; ok {
			if !reflect.DeepEqual(value1, value2) {
				differences = append(differences, fmt.Sprintf("Difference in %s: %v vs %v", key, value1, value2))
			}
		} else {
			differences = append(differences, fmt.Sprintf("Key %s missing in second response", key))
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

// handleImageDownload handles the image download based on the response properties.
func handleImageDownload(properties map[string]interface{}, serverURL, imageSaveDir, filePrefix string) error {
	if downloadLink, ok := properties["download_link"].(string); ok && downloadLink != "" {
		imageURL := serverURL + "/" + downloadLink
		resp, err := http.Get(imageURL)
		if err != nil {
			return fmt.Errorf("failed to download image from %s: %v", serverURL, err)
		}
		defer resp.Body.Close()

		// Check if the directory exists
		if _, err := os.Stat(imageSaveDir); os.IsNotExist(err) {
			// Create the directory if it does not exist
			err := os.MkdirAll(imageSaveDir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %v", imageSaveDir, err)
			}
			fmt.Printf("Directory %s created successfully\n", imageSaveDir)
		}
		imagePath := filepath.Join(imageSaveDir, fmt.Sprintf("%s_%s", filePrefix, filepath.Base(downloadLink)))
		file, err := os.Create(imagePath)
		if err != nil {
			return fmt.Errorf("failed to create image file: %v", err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save image file: %v", err)
		}
		fmt.Printf("Image saved to %s\n", imagePath)
	} else {
		return fmt.Errorf("no valid download link found for image from %s", serverURL)
	}
	return nil
}
