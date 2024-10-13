package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/reujab/wallpaper"
)

// Config variables
var airport string
var outputImage string
var backgroundImage string
var userName string
var boldFont = "assets/bold.ttf"
var regularFont = "assets/regular.ttf"
var lightFont = "assets/light.ttf"

// METAR struct and parsing
type METAR struct {
	StationID       string
	ObservationTime time.Time
	Wind            Wind
	Visibility      string
	Weather         []string
	CloudLayers     []CloudLayer
	Temperature     float64
	DewPoint        float64
	Altimeter       float64
	Remarks         string
}

type Wind struct {
	Direction int
	Speed     int
	Gust      int
}

type CloudLayer struct {
	Coverage string
	Height   int
}

func ParseMETAR(metarString string) (METAR, error) {
	parts := strings.Fields(metarString)
	metar := METAR{}

	// Ensure the parts array has enough fields to avoid index out of range errors
	if len(parts) < 2 {
		return metar, fmt.Errorf("METAR data is incomplete or malformed: %s", metarString)
	}

	// Station ID
	metar.StationID = parts[0]

	// Observation Time
	if len(parts[1]) >= 6 {
		timeStr := parts[1][:6]
		day, _ := strconv.Atoi(timeStr[:2])
		hour, _ := strconv.Atoi(timeStr[2:4])
		min, _ := strconv.Atoi(timeStr[4:6])
		metar.ObservationTime = time.Date(time.Now().Year(), time.Now().Month(), day, hour, min, 0, 0, time.UTC)
	} else {
		return metar, fmt.Errorf("invalid observation time format in METAR: %s", metarString)
	}

	// Wind
	if len(parts) > 2 {
		windRegex := regexp.MustCompile(`(\d{3}|VRB)(\d{2,3})(G\d{2,3})?KT`)
		if windMatch := windRegex.FindStringSubmatch(parts[2]); windMatch != nil {
			if windMatch[1] != "VRB" {
				metar.Wind.Direction, _ = strconv.Atoi(windMatch[1])
			}
			metar.Wind.Speed, _ = strconv.Atoi(windMatch[2])
			if windMatch[3] != "" {
				metar.Wind.Gust, _ = strconv.Atoi(windMatch[3][1:])
			}
		}
	}

	// Visibility
	visRegex := regexp.MustCompile(`(\d+)SM`)
	for _, part := range parts[3:] {
		if visMatch := visRegex.FindStringSubmatch(part); visMatch != nil {
			metar.Visibility = visMatch[1] + "SM"
			break
		}
	}

	// Cloud Layers
	cloudRegex := regexp.MustCompile(`(FEW|SCT|BKN|OVC)(\d{3})`)
	for _, part := range parts {
		if cloudMatch := cloudRegex.FindStringSubmatch(part); cloudMatch != nil {
			height, _ := strconv.Atoi(cloudMatch[2])
			metar.CloudLayers = append(metar.CloudLayers, CloudLayer{
				Coverage: cloudMatch[1],
				Height:   height * 100,
			})
		}
	}

	// Temperature and Dew Point
	tempRegex := regexp.MustCompile(`(\d{2})/(\d{2})`)
	for _, part := range parts {
		if tempMatch := tempRegex.FindStringSubmatch(part); tempMatch != nil {
			metar.Temperature, _ = strconv.ParseFloat(tempMatch[1], 64)
			metar.DewPoint, _ = strconv.ParseFloat(tempMatch[2], 64)
			break
		}
	}

	// Altimeter
	altRegex := regexp.MustCompile(`A(\d{4})`)
	for _, part := range parts {
		if altMatch := altRegex.FindStringSubmatch(part); altMatch != nil {
			altInHg, _ := strconv.ParseFloat(altMatch[1], 64)
			metar.Altimeter = altInHg / 100
			break
		}
	}

	// Remarks
	if rmkIndex := indexOf(parts, "RMK"); rmkIndex != -1 {
		metar.Remarks = strings.Join(parts[rmkIndex:], " ")
	}

	return metar, nil
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// Fetch METAR from the aviationweather API
func fetchMetarData(airportCode string) string {
	url := fmt.Sprintf("https://aviationweather.gov/api/data/metar?ids=%s", airportCode)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch METAR data: %v", err)
	}
	defer resp.Body.Close()

	// Read the full response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	// Convert the response body to string and log it for debugging
	metarData := string(bodyBytes)
	fmt.Println("API Response:", metarData)

	return metarData
}

// Determine greeting based on the time of day
func getTimeOfDayGreeting() string {
	hour := time.Now().Hour()

	switch {
	case hour >= 5 && hour < 12:
		return "MORNING"
	case hour >= 12 && hour < 17:
		return "AFTERNOON"
	case hour >= 17 && hour < 21:
		return "EVENING"
	default:
		return "NIGHT"
	}
}

// Draw the text on the background image
func createWallpaper(metar METAR) {
	// Open the background image
	im, err := gg.LoadImage(backgroundImage)
	if err != nil {
		log.Fatalf("Failed to load background image: %v", err)
	}

	// Create a new context
	dc := gg.NewContextForImage(im)

	// Offset the whole block by an extra 200px downwards
	offsetY := 200

	// Day of the week in regular font (e.g., "Monday")
	err = dc.LoadFontFace(regularFont, 25)
	if err != nil {
		log.Fatalf("Failed to load regular font: %v", err)
	}
	dayOfWeek := time.Now().Format("Monday")
	dc.SetRGB(192, 168, 143) // Set text color to red for "It's Monday"
	dc.DrawStringAnchored(fmt.Sprintf("It's %s", dayOfWeek), 100, float64(775+offsetY), 0, 0)

	// Main message in bold (with the appropriate greeting)
	err = dc.LoadFontFace(boldFont, 40)
	if err != nil {
		log.Fatalf("Failed to load bold font: %v", err)
	}
	dc.SetRGB(1, 1, 1) // Set text color to white
	dc.DrawStringAnchored(fmt.Sprintf("HOPE YOUR %s", getTimeOfDayGreeting()), 100, float64(800+offsetY), 0, 0.5)
	if userName != "" {
		dc.DrawStringAnchored("IS GOING WELL,", 100, float64(850+offsetY), 0, 0.5)
		upperName := strings.ToUpper(userName)
		dc.DrawStringAnchored(upperName, 100, float64(900+offsetY), 0, 0.5)
	} else {
		dc.DrawStringAnchored("IS GOING WELL", 100, float64(850+offsetY), 0, 0.5)
	}

	// Weather details in light font
	err = dc.LoadFontFace(lightFont, 20)
	if err != nil {
		log.Fatalf("Failed to load light font: %v", err)
	}

	dc.SetRGB(47, 47, 47) // Set text color to grey
	// Display the parsed METAR weather details
	weatherText := fmt.Sprintf("Temperature: %.1f°F, Visibility: %s, Wind: %dKT", metar.Temperature, metar.Visibility, metar.Wind.Speed)
	dc.DrawStringAnchored(weatherText, 100, float64(950+offsetY), 0, 0.5)

	if err != nil {
		log.Fatalf("Failed to load regular font: %v", err)
	}
	// Get both Local and Zulu (UTC) times
	localTime := time.Now().Format("15:04")      // Local time
	zuluTime := time.Now().UTC().Format("15:04") // Zulu (UTC) time

	// Time in light font with Local and Zulu
	dc.DrawStringAnchored(fmt.Sprintf("Updated: %s Local / %s Zulu", localTime, zuluTime), 100, float64(975+offsetY), 0, 0.5)

	// Save the final image using the output path provided by the flag
	dc.SavePNG(outputImage)
}

// Set the desktop wallpaper using the generated image
func setWallpaper() {
	err := wallpaper.SetFromFile(outputImage)
	if err != nil {
		log.Fatalf("Failed to set wallpaper: %v", err)
	}

	// Optional: Set wallpaper mode to crop
	err = wallpaper.SetMode(wallpaper.Crop)
	if err != nil {
		log.Fatalf("Failed to set wallpaper mode: %v", err)
	}
}

// Run the program every 15 minutes relative to the system time
func runEvery15Minutes() {
	for {
		now := time.Now()
		// Calculate the next 15-minute interval (e.g., 00, 15, 30, 45)
		next := now.Truncate(15 * time.Minute).Add(15 * time.Minute)
		sleepDuration := time.Until(next)

		fmt.Printf("Next update in: %v\n", sleepDuration)

		// Sleep until the next 15-minute mark
		time.Sleep(sleepDuration)

		// Fetch and parse METAR data
		metarString := fetchMetarData(airport)
		parsedMETAR, err := ParseMETAR(metarString)
		if err != nil {
			log.Fatalf("Error parsing METAR data: %v", err)
		}

		// Once we wake up, update the wallpaper
		createWallpaper(parsedMETAR)
		setWallpaper()
	}
}

func main() {
	// Set up command-line flags
	flag.StringVar(&airport, "airport", "KBFI", "ICAO airport code (e.g., KBFI for Buffalo)")
	flag.StringVar(&backgroundImage, "background", "", "Absolute path to the base background image")
	flag.StringVar(&outputImage, "output", "", "Absolute path to save the output image")
	flag.StringVar(&userName, "name", "", "The name to call you")

	// Parse the flags
	flag.Parse()

	// Ensure both input and output image paths are provided
	if backgroundImage == "" || outputImage == "" {
		log.Fatal("Both --background and --output flags must be provided with absolute paths.")
	}

	// Initial wallpaper set on program start
	metarString := fetchMetarData(airport)
	parsedMETAR, err := ParseMETAR(metarString)
	if err != nil {
		log.Fatalf("Error parsing METAR data: %v", err)
	}
	createWallpaper(parsedMETAR)
	setWallpaper()

	// Run the periodic update
	go runEvery15Minutes()

	// Keep the program running
	select {}
}
