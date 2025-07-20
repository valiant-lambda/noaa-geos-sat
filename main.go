package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

var (
	base = "https://cdn.star.nesdis.noaa.gov/GOES18/ABI/FD/GEOCOLOR/"
	//base = "https://cdn.star.nesdis.noaa.gov/GOES18/ABI/FD/AirMass/"
	date = 2025191
	end  = "_GOES18-ABI-FD-GEOCOLOR-678x678.jpg"
)

func main() {
	downloadImages()
	if err := createGIF(); err != nil {
		fmt.Println("GIF creation error:", err)
	}

}

func createGIF() error {
	entries, err := os.ReadDir("files")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Filter .jpg entries only
	var jpgEntries []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jpg" {
			jpgEntries = append(jpgEntries, entry)
		}
	}

	total := len(jpgEntries)
	if total == 0 {
		return fmt.Errorf("no .jpg images found in 'files/'")
	}

	var images []*image.Paletted
	var delays []int

	for i, entry := range jpgEntries {
		path := "files/" + entry.Name()
		fmt.Printf("\r[%3d%%] Processing %s", (i+1)*100/total, entry.Name())

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			fmt.Println("Decode error:", err)
			continue
		}

		bounds := img.Bounds()
		palettedImg := image.NewPaletted(bounds, palette.Plan9)
		draw.FloydSteinberg.Draw(palettedImg, bounds, img, image.Point{})

		images = append(images, palettedImg)
		delays = append(delays, 5)
	}

	fmt.Println("Encoding GIF...")

	outFile, err := os.Create("output.gif")
	if err != nil {
		return err
	}
	defer outFile.Close()

	err = gif.EncodeAll(outFile, &gif.GIF{
		Image: images,
		Delay: delays,
	})

	if err != nil {
		return err
	}

	fmt.Println("✅ GIF saved as output.gif")
	return nil
}

func downloadImages() {
	err := os.MkdirAll("files", 0755)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}

	client := http.Client{}
	for x := 191; x < 199; x++ {
		for j := 0; j < 24; j++ {
			for i := 0; i < 60; i += 10 {
				timevar := fmt.Sprintf("%02d%02d", j, i)
				filename := fmt.Sprintf("2025%d%s%s", x, timevar, end) // full filename like: 20251891750_GOES18-...
				filename = filepath.Clean(filename)
				url := base + filename

				fmt.Println("Downloading:", url)

				resp, err := client.Get(url)
				if err != nil {
					fmt.Println("Request error:", err)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					fmt.Println("Non-200 response:", resp.Status)
					continue
				}

				localPath := filepath.Join("files", filename)
				file, err := os.Create(localPath)
				if err != nil {
					fmt.Println("File creation error:", err)
					continue
				}

				_, err = io.Copy(file, resp.Body)
				file.Close()
				if err != nil {
					fmt.Println("File write error:", err)
					continue
				}

				fmt.Println("Saved to:", localPath)
			}
		}
	}

}
