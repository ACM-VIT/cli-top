# Captcha Solver

## Overview
The Captcha Solver module is a sophisticated component that handles automated CAPTCHA recognition.

## Core Components

### Image Preprocessing Functions

#### `preImg(img [][]int) [][]int`
Performs binary thresholding on the input image to separate foreground from background.

**Implementation Details:**
- Calculates the average pixel value across the entire image
- Creates a binary image where pixels above average become 1, below become 0
- Dimensions are preserved in the output

**Usage Example:**
```go
binaryImage := preImg(grayscaleImage)
// Returns a binary image where text is separated from background
```

#### `saturation(d []uint8) [][][]int`
Processes raw image data to extract character regions based on saturation values.

**Process Flow:**
1. Converts RGBA pixel data to saturation values
2. Reshapes the data into a 40x200 image matrix
3. Extracts 6 character blocks using specific coordinate calculations

**Key Parameters:**
- Input expects raw pixel data in RGBA format (4 bytes per pixel)
- Output provides 6 separate character regions as 3D array

### Matrix Operations

#### `copySlice(src [][]int, transform func([]int) []int) [][]int`
Creates a deep copy of a 2D slice with custom transformation.

**Features:**
- Supports custom transformation functions
- Preserves memory independence between source and destination
- Commonly used for character region extraction

#### `flatten(arr [][]int) []int` and `flattenFloat32(arr [][]float32) []float32`
Convert multi-dimensional arrays to single-dimensional arrays.

**Usage Context:**
- Prepare data for neural network input
- Maintain data continuity during matrix operations

#### `matMul(a [][]int, b [][]float32) []float32`
Performs matrix multiplication between integer and float matrices.

**Implementation Notes:**
- Optimized for neural network weight calculations
- Handles type conversion automatically
- Returns flattened result for immediate use

#### `matAdd(a []float32, b []float32) []float32`
Performs element-wise addition of two float arrays.

**Error Handling:**
- Requires equal-length input arrays
- Used primarily for adding bias terms in neural network

### Neural Network Components

#### `maxSoft(a []float32) []float32`
Implements the softmax activation function for classification output.

**Mathematical Process:**
1. Exponentiates each input value
2. Normalizes by sum of exponentials
3. Returns probabilities that sum to 1

#### `argmax(slice []float32) int`
Finds the index of the maximum value in a float array.

**Implementation Details:**
- Uses custom key-value structure for sorting
- Maintains original indices during sorting
- Returns index of highest probability class

### Main Function

#### `SolveCaptcha(imageURL string) string`
Orchestrates the complete CAPTCHA solving process.

**Process Flow:**
1. Validates input URL format (base64 JPEG)
2. Decodes base64 data to image
3. Processes image through neural network
4. Returns recognized characters

**Kill Switch States:**
- 0: Automated solving enabled
- 1: Manual solving required
- 2: Completely disabled

**Error Handling:**
- Handles base64 decoding errors
- Manages file operations safely
- Provides debug output when enabled

## Usage Examples

```go
// Example of solving a CAPTCHA
captchaURL := "data:image/jpeg;base64,..."
result := SolveCaptcha(captchaURL)
```

## Best Practices

1. **Image Processing**
   - Ensure input images are properly formatted JPEG
   - Validate image dimensions before processing
   - Handle memory efficiently for large images

2. **Error Handling**
   - Check debug.Debug flag before logging
   - Clean up temporary files properly
   - Validate all array dimensions before operations

3. **Performance Optimization**
   - Reuse allocated slices when possible
   - Consider batch processing for multiple images
   - Monitor memory usage for large images

## Dependencies
- `cli-top/debug`: Debug configuration
- `encoding/base64`: Base64 encoding/decoding
- `image`, `image/jpeg`: Image processing
- `math`: Mathematical operations
- `sort`: Sorting operations

# Update and Control Module Documentation

## Overview
The Update and Control module is a critical component of the CLI-Top application that manages version control and application behavior. This module implements two primary functions: version checking and application control through a kill switch mechanism. These functions are essential for maintaining application security, ensuring users have access to the latest features, and controlling application behavior remotely when necessary.

## Version Management Function

### `CheckUpdate()`
This function verifies whether the user is running the latest version of CLI-Top by comparing the local version against the remote version information.

**Implementation Details:**

The function follows a systematic process to check for updates:

1. HTTP Client Setup
```go
client := &http.Client{}
req, err := http.NewRequest("GET", "https://cli-top.acmvit.in/latest.json", nil)
```
The function creates a new HTTP client and request to fetch version information from the central server. The request is configured as a GET request to the latest.json endpoint which contains the information about the version of CLI-TOP and the kill-switch state.

2. Version Comparison
```go
if !strings.Contains(string(bodyText), debug.Version) {
    fmt.Println("A new version of cli-top is available.\nCheck out: https://cli-top.acmvit.in/ for the latest release.")
}
```
The function performs a simple string comparison to determine if an update is available. This comparison relies on the debug.Version constant being present in the response if the user has the latest version.

## Application Control Function

### `CheckKillSwitch() int`
This function implements a remote control mechanism for the application, allowing administrators to modify application behavior based on system requirements or security concerns.

**Return Values:**
- `0`: Normal operation - automated captcha solver enabled
- `1`: Restricted operation - manual captcha solving required
- `2`: Complete shutdown - application disabled
- `3`: Complete shutdown + open VTOP in browser

**Implementation Details:**

1. Kill Switch State Detection
```go
if strings.Contains(string(bodyText), "\"killSwitch\": 2") {
    return 2  // Complete shutdown
} else if strings.Contains(string(bodyText), "\"killSwitch\": 0") {
    return 0  // Normal operation
}
return 1  // Restricted operation which requires manual captcha solving
```
The function uses string matching to determine the current kill switch state, implementing a fallback mechanism that defaults to restricted operation.
