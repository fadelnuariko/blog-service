package minio

import (
	"fadel-blog-services/configs/helpers"
	"fmt"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient = connectMinio()

func connectMinio() *miniosdk.Client {
	endpoint := helpers.GetEnvVariable("MINIO_ENDPOINT")
	accessKeyID := helpers.GetEnvVariable("MINIO_ACCESS_KEY_ID")
	secretAccessKey := helpers.GetEnvVariable("MINIO_ACCESS_KEY_PASS")
	useSSL := false

	// Initialize minio client object.
	minioClient, err := miniosdk.New(endpoint, &miniosdk.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("minio connected...")
	return minioClient
}
