package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/pkg/filestation"
)

func main() {
	// Create client configuration
	config := &api.Config{
		Host:     "192.168.1.100", // Your QNAP IP
		Port:     8080,             // QNAP port
		Username: "admin",          // Your username
		Password: "password",       // Your password
		Insecure: true,             // Set to false if using valid certificate
		Logger:   slog.Default(),
	}

	// Create client
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Login
	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Logout(ctx)

	// Create FileStation service
	fs := filestation.NewFileStationService(client)

	// List files in a directory
	files, err := fs.ListFiles(ctx, "/home", &filestation.ListOptions{
		Limit: 100,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Print files
	for _, file := range files {
		fmt.Printf("%s (%d bytes)\n", file.Name(), file.Size())
	}

	// Upload a file
	_, err = fs.UploadFile(ctx, "/local/path/file.txt", "/home/remote", nil)
	if err != nil {
		log.Printf("Upload failed: %v", err)
	}

	// Download a file
	err = fs.DownloadFile(ctx, "/home/remote/file.txt", "/local/download", nil)
	if err != nil {
		log.Printf("Download failed: %v", err)
	}

	// Create a folder
	err = fs.CreateFolder(ctx, "/home/new_folder")
	if err != nil {
		log.Printf("Create folder failed: %v", err)
	}

	// Search for files
	results, err := fs.SearchByPattern(ctx, "/home", "*.txt")
	if err != nil {
		log.Printf("Search failed: %v", err)
	}

	for _, result := range results {
		fmt.Printf("Found: %s\n", result.Name())
	}

	// Create a share link
	link, err := fs.CreateShareLink(ctx, "/home/file.txt", &filestation.ShareLinkOptions{
		Writeable: false,
	})
	if err != nil {
		log.Printf("Create share link failed: %v", err)
	} else {
		fmt.Printf("Share link: %s\n", link.URL)
	}
}
