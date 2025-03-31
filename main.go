package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("No command provided. Try again.")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "init":
		err := os.MkdirAll(".mygit/objects", 0755)
        if err != nil {
            log.Printf("Failed to create objects dir: %v", err)
            os.Exit(1)
        }
        err = os.MkdirAll(".mygit/refs/heads", 0755)
        if err != nil {
            log.Printf("Failed to create refs/heads dir: %v", err)
            os.Exit(1)
        }
        err = os.WriteFile(".mygit/HEAD", []byte("ref: refs/heads/main\n"), 0644)
        if err != nil {
            log.Printf("Failed to write HEAD: %v", err)
            os.Exit(1)
        }
        err = os.WriteFile(".mygit/refs/heads/main", []byte(""), 0644)
        if err != nil {
            log.Printf("Failed to write main ref: %v", err)
            os.Exit(1)
        }

        log.Println("Initialized empty repository")

	case "add":
		if len (os.Args) < 3 {
			log.Println("No file specified for 'add'")
			os.Exit(1)
		}
		fileName := os.Args[2]

		fileContents, err := os.ReadFile(fileName)
		if err != nil {
			log.Printf("Failed to read file contents: %v", err)
			os.Exit(1)
		}

		hasher := sha1.New()
		hasher.Write(fileContents)
		hashBytes := hasher.Sum(nil)
		hashString := hex.EncodeToString(hashBytes)

		objectPath := fmt.Sprintf(".mygit/objects/%s", hashString)
		err = os.WriteFile(objectPath, fileContents, 0644)
        if err != nil {
            log.Printf("Failed to write object: %v", err)
            os.Exit(1)
        }

		indexEntry := fmt.Sprintf("%s %s\n", fileName, hashString)
		err = os.WriteFile(".mygit/index", []byte(indexEntry), 0644)
		if err != nil {
			log.Printf("Failed to write index file: %v", err)
			os.Exit(1)
		}

	case "commit":
		if len (os.Args) < 3 {
			fmt.Println("No message provided with commit")
			os.Exit(1)
		}
		commitMessage := os.Args[2]

		indexFileContents, err := os.ReadFile(".mygit/index")
		if err != nil {
			fmt.Println("Could not read contents of file. Try adding something first")
			os.Exit(1)
		}

		// Get parent commit (empty if first commit)
		var parentHash string
		hashOfPreviousCommit, err := os.ReadFile(".mygit/refs/heads/main")
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Println("Could not read main file for previous commit")
				os.Exit(1)
			}
		} else {
			parentHash = string(hashOfPreviousCommit)
		}

		// Build the commit string
		commitContent := fmt.Sprintf("parent %s\n", parentHash)
		indexLines := strings.Split(string(indexFileContents), "\n")
		for _, line := range indexLines {
			if line == "" {
				continue
			}
			commitContent += fmt.Sprintf("blob %s\n", line)
		}
		commitContent += "author me\n"
		commitContent += fmt.Sprintf("message %s\n", commitMessage)
		commitContent += fmt.Sprintf("date %s\n", time.Now().Format(time.RFC3339))

		// hash the commit content
		hasher := sha1.New()
		hasher.Write([]byte(commitContent))
		commitHash := hex.EncodeToString(hasher.Sum(nil))

		// Store the commit object
		commitPath := fmt.Sprintf(".mygit/objects/%s", commitHash)
		err = os.WriteFile(commitPath, []byte(commitContent), 0644)
		if err != nil {
			fmt.Printf("Failed to write commit object: %v\n", err)
			os.Exit(1)
		}

		// Update refs/heads/main
		err = os.WriteFile(".mygit/refs/heads/main", []byte(commitHash), 0644)
		if err != nil {
			fmt.Printf("Failed to update main ref: %v\n", err)
			os.Exit(1)
		}

		// Clear the index
		err = os.WriteFile(".mygit/index", []byte(""), 0644)
		if err != nil {
			fmt.Printf("Failed to clear index: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Committed with hash %s\n", commitHash)

	case "rm":
		if len(os.Args) < 3 {
			fmt.Println("File/folder to remove not included")
			os.Exit(1)
		}

		fileToDelete := os.Args[2]

		err := os.Remove(fileToDelete)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("File %s does not exist in working directory\n", fileToDelete)
			} else {
				fmt.Printf("Could not remove file/folder: %v\n", err)
				os.Exit(1)
			}
		}

		indexContent, err := os.ReadFile(".mygit/index")
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Removed %s (no index to update)\n", fileToDelete)
				return
			}
			fmt.Printf("Could not read index: %v\n", err)
			os.Exit(1)
		}

		var newIndex string
		lines := strings.Split(string(indexContent), "\n")
		found := false
		for _, line := range lines {
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, fileToDelete+" ") {
				newIndex += line + "\n"
			} else {
				found = true
			}
		}

		err = os.WriteFile(".mygit/index", []byte(newIndex), 0644)
		if err != nil {
			fmt.Printf("Could not update index: %v\n", err)
			os.Exit(1)
		}

		if found || err == nil {
			fmt.Printf("Removed %s\n", fileToDelete)
		} else {
			fmt.Printf("File %s was not staged\n", fileToDelete)
		}

	case "status":
		indexContent, err := os.ReadFile(".mygit/index")
		staged := make(map[string]string)
		if err == nil {
			fmt.Println("Staged for commit:")
			lines := strings.Split(string(indexContent), "\n")
			for _, line := range lines {
				if line != "" {
					parts := strings.Split(line, " ")
					if len(parts) >= 2 {
						fmt.Printf("  %s\n", parts[0])
						staged[parts[0]] = parts[1]
					}
				}
			}
		} else if !os.IsNotExist(err) {
			fmt.Printf("Could not read index: %v\n", err)
			os.Exit(1)
		}	

		// Last commit
		committed := make(map[string]string)
		headsMainHash, err := os.ReadFile(".mygit/refs/heads/main")
		if err == nil {
			commitPath := fmt.Sprintf(".mygit/objects/%s", string(headsMainHash))
			commitContent, err := os.ReadFile(commitPath)
			if err != nil {
				fmt.Printf("Could not read commit object: %v\n", err)
				os.Exit(1)
			}
			lines := strings.Split(string(commitContent), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "blob ") {
					parts := strings.SplitN(line[5:], " ", 2) // Skip "blob "
					if len(parts) == 2 {
						committed[parts[1]] = parts[0] // hash, path
					}
				}
			}
		} else if !os.IsNotExist(err) {
			fmt.Printf("Could not read main ref: %v\n", err)
			os.Exit(1)
		}

		//Scan working dir
		dir, err := os.ReadDir(".")
		if err != nil {
			fmt.Printf("Could not read dir: %v\n", err)
			os.Exit(1)
		}

		modified := []string{}
		untracked := []string{}
		for _, entry := range dir {
			if entry.IsDir() && entry.Name() == ".mygit" {
				continue // Skip .mygit
			}
			if entry.IsDir() {
				continue // Skip dirs for simplicity
			}
			filename := entry.Name()

			// Skip if staged
			if _, ok := staged[filename]; ok {
				continue
			}

			// Check if committed
			if committedHash, ok := committed[filename]; ok {
				// Hash current content
				content, err := os.ReadFile(filename)
				if err != nil {
					continue // Skip if unreadable
				}
				hasher := sha1.New()
				hasher.Write(content)
				currentHash := hex.EncodeToString(hasher.Sum(nil))
				if currentHash != committedHash {
					modified = append(modified, filename)
				}
			} else {
				untracked = append(untracked, filename)
			}
		}

		// Display results
		if len(modified) > 0 {
			fmt.Println("Modified:")
			for _, file := range modified {
				fmt.Printf(" %s\n", file)
			}
		}

		if len(untracked) > 0 {
			fmt.Println("Untracked:")
			for _, file := range untracked {
				fmt.Printf(" %s\n", file)
			}
		}

	default:
		log.Print("Not a valid command")
	}
}