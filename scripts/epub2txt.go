package main

import (
    "archive/zip"
    "fmt"
    "os"
    "github.com/revelaction/segrob/epub"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s <epub-file>\n", os.Args[0])
        os.Exit(1)
    }
    epubPath := os.Args[1]

    z, err := zip.OpenReader(epubPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error opening zip: %v\n", err)
        os.Exit(1)
    }
    defer z.Close()
    
    book, err := epub.New(&z.Reader)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing epub: %v\n", err)
        os.Exit(1)
    }
    
    text, err := book.Text()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error extracting text: %v\n", err)
        os.Exit(1)
    }
    fmt.Print(text)
}
