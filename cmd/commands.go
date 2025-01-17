package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"imagetag/internal/tagging"
	"imagetag/internal/web"
	"log"
	"net/http"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		inputPath := os.Getenv("IMAGETAG_INPUT")
		if inputPath == "" {
			log.Panicln("IMAGETAG_INPUT environment variable not set")
		}
		outputPath := os.Getenv("IMAGETAG_OUTPUT")
		if outputPath == "" {
			log.Panicln("IMAGETAG_OUTPUT environment variable not set")
		}
		i := tagging.BuildAndStart(inputPath, outputPath)
		r := web.BuildRouter(i)
		err := http.ListenAndServe(":8080", r)
		if err != nil {
			log.Panicln(err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(os.Stderr, err)
		os.Exit(1)
	}
}
