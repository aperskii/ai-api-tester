# Ai-Face-Detection-profiles-tester


This program tests each profile with params.json against two Ai-Face-Detection servers (SERVER1_URL and SERVER2_URL) using images from a specified directory (in folder). The results are saved to a CSV file named "differences.csv" and the response images are downloaded to a out folder.

## Installation

first check folder /docs i put there exemple for .env and check param.json in each profile folder in /profiles 

To run this program, you'll need to modify the .env file with your own values for: 


* SERVER1_URL: URL of the first Odapi server
* SERVER2_URL: URL of the second Odapi server
* IMAGE_URL1_OUT: Output image URL from the first Odapi response
* IMAGE_URL2_OUT: Output image URL from the second Odapi response
* PROFILES_DIR: Path to the directory containing profiles for testing

## Usage


1. Clone this repository and navigate into it.
2. Create a .env file with your own values as described above.
3. Check /Profiles folder and param.json for each profiles
4. Add picture what you want to send in server in /in folder
3. Run the program using Go: go run main.go
4. The results will be written to "differences.csv" in the /out directory.

## Requirements


* Go (tested on version 1.x)
* Ai-Face-Detection servers (SERVER1_URL and SERVER2_URL)
* Images for testing in a specified directory
* .env file with required values
