package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
)

func init() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func connectAPI(url string) (interface{}, error) {
	// Read the API key from the environment
	apiKey := os.Getenv("CAT_API_KEY")
	if apiKey == "" {
		log.Fatalf("API key is not set in the environment variables")
		return nil, fmt.Errorf("API key is not set in the environment variables")
	}

	// Create an HTTP GET request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalf("Failed to create request object for /GET endpoint: %v", err)
		return nil, err
	}

	// Add necessary headers
	req.Header.Add("Content-type", "application/json; charset=utf-8")
	req.Header.Add("x-api-key", apiKey) // Replace with your actual API key

	// Send the request using the default HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
		return nil, err
	}

	// Try to unmarshal into a slice of maps
	var dataArray []map[string]interface{}
	err = json.Unmarshal(body, &dataArray)
	if err == nil {
		return dataArray, nil
	}

	// If unmarshalling into a slice fails, try to unmarshal into a map
	var dataObject map[string]interface{}
	err = json.Unmarshal(body, &dataObject)
	if err == nil {
		return dataObject, nil
	}

	// If both unmarshalling attempts fail, return an error
	log.Fatalf("Failed to unmarshal JSON response: %v", err)
	return nil, err
}

func getSingleCatImageByBreed(breed string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.thecatapi.com/v1/images/search?has_breeds=1&limit=1&breed_ids=%s", breed)
	data, err := connectAPI(url)
	if err != nil {
		return nil, err
	}

	// Assert the data to be a slice of maps
	dataArray, ok := data.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	if len(dataArray) > 0 {
		return dataArray[0], nil
	}

	return nil, fmt.Errorf("no data")
}

func getCatImages() ([]map[string]interface{}, error) {
	data, err := connectAPI("https://api.thecatapi.com/v1/images/search?has_breeds=1&limit=20")
	if err != nil {
		return nil, err
	}

	// Assert the data to be a slice of maps
	dataArray, ok := data.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	return dataArray, nil
}

func getCatImageByID(id string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.thecatapi.com/v1/images/%s", id)
	data, err := connectAPI(url)
	if err != nil {
		return nil, err
	}

	// Assert the data to be a map
	dataObject, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	return dataObject, nil
}

func main() {

	engine := html.New("./templates", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(cors.New())

	app.Static("/css", "./css")

	app.Get("/", func(c *fiber.Ctx) error {
		catImages, err := getCatImages()
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString("Failed to get cat images")
		}

		return c.Render("index", fiber.Map{
			"Results": catImages,
		})
	})

	app.Get("/cat/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		catData, err := getCatImageByID(id)
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString("Failed to get cat image by ID")
		}

		// Extract the URL from the catData map
		url, ok := catData["url"].(string)
		if !ok {
			return c.Status(http.StatusInternalServerError).SendString("Failed to get URL from cat data")
		}

		// Extract the name from the breeds array
		var breedName string
		var weightMetric string
		var temperaments []string
		var originCountry string
		if breeds, ok := catData["breeds"].([]interface{}); ok && len(breeds) > 0 {
			if breed, ok := breeds[0].(map[string]interface{}); ok {
				if name, ok := breed["name"].(string); ok {
					breedName = name
				}
				if weight, ok := breed["weight"].(map[string]interface{}); ok {
					if metric, ok := weight["metric"].(string); ok {
						weightMetric = metric
					}
				}
				if temperament, ok := breed["temperament"].(string); ok {
					temperaments = strings.Split(temperament, ", ")
					if len(temperaments) > 3 {
						temperaments = temperaments[:3]
					}
				}
				if origin, ok := breed["origin"].(string); ok {
					originCountry = origin
				}
			}
		}

		return c.Render("cat-detail", fiber.Map{
			"ImageUrl":     url,
			"Name":         capitalize(breedName),
			"Weight":       weightMetric,
			"Temperaments": temperaments,
			"Origin":       originCountry,
		})
	})

	app.Get("/search", func(c *fiber.Ctx) error {
		q := strings.ToLower(c.Query("q"))
		catData, err := getSingleCatImageByBreed(q)
		if err != nil {
			return c.Render("cat-detail", fiber.Map{
				"Error": "Oops! There seems to be no cat by that breed in our database.",
			})
		}
		fmt.Println(catData)

		// Extract the URL from the catData map
		url, ok := catData["url"].(string)
		if !ok {
			return c.Status(http.StatusInternalServerError).SendString("Failed to get URL from cat data")
		}

		// Extract the name from the breeds array
		var breedName string
		var weightMetric string
		var temperaments []string
		var originCountry string
		if breeds, ok := catData["breeds"].([]interface{}); ok && len(breeds) > 0 {
			if breed, ok := breeds[0].(map[string]interface{}); ok {
				if name, ok := breed["name"].(string); ok {
					breedName = name
				}
				if weight, ok := breed["weight"].(map[string]interface{}); ok {
					if metric, ok := weight["metric"].(string); ok {
						weightMetric = metric
					}
				}
				if temperament, ok := breed["temperament"].(string); ok {
					temperaments = strings.Split(temperament, ", ")
					if len(temperaments) > 3 {
						temperaments = temperaments[:3]
					}
				}
				if origin, ok := breed["origin"].(string); ok {
					originCountry = origin
				}
			}
		}

		return c.Render("cat-detail", fiber.Map{
			"ImageUrl":     url,
			"Name":         capitalize(breedName),
			"Weight":       weightMetric,
			"Temperaments": temperaments,
			"Origin":       originCountry,
		})
	})

	log.Fatal(app.Listen(":3000"))
}

func capitalize(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
}
