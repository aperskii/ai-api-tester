package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// ServerClient handles the communication with the servers.
type ServerClient struct {
	URL    string
	OutUrl string
}

// SendImageAndGetResponse sends the image to the server and retrieves the response.
func (client *ServerClient) SendImageAndGetResponse(imagePath string, params RequestParam) (*ResponseData, error) {
	var response ResponseData
	jsonData, err := json.Marshal(params.Params)
	if err != nil {
		return &response, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	jsonFieldName := "params"
	formField, err := writer.CreateFormField(jsonFieldName)
	if err != nil {
		return &response, fmt.Errorf("failed to create form field: %v", err)
	}
	_, err = formField.Write(jsonData)
	if err != nil {
		return &response, fmt.Errorf("failed to write to form field: %v", err)
	}
	fmt.Println("Request Body: ", string(jsonData))
	imageFile, err := os.Open(imagePath)
	if err != nil {
		return &response, fmt.Errorf("failed to open image file: %v", err)
	}
	defer imageFile.Close()

	imageFieldName := "file"
	imageFormFile, err := writer.CreateFormFile(imageFieldName, filepath.Base(imagePath))
	if err != nil {
		return &response, fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = io.Copy(imageFormFile, imageFile)
	if err != nil {
		return &response, fmt.Errorf("failed to copy image file: %v", err)
	}

	if err := writer.Close(); err != nil {
		return &response, fmt.Errorf("failed to close writer: %v", err)
	}

	req, err := http.NewRequest("POST", client.URL, &requestBody)
	if err != nil {
		return &response, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &response, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return &response, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	//read response body
	//body, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println("Response Body: ", string(body))
	return &response, nil
}
