# CLI-TOP Helper Functions Documentation

## Table of Contents
1. [Captcha Solver](#captcha-solver)
2. [Update and Control Module](#update-and-control-module)
3. [Table Renderer](#table-renderer)
4. [Semester Details](#semester-details)
5. [General Helpers](#general-helpers)
6. [Fuzzy Search](#fuzzy-search)
7. [Usage Tracking](#usage-tracking)
8. [Rate Limiting](#rate-limiting)
9. [ICS Generator](#ics-generator)
10. [Selection Helpers](#selection-helpers)
11. [HTTP Request Helpers](#http-request-helpers)
12. [Data Extraction](#data-extraction)
13. [Data Formatting](#data-formatting)

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

# Update and Control Module

## Overview
The Update and Control module manages version control and application behavior.

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
- `4`: Disable facility registration

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

# Table Renderer

## Overview
The Table Renderer module is a sophisticated component that provides functionality for creating, displaying, and interacting with formatted tables in the terminal. It supports both direct numeric selection and fuzzy search capabilities, making it a versatile tool for user interaction.

## Core Components

### Text Processing Functions

#### `StripAnsiCodes(str string) string`
Removes ANSI escape sequences from strings while preserving the visible text.

**Implementation Details:**
- Removes standard ANSI CSI sequences (colors, formatting)
- Handles ANSI hyperlink sequences
- Preserves visible text content
- Uses regular expressions for pattern matching

**Process Flow:**
1. Removes standard color/formatting codes
2. Extracts visible text from hyperlinks
3. Cleans up any remaining hyperlink markers

### Selection Types and Interfaces

#### `FuzzySearchFunc type`
Function type for implementing custom fuzzy search algorithms.

**Definition:**
```go
type FuzzySearchFunc func([][]string, string) []int
```

#### `SelectionResult struct`
Structure representing the outcome of a table selection operation.

**Fields:**
- `Index`: Selected row index
- `Selected`: Whether a selection was made
- `ExitRequest`: Whether user requested to exit

### Selection Functions

#### `TableSelector(subject string, nestedList [][]string, initialQuery string) SelectionResult`
Handles direct numeric selection from a table.

**Implementation Details:**
- Supports initial query processing
- Validates numeric inputs
- Provides interactive selection interface

**Process Flow:**
1. Processes initial query if provided
2. Displays formatted table
3. Handles user input validation
4. Returns selection result

**Error Handling:**
- Validates numeric input range
- Provides clear error messages
- Allows exit command

#### `TableSelectorFuzzy(subject string, nestedList [][]string, initialQuery string, fuzzySearchFunc FuzzySearchFunc) SelectionResult`
Advanced selection interface with fuzzy search support.

**Features:**
- Supports both direct selection and fuzzy search
- Handles initial queries
- Provides filtered results
- Customizable search function

**Process Flow:**
1. Attempts direct selection if numeric
2. Performs fuzzy search for text input
3. Handles single and multiple matches
4. Provides interactive refinement

**Error States:**
- Invalid numeric input
- No search matches
- Out of range selections

### Display Functions

#### `PrintTable(nestedList [][]string, indexStatus int) int`
Renders a formatted table with headers and data rows.

**Implementation Details:**
- Calculates optimal column widths
- Handles Unicode characters
- Supports ANSI color codes
- Manages multi-line content

**Features:**
- Dynamic width calculation
- Header row highlighting
- Index column management
- Proper spacing and alignment

**Usage Example:**
```go
data := [][]string{
    {"Header1", "Header2"},
    {"Data1", "Data2"},
}
PrintTable(data, 1) // 1 indicates to show index column
```

### Search Functions

#### `NewFuzzySearch(nestedList [][]string, stringFlag string) []int`
Default fuzzy search implementation.

**Algorithm Details:**
- Case-insensitive matching
- Partial string matching
- Returns matching indices
- Handles multiple matches

## Usage Examples

```go
// Basic table display
data := [][]string{
    {"ID", "Name", "Status"},
    {"1", "Item One", "Active"},
    {"2", "Item Two", "Inactive"},
}
PrintTable(data, 1)

// Interactive selection with fuzzy search
result := TableSelectorFuzzy("Item", data, "", nil)
if result.Selected {
    fmt.Printf("Selected item: %s\n", data[result.Index][1])
}
```

## Best Practices

1. **Input Validation**
   - Always validate numeric inputs
   - Handle empty or invalid queries gracefully
   - Provide clear error messages

2. **Display Formatting**
   - Consider terminal width constraints
   - Handle Unicode characters properly
   - Maintain consistent spacing

3. **User Experience**
   - Provide clear instructions
   - Support both direct and fuzzy selection
   - Allow easy exit options

4. **Performance**
   - Optimize for large datasets
   - Cache calculated widths
   - Minimize screen redraws

## Dependencies
- `bufio`: Input/output operations
- `fmt`: Formatted I/O
- `os`: Operating system interface
- `regexp`: Regular expression support
- `strconv`: String conversions
- `strings`: String manipulation

## Error Handling
- Validates all user inputs
- Provides clear error messages
- Supports graceful exit
- Handles edge cases (empty tables, invalid indices)

## Security Considerations
- Sanitizes user input
- Handles ANSI escape sequences safely
- Prevents buffer overflows
- Validates array bounds

# Semester Details

## Overview
The Semester Details module is a critical component that manages the retrieval, parsing, and selection of academic semester information. It implements robust error handling and fallback mechanisms to ensure reliable semester data access even when primary methods fail.

## Core Components

### Data Retrieval Functions

#### `GetSemDetails(cookies types.Cookies, regNo string) ([]types.Semester, error)`
Primary function for fetching semester information from the VTOP system.

**Implementation Details:**
- Validates authentication tokens
- Makes HTTP request to attendance endpoint
- Parses HTML response for semester data
- Returns structured semester information

**Process Flow:**
1. Validates cookie presence
2. Fetches data from attendance endpoint
3. Parses HTML document
4. Extracts semester information
5. Reverses list for chronological order

**Error Handling:**
- Validates authentication state
- Handles network errors
- Manages parsing failures
- Provides debug information

#### `GetSemDetailsBackup(cookies types.Cookies, regNo string) ([]types.Semester, error)`
Fallback function that attempts to retrieve semester data from an alternative endpoint.

**Implementation Details:**
- Uses course page endpoint
- Follows same parsing logic as primary function
- Maintains consistent data structure

**Usage Context:**
- Called when primary function fails
- Uses alternative data source
- Provides redundancy

### HTML Parsing Functions

#### `FindAndSaveSemIds(doc *goquery.Document) ([]types.Semester, error)`
Extracts semester information from HTML document using multiple selector strategies.

**Implementation Details:**
- Uses multiple CSS selectors for robustness
- Extracts semester IDs and names
- Validates extracted data
- Handles empty results

**Selector Strategy:**
1. Tries form-select options
2. Attempts semester-specific selectors
3. Falls back to generic selectors
4. Validates found data

**Error States:**
- Empty document handling
- Invalid selector handling
- Missing attribute handling
- Debug output for troubleshooting

### Selection Interface

#### `SelectSemester(regNo string, cookies types.Cookies, sem_choice int) (types.Semester, error)`
Provides interactive semester selection with both automatic and manual options.

**Features:**
- Supports direct selection via parameter
- Falls back to interactive selection
- Handles user cancellation
- Validates selections

**Process Flow:**
1. Retrieves semester list
2. Creates formatted table
3. Handles user input
4. Validates selection
5. Returns chosen semester

**Error Handling:**
- Invalid selection handling
- User cancellation handling
- Data retrieval failures
- Input buffer management

### Utility Functions

#### `clearInputBuffer() error`
Manages input stream cleanliness for reliable user interaction.

**Implementation Details:**
- Clears pending input
- Handles buffer states
- Manages error conditions
- Supports debug logging

## Usage Examples

```go
// Fetch semester details
cookies := types.Cookies{...}
semesters, err := GetSemDetails(cookies, "12345")
if err != nil {
    // Handle error or try backup
    semesters, err = GetSemDetailsBackup(cookies, "12345")
}

// Select semester with automatic choice
selectedSem, err := SelectSemester(regNo, cookies, 1)
if err != nil {
    // Handle error
}
```

## Best Practices

1. **Error Handling**
   - Always check authentication state
   - Implement fallback mechanisms
   - Provide clear error messages
   - Enable debug information when needed

2. **Data Validation**
   - Verify semester IDs
   - Validate semester names
   - Check for empty results
   - Ensure chronological order

3. **User Experience**
   - Support both automatic and manual selection
   - Provide clear selection interface
   - Handle cancellation gracefully
   - Maintain input buffer cleanliness

4. **Performance**
   - Cache semester data when appropriate
   - Minimize network requests
   - Optimize HTML parsing
   - Handle large datasets efficiently

## Dependencies
- `goquery`: HTML parsing
- `bufio`: Input handling
- `strings`: String manipulation
- `strconv`: Number conversion
- `fmt`: Formatted I/O
- `os`: System operations

## Error Handling
- Authentication validation
- Network error management
- Parsing error handling
- Selection validation
- Buffer management

## Security Considerations
- Validates authentication tokens
- Sanitizes user input
- Handles sensitive data appropriately
- Implements proper access controls

# General Helpers

## Overview
The General Helpers module provides a comprehensive suite of utility functions that support core application functionality. These helpers handle everything from string manipulation and file operations to calendar generation and UI formatting, ensuring consistent behavior and robust error handling across the application.

## Core Components

### String Manipulation

#### `StrToInt(str string) int`
Converts string representations of numbers to integers with debug-aware error handling.

**Implementation Details:**
- Uses `strconv.Atoi` for conversion
- Handles conversion errors gracefully
- Provides debug output when enabled
- Returns 0 for invalid inputs

#### `TruncateWithEllipsis(s string, maxLength int) string`
Smart string truncation that preserves Unicode characters.

**Features:**
- Handles multi-byte characters correctly
- Preserves string integrity
- Adds ellipsis when truncated
- Handles edge cases for small lengths

#### `SanitizeString(input string) string`
Cleanses strings for safe usage in various contexts.

**Implementation:**
- Removes control characters
- Preserves Unicode letters and numbers
- Maintains whitespace where appropriate
- Handles special characters safely

### Date and Time Formatting

#### `FormatDateTime(dateStr string) string`
Flexible date-time parsing and formatting.

**Supported Formats:**
- "02-Jan-2006 03:04 PM"
- "02-Jan-2006 15:04"
- "02-Jan-2006"
- "02/01/2006"
- "02/01/06"

**Features:**
- Multi-format parsing
- Consistent output format
- Graceful fallback
- Time zone handling

### File Operations

#### `SaveFile(data []byte, filePath string) error`
Secure file saving with proper error handling.

**Implementation Details:**
- Creates directories if needed
- Handles permissions
- Atomic write operations
- Validates file path

#### `GetFileExtension(filename string, body []byte, headers http.Header) string`
Smart file extension detection.

**Detection Methods:**
1. Original filename extension
2. MIME type from headers
3. Content-based detection
4. OOXML detection for Office files

### Calendar Integration

#### `GenerateCalendarImportLinks(icsURL string, calendarName string)`
Creates calendar import links for major platforms.

**Supported Platforms:**
- Google Calendar
- Microsoft Outlook
- Generic ICS import

**Features:**
- URL encoding
- ANSI-colored output
- Clickable links
- Platform-specific formatting

### HTTP Operations

#### `FetchReqClient(client *http.Client, ...) ([]byte, http.Header, error)`
Advanced HTTP request handling.

Use `helpers.GetHTTPClient()` to obtain the shared client when making requests.

**Features:**
- Cookie management
- Custom headers
- Multiple HTTP methods
- Response validation
- Error handling

### UI Formatting

#### `ColorStatus(status string) string`
Status text coloring for better visibility.

**Color Coding:**
- Red: Pending/Error states
- Green: Approved/Success states
- Default: Neutral states

#### `AddLeftPadding(text string, padding int) string`
Consistent text padding for UI elements.

**Features:**
- Multi-line support
- Preserves string content
- Configurable padding width
- Unicode character support

## Usage Examples

```go
// String manipulation
truncated := TruncateWithEllipsis("Long text here", 10)
sanitized := SanitizeString("User input <script>")

// Date formatting
formattedDate := FormatDateTime("15-Jan-2024 14:30")

// File operations
extension := GetFileExtension("document.docx", fileBytes, headers)
err := SaveFile(data, "downloads/file.pdf")

// Calendar integration
GenerateCalendarImportLinks("https://example.com/calendar.ics", "My Calendar")

// UI formatting
status := ColorStatus("PENDING")
paddedText := AddLeftPadding("Menu item", 4)
```

## Best Practices

1. **Error Handling**
   - Use debug-aware error reporting
   - Implement graceful fallbacks
   - Provide meaningful error contexts
   - Log errors appropriately

2. **String Processing**
   - Handle Unicode correctly
   - Validate input lengths
   - Sanitize user inputs
   - Preserve string integrity

3. **File Operations**
   - Validate paths before operations
   - Handle permissions properly
   - Use atomic operations
   - Clean up temporary files

4. **Performance**
   - Cache repeated operations
   - Use efficient algorithms
   - Minimize memory allocations
   - Implement timeouts

## Dependencies
- `strconv`: Number conversion
- `strings`: String manipulation
- `time`: Date/time handling
- `os`: File operations
- `http`: Network requests
- `filepath`: Path manipulation
- `goquery`: HTML parsing
- `filetype`: File type detection

## Error Handling
- Input validation
- File operation errors
- Network failures
- Format conversion errors
- Resource cleanup

## Security Considerations
- Path traversal prevention
- Input sanitization
- Safe file operations
- Secure HTTP handling
- Resource limits

# Fuzzy Search

## Overview
The Fuzzy Search module provides advanced string matching capabilities with support for acronyms, partial matches, and flexible search patterns. It is designed to enhance user experience by providing intelligent search functionality that understands various ways users might search for content.

## Core Components

### Basic Fuzzy Matching

#### `FuzzyMatch(query, target string) bool`
Simple subsequence-based fuzzy matching algorithm.

**Implementation Details:**
- Case-insensitive comparison
- Subsequence matching
- Empty query handling
- Length optimization checks

**Process Flow:**
1. Convert strings to lowercase
2. Validate query length
3. Check for subsequence match
4. Return match status

### Advanced Matching

#### `FuzzyMatchWithAcronym(query, target string) int`
Multi-strategy matching with priority scoring.

**Match Types:**
1. Exact Acronym Match (Score: 3)
2. Partial Acronym Match (Score: 2)
3. Fuzzy Match (Score: 1)
4. No Match (Score: 0)

**Features:**
- Case-insensitive matching
- Multi-strategy approach
- Prioritized results
- Minimum query length enforcement

### Search Implementation

#### `FuzzySearchWithAcronym(nestedList [][]string, stringFlag string) []int`
Comprehensive search across nested string arrays.

**Search Strategy:**
1. Empty Query Handling
   - Returns all non-header rows
   - Preserves data structure

2. Priority-based Search
   - Course title and code priority
   - Exact acronym matches
   - Partial acronym matches
   - Fuzzy matches

3. Result Aggregation
   - Exclusive exact matches
   - Combined partial and fuzzy matches
   - Index-based result tracking

### Acronym Processing

#### `computeAcronyms(target string) (filtered, unfiltered string)`
Intelligent acronym generation with filler word handling.

**Filler Words:**
```go
fillerWords := map[string]bool{
    "and":  true,
    "of":   true,
    "for":  true,
    "the":  true,
    "in":   true,
    "with": true,
    "to":   true,
    "a":    true,
    "an":   true,
}
```

**Processing Steps:**
1. Word splitting
2. First letter extraction
3. Filler word filtering
4. Case normalization

### Acronym Matching

#### `exactAcronymMatch(query, target string) bool`
Precise acronym comparison with multiple variants.

**Match Types:**
- Filtered acronym match
- Unfiltered acronym match
- Case-insensitive comparison

#### `acronymMatch(query, target string) bool`
Flexible acronym matching with compound word support.

**Variants Generated:**
1. Simple Filtered
2. Compound Filtered
3. Simple Unfiltered
4. Compound Unfiltered

**Features:**
- Camel case handling
- Multi-word processing
- Filler word filtering
- Case normalization

## Usage Examples

```go
// Basic fuzzy matching
matched := FuzzyMatch("prgm", "Programming")  // true

// Advanced matching with scoring
score := FuzzyMatchWithAcronym("AI", "Artificial Intelligence")  // 3
score = FuzzyMatchWithAcronym("ML", "Machine Learning")         // 3

// Searching in nested data
data := [][]string{
    {"Course", "Code"},
    {"Artificial Intelligence", "CSE4001"},
    {"Machine Learning", "CSE4002"},
}
results := FuzzySearchWithAcronym(data, "AI")  // [1]
```

## Best Practices

1. **Query Optimization**
   - Use minimum query length (3+) for fuzzy matches
   - Prioritize exact matches
   - Handle empty queries appropriately
   - Consider case sensitivity

2. **Performance**
   - Implement early returns
   - Use length checks
   - Cache computed acronyms
   - Optimize string operations

3. **User Experience**
   - Provide feedback on match quality
   - Sort results by relevance
   - Handle partial matches gracefully
   - Support multiple search patterns

4. **Maintenance**
   - Update filler words list
   - Monitor match quality
   - Adjust scoring weights
   - Document edge cases

## Dependencies
- `strings`: String manipulation
- `unicode`: Character processing
- Built-in Go types and functions

## Error Handling
- Empty string handling
- Length validation
- Index bounds checking
- Type safety checks

## Performance Considerations
- Early termination
- Memory efficiency
- String allocation
- Algorithm complexity
- Cache utilization

## Security Considerations
- Input validation
- Resource limits
- Memory constraints
- Processing timeouts

# Usage Tracking

## Overview
The Usage Tracking module provides robust functionality for registering and managing unique identifiers (UUIDs) for application instances. It implements a reliable registration system with retry mechanisms, error handling, and persistent storage of registration status.

## Core Components

### UUID Registration

#### `RegisterUUID(uuid string) error`
Primary function for registering application instances with the tracking server.

**Implementation Details:**
- JSON-based registration payload
- HTTP POST request handling
- Configurable retry mechanism
- Persistent configuration storage

**Process Flow:**
1. Data Preparation
   - UUID marshaling
   - JSON payload creation
   - Request configuration

2. Registration Attempt
   - HTTP request creation
   - Timeout configuration
   - Response handling
   - Status code processing

3. Configuration Update
   - Success status storage
   - UUID persistence
   - Unregistered status clearing

**Constants:**
```go
const maxRetries = 3  // Maximum registration attempts
```

### Request Handling

#### HTTP Configuration
- Method: POST
- Endpoint: https://cli-calendar.acmvit.in/register
- Content-Type: application/json
- Timeout: 5 seconds

#### Retry Mechanism
- Maximum attempts: 3
- Delay between retries: 2 seconds
- Progressive error reporting
- Debug-aware logging

### Response Processing

#### Status Code Handling
1. **201 Created**
   - Successful registration
   - Configuration update
   - Debug logging
   - UUID persistence

2. **409 Conflict**
   - Already registered
   - Configuration update
   - Debug logging
   - Status normalization

3. **Other Codes**
   - Error reporting
   - Retry triggering
   - Debug information
   - Failure tracking

### Configuration Management

#### Viper Integration
- UUID storage
- Unregistered status tracking
- Configuration persistence
- Error handling

## Usage Examples

```go
// Register a new UUID
uuid := "550e8400-e29b-41d4-a716-446655440000"
err := RegisterUUID(uuid)
if err != nil {
    // Handle registration failure
}

// Registration with debug logging
debug.Debug = true
err = RegisterUUID(uuid)
if err != nil {
    // Detailed error information available
}
```

## Best Practices

1. **Error Handling**
   - Implement retry logic
   - Log failures appropriately
   - Provide debug information
   - Handle network issues

2. **Configuration Management**
   - Persist registration status
   - Handle write failures
   - Maintain state consistency
   - Clean up old data

3. **Network Operations**
   - Use appropriate timeouts
   - Handle connection failures
   - Implement backoff strategy
   - Validate responses

4. **Security**
   - Validate UUIDs
   - Use HTTPS
   - Handle sensitive data
   - Protect configuration

## Dependencies
- `encoding/json`: JSON handling
- `net/http`: Network operations
- `time`: Timeout and retry delays
- `github.com/spf13/viper`: Configuration
- `cli-top/debug`: Debug logging
- `cli-top/types`: Data structures

## Error Handling
- JSON marshaling errors
- Network request failures
- Configuration write errors
- Invalid response codes
- Timeout handling

## Performance Considerations
- Retry limits
- Timeout configuration
- Response processing
- Configuration updates
- Memory usage

## Security Considerations
- HTTPS usage
- UUID validation
- Configuration protection
- Error message safety
- Rate limiting compliance

# Rate Limiting

## Overview
The Rate Limiting module implements a sliding window rate limiter to control download operations and prevent server overload. It provides thread-safe operation tracking with automatic window management and configurable limits.

## Core Components

### Rate Limiter Configuration

#### Constants
```go
const MaxDownloads = 100               // Maximum operations per window
const RateLimitWindow = 10 * time.Minute  // Time window size
```

#### Global State
```go
var (
    mu                 sync.Mutex    // Thread synchronization
    downloadTimestamps []time.Time   // Operation tracking
)
```

### Rate Limiting Functions

#### `IsRateLimitExceeded() bool`
Thread-safe rate limit checker with sliding window implementation.

**Implementation Details:**
- Mutex-based synchronization
- Sliding window tracking
- Automatic window cleanup
- Operation counting

**Process Flow:**
1. Lock Acquisition
   - Mutex locking
   - Deferred unlock
   - Thread safety

2. Window Management
   - Current time capture
   - Expired entry removal
   - Window sliding
   - Timestamp cleanup

3. Limit Checking
   - Count verification
   - Threshold comparison
   - New entry addition
   - Status return

### Operation Tracking

#### Timestamp Management
- Ordered timestamp storage
- FIFO queue behavior
- Automatic expiration
- Memory optimization

#### Window Sliding
- Dynamic window adjustment
- Expired entry removal
- Continuous operation
- Resource efficiency

## Usage Examples

```go
// Check rate limit before operation
if IsRateLimitExceeded() {
    // Handle rate limit exceeded
    return errors.New("rate limit exceeded")
}

// Perform operation if allowed
// The operation is automatically tracked

// Multiple goroutine usage
go func() {
    if !IsRateLimitExceeded() {
        // Perform concurrent operation
    }
}()
```

## Best Practices

1. **Thread Safety**
   - Always use mutex protection
   - Implement proper locking
   - Handle concurrent access
   - Avoid deadlocks

2. **Memory Management**
   - Clean expired timestamps
   - Optimize slice capacity
   - Prevent memory leaks
   - Monitor growth

3. **Performance**
   - Minimize lock duration
   - Optimize cleanup operations
   - Handle high concurrency
   - Reduce allocations

4. **Error Handling**
   - Handle limit exceeded cases
   - Provide clear feedback
   - Implement backoff strategies
   - Monitor rate limits

## Dependencies
- `sync`: Mutex operations
- `time`: Timestamp handling
- Built-in Go types

## Implementation Details

### Mutex Operations
- Lock acquisition
- Deferred unlocking
- Critical section protection
- Race condition prevention

### Timestamp Management
- Slice-based storage
- FIFO queue behavior
- Automatic cleanup
- Memory efficiency

### Window Operations
- Sliding window implementation
- Dynamic size adjustment
- Expired entry removal
- Continuous operation

## Performance Considerations

1. **Lock Contention**
   - Minimize critical section
   - Optimize lock duration
   - Handle high concurrency
   - Monitor lock waiting

2. **Memory Usage**
   - Efficient slice usage
   - Regular cleanup
   - Capacity management
   - Allocation optimization

3. **Time Complexity**
   - O(1) limit checking
   - O(n) cleanup operation
   - Efficient sorting
   - Quick response time

## Security Considerations

1. **Rate Limit Enforcement**
   - Strict limit adherence
   - No bypass mechanisms
   - Secure counter
   - Protected state

2. **Resource Protection**
   - Memory bounds
   - CPU usage limits
   - Storage constraints
   - Network protection

3. **Thread Safety**
   - Race condition prevention
   - Deadlock avoidance
   - State protection
   - Concurrent access control

## Error States
- Rate limit exceeded
- Resource exhaustion
- Concurrent overload
- System constraints

# ICS Generator

## Overview
The ICS Generator module provides comprehensive functionality for creating, managing, and sharing calendar files in the iCalendar format (ICS). It supports various calendar event types, handles file operations, and implements secure file sharing through a dedicated server.

## Core Components

### Calendar File Generation

#### `GenerateICSFileDateOnly(events []types.ICSEvent, filePath string, calName string) error`
Creates ICS files with date-only events.

**Implementation Details:**
- iCalendar 2.0 compliance
- UTF-8 encoding
- CRLF line endings
- Proper character escaping

**File Structure:**
```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//CLI-TOP//EN
X-WR-CALNAME:[Calendar Name]
[Event Blocks]
END:VCALENDAR
```

**Event Properties:**
- UID: Unique identifier
- DTSTAMP: Creation timestamp
- DTSTART: Event start (DATE)
- DTEND: Event end (DATE)
- SUMMARY: Event title
- DESCRIPTION: Event details

### Location-Aware Events

#### `VenueAdd(events []types.ICSWithLocation, filePath string, calName string) error`
Generates ICS files with location information.

**Features:**
- Location support
- Timezone handling
- Extended properties
- Rich descriptions

**Event Format:**
```
BEGIN:VEVENT
DTSTART;[Timezone Info]
DTEND;[Timezone Info]
SUMMARY:[Event Title]
DESCRIPTION:[Details]
DTSTAMP:[UTC Timestamp]
END:VEVENT
```

### Utility Functions

#### `GenerateUID(prefix string) string`
Creates unique event identifiers.

**Implementation:**
- Cryptographic randomness
- Prefix customization
- Fallback mechanism
- Hex encoding

#### `GetDownloadsDir() string`
Determines system download directory.

**Platform Support:**
- Windows
- Unix/Linux
- macOS
- Fallback handling

### File Sharing

#### `UploadICSFile(filePath string, serverURL string) (string, error)`
Handles secure file uploads for sharing.

**Features:**
- Multipart form upload
- HTTP client configuration
- Response handling
- URL generation

**Process Flow:**
1. File preparation
2. Form data creation
3. HTTP request
4. Response parsing

## Usage Examples

```go
// Generate basic calendar
events := []types.ICSEvent{
    {
        UID:         GenerateUID("EVENT"),
        DtStart:     "20240315",
        DtEnd:       "20240316",
        Summary:     "Team Meeting",
        Description: "Monthly review",
    },
}
err := GenerateICSFileDateOnly(events, "calendar.ics", "Team Calendar")

// Generate calendar with locations
locEvents := []types.ICSWithLocation{
    {
        Event: types.ICSEvent{
            DtStart:     "20240315T100000",
            DtEnd:       "20240315T110000",
            Summary:     "Conference",
            Description: "Annual meeting",
        },
    },
}
err = VenueAdd(locEvents, "conference.ics", "Conference Schedule")

// Share calendar
url, err := UploadICSFile("calendar.ics", "https://calendar-server.com")
```

## Best Practices

1. **File Generation**
   - Validate event data
   - Handle character escaping
   - Use proper line endings
   - Implement error checking

2. **Date Handling**
   - Use UTC for timestamps
   - Validate date formats
   - Handle timezones properly
   - Consider DST

3. **File Operations**
   - Clean up resources
   - Handle permissions
   - Validate paths
   - Implement timeouts

4. **Error Handling**
   - Validate input data
   - Handle network errors
   - Provide clear messages
   - Implement retries

## Dependencies
- `crypto/rand`: UID generation
- `encoding/hex`: UID encoding
- `encoding/json`: Response parsing
- `net/http`: File upload
- `os`: File operations
- `time`: Timestamp handling

## Implementation Details

### Event Generation
- iCalendar compliance
- Property formatting
- Character escaping
- Line wrapping

### File Operations
- Resource management
- Error handling
- Path resolution
- Permission checking

### Upload Process
- Form data creation
- HTTP client setup
- Response handling
- Error management

## Performance Considerations

1. **File Writing**
   - Buffer usage
   - Memory management
   - I/O optimization
   - Resource cleanup

2. **Upload Operations**
   - Connection pooling
   - Timeout configuration
   - Memory efficiency
   - Response handling

3. **Data Processing**
   - String operations
   - Memory allocation
   - Slice management
   - Buffer usage

## Security Considerations

1. **File Operations**
   - Path validation
   - Permission checks
   - Resource limits
   - Cleanup handling

2. **Upload Security**
   - HTTPS usage
   - File validation
   - Size limits
   - Error handling

3. **Data Protection**
   - Input validation
   - Output escaping
   - Secure random
   - Error messages

# Selection Helpers

## Overview
The Selection Helpers module provides a comprehensive suite of functions for handling course and faculty data manipulation, text formatting, and interactive selection interfaces. It implements sophisticated string processing, user interaction handling, and data presentation capabilities.

## Core Components

### String Processing

#### `RemoveCourseCode(courseName string) string`
Strips course codes from course names using regex.

**Implementation Details:**
- Regex pattern: `^[A-Z]{4}\d{3}[A-Z]?\s*[─-]\s*`
- Handles various dash types
- Preserves course name
- Trims whitespace

#### `SplitCourseName(courseName string) (string, string)`
Separates course code from course name.

**Features:**
- Multiple separator support
- Whitespace handling
- Empty case handling
- Clean output format

#### `RedactERPID(facultyName string) string`
Removes ERP IDs from faculty names.

**Implementation:**
- Regex-based removal
- Multiple separator support
- Whitespace cleanup
- Format preservation

### Text Formatting

#### `TruncateString(str string, maxLength int) string`
Smart string truncation with ellipsis.

**Features:**
- Length preservation
- Ellipsis handling
- Edge case management
- Unicode support

#### `HighlightMatches(text, query string) string`
Highlights search matches in text.

**Implementation:**
- Case-insensitive matching
- ANSI color support
- Regex-based replacement
- Pattern escaping

### Data Processing

#### `SplitCourseNameFull(courseName string) []string`
Complete course name parsing.

**Processing Steps:**
1. Separator detection
2. String splitting
3. Whitespace cleanup
4. Array generation

#### `RemoveDuplicateFaculties(faculties []types.Faculty) []types.Faculty`
Deduplicates faculty entries.

**Features:**
- Maintains order
- Preserves data
- Memory efficient
- Fast lookup

### Selection Interface

#### `SelectCourseMaterials(materials []types.CourseMaterial) ([]types.CourseMaterial, error)`
Interactive material selection.

**Features:**
- Multi-select support
- Search functionality
- Cancel operation
- Error handling

## Usage Examples

```go
// String processing
courseName := RemoveCourseCode("CSE1001 - Introduction to Programming")
code, name := SplitCourseName("CSE1001 - Programming Fundamentals")

// Text formatting
truncated := TruncateString("Long course name", 10)
highlighted := HighlightMatches("Programming", "prog")

// Faculty processing
facultyName := RedactERPID("12345 - Dr. Smith")
parts := SplitFacultyNameFull("12345 - Smith, John - CS")

// Selection handling
materials, err := SelectCourseMaterials(courseData)
if err != nil {
    // Handle selection error
}
```

## Best Practices

1. **String Processing**
   - Use consistent regex patterns
   - Handle edge cases
   - Validate input
   - Clean output

2. **User Interface**
   - Clear instructions
   - Error feedback
   - Cancel options
   - Input validation

3. **Data Handling**
   - Validate input data
   - Handle duplicates
   - Preserve order
   - Clean output

4. **Error Management**
   - Descriptive messages
   - Debug logging
   - Recovery options
   - User feedback

## Dependencies
- `regexp`: Pattern matching
- `strings`: String manipulation
- `fmt`: I/O operations
- `sort`: Data ordering
- `cli-top/types`: Data structures
- `cli-top/debug`: Debug logging

## Implementation Details

### Regular Expressions
- Course code pattern
- Faculty ID pattern
- Separator patterns
- Match highlighting

### String Operations
- Splitting logic
- Truncation rules
- Highlighting implementation
- Character replacement

### Data Structures
- Faculty types
- Course materials
- Selection results
- Search indices

## Performance Considerations

1. **Regex Operations**
   - Pattern compilation
   - Caching
   - Optimization
   - Memory usage

2. **String Processing**
   - Buffer management
   - Memory allocation
   - String building
   - UTF-8 handling

3. **Data Management**
   - Efficient sorting
   - Quick lookups
   - Memory efficiency
   - Cache usage

## Security Considerations

1. **Input Validation**
   - Sanitize input
   - Length limits
   - Pattern checks
   - Type safety

2. **Output Safety**
   - Escape special chars
   - Format validation
   - Buffer limits
   - Safe display

3. **Error Handling**
   - Safe messages
   - Resource cleanup
   - State recovery
   - Debug safety

## Error States
- Invalid input
- Selection cancelled
- Parse failures
- Resource limits
- Network issues

# HTTP Request Helpers

## Overview
The HTTP Request Helpers module provides a robust and flexible HTTP client implementation specifically designed for interacting with VTOP services. It handles authentication, request formatting, and response processing with comprehensive error handling and debug support.

## Core Components

### Request Handler

#### `FetchReq(regNo string, cookies types.Cookies, url string, semID string, payload string, method string, header string) ([]byte, error)`
Primary function for making HTTP requests to VTOP services.

**Implementation Details:**
- Custom HTTP client configuration
- Cookie management
- Payload formatting
- Method handling

**Parameters:**
```go
regNo string       // Registration number
cookies types.Cookies  // Session cookies
url string         // Target endpoint
semID string       // Semester identifier
payload string     // Request payload
method string      // HTTP method
header string      // Special header flags
```

### Request Configuration

#### Payload Generation
Handles different payload formats:

1. **Default Format**
```go
verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d
```

2. **UTC Format**
```go
authorizedID=%s&_csrf=%s&semesterSubId=%s&x=%s
```

#### Header Management
- Content-Type handling
- Cookie formatting
- Special headers
- Debug information

### Method Support

#### POST Requests
**Features:**
- Payload encoding
- Form data handling
- Boundary setting
- Content-Type selection

#### GET Requests
**Features:**
- Query parameter handling
- URL encoding
- No payload
- Header configuration

## Usage Examples

```go
// Basic GET request
cookies := types.Cookies{
    SERVERID:   "server123",
    JSESSIONID: "session456",
    CSRF:       "token789",
}
body, err := FetchReq(
    "12345",     // regNo
    cookies,     // cookies
    "https://vtop.example.com/endpoint",  // url
    "",          // semID
    "",          // payload
    "GET",       // method
    "",          // header
)

// POST request with custom payload
body, err = FetchReq(
    "12345",
    cookies,
    "https://vtop.example.com/marks",
    "FAL2023",
    "custom_payload",
    "POST",
    "marks",
)
```

## Best Practices

1. **Error Handling**
   - Check response status
   - Handle network errors
   - Validate responses
   - Debug logging

2. **Cookie Management**
   - Validate cookies
   - Handle expiration
   - Secure storage
   - Session tracking

3. **Request Formation**
   - Validate parameters
   - Encode data properly
   - Set correct headers
   - Handle timeouts

4. **Response Processing**
   - Check content type
   - Handle large responses
   - Process errors
   - Clean up resources

## Dependencies
- `bytes`: Buffer management
- `net/http`: HTTP client
- `io`: Response reading
- `time`: Timestamps
- `cli-top/types`: Data structures
- `cli-top/debug`: Debug logging

## Implementation Details

### HTTP Client
- Custom configuration
- Connection pooling
- Timeout handling
- Keep-alive settings

### Request Building
- Method validation
- Header construction
- Payload formatting
- Cookie formatting

### Response Handling
- Body reading
- Error checking
- Resource cleanup
- Debug output

## Performance Considerations

1. **Connection Management**
   - Connection reuse
   - Keep-alive settings
   - Connection pooling
   - Resource cleanup

2. **Memory Usage**
   - Buffer management
   - Response streaming
   - Memory allocation
   - Garbage collection

3. **Error Handling**
   - Quick failure
   - Resource release
   - Connection reset
   - Timeout management

## Security Considerations

1. **Data Protection**
   - HTTPS usage
   - Cookie security
   - Payload encryption
   - Header sanitization

2. **Authentication**
   - Cookie validation
   - Session management
   - Token handling
   - Secure storage

3. **Error Messages**
   - Safe error reporting
   - Debug mode control
   - Information hiding
   - Secure logging

## Error States
- Network failures
- Invalid methods
- Bad responses
- Timeout errors
- Cookie issues
- Payload errors

# Data Extraction

## Overview
The Data Extraction module provides specialized functions for parsing and extracting critical data from HTML responses and HTTP headers. It implements robust pattern matching, error handling, and debug-aware extraction for various data types including CSRF tokens, cookies, images, and registration numbers.

## Core Components

### Cookie Extraction

#### `ExtractCookies(resp *http.Response) types.Cookies`
Extracts and processes session cookies from HTTP responses.

**Implementation Details:**
- Cookie map creation
- Response validation
- Cookie filtering
- Type conversion

**Cookie Structure:**
```go
type Cookies struct {
    SERVERID   string
    CSRF       string
    JSESSIONID string
}
```

### CSRF Token Extraction

#### `ExtractCSRF(bodyString string) string`
Extracts CSRF token using regex pattern matching.

**Implementation Details:**
- Regex-based extraction
- Multiple pattern support
- Error handling
- Debug logging

**Pattern Matching:**
```go
// Primary pattern
var csrfValue = /*(.*?)*/'.*';

// Secondary pattern (ExtractCSRF2)
var csrfValue = "([a-fA-F0-9-]+)";
```

### Image Extraction

#### `ExtractImage(html string) string`
Extracts captcha image source from HTML.

**Features:**
- DOM parsing
- Attribute extraction
- Error handling
- Retry support

**Implementation:**
```go
func extractImageSrc(html string) (string, error) {
    doc.Find("#captchaBlock img").AttrOr("src", "")
    // ... error handling and processing
}
```

### Registration Number Extraction

#### `ExtractRegNo(bodyString string) (string, error)`
Extracts student registration number from HTML.

**Features:**
- Pattern matching
- Error reporting
- Validation
- Clean output

**Pattern:**
```go
let id\s*=\s*"(.*?)";
```

### Body Text Extraction

#### `ExtractBodyText(resp *http.Response) string`
Extracts and processes response body text.

**Features:**
- Stream reading
- Error handling
- Memory management
- Debug support

## Usage Examples

```go
// Extract cookies from response
cookies := ExtractCookies(response)
fmt.Printf("JSESSIONID: %s\n", cookies.JSESSIONID)

// Extract CSRF token
csrf := ExtractCSRF(bodyString)
if csrf == "" {
    csrf = ExtractCSRF2(bodyString)  // Try alternate pattern
}

// Extract captcha image
imageSrc := ExtractImage(html)
if imageSrc != "nocaptcha" {
    // Process image source
}

// Extract registration number
regNo, err := ExtractRegNo(bodyString)
if err != nil {
    // Handle extraction error
}
```

## Best Practices

1. **Pattern Matching**
   - Use precise patterns
   - Handle edge cases
   - Validate matches
   - Implement fallbacks

2. **Error Handling**
   - Check input validity
   - Handle nil responses
   - Provide debug info
   - Clean error messages

3. **Performance**
   - Optimize regex
   - Manage memory
   - Handle large inputs
   - Cache patterns

4. **Security**
   - Validate output
   - Sanitize data
   - Handle sensitive info
   - Secure storage

## Dependencies
- `regexp`: Pattern matching
- `strings`: String manipulation
- `io`: Response reading
- `net/http`: HTTP types
- `github.com/PuerkitoBio/goquery`: HTML parsing
- `cli-top/types`: Custom types
- `cli-top/debug`: Debug support

## Implementation Details

### Pattern Compilation
- Regex optimization
- Pattern caching
- Error handling
- Debug support

### HTML Parsing
- DOM traversal
- Attribute extraction
- Node selection
- Error recovery

### Cookie Processing
- Header parsing
- Value extraction
- Type conversion
- Validation

## Performance Considerations

1. **Regex Operations**
   - Pattern compilation
   - Match efficiency
   - Memory usage
   - Cache usage

2. **HTML Processing**
   - DOM parsing
   - Memory management
   - Node traversal
   - Attribute lookup

3. **Response Handling**
   - Stream processing
   - Buffer management
   - Memory allocation
   - Resource cleanup

## Security Considerations

1. **Data Validation**
   - Input sanitization
   - Output validation
   - Pattern safety
   - Error handling

2. **Sensitive Data**
   - Cookie handling
   - Token protection
   - Debug logging
   - Data cleanup

3. **Error Messages**
   - Safe reporting
   - Debug control
   - Information hiding
   - Secure logging

## Error States
- Invalid patterns
- Missing data
- Parse failures
- Nil responses
- Empty results
- Invalid formats

# Data Formatting

## Overview
The Data Formatting module provides efficient and secure functions for formatting data structures into standardized string formats. It specializes in converting map-based data into URL-encoded form data and cookie strings, with optimized string building and proper delimiter handling.

## Core Components

### Form Data Formatting

#### `FormatBodyData(bodyData map[string]string) string`
Converts map data into URL-encoded form data string.

**Implementation Details:**
- String builder usage
- Key-value pairing
- Delimiter handling
- Trailing cleanup

**Format Pattern:**
```
key1=value1&key2=value2&key3=value3
```

**Features:**
- Memory efficient
- No allocations
- Clean output
- Safe encoding

### Cookie String Formatting

#### `FormatCookies(cookies map[string]string) string`
Formats cookie map into HTTP cookie header string.

**Implementation Details:**
- String builder usage
- Cookie pair formatting
- Delimiter handling
- Whitespace management

**Format Pattern:**
```
key1=value1; key2=value2; key3=value3
```

**Features:**
- RFC compliance
- Memory efficient
- Clean formatting
- Safe encoding

## Usage Examples

```go
// Format form data
bodyData := map[string]string{
    "username": "john_doe",
    "action":   "login",
    "token":    "abc123",
}
formString := FormatBodyData(bodyData)
// Output: username=john_doe&action=login&token=abc123

// Format cookies
cookies := map[string]string{
    "session": "xyz789",
    "theme":   "dark",
    "lang":    "en",
}
cookieString := FormatCookies(cookies)
// Output: session=xyz789; theme=dark; lang=en
```

## Best Practices

1. **Memory Management**
   - Use string builder
   - Minimize allocations
   - Pre-size buffers
   - Clean up resources

2. **String Formatting**
   - Handle empty values
   - Escape special chars
   - Validate output
   - Check lengths

3. **Performance**
   - Optimize building
   - Reduce allocations
   - Cache results
   - Handle large maps

4. **Security**
   - Validate input
   - Escape output
   - Check lengths
   - Handle special chars

## Dependencies
- `strings`: String operations
- Built-in Go types

## Implementation Details

### String Building
- Builder initialization
- Key-value writing
- Delimiter insertion
- Trailing cleanup

### Format Processing
- Map iteration
- String concatenation
- Delimiter management
- Length handling

### Output Cleaning
- Trailing removal
- Whitespace handling
- Format validation
- Safety checks

## Performance Considerations

1. **String Building**
   - Minimize allocations
   - Optimize capacity
   - Reduce copying
   - Buffer reuse

2. **Map Processing**
   - Efficient iteration
   - Memory usage
   - Key ordering
   - Value handling

3. **Output Generation**
   - Clean formatting
   - Memory efficiency
   - String slicing
   - Buffer management

## Security Considerations

1. **Input Validation**
   - Key validation
   - Value checking
   - Length limits
   - Character sets

2. **Output Safety**
   - Proper escaping
   - Format validation
   - Length checking
   - Character encoding

3. **Data Protection**
   - Sensitive data
   - Cookie safety
   - Form security
   - Error handling

## Error States
- Invalid input
- Empty maps
- Large values
- Special characters
- Format errors
- Length limits
