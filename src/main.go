package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	importKeystore := flag.Bool("importkeystore", false, "Import one or all entries from another keystore")
	srcKeystore := flag.String("srckeystore", "", "Source keystore file path")
	srcStoreType := flag.String("srcstoretype", "", "Source keystore type (PKCS12 or PEM)")
	srcStorePass := flag.String("srcstorepass", "", "Source keystore password")
	srcKeyPass := flag.String("srckeypass", "", "Source key password (defaults to source keystore password)")
	srcKeyFile := flag.String("srckeyfile", "", "Source private key PEM file path (only for PEM with separate key file)")
	destKeystore := flag.String("destkeystore", "", "Destination keystore file path")
	destStoreType := flag.String("deststoretype", "JKS", "Destination keystore type")
	destStorePass := flag.String("deststorepass", "", "Destination keystore password")
	destKeyPass := flag.String("destkeypass", "", "Destination key password (defaults to destination keystore password)")
	alias := flag.String("alias", "", "Alias for the entry")
	noprompt := flag.Bool("noprompt", false, "Do not prompt for confirmation")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "jks-go %s\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage: jks-go -importkeystore [options]\n\n")
		fmt.Fprintf(os.Stderr, "A drop-in replacement for keytool -importkeystore.\n")
		fmt.Fprintf(os.Stderr, "Converts PKCS12 (.p12/.pfx) or PEM certificates to JKS format.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("jks-go %s\n", Version)
		fmt.Printf("  commit:    %s\n", Commit)
		fmt.Printf("  built:     %s\n", BuildDate)
		return
	}

	if !*importKeystore {
		flag.Usage()
		os.Exit(2)
	}

	if *srcKeystore == "" {
		fmt.Fprintln(os.Stderr, "Error: -srckeystore is required")
		flag.Usage()
		os.Exit(2)
	}

	if *srcStoreType == "" {
		fmt.Fprintln(os.Stderr, "Error: -srcstoretype is required (PKCS12 or PEM)")
		flag.Usage()
		os.Exit(2)
	}

	if *destKeystore == "" {
		fmt.Fprintln(os.Stderr, "Error: -destkeystore is required")
		flag.Usage()
		os.Exit(2)
	}

	if *destStorePass == "" {
		fmt.Fprintln(os.Stderr, "Error: -deststorepass is required")
		flag.Usage()
		os.Exit(2)
	}

	srcPass := *srcStorePass
	keyPass := *srcKeyPass
	if keyPass == "" {
		keyPass = srcPass
	}

	dstPass := *destStorePass
	dstKey := *destKeyPass
	if dstKey == "" {
		dstKey = dstPass
	}

	srcType := strings.ToUpper(*srcStoreType)
	switch srcType {
	case "PKCS12":
		if srcPass == "" {
			fmt.Fprintln(os.Stderr, "Error: -srcstorepass is required for PKCS12")
			os.Exit(2)
		}
		_ = keyPass
		if err := convertPKCS12ToJKS(*srcKeystore, srcPass, *destKeystore, dstPass, *alias); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "PEM":
		if err := convertPEMToJKS(*srcKeystore, *srcKeyFile, keyPass, *destKeystore, dstPass, *alias); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported srcstoretype '%s', must be PKCS12 or PEM\n", *srcStoreType)
		os.Exit(2)
	}

	if !*noprompt {
		fmt.Fprintf(os.Stderr, "Import successful: %s -> %s\n", *srcKeystore, *destKeystore)
	}

	_ = destStoreType
	_ = dstKey
}
