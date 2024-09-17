package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Load .env file
	err := godotenv.Load("docs/.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Get environment variables
	server1URL := os.Getenv("SERVER1_URL")
	server2URL := os.Getenv("SERVER2_URL")
	imageUrl1Out := os.Getenv("IMAGE_URL1_OUT")
	imageUrl2Out := os.Getenv("IMAGE_URL2_OUT")
	profilesDir := os.Getenv("PROFILES_DIR")

	folders, err := ioutil.ReadDir(profilesDir)
	if err != nil {
		fmt.Errorf("error reading root folder: %v", err)
	}
	for _, folder := range folders {
		if folder.IsDir() {
			folderPath := filepath.Join(profilesDir, folder.Name())
			fmt.Println("Processing folder:", folderPath)
			err := processFolder(folderPath, server1URL, server2URL, imageUrl1Out, imageUrl2Out)
			if err != nil {
				fmt.Printf("Error processing folder %s: %v\n", folderPath, err)
			}
		}
	}
	fmt.Println("Processing completed. Differences saved to out/differences.csv")
}

// Process a single folder
func processFolder(folderPath, server1URL, server2URL, imageUrl1Out, imageUrl2Out string) error {
	// Read the params.json
	paramsFilePath := filepath.Join(folderPath, "params.json")
	paramsFile, err := os.Open(paramsFilePath)
	if err != nil {
		return fmt.Errorf("error opening params.json: %v", err)
	}
	defer paramsFile.Close()

	var params RequestParam
	if err := json.NewDecoder(paramsFile).Decode(&params); err != nil {
		return fmt.Errorf("error decoding params.json: %v", err)
	}

	// Iterate over all the images in the folder
	imagesFolderPath := filepath.Join(folderPath, "in")
	imageFiles, err := ioutil.ReadDir(imagesFolderPath)
	if err != nil {
		return fmt.Errorf("error reading images folder: %v", err)
	}

	// Create an out folder for the results
	outFolderPath := filepath.Join(folderPath, "out")
	if _, err := os.Stat(outFolderPath); os.IsNotExist(err) {
		os.Mkdir(outFolderPath, 0755)
	}
	// Initialize the ImageProcessor
	imageProcessor, err := NewImageProcessor(params, server1URL, server2URL, imageUrl1Out, imageUrl2Out, outFolderPath+"/differences.csv", outFolderPath)
	if err != nil {
		log.Fatalf("Failed to initialize ImageProcessor: %v", err)
	}
	defer imageProcessor.CSVWriter.Close()

	// Process each image
	for _, file := range imageFiles {
		if !file.IsDir() {
			imagePath := filepath.Join(imagesFolderPath, file.Name())
			fmt.Println("Processing Image:", imagePath)
			if err := imageProcessor.ProcessImage(imagePath, file.Name()); err != nil {
				log.Printf("Failed to process image %s: %v", file.Name(), err)
			}
		}
	}

	return nil
}
