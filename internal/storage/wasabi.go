package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

type WasabiStorage struct {
	Client    *s3.Client
	Presigner *s3.PresignClient

	Bucket   string
	Region   string
	Endpoint string
}

func NewWasabiStorage() (*WasabiStorage, error) {
	bucket := getEnv("WASABI_BUCKET")
	region := getEnv("WASABI_REGION")
	endpoint := getEnv("WASABI_ENDPOINT")
	accessKey := getEnv("WASABI_ACCESS_KEY")
	secretKey := getEnv("WASABI_SECRET_KEY")

	if bucket == "" {
		return nil, fmt.Errorf("WASABI_BUCKET is missing")
	}

	if region == "" {
		return nil, fmt.Errorf("WASABI_REGION is missing")
	}

	if endpoint == "" {
		return nil, fmt.Errorf("WASABI_ENDPOINT is missing")
	}

	if accessKey == "" {
		return nil, fmt.Errorf("WASABI_ACCESS_KEY is missing")
	}

	if secretKey == "" {
		return nil, fmt.Errorf("WASABI_SECRET_KEY is missing")
	}

	creds := credentials.NewStaticCredentialsProvider(
		accessKey,
		secretKey,
		"",
	)

	client := s3.New(s3.Options{
		Region:      region,
		Credentials: aws.NewCredentialsCache(creds),

		BaseEndpoint: aws.String(endpoint),

		UsePathStyle: true,
	})

	presigner := s3.NewPresignClient(client)

	return &WasabiStorage{
		Client:    client,
		Presigner: presigner,
		Bucket:    bucket,
		Region:    region,
		Endpoint:  endpoint,
	}, nil
}

func getEnv(key string) string {
	value := os.Getenv(key)

	if value == "" {
		panic(fmt.Sprintf(
			"Environment variable %s is missing",
			key,
		))
	}

	return value
}

func generateUniqueID() string {
	id, err := uuid.NewV7()

	if err != nil {
		return uuid.New().String()
	}

	return id.String()
}

func (w *WasabiStorage) UploadFile(
	ctx context.Context,
	data []byte,
	originalName string,
	contentType string,
	folder string,
) (string, error) {

	// ============================================================
	// VALIDASI DATA
	// ============================================================

	if len(data) == 0 {
		return "", fmt.Errorf("file kosong")
	}

	// ============================================================
	// DETECT CONTENT TYPE DARI ISI FILE
	// Jangan sepenuhnya percaya Content-Type dari client.
	// ============================================================

	detectedContentType := http.DetectContentType(data)

	log.Printf(
		"Upload image: filename=%s client_content_type=%s detected_content_type=%s size=%d",
		originalName,
		contentType,
		detectedContentType,
		len(data),
	)

	// ============================================================
	// VALIDASI IMAGE
	// ============================================================

	switch detectedContentType {

	case "image/jpeg":
	case "image/png":
	case "image/gif":
		// GIF terdeteksi sebagai image, tetapi untuk sekarang
		// kita tidak melakukan encoding GIF.
		return "", fmt.Errorf(
			"format image tidak didukung: %s",
			detectedContentType,
		)

	default:

		// Fallback berdasarkan ekstensi.
		// Ini berguna untuk beberapa format yang tidak dikenali
		// oleh http.DetectContentType.

		ext := strings.ToLower(
			filepath.Ext(originalName),
		)

		switch ext {

		case ".jpg", ".jpeg":
			detectedContentType = "image/jpeg"

		case ".png":
			detectedContentType = "image/png"

		default:
			return "", fmt.Errorf(
				"file harus berupa gambar",
			)
		}
	}

	// ============================================================
	// NORMALIZE FILENAME
	// ============================================================

	filename := filepath.Base(originalName)

	if filename == "." || filename == "/" || filename == "" {
		filename = "image"
	}

	// ============================================================
	// GENERATE UNIQUE ID
	// ============================================================

	uniqueID := generateUniqueID()

	// ============================================================
	// FOLDER
	//
	// Contoh:
	//
	// folder = chat/images
	//
	// hasil:
	//
	// chat/images/0198xxxx_xxx.jpg
	//
	// ============================================================

	cleanFolder := strings.Trim(
		folder,
		"/",
	)

	if cleanFolder == "" {
		return "", fmt.Errorf(
			"folder upload wajib diisi",
		)
	}

	key := fmt.Sprintf(
		"%s/%s_%s",
		cleanFolder,
		uniqueID,
		filename,
	)

	// ============================================================
	// COMPRESS
	// ============================================================

	compressed, finalContentType, err := compressImage(
		data,
		detectedContentType,
		300*1024,
	)

	if err != nil {
		log.Printf(
			"image compression error: %v",
			err,
		)

		return "", err
	}

	// ============================================================
	// UPLOAD WASABI
	// ============================================================

	_, err = w.Client.PutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket: aws.String(
				w.Bucket,
			),

			Key: aws.String(
				key,
			),

			Body: bytes.NewReader(
				compressed,
			),

			ContentType: aws.String(
				finalContentType,
			),

			ContentDisposition: aws.String(
				"inline",
			),
		},
	)

	if err != nil {

		log.Printf(
			"Upload error Wasabi: %v",
			err,
		)

		return "", fmt.Errorf(
			"error uploading file to Wasabi: %w",
			err,
		)
	}

	// ============================================================
	// SUCCESS
	// ============================================================

	log.Printf(
		"Wasabi upload success: key=%s original_size=%d compressed_size=%d content_type=%s",
		key,
		len(data),
		len(compressed),
		finalContentType,
	)

	// Kita tetap return KEY.
	//
	// Contoh:
	//
	// chat/images/0198xxxx_uuid_photo.jpg
	//
	// BUKAN URL.
	//
	return key, nil
}

func compressImage(
	data []byte,
	contentType string,
	maxSize int,
) ([]byte, string, error) {

	img, _, err := image.Decode(
		bytes.NewReader(data),
	)

	if err != nil {
		return nil, "", fmt.Errorf(
			"failed to decode image: %w",
			err,
		)
	}

	switch strings.ToLower(contentType) {

	case "image/jpeg", "image/jpg":

		return compressJPEG(
			img,
			maxSize,
		)

	case "image/png":

		return compressPNG(
			img,
			maxSize,
		)

	default:

		return nil, "", fmt.Errorf(
			"unsupported image type: %s",
			contentType,
		)
	}
}

func compressJPEG(
	img image.Image,
	maxSize int,
) ([]byte, string, error) {

	quality := 90

	var result []byte

	for quality >= 10 {

		var buffer bytes.Buffer

		err := jpeg.Encode(
			&buffer,
			img,
			&jpeg.Options{
				Quality: quality,
			},
		)

		if err != nil {
			return nil, "", err
		}

		result = buffer.Bytes()

		if len(result) <= maxSize {
			return result, "image/jpeg", nil
		}

		quality -= 10
	}

	// Kalau quality sudah mentok,
	// mulai resize image.
	resized := img

	for len(result) > maxSize {

		bounds := resized.Bounds()

		width := bounds.Dx()
		height := bounds.Dy()

		if width <= 200 || height <= 200 {
			break
		}

		width = int(
			math.Floor(float64(width) * 0.9),
		)

		height = int(
			math.Floor(float64(height) * 0.9),
		)

		resized = imaging.Resize(
			resized,
			width,
			height,
			imaging.Lanczos,
		)

		var buffer bytes.Buffer

		err := jpeg.Encode(
			&buffer,
			resized,
			&jpeg.Options{
				Quality: 50,
			},
		)

		if err != nil {
			return nil, "", err
		}

		result = buffer.Bytes()
	}

	return result, "image/jpeg", nil
}

func compressPNG(
	img image.Image,
	maxSize int,
) ([]byte, string, error) {

	current := img

	for {

		var buffer bytes.Buffer

		err := png.Encode(
			&buffer,
			current,
		)

		if err != nil {
			return nil, "", err
		}

		result := buffer.Bytes()

		if len(result) <= maxSize {
			return result, "image/png", nil
		}

		bounds := current.Bounds()

		width := bounds.Dx()
		height := bounds.Dy()

		if width <= 200 || height <= 200 {
			return result, "image/png", nil
		}

		width = int(
			math.Floor(float64(width) * 0.9),
		)

		height = int(
			math.Floor(float64(height) * 0.9),
		)

		current = imaging.Resize(
			current,
			width,
			height,
			imaging.Lanczos,
		)
	}
}

func (w *WasabiStorage) GetPresignedURL(
	ctx context.Context,
	fileURLOrKey string,
	expiresInSeconds int64,
) (string, error) {

	if strings.TrimSpace(fileURLOrKey) == "" {
		return "", nil
	}

	key := extractKey(fileURLOrKey)

	if key == "" {
		return "", nil
	}

	result, err := w.Presigner.PresignGetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(w.Bucket),
			Key:    aws.String(key),
		},
		func(options *s3.PresignOptions) {
			options.Expires = time.Duration(
				expiresInSeconds,
			) * time.Second
		},
	)

	if err != nil {
		log.Printf(
			"Error generating presigned URL: %v",
			err,
		)

		return "", nil
	}

	return result.URL, nil
}

func (w *WasabiStorage) DeleteFile(
	ctx context.Context,
	fileURLOrKey string,
) error {

	key := extractKey(fileURLOrKey)

	if key == "" {
		return nil
	}

	_, err := w.Client.DeleteObject(
		ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(w.Bucket),
			Key:    aws.String(key),
		},
	)

	if err != nil {

		log.Printf(
			"Error deleting Wasabi object: %v",
			err,
		)

		return err
	}

	log.Printf(
		"Deleted from Wasabi: %s",
		key,
	)

	return nil
}

func extractKey(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	if index := strings.Index(value, ".com/"); index >= 0 {
		return value[index+5:]
	}

	return strings.TrimPrefix(value, "/")
}
