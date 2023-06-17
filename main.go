package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-json"

	"github.com/juju/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type JsonMap map[string]map[string]any

func main() {
	log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Caller().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// get python file paths
	python_paths, err := filepath.Glob(filepath.Join("input", "*.py"))
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading Python file")
	}

	var wg sync.WaitGroup
	wg.Add(len(python_paths))
	for _, pythonFilePath := range python_paths {
		go func(pythonFilePath string) {
			log := log.Logger.With().Str("filepath", pythonFilePath).Logger()
			defer wg.Done()
			// Read Python file
			content, err := os.ReadFile(pythonFilePath)
			if err != nil {
				log.Fatal().Err(err).Msg("Error reading Python file")
				return
			}

			// Extract JSON variables
			jsonMap, err := extractJSONVariables(string(content))
			if err != nil {
				log.Fatal().Err(err).Msg("Error extracting JSON variables")
			}

			// Write parsed output to JSON file
			jsonData, err := json.MarshalIndent(&jsonMap, "", "  ")
			if err != nil {
				log.Fatal().Err(err).Msg("Error marshaling JSON")
				return
			}
			filepath := strings.Split(pythonFilePath, "/")
			filename := filepath[len(filepath)-1]
			jsonOutputFilePath := fmt.Sprintf("output/%s.json", strings.Split(filename, ".")[0])

			err = os.WriteFile(jsonOutputFilePath, jsonData, 0644)
			if err != nil {
				log.Fatal().Err(err).Msg("Error writing JSON output file")
			}
			log.Info().Str("filepath", jsonOutputFilePath).Msg("Parsed output written")
		}(pythonFilePath)
	}
	wg.Wait()
}

// Extracts JSON variables from Python file
func extractJSONVariables(content string) (output JsonMap, err error) {
	output = make(JsonMap)
	var json_builder strings.Builder
	// write the open brace
	json_builder.WriteString("{")

	lines := strings.Split(content, "\n")
	object_count := 0
	for i, line := range lines {
		// if we detect the beginning of a JSON variable assignment
		if strings.HasSuffix(line, " = {") && !strings.HasPrefix(line, "#") {
			// create a string builder to capture the JSON
			// line1 will be the first key in the output map
			line1 := strings.Split(line, " = ")
			j_log := log.Logger.With().Str("variable", line1[0]).Logger()
			j_log.Info().Msg("JSON var detected")
			// write the first key and open a new object

			json_builder.WriteString(fmt.Sprintf("\"%s\":{", line1[0]))

			// look ahead and write the next lines until we find the closing brace
			open_braces := 1
			closed_braces := 0
			for j := i + 1; j < len(lines); j++ {
				// if we find an open brace
				if strings.HasSuffix(lines[j], "{") {
					key := strings.Split(lines[j], ":")[0]
					// write the open brace and key
					key = strings.ReplaceAll(key, "'", "\"")
					json_builder.WriteString(key + ":{")
					// increment the open brace counter
					open_braces++
					j_log.Debug().
						Int("open_braces", open_braces).
						Int("closed_braces", closed_braces).
						Msg("Open brace detected")
					continue
				}

				// if we find the closing brace
				if strings.HasSuffix(lines[j], "}") {
					// increment the closed brace counter
					closed_braces++
					// if the open and closed brace counters are equal, we have reached the end of the JSON variable
					if open_braces == closed_braces {
						// write the closing brace
						json_builder.WriteString("},")
						j_log.Info().
							Int("open_braces", open_braces).
							Int("closed_braces", closed_braces).
							Msg("Last Closed brace; JSON var parsed")
						// break out of the loop
						break
					} else {
						// write the closing brace
						json_builder.WriteString("}")
						j_log.Debug().
							Int("open_braces", open_braces).
							Int("closed_braces", closed_braces).
							Msg("Closed brace detected")
						continue
					}
				}
				// otherwise, write the line
				line := strings.TrimSpace(lines[j])
				line = strings.ReplaceAll(line, "'", "\"")
				json_builder.WriteString(line)
				j_log.Debug().Str("line", line).Msg("Line written")
			}
			// increment the object counter
			object_count++
		}
	}

	// write the closing brace
	json_builder.WriteString("}")

	out_string := json_builder.String()
	// trim the last comma
	out_string = strings.TrimSuffix(out_string, ",}")
	out_string += "}"

	log.Trace().Str("json", out_string).Msg("JSON output")

	// unmarshal the JSON into a map
	err = json.Unmarshal([]byte(out_string), &output)
	if err != nil {
		return nil, errors.Annotate(err, "unmarsal python json")
	}

	return output, nil
}
