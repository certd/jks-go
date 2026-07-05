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

	cleanArgs := []string{os.Args[0]}
	importKeystoreFromPos := false
	for _, arg := range os.Args[1:] {
		if !importKeystoreFromPos && !strings.HasPrefix(arg, "-") && arg == "import" {
			importKeystoreFromPos = true
			continue
		}
		cleanArgs = append(cleanArgs, arg)
	}
	os.Args = cleanArgs

	var (
		importKeystore bool
		srcKeystore    string
		srcStoreType   string
		srcStorePass   string
		srcKeyPass     string
		srcKeyFile     string
		destKeystore   string
		destStoreType  string
		destStorePass  string
		destKeyPass    string
		alias          string
		noprompt       bool
	)

	flag.BoolVar(&importKeystore, "importkeystore", importKeystoreFromPos, "Import one or all entries from another keystore")
	flag.StringVar(&srcKeystore, "srckeystore", "", "Source keystore file path")
	flag.StringVar(&srcKeystore, "src", "", "Alias for -srckeystore")
	flag.StringVar(&srcStoreType, "srcstoretype", "", "Source keystore type (PKCS12 or PEM)")
	flag.StringVar(&srcStorePass, "srcstorepass", "", "Source keystore password")
	flag.StringVar(&srcKeyPass, "srckeypass", "", "Source key password (defaults to source keystore password)")
	flag.StringVar(&srcKeyFile, "srckeyfile", "", "Source private key PEM file path (only for PEM with separate key file)")
	flag.StringVar(&destKeystore, "destkeystore", "", "Destination keystore file path")
	flag.StringVar(&destKeystore, "dest", "", "Alias for -destkeystore")
	flag.StringVar(&destStoreType, "deststoretype", "JKS", "Destination keystore type")
	flag.StringVar(&destStorePass, "deststorepass", "", "Destination keystore password")
	flag.StringVar(&destKeyPass, "destkeypass", "", "Destination key password (defaults to destination keystore password)")
	flag.StringVar(&alias, "alias", "", "Alias for the entry")
	flag.BoolVar(&noprompt, "noprompt", false, "Do not prompt for confirmation")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "jks-go %s\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage: jks-go -importkeystore [options]\n")
		fmt.Fprintf(os.Stderr, "   or: jks-go import [options]\n\n")
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

	if !importKeystore {
		flag.Usage()
		os.Exit(2)
	}

	if srcKeystore == "" {
		fmt.Fprintln(os.Stderr, "Error: -srckeystore is required")
		flag.Usage()
		os.Exit(2)
	}

	if srcStoreType == "" {
		fmt.Fprintln(os.Stderr, "Error: -srcstoretype is required (PKCS12 or PEM)")
		flag.Usage()
		os.Exit(2)
	}

	if destKeystore == "" {
		fmt.Fprintln(os.Stderr, "Error: -destkeystore is required")
		flag.Usage()
		os.Exit(2)
	}

	if destStorePass == "" {
		fmt.Fprintln(os.Stderr, "Error: -deststorepass is required")
		flag.Usage()
		os.Exit(2)
	}

	srcPass := srcStorePass
	keyPass := srcKeyPass
	if keyPass == "" {
		keyPass = srcPass
	}

	dstPass := destStorePass
	dstKey := destKeyPass
	if dstKey == "" {
		dstKey = dstPass
	}

	srcType := strings.ToUpper(srcStoreType)
	switch srcType {
	case "PKCS12":
		if srcPass == "" {
			fmt.Fprintln(os.Stderr, "Error: -srcstorepass is required for PKCS12")
			os.Exit(2)
		}
		_ = keyPass
		if err := convertPKCS12ToJKS(srcKeystore, srcPass, destKeystore, dstPass, alias); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "PEM":
		if err := convertPEMToJKS(srcKeystore, srcKeyFile, keyPass, destKeystore, dstPass, alias); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported srcstoretype '%s', must be PKCS12 or PEM\n", srcStoreType)
		os.Exit(2)
	}

	if !noprompt {
		fmt.Fprintf(os.Stderr, "Import successful: %s -> %s\n", srcKeystore, destKeystore)
	}

	_ = destStoreType
	_ = dstKey
}
