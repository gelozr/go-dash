package request_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/gelozr/go-dash/internal/http/request"
)

func TestCreateCustomerImageURLBodyFormats(t *testing.T) {
	const imageURL = "https://example.com/image.png"

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		var req request.CreateCustomer
		if err := c.Bind().Body(&req); err != nil {
			return err
		}
		return c.JSON(req.ToCustomer().ImageURL)
	})

	for _, tt := range []struct {
		name        string
		contentType string
		body        string
		wantImage   bool
	}{
		{"form image", fiber.MIMEApplicationForm, "name=Ada&image_url=https%3A%2F%2Fexample.com%2Fimage.png", true},
		{"form omitted", fiber.MIMEApplicationForm, "name=Ada", false},
		{"JSON image", fiber.MIMEApplicationJSON, `{"image_url":"` + imageURL + `"}`, true},
		{"XML image", fiber.MIMEApplicationXML, "<CreateCustomer><ImageURL>" + imageURL + "</ImageURL></CreateCustomer>", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("got HTTP %d, want HTTP 200", resp.StatusCode)
			}

			var image *string
			if err := json.NewDecoder(resp.Body).Decode(&image); err != nil {
				t.Fatal(err)
			}
			if tt.wantImage {
				if image == nil {
					t.Fatal("image URL is nil")
				}
				if *image != imageURL {
					t.Fatalf("image URL = %q, want %q", *image, imageURL)
				}
			} else if image != nil {
				t.Fatalf("image URL = %q, want nil", *image)
			}
		})
	}
}
