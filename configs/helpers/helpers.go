package helpers

import (
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	uuid "github.com/nu7hatch/gouuid"
)

func GetEnvVariable(key string) string {
	// load .env file and get env variable value based on "key"
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

func CreateUUIDStr() (string, error) {
	// generate an uuid and return it as a string
	u4, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return u4.String(), nil
}

func CreateSlug(str string) string {
	reg, err := regexp.Compile("[^a-z0-9 ]+")
	if err != nil {
		log.Fatal(err)
	}

	str = strings.ToLower(str)
	str = strings.Trim(str, " ")
	str = reg.ReplaceAllString(str, "")
	str = strings.ReplaceAll(str, " ", "-")

	return str
}

func ValidateImage(c *gin.Context, file string) error {
	var imageType, _ = regexp.Compile(`^.*\.(jpeg|JPEG|jpg|JPG|gif|GIF|png|PNG|svg|SVG|webp|WebP|WEBP)$`)
	if isImage := imageType.MatchString(file); !isImage {
		c.JSON(http.StatusOK, gin.H{
			"status":  "failed",
			"message": "tolong masukkan file dalam format gambar (png, jpg, dsb)",
		})
		c.Abort()

		return errors.New("invalid type")
	}

	return nil
}

func ValidatePdf(c *gin.Context, file string) error {
	var pdfType, _ = regexp.Compile(`^.*\.(pdf|PDF)$`)
	if isImage := pdfType.MatchString(file); !isImage {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "tolong masukkan cv dalam format pdf",
		})
		c.Abort()

		return errors.New("invalid type")
	}

	return nil
}

// HTTP Response
func SendInternalServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"status":  "failed",
		"message": err.Error(),
	})
	c.Abort()
}
